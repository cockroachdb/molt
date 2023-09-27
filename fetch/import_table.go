package fetch

import (
	"context"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
	"github.com/cockroachdb/molt/retry"
	"github.com/rs/zerolog"
)

type importResult struct {
	StartTime time.Time
	EndTime   time.Time
}

func importTable(
	ctx context.Context,
	cfg Config,
	baseConn dbconn.Conn,
	logger zerolog.Logger,
	table dbtable.VerifiedTable,
	resources []datablobstorage.Resource,
) (importResult, error) {
	ret := importResult{
		StartTime: time.Now(),
	}

	var locs []string
	for _, resource := range resources {
		u, err := resource.ImportURL()
		if err != nil {
			return importResult{}, err
		}
		locs = append(locs, u)
	}
	conn := baseConn.(*dbconn.PGConn)
	r, err := retry.NewRetry(retry.Settings{
		InitialBackoff: time.Second,
		Multiplier:     2,
		MaxRetries:     4,
	})
	if err != nil {
		return ret, err
	}
	if err := r.Do(func() error {
		kvOptions := tree.KVOptions{}
		if cfg.Compression == compression.GZIP {
			kvOptions = append(kvOptions, tree.KVOption{
				Key:   "decompress",
				Value: tree.NewStrVal("gzip"),
			})
		}

		if _, err := conn.Exec(
			ctx,
			dataquery.ImportInto(table, locs, kvOptions),
		); err != nil {
			return errors.Wrap(err, "error importing data")
		}
		return nil
	}, func(err error) {
		logger.Err(err).Msgf("error importing data, retrying")
	}); err != nil {
		return ret, err
	}
	ret.EndTime = time.Now()
	return ret, nil
}
