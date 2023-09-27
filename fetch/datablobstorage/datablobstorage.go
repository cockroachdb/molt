package datablobstorage

import (
	"context"
	"io"

	"github.com/cockroachdb/molt/dbtable"
)

type Store interface {
	CreateFromReader(ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string) (Resource, error)
	CanBeTarget() bool
	DefaultFlushBatchSize() int
	Cleanup(ctx context.Context) error
	TelemetryName() string
}

type Resource interface {
	ImportURL() (string, error)
	MarkForCleanup(ctx context.Context) error
	Reader(ctx context.Context) (io.ReadCloser, error)
}
