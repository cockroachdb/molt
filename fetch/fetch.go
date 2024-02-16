package fetch

import (
	"context"
	"fmt"
	"strings"
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
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/cockroachdb/molt/verify/tableverify"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	FlushSize            int
	FlushRows            int
	Cleanup              bool
	Live                 bool
	Concurrency          int
	FetchID              string
	ContinuationToken    string
	ContinuationFileName string
	// TestOnly means this fetch attempt is just for test, and hence all time/duration
	// stats are deterministic.
	TestOnly bool

	// The target table handling configs.
	Truncate bool

	Compression    compression.Flag
	ExportSettings dataexport.Settings
}

type SchemaCreationConfig struct {
	TableFilter  string
	SchemaFilter string
}

func Fetch(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	conns dbconn.OrderedConns,
	blobStore datablobstorage.Store,
	tableFilter utils.FilterConfig,
	testingKnobs testutils.FetchTestingKnobs,
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
	if dbTables, err = utils.FilterResult(tableFilter, dbTables); err != nil {
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

	// Wait until all the verification portions are completed first before deferring this.
	// If verify fails, we don't need to report fetch_id.
	// We only want to log out if the fetch fails.
	defer func() {
		if retErr != nil {
			logger.Info().
				Str("fetch_id", utils.MaybeFormatFetchID(cfg.TestOnly, fetchStatus.ID.String())).Msg("continue from this fetch ID")
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

	exceptionLogMapping, err := getExceptionLogMapping(ctx, cfg, targetPgxConn)
	if err != nil {
		return err
	}

	// TODO(janexing): ingest the schema creation logic here. We create the schemas
	// one by one, and exit if any of them errors out.

	workCh := make(chan tableverify.Result)
	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < cfg.Concurrency; i++ {
		g.Go(func() error {
			for {
				table, ok := <-workCh
				if !ok {
					return nil
				}

				var relevantExceptionLog *status.ExceptionLog
				if v, ok := exceptionLogMapping[table.SafeString()]; ok {
					relevantExceptionLog = v
				}

				// We want to run the fetch in only two cases:
				// 1. Export + import mode combined (when fetch ID is not passed in; means new fetch)
				// 2. When the fetch ID is passed in and exception log is not nil, which means it is a table we want to continue from.
				// This means we want to skip if we are trying to continue but there is no entry that specifies where to continue from.
				if (cfg.FetchID != "" && relevantExceptionLog != nil) || (cfg.FetchID == "") {
					if err := fetchTable(ctx, cfg, logger, conns, blobStore, sqlSrc, table, relevantExceptionLog, testingKnobs); err != nil {
						return err
					}
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
		Str("fetch_id", utils.MaybeFormatFetchID(cfg.TestOnly, fetchStatus.ID.String())).
		Int("num_tables", stats.numImportedTables).
		Strs("tables", stats.importedTables).
		Str("cdc_cursor", utils.MaybeFormatCDCCursor(cfg.TestOnly, sqlSrc.CDCCursor())).
		Dur("net_duration_ms", ovrDuration).
		Str("net_duration", utils.FormatDurationToTimeString(ovrDuration)).
		Msgf("fetch complete")
	return nil
}

func truncateTable(
	ctx context.Context, logger zerolog.Logger, table tableverify.Result, conns dbconn.OrderedConns,
) error {
	truncateTargetTableConn, err := conns[1].Clone(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to clone a connection to truncate the table on the target db")
	}
	logger.Info().Msgf("truncating table")
	_, err = truncateTargetTableConn.(*dbconn.PGConn).Conn.Exec(ctx, "TRUNCATE TABLE "+table.SafeString())
	if err != nil {
		return errors.Wrap(err, "failed executing the TRUNCATE TABLE statement")
	}
	if err := truncateTargetTableConn.Close(ctx); err != nil {
		return errors.Wrap(err, "unable to close the connection that is used to truncate the table on the target db")
	}
	logger.Info().Msgf("finished truncating table")
	return nil
}

// Note that if `ExceptionLog` is not nil, then that means
// there is an exception log and import/copy only mode
// was specified.
func fetchTable(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	conns dbconn.OrderedConns,
	blobStore datablobstorage.Store,
	sqlSrc dataexport.Source,
	table tableverify.Result,
	exceptionLog *status.ExceptionLog,
	testingKnobs testutils.FetchTestingKnobs,
) (retErr error) {
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

	// Truncate table on the target side, if applicable.
	if cfg.Truncate {
		if err := truncateTable(ctx, logger, table, conns); err != nil {
			return err
		}
	}

	logger.Info().Msgf("data extraction phase starting")

	var e exportResult
	// In the case that exception log is nil or fetch id is empty,
	// this means that we want to export the table because it means
	// we want export + copy mode.
	resourceCh := make(chan datablobstorage.Resource)
	summaryLogger := moltlogger.GetSummaryLogger(logger)
	var exportDuration time.Duration
	exportWG, _ := errgroup.WithContext(ctx)
	exportWG.Go(func() error {
		defer close(resourceCh)
		if exceptionLog == nil || cfg.FetchID == "" {
			er, err := exportTable(ctx, cfg, logger, sqlSrc, blobStore, table.VerifiedTable, testingKnobs, resourceCh)
			if err != nil {
				return err
			}
			e = er
		} else {
			if exceptionLog.FileName == "" {
				logger.Warn().Msgf("skipping table %s because no file name is present in the exception log", table.SafeString())
				return errors.Newf("table %s not imported because no file name is present in the exception log", table.SafeString())
			}
			logger.Warn().Msgf("skipping export for table %s due to running in import-copy only mode", table.SafeString())

			// TODO: future PR needs to add number of rows estimation. and populate exportResult.NumRows
			// TODO: need to figure out start and end time too.
			rsc, err := blobStore.ListFromContinuationPoint(ctx, table.VerifiedTable, exceptionLog.FileName)
			if err != nil {
				return err
			}
			e.Resources = rsc

			if len(e.Resources) == 0 {
				return errors.Newf("exported resources for table %s is empty, please make sure you did not accidentally delete from the intermediate store", table.SafeString())
			}
		}
		// TODO: consider if we want to skip this portion since we don't export anything....
		exportDuration = utils.MaybeFormatDurationForTest(cfg.TestOnly, e.EndTime.Sub(e.StartTime))
		summaryLogger.Info().
			Int("num_rows", e.NumRows).
			Dur("export_duration_ms", exportDuration).
			Str("export_duration", utils.FormatDurationToTimeString(exportDuration)).
			Msgf("data extraction from source complete")
		fetchmetrics.TableExportDuration.WithLabelValues(table.SafeString()).Set(float64(exportDuration.Milliseconds()))
		return nil
	})

	// We actually need to skip the cleanup for something that has an error
	// On a continuation run we can cleanup so long as it's successful.
	if cfg.Cleanup {
		defer func() {
			if retErr != nil {
				logger.Info().Msg("skipping cleanup because an error occurred and files may need to be kept for continuation")
				return
			}

			logger.Info().Msg("cleaning up resources created during fetch run")
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

	importWG, _ := errgroup.WithContext(ctx)
	var importDuration time.Duration
	var netDuration time.Duration
	var cdcCursor string
	exitCh := make(chan error, 1)
	importWG.Go(func() error {
		if blobStore.CanBeTarget() {
			targetConn, err := conns[1].Clone(ctx)
			if err != nil {
				return err
			}
			if err := func() error {
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
					for {
						rs, ok := <-resourceCh
						if !ok {
							select {
							// If there was an error on the exportSide, stop importing.
							case e := <-exitCh:
								return e
							default:
								return nil
							}
						}
						r, err := importTable(ctx, cfg, targetConn, logger, table.VerifiedTable, []datablobstorage.Resource{rs}, testingKnobs)
						if err != nil {
							return err
						}
						fetchmetrics.ImportedRows.WithLabelValues(table.SafeString()).Add(float64(rs.Rows()))
						importDuration = utils.MaybeFormatDurationForTest(cfg.TestOnly, r.EndTime.Sub(r.StartTime))

					}
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
			netDuration = utils.MaybeFormatDurationForTest(cfg.TestOnly, time.Since(tableStartTime))
			cdcCursor = utils.MaybeFormatCDCCursor(cfg.TestOnly, sqlSrc.CDCCursor())
		}
		return nil
	})

	if err := exportWG.Wait(); err != nil {
		fmt.Println("writin to exitCh")
		exitCh <- err
	}
	err := importWG.Wait()
	if err != nil {
		return err
	}

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

// TODO: handle the case where the file override happens.
func getExceptionLogMapping(
	ctx context.Context, cfg Config, targetPgxConn *pgx.Conn,
) (map[string]*status.ExceptionLog, error) {
	exceptionLogMapping := map[string]*status.ExceptionLog{}
	if IsImportCopyOnlyMode(cfg) {
		exceptionLogs := []*status.ExceptionLog{}
		if strings.TrimSpace(cfg.ContinuationToken) == "" {
			exceptionLogsFID, err := status.GetAllExceptionLogsByFetchID(ctx, targetPgxConn, cfg.FetchID)
			if err != nil {
				return nil, err
			}
			exceptionLogs = append(exceptionLogs, exceptionLogsFID...)
		} else {
			exceptionLog, err := status.GetExceptionLogByToken(ctx, targetPgxConn, cfg.ContinuationToken)
			if err != nil {
				return nil, err
			}

			if cfg.ContinuationFileName != "" {
				exceptionLog.FileName = cfg.ContinuationFileName
			}

			exceptionLogs = append(exceptionLogs, exceptionLog)
		}

		exceptionLogMapping = status.GetTableSchemaToExceptionLog(exceptionLogs)
	}

	return exceptionLogMapping, nil
}

func IsImportCopyOnlyMode(cfg Config) bool {
	return strings.TrimSpace(cfg.FetchID) != ""
}
