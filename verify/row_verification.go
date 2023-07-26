package verify

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/verify/internal/rowiterator"
	"github.com/cockroachdb/molt/verify/verifybase"
	"github.com/rs/zerolog"
)

type rowStats struct {
	numVerified   int
	numSuccess    int
	numMissing    int
	numMismatch   int
	numExtraneous int
	numLiveRetry  int
}

func (s *rowStats) String() string {
	return fmt.Sprintf(
		"truth rows seen: %d, success: %d, missing: %d, mismatch: %d, extraneous: %d, live_retry: %d",
		s.numVerified,
		s.numSuccess,
		s.numMissing,
		s.numMismatch,
		s.numExtraneous,
		s.numLiveRetry,
	)
}

// compareContext implements tree.CompareContext
type compareContext struct{}

func (c *compareContext) UnwrapDatum(d tree.Datum) tree.Datum {
	return d
}

func (c *compareContext) GetLocation() *time.Location {
	return time.UTC
}

func (c *compareContext) GetRelativeParseTime() time.Time {
	return time.Now().UTC()
}

func (c *compareContext) MustGetPlaceholderValue(p *tree.Placeholder) tree.Datum {
	return p
}

type TableShard struct {
	verifybase.VerifiableTable

	StartPKVals []tree.Datum
	EndPKVals   []tree.Datum

	ShardNum    int
	TotalShards int
}

func verifyRowsOnShard(
	ctx context.Context,
	conns dbconn.OrderedConns,
	table TableShard,
	rowBatchSize int,
	reporter Reporter,
	logger zerolog.Logger,
	live bool,
) error {
	var iterators [2]rowiterator.Iterator
	for i, conn := range conns {
		var err error
		iterators[i], err = rowiterator.NewScanIterator(
			ctx,
			conn,
			rowiterator.Table{
				TableName:         table.TableName,
				ColumnNames:       table.Columns,
				ColumnOIDs:        table.ColumnOIDs[i],
				PrimaryKeyColumns: table.PrimaryKeyColumns,
				StartPKVals:       table.StartPKVals,
				EndPKVals:         table.EndPKVals,
			},
			rowBatchSize,
		)
		if err != nil {
			return errors.Wrapf(err, "error initializing row iterator on %s", conn.ID())
		}
	}

	var evl RowEventListener = &nonLiveRowEventListener{reporter: reporter, table: table}
	var reverifier *Reverifier
	if live {
		var err error
		reverifier, err = NewReverifier(ctx, logger, conns, table, evl)
		if err != nil {
			return err
		}
		evl = &liveRowEventListener{base: evl.(*nonLiveRowEventListener), r: reverifier}
	}
	if err := verifyRows(ctx, iterators, table, evl); err != nil {
		return err
	}
	switch evl := evl.(type) {
	case *nonLiveRowEventListener:
		reporter.Report(StatusReport{
			Info: fmt.Sprintf("finished row verification on %s.%s (shard %d/%d): %s", table.Schema, table.Table, table.ShardNum, table.TotalShards, evl.stats.String()),
		})
	case *liveRowEventListener:
		logger.Trace().Msgf("flushing remaining reverifier objects")
		evl.Flush()
		reverifier.ScanComplete()
		logger.Trace().Msgf("waiting for reverifier to complete")
		reverifier.WaitForDone()
		reporter.Report(StatusReport{
			Info: fmt.Sprintf("finished LIVE row verification on %s.%s (shard %d/%d): %s", table.Schema, table.Table, table.ShardNum, table.TotalShards, evl.base.stats.String()),
		})
	}
	return nil
}

func verifyRows(
	ctx context.Context, iterators [2]rowiterator.Iterator, table TableShard, evl RowEventListener,
) error {
	cmpCtx := &compareContext{}

	truth := iterators[0]
	for truth.HasNext(ctx) {
		evl.OnRowScan()

		truthVals := truth.Next(ctx)
		it := iterators[1]

	itLoop:
		for {
			if !it.HasNext(ctx) {
				if err := it.Error(); err == nil {
					evl.OnMissingRow(MissingRow{
						ConnID:            it.Conn().ID(),
						Schema:            table.Schema,
						Table:             table.Table,
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
				if compareVal = truthVals[i].Compare(cmpCtx, targetVals[i]); compareVal != 0 {
					break
				}
			}
			switch compareVal {
			case 1:
				// Extraneous row. Log and continue.
				it.Next(ctx)
				evl.OnExtraneousRow(ExtraneousRow{
					ConnID:            it.Conn().ID(),
					Schema:            table.Schema,
					Table:             table.Table,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				})
			case 0:
				// Matching primary key. Compare values and break loop.
				targetVals = it.Next(ctx)
				mismatches := MismatchingRow{
					ConnID:            it.Conn().ID(),
					Schema:            table.Schema,
					Table:             table.Table,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				}
				for valIdx := len(table.PrimaryKeyColumns); valIdx < len(targetVals); valIdx++ {
					if targetVals[valIdx].Compare(cmpCtx, truthVals[valIdx]) != 0 {
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
				evl.OnMissingRow(MissingRow{
					ConnID:            it.Conn().ID(),
					Schema:            table.Schema,
					Table:             table.Table,
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
				evl.OnExtraneousRow(ExtraneousRow{
					ConnID:            it.Conn().ID(),
					Schema:            table.Schema,
					Table:             table.Table,
					PrimaryKeyColumns: table.PrimaryKeyColumns,
					PrimaryKeyValues:  targetVals[:len(table.PrimaryKeyColumns)],
				})
			}
		}
	}

	return nil
}

type RowEventListener interface {
	OnExtraneousRow(row ExtraneousRow)
	OnMissingRow(row MissingRow)
	OnMismatchingRow(row MismatchingRow)
	OnMatch()
	OnRowScan()
}

type nonLiveRowEventListener struct {
	reporter Reporter
	stats    rowStats
	table    TableShard
}

func (n *nonLiveRowEventListener) OnExtraneousRow(row ExtraneousRow) {
	n.reporter.Report(row)
	n.stats.numExtraneous++
}

func (n *nonLiveRowEventListener) OnMissingRow(row MissingRow) {
	n.stats.numMissing++
	n.reporter.Report(row)
}

func (n *nonLiveRowEventListener) OnMismatchingRow(row MismatchingRow) {
	n.reporter.Report(row)
	n.stats.numMismatch++
}

func (n *nonLiveRowEventListener) OnMatch() {
	n.stats.numSuccess++
}

func (n *nonLiveRowEventListener) OnRowScan() {
	if n.stats.numVerified%10000 == 0 && n.stats.numVerified > 0 {
		n.reporter.Report(StatusReport{
			Info: fmt.Sprintf("progress on %s.%s (shard %d/%d): %s", n.table.Schema, n.table.Table, n.table.ShardNum, n.table.TotalShards, n.stats.String()),
		})
	}
	n.stats.numVerified++
}

type liveRowEventListener struct {
	base *nonLiveRowEventListener
	pks  []tree.Datums
	r    *Reverifier
}

func (n *liveRowEventListener) OnExtraneousRow(row ExtraneousRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.numLiveRetry++
}

func (n *liveRowEventListener) OnMissingRow(row MissingRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.numLiveRetry++
}

func (n *liveRowEventListener) OnMismatchingRow(row MismatchingRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.numLiveRetry++
}

func (n *liveRowEventListener) OnMatch() {
	n.base.OnMatch()
}

func (n *liveRowEventListener) OnRowScan() {
	n.base.OnRowScan()
	if n.base.stats.numVerified%10000 == 0 {
		n.Flush()
	}
}

func (n *liveRowEventListener) Flush() {
	if len(n.pks) > 0 {
		r, err := retry.NewRetry(retry.Settings{
			InitialBackoff: time.Millisecond * 200,
			Multiplier:     4,
			MaxBackoff:     3 * time.Second,
			MaxRetries:     6,
		})
		if err != nil {
			panic(err)
		}
		n.r.Push(&RetryItem{
			PrimaryKeys: n.pks,
			// TODO: configurable.
			Retry: r,
		})
		n.pks = nil
	}
}