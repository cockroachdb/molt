package fetch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/molttelemetry"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/cockroachdb/molt/verify/tableverify"
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

	ExportSettings dataexport.Settings
}

func Fetch(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	conns dbconn.OrderedConns,
	blobStore datablobstorage.Store,
	tableFilter dbverify.FilterConfig,
) error {
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

	logger.Debug().
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
	logger.Info().
		Int("num_tables", len(tables)).
		Str("cdc_cursor", sqlSrc.CDCCursor()).
		Msgf("starting fetch")

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

	logger.Info().
		Int("num_tables", stats.numImportedTables).
		Strs("tables", stats.importedTables).
		Str("cdc_cursor", sqlSrc.CDCCursor()).
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
	logger = logger.With().Str("table", table.SafeString()).Logger()

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

	e, err := exportTable(ctx, logger, sqlSrc, blobStore, table.VerifiedTable, cfg.FlushSize, cfg.FlushRows)
	if err != nil {
		return err
	}

	if cfg.Cleanup {
		defer func() {
			for _, r := range e.Resources {
				if err := r.MarkForCleanup(ctx); err != nil {
					logger.Err(err).Msgf("error cleaning up resource")
				}
			}
		}()
	}

	logger.Info().
		Int("num_rows", e.NumRows).
		Dur("export_duration", e.EndTime.Sub(e.StartTime)).
		Msgf("data extraction from source complete")

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
				r, err := importTable(ctx, targetConn, logger, table.VerifiedTable, e.Resources)
				if err != nil {
					return err
				}
				importDuration = r.EndTime.Sub(r.StartTime)
			} else {
				r, err := Copy(ctx, targetConn, logger, table.VerifiedTable, e.Resources)
				if err != nil {
					return err
				}
				importDuration = r.EndTime.Sub(r.StartTime)
			}
			return nil
		}(); err != nil {
			return errors.CombineErrors(err, targetConn.Close(ctx))
		}
		if err := targetConn.Close(ctx); err != nil {
			return err
		}
		logger.Info().
			Dur("net_duration", time.Since(tableStartTime)).
			Dur("import_duration", importDuration).
			Str("cdc_cursor", sqlSrc.CDCCursor()).
			Msgf("data import on target for table complete")
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
