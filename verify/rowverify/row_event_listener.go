package rowverify

import (
	"fmt"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/cockroachdb/molt/verify/verifymetrics"
)

type RowEventListener interface {
	OnExtraneousRow(row inconsistency.ExtraneousRow)
	OnMissingRow(row inconsistency.MissingRow)
	OnMismatchingRow(row inconsistency.MismatchingRow)
	OnColumnMismatchNoOtherIssues(row inconsistency.MismatchingColumn, reportLog bool)
	OnMatch()
	OnRowScan()
}

// defaultRowEventListener is the default invocation of the row event listener.
type defaultRowEventListener struct {
	reporter inconsistency.Reporter
	stats    inconsistency.RowStats
	table    TableShard
}

func (n *defaultRowEventListener) OnExtraneousRow(row inconsistency.ExtraneousRow) {
	n.reporter.Report(row)
	n.stats.NumExtraneous++
	verifymetrics.RowStatus.WithLabelValues("extraneous").Inc()
}

func (n *defaultRowEventListener) OnMissingRow(row inconsistency.MissingRow) {
	n.stats.NumMissing++
	n.reporter.Report(row)
	verifymetrics.RowStatus.WithLabelValues("missing").Inc()
}

func (n *defaultRowEventListener) OnMismatchingRow(row inconsistency.MismatchingRow) {
	n.reporter.Report(row)
	n.stats.NumMismatch++
	verifymetrics.RowStatus.WithLabelValues("mismatching").Inc()
}

func (n *defaultRowEventListener) OnMatch() {
	n.stats.NumSuccess++
	verifymetrics.RowStatus.WithLabelValues("success").Inc()
}

func (n *defaultRowEventListener) OnColumnMismatchNoOtherIssues(
	row inconsistency.MismatchingColumn, reportLog bool,
) {
	// This logic happens at most once per shard per table
	// so we don't double count mismatching columns and reporting for mismatching columns.
	if reportLog {
		n.reporter.Report(row)
		numMismatchingCols := len(row.MismatchingColumns)
		verifymetrics.RowStatus.WithLabelValues("mismatching_column").Add(float64(numMismatchingCols))
		n.stats.NumColumnMismatch += numMismatchingCols
	}
	n.stats.NumConditionalSuccess++
	verifymetrics.RowStatus.WithLabelValues("conditional_success").Inc()
}

func (n *defaultRowEventListener) OnRowScan() {
	if n.stats.NumVerified%10000 == 0 && n.stats.NumVerified > 0 {
		n.reporter.Report(inconsistency.SummaryReport{
			Info:  fmt.Sprintf("progress on %s.%s (shard %d/%d)", n.table.Schema, n.table.Table, n.table.ShardNum, n.table.TotalShards),
			Stats: n.stats,
		})
	}
	verifymetrics.RowsRead.Inc()
	n.stats.NumVerified++
}

// liveRowEventListener is used when `live` mode is enabled.
type liveRowEventListener struct {
	base *defaultRowEventListener
	pks  []tree.Datums
	r    *liveReverifier

	settings  LiveReverificationSettings
	lastFlush time.Time
}

func (n *liveRowEventListener) OnExtraneousRow(row inconsistency.ExtraneousRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.NumLiveRetry++
}

func (n *liveRowEventListener) OnMissingRow(row inconsistency.MissingRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.NumLiveRetry++
}

func (n *liveRowEventListener) OnMismatchingRow(row inconsistency.MismatchingRow) {
	n.pks = append(n.pks, row.PrimaryKeyValues)
	n.base.stats.NumLiveRetry++
}

func (n *liveRowEventListener) OnMatch() {
	n.base.OnMatch()
}

func (n *liveRowEventListener) OnColumnMismatchNoOtherIssues(
	row inconsistency.MismatchingColumn, reportLog bool,
) {
	n.base.OnColumnMismatchNoOtherIssues(row, reportLog)
}

func (n *liveRowEventListener) OnRowScan() {
	n.base.OnRowScan()
	if time.Since(n.lastFlush) > n.settings.FlushInterval || len(n.pks) >= n.settings.MaxBatchSize {
		n.Flush()
	}
}

func (n *liveRowEventListener) Flush() {
	n.lastFlush = time.Now()
	if len(n.pks) > 0 {
		r, err := retry.NewRetry(n.settings.RetrySettings)
		if err != nil {
			panic(err)
		}
		n.r.Push(&liveRetryItem{
			PrimaryKeys: n.pks,
			Retry:       r,
		})
		n.pks = nil
	}
}
