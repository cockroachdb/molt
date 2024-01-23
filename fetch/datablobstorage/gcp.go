package datablobstorage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
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
	s.logger.Debug().Str("file", key).Msgf("gcp file creation complete complete")
	return &gcpResource{
		store: s,
		key:   key,
	}, nil
}

func (s *gcpStore) ListFromContinuationPoint(
	ctx context.Context, table dbtable.VerifiedTable, fileName string,
) ([]Resource, error) {
	key, prefix := getKeyAndPrefix(fileName, s.bucketPath, table)
	return listFromContinuationPointGCP(ctx, stiface.AdaptClient(s.client), key, prefix, s.bucket)
}

func listFromContinuationPointGCP(
	ctx context.Context, client stiface.Client, key, prefix, bucket string,
) ([]Resource, error) {
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
		// The StartOffeset parameter is similar to the StartAfter flag
		// for S3 except that it is inclusive of the key so
		// we don't need to do any extra filtering of the
		// results.
		StartOffset: key,
	})

	resources := []Resource{}
	for {
		if attrs, err := it.Next(); err != nil {
			if err == iterator.Done {
				return resources, nil
			}
			return nil, err
		} else {
			resources = append(resources, &gcpResource{
				key: attrs.Name,
			})
		}
	}
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

func (r *gcpResource) Reader(ctx context.Context) (io.ReadCloser, error) {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).NewReader(ctx)
}

func (r *gcpResource) MarkForCleanup(ctx context.Context) error {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).Delete(ctx)
}
