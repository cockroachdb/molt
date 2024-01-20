package fetch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/fetch/fetchcontext"
	"github.com/cockroachdb/molt/fetch/fetchmetrics"
	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/molttelemetry"
	"github.com/cockroachdb/molt/utils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/cockroachdb/molt/verify/tableverify"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	FlushSize   int
	FlushRows   int
	Cleanup     bool
	Live        bool
	Truncate    bool
	Concurrency int

	// TestOnly means this fetch attempt is just for test, and hence all time/duration
	// stats are deterministic.
	TestOnly bool

	Compression    compression.Flag
	ExportSettings dataexport.Settings
}

func Fetch(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	conns dbconn.OrderedConns,
	blobStore datablobstorage.Store,
	tableFilter dbverify.FilterConfig,
) (retErr error) {
	// Setup fetch status tracking.
	targetPgConn, valid := conns[1].(*dbconn.PGConn)
	if !valid {
		return errors.New("failed to assert conn as a pgconn")
	}
	targetPgxConn := targetPgConn.Conn
	fetchStatus, err := initStatusEntry(ctx, targetPgxConn, conns[0].Dialect())
	if err != nil {
		return err
	}
	ctx = fetchcontext.ContextWithFetchData(ctx, fetchcontext.FetchContextData{
		RunID:     fetchStatus.ID,
		StartedAt: fetchStatus.StartedAt,
	})

	timer := prometheus.NewTimer(prometheus.ObserverFunc(fetchmetrics.OverallDuration.Set))

	if cfg.FlushSize == 0 {
		cfg.FlushSize = blobStore.DefaultFlushBatchSize()
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}

	if cfg.Cleanup {
		defer func() {
			if err := blobStore.Cleanup(ctx); err != nil {
				logger.Err(err).Msgf("error marking object for cleanup")
			}
		}()
	}

	if err := dbconn.RegisterTelemetry(conns); err != nil {
		return err
	}
	reportTelemetry(logger, cfg, conns, blobStore)

	dataLogger := moltlogger.GetDataLogger(logger)
	dataLogger.Debug().
		Int("flush_size", cfg.FlushSize).
		Int("flush_num_rows", cfg.FlushRows).
		Str("store", fmt.Sprintf("%T", blobStore)).
		Msg("initial config")

	logger.Info().Msgf("checking database details")
	dbTables, err := dbverify.Verify(ctx, conns)
	if err != nil {
		return err
	}
	if dbTables, err = dbverify.FilterResult(tableFilter, dbTables); err != nil {
		return err
	}
	for _, tbl := range dbTables.ExtraneousTables {
		logger.Warn().
			Str("table", tbl.SafeString()).
			Msgf("ignoring table as it is missing a definition on the source")
	}
	for _, tbl := range dbTables.MissingTables {
		logger.Warn().
			Str("table", tbl.SafeString()).
			Msgf("ignoring table as it is missing a definition on the target")
	}
	for _, tbl := range dbTables.Verified {
		logger.Info().
			Str("source_table", tbl[0].SafeString()).
			Str("target_table", tbl[1].SafeString()).
			Msgf("found matching table")
	}

	logger.Info().Msgf("verifying common tables")
	tables, err := tableverify.VerifyCommonTables(ctx, conns, dbTables.Verified)
	if err != nil {
		return err
	}
	logger.Info().Msgf("establishing snapshot")
	sqlSrc, err := dataexport.InferExportSource(ctx, cfg.ExportSettings, conns[0])
	if err != nil {
		return err
	}
	defer func() {
		if err := sqlSrc.Close(ctx); err != nil {
			logger.Err(err).Msgf("error closing export source")
		}
	}()

	numTables := len(tables)
	summaryLogger := moltlogger.GetSummaryLogger(logger)
	summaryLogger.Info().
		Int("num_tables", numTables).
		Str("cdc_cursor", utils.MaybeFormatCDCCursor(cfg.TestOnly, sqlSrc.CDCCursor())).
		Msgf("starting fetch")
	fetchmetrics.NumTablesProcessed.Add(float64(numTables))

	type statsMu struct {
		sync.Mutex
		numImportedTables int
		importedTables    []string
	}
	var stats statsMu

	workCh := make(chan tableverify.Result)
	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < cfg.Concurrency; i++ {
		g.Go(func() error {
			for {
				table, ok := <-workCh
				if !ok {
					return nil
				}
				if err := fetchTable(ctx, cfg, logger, conns, blobStore, sqlSrc, table); err != nil {
					return err
				}

				stats.Lock()
				stats.numImportedTables++
				stats.importedTables = append(stats.importedTables, table.SafeString())
				stats.Unlock()
			}
		})
	}

	go func() {
		defer close(workCh)
		for _, table := range tables {
			workCh <- table
		}
	}()

	if err := g.Wait(); err != nil {
		return err
	}

	ovrDuration := utils.MaybeFormatDurationForTest(cfg.TestOnly, timer.ObserveDuration())
	summaryLogger.Info().
		Int("num_tables", stats.numImportedTables).
		Strs("tables", stats.importedTables).
		Str("cdc_cursor", utils.MaybeFormatCDCCursor(cfg.TestOnly, sqlSrc.CDCCursor())).
		Dur("net_duration_ms", ovrDuration).
		Str("net_duration", utils.FormatDurationToTimeString(ovrDuration)).
		Msgf("fetch complete")
	return nil
}

func fetchTable(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	conns dbconn.OrderedConns,
	blobStore datablobstorage.Store,
	sqlSrc dataexport.Source,
	table tableverify.Result,
) error {
	tableStartTime := time.Now()
	// Initialize metrics for this table so we can calculate a rate later.
	fetchmetrics.ExportedRows.WithLabelValues(table.SafeString())
	fetchmetrics.ImportedRows.WithLabelValues(table.SafeString())

	for _, col := range table.MismatchingTableDefinitions {
		logger.Warn().
			Str("reason", col.Info).
			Msgf("not migrating column %s as it mismatches", col.Name)
	}
	if !table.RowVerifiable {
		logger.Error().Msgf("table %s do not have matching primary keys, cannot migrate", table.SafeString())
		return nil
	}

	logger.Info().Msgf("data extraction phase starting")

	e, err := exportTable(ctx, cfg, logger, sqlSrc, blobStore, table.VerifiedTable)
	if err != nil {
		return err
	}

	if cfg.Cleanup {
		defer func() {
			for _, r := range e.Resources {
				if r == nil {
					continue
				}

				if err := r.MarkForCleanup(ctx); err != nil {
					logger.Err(err).Msgf("error cleaning up resource")
				}
			}
		}()
	}

	exportDuration := utils.MaybeFormatDurationForTest(cfg.TestOnly, e.EndTime.Sub(e.StartTime))
	summaryLogger := moltlogger.GetSummaryLogger(logger)
	summaryLogger.Info().
		Int("num_rows", e.NumRows).
		Dur("export_duration_ms", exportDuration).
		Str("export_duration", utils.FormatDurationToTimeString(exportDuration)).
		Msgf("data extraction from source complete")
	fetchmetrics.TableExportDuration.WithLabelValues(table.SafeString()).Set(float64(exportDuration.Milliseconds()))

	if blobStore.CanBeTarget() {
		targetConn, err := conns[1].Clone(ctx)
		if err != nil {
			return err
		}
		var importDuration time.Duration
		if err := func() error {
			if cfg.Truncate {
				logger.Info().Msgf("truncating table")
				_, err := targetConn.(*dbconn.PGConn).Conn.Exec(ctx, "TRUNCATE TABLE "+table.SafeString())
				if err != nil {
					return err
				}
			}

			logger.Info().
				Msgf("starting data import on target")

			if !cfg.Live {
				go func() {
					err := reportImportTableProgress(ctx,
						targetConn,
						logger,
						table.VerifiedTable,
						time.Now(),
						false /*testing*/)
					if err != nil {
						logger.Err(err).Msg("failed to report import table progress")
					}
				}()

				r, err := importTable(ctx, cfg, targetConn, logger, table.VerifiedTable, e.Resources)
				if err != nil {
					return err
				}
				fetchmetrics.ImportedRows.WithLabelValues(table.SafeString()).Add(float64(e.NumRows))
				importDuration = utils.MaybeFormatDurationForTest(cfg.TestOnly, r.EndTime.Sub(r.StartTime))
			} else {
				r, err := Copy(ctx, targetConn, logger, table.VerifiedTable, e.Resources)
				if err != nil {
					return err
				}
				importDuration = utils.MaybeFormatDurationForTest(cfg.TestOnly, r.EndTime.Sub(r.StartTime))
			}
			return nil
		}(); err != nil {
			return errors.CombineErrors(err, targetConn.Close(ctx))
		}
		if err := targetConn.Close(ctx); err != nil {
			return err
		}

		netDuration := utils.MaybeFormatDurationForTest(cfg.TestOnly, time.Since(tableStartTime))
		cdcCursor := utils.MaybeFormatCDCCursor(cfg.TestOnly, sqlSrc.CDCCursor())
		summaryLogger.Info().
			Dur("net_duration_ms", netDuration).
			Str("net_duration", utils.FormatDurationToTimeString(netDuration)).
			Dur("import_duration_ms", importDuration).
			Str("import_duration", utils.FormatDurationToTimeString(importDuration)).
			Dur("export_duration_ms", exportDuration).
			Str("export_duration", utils.FormatDurationToTimeString(exportDuration)).
			Int("num_rows", e.NumRows).
			Str("cdc_cursor", cdcCursor).
			Msgf("data import on target for table complete")
		fetchmetrics.TableImportDuration.WithLabelValues(table.SafeString()).Set(float64(importDuration.Milliseconds()))
		fetchmetrics.TableOverallDuration.WithLabelValues(table.SafeString()).Set(float64(netDuration.Milliseconds()))

		return nil
	}
	return nil
}

func reportTelemetry(
	logger zerolog.Logger, cfg Config, conns dbconn.OrderedConns, store datablobstorage.Store,
) {
	dialect := "CockroachDB"
	for _, conn := range conns {
		if !conn.IsCockroach() {
			dialect = conn.Dialect()
			break
		}
	}
	ingestMethod := "import"
	if cfg.Live {
		ingestMethod = "copy"
	}
	molttelemetry.ReportTelemetryAsync(
		logger,
		"molt_fetch_dialect_"+dialect,
		"molt_fetch_ingest_method_"+ingestMethod,
		"molt_fetch_blobstore_"+store.TelemetryName(),
	)
}

func initStatusEntry(
	ctx context.Context, conn *pgx.Conn, dialect string,
) (*status.FetchStatus, error) {
	// Setup the status and exception tables.
	if err := status.CreateStatusAndExceptionTables(ctx, conn); err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC()
	fetchStatus := &status.FetchStatus{
		Name:          fmt.Sprintf("run at %d", createdAt.Unix()),
		StartedAt:     createdAt,
		SourceDialect: dialect,
	}
	if err := fetchStatus.CreateEntry(ctx, conn); err != nil {
		return nil, err
	}

	return fetchStatus, nil
}
