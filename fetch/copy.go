package fetch

import (
	"context"
	"time"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
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
) (CopyResult, error) {
	dataLogger := moltlogger.GetDataLogger(logger)
	ret := CopyResult{
		StartTime: time.Now(),
	}

	conn := baseConn.(*dbconn.PGConn).Conn

	for i, resource := range resources {
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
			if _, err := conn.PgConn().CopyFrom(
				ctx,
				r,
				dataquery.CopyFrom(table),
			); err != nil {
				return err
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
