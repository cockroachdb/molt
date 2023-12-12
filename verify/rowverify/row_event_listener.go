package rowverify

import (
	"fmt"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type RowEventListener interface {
	OnExtraneousRow(row inconsistency.ExtraneousRow)
	OnMissingRow(row inconsistency.MissingRow)
	OnMismatchingRow(row inconsistency.MismatchingRow)
	OnMatch()
	OnRowScan()
}

var (
	rowStatusMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "molt",
		Subsystem: "verify",
		Name:      "row_verification_status",
		Help:      "Status of rows that have been verified.",
	}, []string{"status"})
	rowsReadMetric = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "molt",
		Subsystem: "verify",
		Name:      "rows_read",
		Help:      "Rate of rows that are being read from source database.",
	})
)

func init() {
	// Initialise each metric by default.
	for _, s := range []string{"extraneous", "missing", "mismatching", "success"} {
		rowStatusMetric.WithLabelValues(s)
	}
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
	rowStatusMetric.WithLabelValues("extraneous").Inc()
}

func (n *defaultRowEventListener) OnMissingRow(row inconsistency.MissingRow) {
	n.stats.NumMissing++
	n.reporter.Report(row)
	rowStatusMetric.WithLabelValues("missing").Inc()
}

func (n *defaultRowEventListener) OnMismatchingRow(row inconsistency.MismatchingRow) {
	n.reporter.Report(row)
	n.stats.NumMismatch++
	rowStatusMetric.WithLabelValues("mismatching").Inc()
}

func (n *defaultRowEventListener) OnMatch() {
	n.stats.NumSuccess++
	rowStatusMetric.WithLabelValues("success").Inc()
}

func (n *defaultRowEventListener) OnRowScan() {
	if n.stats.NumVerified%10000 == 0 && n.stats.NumVerified > 0 {
		n.reporter.Report(inconsistency.SummaryReport{
			Info:  fmt.Sprintf("progress on %s.%s (shard %d/%d)", n.table.Schema, n.table.Table, n.table.ShardNum, n.table.TotalShards),
			Stats: n.stats,
		})
	}
	rowsReadMetric.Inc()
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
