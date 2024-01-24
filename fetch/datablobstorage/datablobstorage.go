package datablobstorage

import (
	"context"
	"fmt"
	"io"

	"github.com/cockroachdb/molt/dbtable"
)

type Store interface {
	// CreateFromReader is responsible for the creation of the individual
	// CSVs from the data export process. It will create the file and upload
	// it to the respetive data store and return the resource object which
	// will be used in the data import phase.
	CreateFromReader(ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string) (Resource, error)
	// ListFromContinuationPoint is used when restarting Fetch from
	// a continuation point. It will query the respective data store
	// and create the slice of resources that will be used by the
	// import process. Note that NO files are created from the method.
	// It simply lists all files in the data store and filters and returns
	// the files that are needed.
	ListFromContinuationPoint(ctx context.Context, table dbtable.VerifiedTable, fileName string) ([]Resource, error)
	CanBeTarget() bool
	DefaultFlushBatchSize() int
	Cleanup(ctx context.Context) error
	TelemetryName() string
}

type Resource interface {
	Key() (string, error)
	ImportURL() (string, error)
	MarkForCleanup(ctx context.Context) error
	Reader(ctx context.Context) (io.ReadCloser, error)
}

func getKeyAndPrefix(fileName, bucketPath string, table dbtable.VerifiedTable) (string, string) {
	key := fmt.Sprintf("%s/%s", table.SafeString(), fileName)
	prefix := table.SafeString()
	// If bucketPath is not "", then this means the files are not at
	// the root level of the bucket. It must be structured like
	// s3://bucket/public.table/part_000000001.csv.tar.gz.
	// This means we will need to prepend whatever the full path is
	// to have the proper key to the object.
	if bucketPath != "" {
		key = fmt.Sprintf("%s/%s/%s", bucketPath, table.SafeString(), fileName)
		prefix = fmt.Sprintf("%s/%s", bucketPath, table.SafeString())
	}
	return key, prefix
}
