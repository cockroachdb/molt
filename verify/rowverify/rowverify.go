package rowverify

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
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
				targetVals = it.Next(ctx)
				mismatches := inconsistency.MismatchingRow{
					Name:              table.Name,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				}
				for valIdx := len(table.PrimaryKeyColumns); valIdx < len(targetVals); valIdx++ {
					if targetVals[valIdx].Compare(comparectx.CompareContext, truthVals[valIdx]) != 0 {
						mismatches.MismatchingColumns = append(mismatches.MismatchingColumns, table.Columns[valIdx])
						mismatches.TargetVals = append(mismatches.TargetVals, targetVals[valIdx])
						mismatches.TruthVals = append(mismatches.TruthVals, truthVals[valIdx])
					}
				}
				if len(mismatches.MismatchingColumns) > 0 {
					evl.OnMismatchingRow(mismatches)
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
