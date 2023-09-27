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
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/rs/zerolog"
)

type s3Store struct {
	logger      zerolog.Logger
	bucket      string
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
	logger zerolog.Logger, session *session.Session, creds credentials.Value, bucket string,
) *s3Store {
	return &s3Store{
		bucket:  bucket,
		session: session,
		logger:  logger,
		creds:   creds,
	}
}

func (s *s3Store) CreateFromReader(
	ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string,
) (Resource, error) {
	key := fmt.Sprintf("%s/part_%08d.%s", table.SafeString(), iteration, fileExt)
	s.logger.Debug().Str("file", key).Msgf("creating new file")
	if _, err := s3manager.NewUploader(s.session).UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	}); err != nil {
		return nil, err
	}
	s.logger.Debug().Str("file", key).Msgf("s3 file creation batch complete")
	return &s3Resource{
		session: s.session,
		store:   s,
		key:     key,
	}, nil
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
