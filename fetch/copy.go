package fetch

import (
	"context"
	"path"
	"time"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/fetchmetrics"
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/moltlogger"
	"github.com/rs/zerolog"
)

type CopyResult struct {
	StartTime time.Time
	EndTime   time.Time
}

func Copy(
	ctx context.Context,
	baseConn dbconn.Conn,
	logger zerolog.Logger,
	table dbtable.VerifiedTable,
	resources []datablobstorage.Resource,
	isLocal bool,
	isClearContinuationTokenMode bool,
	exceptionLog *status.ExceptionLog,
) (CopyResult, error) {
	dataLogger := moltlogger.GetDataLogger(logger)
	ret := CopyResult{
		StartTime: time.Now(),
	}

	rowsSoFar := 0
	conn := baseConn.(*dbconn.PGConn).Conn

	for i, resource := range resources {
		key, err := resource.Key()
		if err != nil {
			return ret, err
		}

		dataLogger.Debug().
			Int("idx", i+1).
			Msgf("reading resource")
		if err := func() error {
			r, err := resource.Reader(ctx)
			if err != nil {
				return err
			}
			dataLogger.Debug().
				Int("idx", i+1).
				Msgf("running copy from resource")
			if copyRet, err := conn.PgConn().CopyFrom(
				ctx,
				r,
				dataquery.CopyFrom(table, isLocal /*skips header if local */),
			); err != nil {
				fileName := path.Base(key)
				return status.MaybeReportException(ctx, logger, conn, table.Name, err, fileName, status.StageDataLoad, isClearContinuationTokenMode, exceptionLog)
			} else {
				rowsSoFar += int(copyRet.RowsAffected())
				dataLogger.Info().
					Int("num_rows", rowsSoFar).
					Str("table", table.SafeString()).
					Str("file", key).
					Msg("row copy status")
				fetchmetrics.ImportedRows.WithLabelValues(table.SafeString()).Add(float64(copyRet.RowsAffected()))
			}
			return nil
		}(); err != nil {
			return ret, err
		}
	}

	ret.EndTime = time.Now()
	dataLogger.Info().
		Dur("duration", ret.EndTime.Sub(ret.StartTime)).
		Msgf("table COPY complete")
	return ret, nil
}
