package datablobstorage

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2/google"
)

type Store interface {
	// CreateFromReader is responsible for the creation of the individual
	// CSVs from the data export process. It will create the file and upload
	// it to the respetive data store and return the resource object which
	// will be used in the data import phase.
	CreateFromReader(ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string, numRows chan int) (Resource, error)
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
	Rows() int
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

type DirectCopyPayload struct {
	TargetConnForCopy *pgx.Conn
}

type LocalPathPayload struct {
	LocalPath               string
	LocalPathListenAddr     string
	LocalPathCRDBAccessAddr string
}

type GCPPayload struct {
	GCPBucket  string
	BucketPath string
}

type S3Payload struct {
	S3Bucket   string
	BucketPath string
}

type DatastoreCreationPayload struct {
	DirectCopyPl *DirectCopyPayload
	GCPPl        *GCPPayload
	S3Pl         *S3Payload
	LocalPathPl  *LocalPathPayload

	logger zerolog.Logger
}

func GenerateDatastore(
	ctx context.Context, cfg any, logger zerolog.Logger, testFailedWriteToBucket bool,
) (Store, error) {
	var src Store
	var err error

	switch t := cfg.(type) {
	case *DirectCopyPayload:
		src = NewCopyCRDBDirect(logger, t.TargetConnForCopy)
	case *GCPPayload:
		var creds *google.Credentials
		var err error
		var emptyClient storage.Client
		gcpClient := &emptyClient
		// For this test, we don't need the real credentials or client.
		if !testFailedWriteToBucket {
			creds, err = google.FindDefaultCredentials(ctx)
			if err != nil {
				return nil, err
			}
			gcpClient, err = storage.NewClient(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to make new gcp client")
			}
		}
		src = NewGCPStore(logger, gcpClient, creds, t.GCPBucket, t.BucketPath)
	case *S3Payload:
		var sess *session.Session
		var creds credentials.Value
		var err error

		if sess, err = session.NewSession(); err != nil {
			return nil, err
		}
		if t.Region != "" {
			sess.Config.Region = &t.Region
		}
		// For this test, we don't need the real credentials or client.
		if !testFailedWriteToBucket {
			if creds, err = sess.Config.Credentials.Get(); err != nil {
				return nil, err
			}
		}
		src = NewS3Store(logger, sess, creds, t.S3Bucket, t.BucketPath)
	case *LocalPathPayload:
		src, err = NewLocalStore(logger, t.LocalPath, t.LocalPathListenAddr, t.LocalPathCRDBAccessAddr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.AssertionFailedf("data source must be configured (--s3-bucket, --gcp-bucket, --direct-copy)")
	}

	return src, err
}

func GenerateDatastoreNew(ctx context.Context, cfg DatastoreCreationPayload) (Store, error) {
	var src Store
	var err error
	switch {
	case cfg.DirectCopyPl != nil:
		src = NewCopyCRDBDirect(cfg.logger, cfg.DirectCopyPl.TargetConnForCopy)
	case cfg.GCPPl != nil:
		creds, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, err
		}
		gcpClient, err := storage.NewClient(context.Background())
		if err != nil {
			return nil, err
		}
		src = NewGCPStore(cfg.logger, gcpClient, creds, cfg.GCPPl.GCPBucket, cfg.GCPPl.BucketPath)
	case cfg.S3Pl != nil:
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		creds, err := sess.Config.Credentials.Get()
		if err != nil {
			return nil, err
		}
		src = NewS3Store(cfg.logger, sess, creds, cfg.S3Pl.S3Bucket, cfg.S3Pl.BucketPath)
	case cfg.LocalPathPl != nil:
		src, err = NewLocalStore(cfg.logger, cfg.LocalPathPl.LocalPath, cfg.LocalPathPl.LocalPathListenAddr, cfg.LocalPathPl.LocalPathCRDBAccessAddr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.AssertionFailedf("data source must be configured (--s3-bucket, --gcp-bucket, --direct-copy)")
	}
	return src, nil
}
