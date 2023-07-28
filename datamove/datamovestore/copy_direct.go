package datamovestore

import (
	"context"
	"io"

	"github.com/cockroachdb/molt/dbtable"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

type copyCRDBDirect struct {
	logger zerolog.Logger
	target *pgx.Conn
}

func (c *copyCRDBDirect) CreateFromReader(
	ctx context.Context, r io.Reader, table dbtable.Name, iteration int,
) (Resource, error) {
	c.logger.Debug().Int("batch", iteration).Msgf("csv batch starting")
	if _, err := c.target.PgConn().CopyFrom(ctx, r, "COPY "+table.SafeString()+" FROM STDIN CSV"); err != nil {
		return nil, err
	}
	c.logger.Debug().Int("batch", iteration).Msgf("csv batch complete")
	return nil, nil
}

func (c *copyCRDBDirect) CanBeTarget() bool {
	return false
}

func (c *copyCRDBDirect) DefaultFlushBatchSize() int {
	return 1 * 1024 * 1024
}

func (c *copyCRDBDirect) Cleanup(ctx context.Context) error {
	return nil
}

func NewCopyCRDBDirect(logger zerolog.Logger, target *pgx.Conn) *copyCRDBDirect {
	return &copyCRDBDirect{
		logger: logger,
		target: target,
	}
}
