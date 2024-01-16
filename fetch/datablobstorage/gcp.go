package datablobstorage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2/google"
)

type gcpStore struct {
	logger     zerolog.Logger
	bucket     string
	bucketPath string
	client     *storage.Client
	creds      *google.Credentials
}

func NewGCPStore(
	logger zerolog.Logger,
	client *storage.Client,
	creds *google.Credentials,
	bucket string,
	bucketPath string,
) *gcpStore {
	return &gcpStore{
		bucket:     bucket,
		bucketPath: bucketPath,
		client:     client,
		logger:     logger,
		creds:      creds,
	}
}

func (s *gcpStore) CreateFromReader(
	ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string,
) (Resource, error) {
	key := fmt.Sprintf("%s/part_%08d.%s", table.SafeString(), iteration, fileExt)
	if s.bucketPath != "" {
		key = fmt.Sprintf("%s/%s/part_%08d.%s", s.bucketPath, table.SafeString(), iteration, fileExt)
	}

	s.logger.Debug().Str("file", key).Msgf("creating new file")
	wc := s.client.Bucket(s.bucket).Object(key).NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		return nil, err
	}
	if err := wc.Close(); err != nil {
		return nil, err
	}

	rows := <-numRows
	s.logger.Debug().Str("file", key).Int("rows", rows).Msgf("gcp file creation complete complete")
	return &gcpResource{
		store: s,
		key:   key,
		rows:  rows,
	}, nil
}

func (s *gcpStore) CanBeTarget() bool {
	return true
}

func (s *gcpStore) DefaultFlushBatchSize() int {
	return 256 * 1024 * 1024
}

func (s *gcpStore) Cleanup(ctx context.Context) error {
	// Folders are deleted when the final object is deleted.
	return nil
}

func (r *gcpStore) TelemetryName() string {
	return "gcp"
}

type gcpResource struct {
	store *gcpStore
	key   string
}

func (r *gcpResource) ImportURL() (string, error) {
	return fmt.Sprintf(
		"gs://%s/%s?CREDENTIALS=%s",
		r.store.bucket,
		r.key,
		base64.StdEncoding.EncodeToString(r.store.creds.JSON),
	), nil
}

func (r *gcpResource) Key() (string, error) {
	return r.key, nil
}

func (r *gcpResource) NumRows() (int, error) {
	return r.rows, nil
}

func (r *gcpResource) Reader(ctx context.Context) (io.ReadCloser, error) {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).NewReader(ctx)
}

func (r *gcpResource) MarkForCleanup(ctx context.Context) error {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).Delete(ctx)
}
