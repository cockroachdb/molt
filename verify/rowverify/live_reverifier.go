package rowverify

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/rowiterator"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

type LiveReverificationSettings struct {
	MaxBatchSize  int
	FlushInterval time.Duration
	RetrySettings retry.Settings
	RunsPerSecond int
}

func (s LiveReverificationSettings) rateLimit() rate.Limit {
	if s.RunsPerSecond == 0 {
		return rate.Inf
	}
	return rate.Every(time.Duration(float64(time.Second) / float64(s.RunsPerSecond)))
}

type liveRetryItem struct {
	PrimaryKeys []tree.Datums
	Retry       *retry.Retry
}

type liveReverifier struct {
	insertQueue  chan *liveRetryItem
	scanComplete chan struct{}
	done         chan struct{}
	logger       zerolog.Logger
	table        TableShard
	limiter      *rate.Limiter
	testingKnobs struct {
		beforeScan func(*retry.Retry, []tree.Datums)
		blockScan  atomic.Bool
	}
}

var (
	liveReverifiedRows = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "molt",
		Subsystem: "verify",
		Name:      "live_reverified_rows",
		Help:      "Number of rows that require reverification by the live reverifier.",
	})
	liveReverifyRemainingPKs = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "molt",
		Subsystem: "verify",
		Name:      "live_queued_pks",
		Help:      "Number of rows that are queued by the live reverifier.",
	})
	liveReverifyRemainingBatches = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "molt",
		Subsystem: "verify",
		Name:      "live_queued_batches",
		Help:      "Number of batches of rows that require the live reverifier.",
	})
)

func newLiveReverifier(
	ctx context.Context,
	logger zerolog.Logger,
	baseConns dbconn.OrderedConns,
	table TableShard,
	baseListener RowEventListener,
	rateLimiter *rate.Limiter,
) (*liveReverifier, error) {
	r := &liveReverifier{
		insertQueue:  make(chan *liveRetryItem),
		logger:       logger,
		table:        table,
		done:         make(chan struct{}),
		scanComplete: make(chan struct{}),
		limiter:      rateLimiter,
	}

	var conns dbconn.OrderedConns
	for i, conn := range baseConns {
		var err error
		conns[i], err = conn.Clone(ctx)
		if err != nil {
			return nil, err
		}
		// Initialize the type map on the connection.
		for _, typOID := range table.ColumnOIDs[i] {
			if _, err := dbconn.GetDataType(ctx, conn, typOID); err != nil {
				return nil, errors.Wrapf(err, "error initializing type oid %d", typOID)
			}
		}
	}

	go func() {
		defer func() {
			logger.Debug().Msgf("live reverifier exiting")
			for _, conn := range conns {
				if err := conn.Close(ctx); err != nil {
					logger.Err(err).Msgf("error closing live reverifier connection")
				}
			}
			close(r.done)
		}()
		var queue liveRetryQueue
		noTrafficChannel := make(chan time.Time)
		var done bool
		for !done || len(queue.items) > 0 {
			if done {
				logger.Info().
					Int("num_batches", len(queue.items)).
					Int("num_pks", queue.numPKs).
					Msgf("waiting for live reverifier to complete")
			}
			liveReverifyRemainingPKs.Set(float64(queue.numPKs))
			liveReverifyRemainingBatches.Set(float64(len(queue.items)))

			// By default, block until we get work.
			var nextWorkCh <-chan time.Time = noTrafficChannel
			if len(queue.items) > 0 {
				// If we have work queued up, wait for it.
				nextWorkCh = time.After(time.Until(queue.items[0].Retry.NextRetry))
			}
			select {
			case <-r.scanComplete:
				done = true
				logger.Debug().Msgf("live reverifier marked scan as complete")
			case it, ok := <-r.insertQueue:
				// If the queue is closed, we can exit.
				if !ok {
					return
				}
				liveReverifiedRows.Add(float64(len(it.PrimaryKeys)))
				queue.heapPush(it)
			case <-nextWorkCh:
				if r.testingKnobs.blockScan.Load() {
					continue
				}
				it := queue.heapPop()
				if it == nil {
					continue
				}
				if err := r.limiter.Wait(ctx); err != nil {
					logger.Err(err).Msgf("error applying live reverifier rate limit")
				}
				if r.testingKnobs.beforeScan != nil {
					r.testingKnobs.beforeScan(it.Retry, it.PrimaryKeys)
				}
				logger.Trace().
					Int("iteration", it.Retry.Iteration).
					Time("start_time", it.Retry.StartTime).
					Time("next_retry", it.Retry.NextRetry).
					Int("num_failed_keys", len(it.PrimaryKeys)).
					Msgf("live reverifying primary keys")
				var iterators [2]rowiterator.Iterator
				for i, conn := range conns {
					iterators[i] = rowiterator.NewPointLookupIterator(
						conn,
						rowiterator.Table{
							Name:              table.Name,
							ColumnNames:       table.Columns,
							ColumnOIDs:        table.ColumnOIDs[i],
							PrimaryKeyColumns: table.PrimaryKeyColumns,
						},
						it.PrimaryKeys,
					)
				}
				it.PrimaryKeys = it.PrimaryKeys[:0]
				if err := verifyRows(ctx, iterators, table, &reverifyEventListener{RetryItem: it, BaseListener: baseListener}); err != nil {
					logger.Err(err).Msgf("error during live verification")
					continue
				}
				if len(it.PrimaryKeys) > 0 {
					it.Retry.Next()
					queue.heapPush(it)
					logger.Trace().
						Int("iteration", it.Retry.Iteration-1).
						Time("start_time", it.Retry.StartTime).
						Time("next_retry", it.Retry.NextRetry).
						Int("num_failed_keys", len(it.PrimaryKeys)).
						Msgf("not everything reverified successfully")
				}
			}
		}
	}()
	return r, nil
}

func (r *liveReverifier) Push(it *liveRetryItem) {
	r.insertQueue <- it
}

func (r *liveReverifier) ScanComplete() {
	r.scanComplete <- struct{}{}
}

func (r *liveReverifier) WaitForDone() {
	<-r.done
}

type reverifyEventListener struct {
	RetryItem    *liveRetryItem
	BaseListener RowEventListener
}

func (r *reverifyEventListener) OnExtraneousRow(row inconsistency.ExtraneousRow) {
	if !r.RetryItem.Retry.ShouldContinue() {
		r.BaseListener.OnExtraneousRow(row)
	} else {
		r.RetryItem.PrimaryKeys = append(r.RetryItem.PrimaryKeys, row.PrimaryKeyValues)
	}
}

func (r *reverifyEventListener) OnMissingRow(row inconsistency.MissingRow) {
	if !r.RetryItem.Retry.ShouldContinue() {
		r.BaseListener.OnMissingRow(row)
	} else {
		r.RetryItem.PrimaryKeys = append(r.RetryItem.PrimaryKeys, row.PrimaryKeyValues)
	}
}

func (r *reverifyEventListener) OnMismatchingRow(row inconsistency.MismatchingRow) {
	if !r.RetryItem.Retry.ShouldContinue() {
		r.BaseListener.OnMismatchingRow(row)
	} else {
		r.RetryItem.PrimaryKeys = append(r.RetryItem.PrimaryKeys, row.PrimaryKeyValues)
	}
}

func (r *reverifyEventListener) OnMatch() {
	r.BaseListener.OnMatch()
}

func (r *reverifyEventListener) OnColumnMismatchNoOtherIssues(
	row inconsistency.MismatchingColumn, reportLog bool,
) {
	r.BaseListener.OnColumnMismatchNoOtherIssues(row, reportLog)
}

func (r *reverifyEventListener) OnRowScan() {}
