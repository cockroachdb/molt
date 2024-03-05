package verify

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/molttelemetry"
	"github.com/cockroachdb/molt/utils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/cockroachdb/molt/verify/tableverify"
	"github.com/cockroachdb/molt/verify/verifymetrics"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const DefaultConcurrency = 8
const DefaultRowBatchSize = 1000
const DefaultTableSplits = 8

type VerifyOpt func(*verifyOpts)

type verifyOpts struct {
	concurrency              int
	rowBatchSize             int
	tableSplits              int
	rowsPerSecond            int
	continuous               bool
	continuousPause          time.Duration
	rows                     bool
	dbFilter                 utils.FilterConfig
	liveVerificationSettings *rowverify.LiveReverificationSettings

	testOnly bool
}

func (o verifyOpts) rateLimit() rate.Limit {
	if o.rowsPerSecond == 0 {
		return rate.Inf
	}
	perSecond := float64(o.rowBatchSize) / float64(o.rowsPerSecond)
	return rate.Every(time.Duration(float64(time.Second) * perSecond))
}

func WithConcurrency(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.concurrency = c
	}
}

func WithRowBatchSize(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.rowBatchSize = c
	}
}

func WithRowsPerSecond(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.rowsPerSecond = c
	}
}

func WithTableSplits(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.tableSplits = c
	}
}

func WithContinuous(c bool, pauseLength time.Duration) VerifyOpt {
	return func(o *verifyOpts) {
		o.continuous = c
		o.continuousPause = pauseLength
	}
}

func WithLive(live bool, settings rowverify.LiveReverificationSettings) VerifyOpt {
	return func(o *verifyOpts) {
		if live {
			o.liveVerificationSettings = &settings
		}
	}
}

func WithDBFilter(filter utils.FilterConfig) VerifyOpt {
	return func(o *verifyOpts) {
		o.dbFilter = filter
	}
}

func WithRows(b bool) VerifyOpt {
	return func(o *verifyOpts) {
		o.rows = b
	}
}

func WithTestOnly(b bool) VerifyOpt {
	return func(o *verifyOpts) {
		o.testOnly = b
	}
}

// Verify verifies the given connections have matching tables and contents.
func Verify(
	ctx context.Context,
	conns dbconn.OrderedConns,
	logger zerolog.Logger,
	reporter inconsistency.Reporter,
	inOpts ...VerifyOpt,
) error {
	opts := verifyOpts{
		concurrency:  DefaultConcurrency,
		rowBatchSize: DefaultRowBatchSize,
		tableSplits:  DefaultTableSplits,
		rows:         true,
		dbFilter:     utils.DefaultFilterConfig(),
	}
	for _, applyOpt := range inOpts {
		applyOpt(&opts)
	}

	if err := dbconn.RegisterTelemetry(conns); err != nil {
		return err
	}
	reportTelemetry(logger, opts, conns)

	dbTables, err := dbverify.Verify(ctx, conns)
	if err != nil {
		return errors.Wrap(err, "error comparing database tables")
	}
	if dbTables, err = utils.FilterResult(opts.dbFilter, dbTables); err != nil {
		return err
	}

	for _, missingTable := range dbTables.MissingTables {
		reporter.Report(missingTable)
	}
	for _, extraneousTable := range dbTables.ExtraneousTables {
		reporter.Report(extraneousTable)
	}

	// Grab columns for each table on both sides.
	tbls, err := tableverify.VerifyCommonTables(ctx, conns, logger, dbTables.Verified)
	if err != nil {
		return err
	}

	reportTableCategories(dbTables)

	// Report mismatching table definitions.
	for _, tbl := range tbls {
		for _, d := range tbl.MismatchingTableDefinitions {
			reporter.Report(d)
		}
	}

	if !opts.rows {
		logger.Info().Msgf("skipping row based verification")
		return nil
	}

	shards := make([]rowverify.TableShard, 0, len(tbls))
	for _, tbl := range tbls {
		if !tbl.RowVerifiable {
			logger.Warn().Msgf("skipping unverifiable table %s.%s", tbl.Schema, tbl.Table)
			continue
		}
		// Get and first and last of each PK.
		tableShards, err := ShardTable(ctx, conns[0], tbl, reporter, opts.tableSplits)
		if err != nil {
			return errors.Wrapf(err, "error splitting tables")
		}
		shards = append(shards, tableShards...)
	}

	numGoroutines := opts.concurrency
	if opts.continuous {
		if opts.concurrency != 0 && opts.concurrency < len(shards) {
			logger.Warn().Msgf("--concurrency is set to a higher value %d so that continuous mode can run successfully", len(shards))
		}
		// For continuous mode, we default to running each shard on a continous loop.
		numGoroutines = len(shards)
	} else if numGoroutines == 0 {
		numGoroutines = runtime.NumCPU()
		logger.Debug().Int("concurrency", numGoroutines).
			Msgf("no concurrency set; defaulting to number of CPUs")
	}

	if numGoroutines > runtime.NumCPU()*4 {
		logger.Warn().Msgf(
			"running %d row verifications concurrently, which may thrash your CPU (consider reducing using --concurrency or using less tables)",
			numGoroutines,
		)
	}

	// Compare rows up to the numGoroutines specified.
	g, _ := errgroup.WithContext(ctx)
	workQueue := make(chan rowverify.TableShard)
	for goroutineIdx := 0; goroutineIdx < numGoroutines; goroutineIdx++ {
		g.Go(func() error {
			verifymetrics.NumShards.Inc()
			defer verifymetrics.NumShards.Dec()

			for {
				shard, ok := <-workQueue
				if !ok {
					return nil
				}
				for runNum := 1; opts.continuous || runNum <= 1; runNum++ {
					msg := fmt.Sprintf(
						"starting verify on %s.%s, shard %d/%d",
						shard.Schema,
						shard.Table,
						shard.ShardNum,
						shard.TotalShards,
					)
					if opts.continuous {
						msg += fmt.Sprintf(", run #%d", runNum)
					}
					if shard.TotalShards > 1 {
						msg += ", range: ["
						if len(shard.StartPKVals) > 0 {
							for i, val := range shard.StartPKVals {
								if i > 0 {
									msg += ","
								}
								msg += val.String()
							}
						} else {
							msg += "<beginning>"
						}
						msg += " - "
						if len(shard.EndPKVals) > 0 {
							for i, val := range shard.EndPKVals {
								if i > 0 {
									msg += ", "
								}
								msg += val.String()
							}
							msg += ")"
						} else {
							msg += "<end>]"
						}
					}
					reporter.Report(inconsistency.StatusReport{
						Info: msg,
					})
					if err := verifyRowShard(
						ctx,
						conns,
						reporter,
						logger,
						opts.rowBatchSize,
						shard,
						opts.liveVerificationSettings,
						rate.NewLimiter(opts.rateLimit(), 1),
					); err != nil {
						logger.Err(err).
							Str("schema", string(shard.Schema)).
							Str("table", string(shard.Table)).
							Msgf("error verifying rows")
						reporter.Report(inconsistency.StatusReport{
							Info: "failed to verify",
						})
					}
					time.Sleep(opts.continuousPause)
				}
			}
		})
	}
	for _, shard := range shards {
		workQueue <- shard
	}
	close(workQueue)
	return g.Wait()
}

func verifyRowShard(
	ctx context.Context,
	conns dbconn.OrderedConns,
	reporter inconsistency.Reporter,
	logger zerolog.Logger,
	rowBatchSize int,
	tbl rowverify.TableShard,
	liveVerifySettings *rowverify.LiveReverificationSettings,
	rateLimiter *rate.Limiter,
) error {
	// Copy connections over naming wise, but initialize a new connection
	// for each table.
	var workerConns dbconn.OrderedConns
	for i := range workerConns {
		// Make a copy of i so the worker closes correctly.
		i := i
		var err error
		workerConns[i], err = conns[i].Clone(ctx)
		if err != nil {
			return errors.Wrap(err, "error establishing connection to compare")
		}
		defer func() {
			_ = workerConns[i].Close(ctx)
		}()
	}
	return rowverify.VerifyRowsOnShard(
		ctx,
		workerConns,
		tbl,
		rowBatchSize,
		reporter,
		logger,
		liveVerifySettings,
		rateLimiter,
	)
}

func reportTableCategories(result utils.Result) {
	verifymetrics.NumTablesProcessed.WithLabelValues("missing").Add(float64(len(result.MissingTables)))
	verifymetrics.NumTablesProcessed.WithLabelValues("extraneous").Add(float64(len(result.ExtraneousTables)))
	verifymetrics.NumTablesProcessed.WithLabelValues("verified").Add(float64(len(result.Verified)))
}

func reportTelemetry(logger zerolog.Logger, opts verifyOpts, conns dbconn.OrderedConns) {
	dialect := "CockroachDB"
	for _, conn := range conns {
		if !conn.IsCockroach() {
			dialect = conn.Dialect()
			break
		}
	}
	features := []string{"molt_verify_dialect_" + dialect}
	if opts.continuous {
		features = append(features, "molt_verify_continuous")
	}
	if opts.liveVerificationSettings != nil {
		features = append(features, "molt_verify_live")
	}
	molttelemetry.ReportTelemetryAsync(logger, features...)
}
