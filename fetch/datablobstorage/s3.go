package datablobstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/rs/zerolog"
)

type s3Store struct {
	logger      zerolog.Logger
	bucket      string
	bucketPath  string
	session     *session.Session
	creds       credentials.Value
	batchDelete struct {
		sync.Mutex
		batch []s3manager.BatchDeleteObject
	}
}

type s3Resource struct {
	session *session.Session
	store   *s3Store
	key     string
	rows    int
}

func (s *s3Resource) ImportURL() (string, error) {
	return fmt.Sprintf(
		"s3://%s/%s?AWS_ACCESS_KEY_ID=%s&AWS_SECRET_ACCESS_KEY=%s",
		s.store.bucket,
		s.key,
		url.QueryEscape(s.store.creds.AccessKeyID),
		url.QueryEscape(s.store.creds.SecretAccessKey),
	), nil
}

func (s *s3Resource) Key() (string, error) {
	return s.key, nil
}

func (s *s3Resource) Rows() int {
	return s.rows
}

func (s *s3Resource) MarkForCleanup(ctx context.Context) error {
	s.store.batchDelete.Lock()
	defer s.store.batchDelete.Unlock()
	s.store.batchDelete.batch = append(s.store.batchDelete.batch, s3manager.BatchDeleteObject{
		Object: &s3.DeleteObjectInput{
			Key:    aws.String(s.key),
			Bucket: aws.String(s.store.bucket),
		},
	})
	return nil
}

func (s *s3Resource) Reader(ctx context.Context) (io.ReadCloser, error) {
	b := aws.NewWriteAtBuffer(nil)
	if _, err := s3manager.NewDownloader(s.store.session).DownloadWithContext(
		ctx,
		b,
		&s3.GetObjectInput{
			Key:    aws.String(s.key),
			Bucket: aws.String(s.store.bucket),
		},
	); err != nil {
		return nil, err
	}
	return s3Reader{Reader: bytes.NewReader(b.Bytes())}, nil
}

type s3Reader struct {
	*bytes.Reader
}

func (r s3Reader) Close() error {
	return nil
}

func NewS3Store(
	logger zerolog.Logger,
	session *session.Session,
	creds credentials.Value,
	bucket string,
	bucketPath string,
) *s3Store {
	return &s3Store{
		bucket:     bucket,
		bucketPath: bucketPath,
		session:    session,
		logger:     logger,
		creds:      creds,
	}
}

func (s *s3Store) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	table dbtable.VerifiedTable,
	iteration int,
	fileExt string,
	numRows chan int,
) (Resource, error) {
	key := fmt.Sprintf("%s/part_%08d.%s", table.SafeString(), iteration, fileExt)
	if s.bucketPath != "" {
		key = fmt.Sprintf("%s/%s/part_%08d.%s", s.bucketPath, table.SafeString(), iteration, fileExt)
	}

	s.logger.Debug().Str("file", key).Msgf("creating new file")
	if _, err := s3manager.NewUploader(s.session).UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	}); err != nil {
		return nil, err
	}

	rows := <-numRows
	s.logger.Debug().Str("file", key).Int("rows", rows).Msgf("s3 file creation batch complete")
	return &s3Resource{
		session: s.session,
		store:   s,
		key:     key,
		rows:    rows,
	}, nil
}

// ListFromContinuationPoint will create the list of s3 resources
// that will be processed for this iteration of fetch. It uses the
// passed in table name to construct the key and prefix to look at
// in the s3 bucket.
func (s *s3Store) ListFromContinuationPoint(
	ctx context.Context, table dbtable.VerifiedTable, fileName string,
) ([]Resource, error) {
	key, prefix := getKeyAndPrefix(fileName, s.bucketPath, table)
	s3client := s3.New(s.session)
	return listFromContinuationPointAWS(ctx, s3client, key, prefix, s)
}

// listFromContinuationPoint is a helper for listFromContinuationPoint
// to allow dependancy injection of the S3API since ListFromContinuationPoint
// needs to satisfy the datablobstore interface, we can't put a s3 specific API
// as part of the function signature. The helper will make the API call to S3 and
// create the s3Resource objects that Import or Copy will use.
func listFromContinuationPointAWS(
	ctx context.Context, s3Client s3iface.S3API, key, prefix string, s3Store *s3Store,
) ([]Resource, error) {
	// Note: There is a StartAfter parameter in ListObjectV2Input
	// but it is non inclusive of the provided key so we can't use it as we
	// need to include the file we are starting from.
	s3Objects, err := s3Client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3Store.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}

	resources := []Resource{}
	for _, obj := range s3Objects.Contents {
		// Find the key we want to start at. Because we name the files
		// in a specific pattern, we can guarantee lexicographical ordering
		// based on the guarantee of return order from the S3 API.
		// eg. If key = fetch/public.inventory/part_00000004.tar.gz,
		// fetch/public.inventory/part_00000005.tar.gz is >= to key meaning,
		// it is a file we need to include.
		if aws.StringValue(obj.Key) >= key {
			resources = append(resources, &s3Resource{
				key:     aws.StringValue(obj.Key),
				session: s3Store.session,
				store:   s3Store,
			})
		}
	}
	return resources, nil
}

func (s *s3Store) CanBeTarget() bool {
	return true
}

func (s *s3Store) DefaultFlushBatchSize() int {
	return 256 * 1024 * 1024
}

func (s *s3Store) Cleanup(ctx context.Context) error {
	s.batchDelete.Lock()
	defer s.batchDelete.Unlock()

	batcher := s3manager.NewBatchDelete(s.session)
	if err := batcher.Delete(
		aws.BackgroundContext(),
		&s3manager.DeleteObjectsIterator{Objects: s.batchDelete.batch},
	); err != nil {
		return err
	}
	s.batchDelete.batch = s.batchDelete.batch[:0]
	return nil
}

func (s *s3Store) TelemetryName() string {
	return "s3"
}
