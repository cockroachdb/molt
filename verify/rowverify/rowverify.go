package rowverify

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/comparectx"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/rowiterator"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

type TableShard struct {
	dbtable.VerifiedTable

	StartPKVals []tree.Datum
	EndPKVals   []tree.Datum

	ShardNum    int
	TotalShards int
}

func (s *TableShard) String() string {
	return fmt.Sprintf("shard [%d/%d] for table %q, pk range %q -> %q", s.ShardNum, s.TotalShards, s.VerifiedTable, s.StartPKVals, s.EndPKVals)
}

func VerifyRowsOnShard(
	ctx context.Context,
	conns dbconn.OrderedConns,
	table TableShard,
	rowBatchSize int,
	reporter inconsistency.Reporter,
	logger zerolog.Logger,
	liveReverifySettings *LiveReverificationSettings,
	rateLimiter *rate.Limiter,
) error {
	var iterators [2]rowiterator.Iterator
	for i, conn := range conns {
		var err error
		iterators[i], err = rowiterator.NewScanIterator(
			ctx,
			conn,
			rowiterator.ScanTable{
				Table: rowiterator.Table{
					Name:              table.Name,
					ColumnNames:       table.Columns,
					ColumnOIDs:        table.ColumnOIDs[i],
					PrimaryKeyColumns: table.PrimaryKeyColumns,
				},
				StartPKVals: table.StartPKVals,
				EndPKVals:   table.EndPKVals,
			},
			rowBatchSize,
			rateLimiter,
		)
		if err != nil {
			return errors.Wrapf(err, "error initializing row iterator on %s", conn.ID())
		}
	}

	defaultRowEVL := &defaultRowEventListener{reporter: reporter, table: table,
		stats: inconsistency.RowStats{
			Schema: table.Schema.String(),
			Table:  table.Table.String(),
		}}
	var rowEVL RowEventListener = defaultRowEVL
	var liveReverifier *liveReverifier
	if liveReverifySettings != nil {
		var err error
		liveReverifier, err = newLiveReverifier(ctx, logger, conns, table, rowEVL, rate.NewLimiter(liveReverifySettings.rateLimit(), 1))
		if err != nil {
			return err
		}
		rowEVL = &liveRowEventListener{
			base:      defaultRowEVL,
			r:         liveReverifier,
			settings:  *liveReverifySettings,
			lastFlush: time.Now(),
		}
	}
	if err := verifyRows(ctx, iterators, table, rowEVL); err != nil {
		return err
	}
	switch rowEVL := rowEVL.(type) {
	case *defaultRowEventListener:
		reporter.Report(inconsistency.SummaryReport{
			Info:  fmt.Sprintf("finished row verification on %s.%s (shard %d/%d)", table.Schema, table.Table, table.ShardNum, table.TotalShards),
			Stats: rowEVL.stats,
		})
	case *liveRowEventListener:
		logger.Trace().Msgf("flushing remaining live reverifier objects")
		rowEVL.Flush()
		liveReverifier.ScanComplete()
		logger.Trace().Msgf("waiting for live reverifier to complete")
		liveReverifier.WaitForDone()
		reporter.Report(inconsistency.SummaryReport{
			Info:  fmt.Sprintf("finished LIVE row verification on %s.%s (shard %d/%d)", table.Schema, table.Table, table.ShardNum, table.TotalShards),
			Stats: rowEVL.base.stats,
		})
	default:
		return errors.Newf("unknown row event listener: %T", rowEVL)
	}
	return nil
}

func verifyRows(
	ctx context.Context, iterators [2]rowiterator.Iterator, table TableShard, evl RowEventListener,
) error {
	truth := iterators[0]
	colIdxToSeen := map[int]bool{}

	for truth.HasNext(ctx) {
		evl.OnRowScan()

		truthVals := truth.Next(ctx)
		it := iterators[1]

	itLoop:
		for {
			if !it.HasNext(ctx) {
				if err := it.Error(); err == nil {
					evl.OnMissingRow(inconsistency.MissingRow{
						Name:              table.Name,
						PrimaryKeyColumns: table.PrimaryKeyColumns,
						PrimaryKeyValues:  truthVals[:len(table.PrimaryKeyColumns)],
						Columns:           table.Columns,
						Values:            truthVals,
					})
				}
				break
			}

			// Check the primary key.
			targetVals := it.Peek(ctx)
			var compareVal int
			for i := range table.PrimaryKeyColumns {
				if compareVal = truthVals[i].Compare(comparectx.CompareContext, targetVals[i]); compareVal != 0 {
					break
				}
			}
			switch compareVal {
			case 1:
				// Extraneous row. Log and continue.
				it.Next(ctx)
				evl.OnExtraneousRow(inconsistency.ExtraneousRow{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				})
			case 0:
				// Matching primary key. Compare values and break loop.
				reportLog := true
				targetVals = it.Next(ctx)
				mismatches := inconsistency.MismatchingRow{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				}
				mismatchColumns := inconsistency.MismatchingColumn{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				}

				// Currently, CRDB or PG doesn't support comparison between boolean and int, i.e. `SELECT true=1;` will
				// return a comparison error. So if we need to explicitly convert a bool datum to an int to enable its
				// comparison with an int datum.
				boolDatumToIntDatum := func(dbool *tree.DBool) *tree.DInt {
					var res tree.DInt
					if *dbool {
						res = tree.DInt(1)
					}
					return &res
				}

				uuidDatumToStringDatum := func(duuid *tree.DUuid) *tree.DString {
					// We need to use the bare string fmt so that no extra single quotes are added for the string.
					fmtCtx := tree.NewFmtCtx(tree.FmtBareStrings)
					fmtCtx.FormatNode(duuid)
					res := tree.DString(fmtCtx.CloseAndGetString())
					return &res
				}

				for valIdx := len(table.PrimaryKeyColumns); valIdx < len(targetVals); valIdx++ {
					datumsToProcess := [2]tree.Datum{truthVals[valIdx], targetVals[valIdx]}

					for i, d := range datumsToProcess {
						switch d.ResolvedType().Family() {
						case types.BoolFamily:
							if datumsToProcess[1-i].ResolvedType().Family() == types.IntFamily {
								datumsToProcess[i] = boolDatumToIntDatum(d.(*tree.DBool))
								break
							}
						case types.UuidFamily:
							if datumsToProcess[1-i].ResolvedType().Family() == types.StringFamily {
								datumsToProcess[i] = uuidDatumToStringDatum(d.(*tree.DUuid))
								break
							}
						}
					}

					if retVal, err := datumsToProcess[0].CompareError(comparectx.CompareContext, datumsToProcess[1]); err != nil {
						if _, ok := colIdxToSeen[valIdx]; ok {
							reportLog = false
						}

						mismatchColumns.MismatchingColumns = append(mismatchColumns.MismatchingColumns, table.Columns[valIdx])
						mismatchColumns.TargetVals = append(mismatchColumns.TargetVals, targetVals[valIdx])
						mismatchColumns.TruthVals = append(mismatchColumns.TruthVals, truthVals[valIdx])

						msg := fmt.Sprintf("%s (%s)", table.Columns[valIdx].String(), err.Error())
						mismatchColumns.Info = append(mismatchColumns.Info, msg)
						colIdxToSeen[valIdx] = true
					} else if retVal != 0 {
						mismatches.MismatchingColumns = append(mismatches.MismatchingColumns, table.Columns[valIdx])
						mismatches.TargetVals = append(mismatches.TargetVals, targetVals[valIdx])
						mismatches.TruthVals = append(mismatches.TruthVals, truthVals[valIdx])
					}
				}

				if len(mismatches.MismatchingColumns) > 0 {
					evl.OnMismatchingRow(mismatches)
				} else if len(mismatchColumns.MismatchingColumns) > 0 {
					evl.OnColumnMismatchNoOtherIssues(mismatchColumns, reportLog)
				} else {
					evl.OnMatch()
				}
				break itLoop
			case -1:
				evl.OnMissingRow(inconsistency.MissingRow{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  truthVals[:len(table.PrimaryKeyColumns)],
					Columns:           table.Columns,
					Values:            truthVals,
				})
				break itLoop
			}
		}
	}

	for idx, it := range iterators {
		if err := it.Error(); err != nil {
			return err
		}
		// If we still have rows in our iterator, they're all extraneous.
		if idx > 0 {
			for it.HasNext(ctx) {
				targetVals := it.Next(ctx)
				evl.OnExtraneousRow(inconsistency.ExtraneousRow{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				})
			}
		}
	}

	return nil
}
