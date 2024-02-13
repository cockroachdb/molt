package datablobstorage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
)

const numRowsKeyGCP = "numRows"

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
	utils.RedactedQueryParams = map[string]struct{}{utils.GCPCredentials: {}}
	return &gcpStore{
		bucket:     bucket,
		bucketPath: bucketPath,
		client:     client,
		logger:     logger,
		creds:      creds,
	}
}

func (s *gcpStore) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	table dbtable.VerifiedTable,
	iteration int,
	fileExt string,
	numRows chan int,
	testingKnobs testutils.FetchTestingKnobs,
) (Resource, error) {
	key := fmt.Sprintf("%s/part_%08d.%s", table.SafeString(), iteration, fileExt)
	if s.bucketPath != "" {
		key = fmt.Sprintf("%s/%s/part_%08d.%s", s.bucketPath, table.SafeString(), iteration, fileExt)
	}

	s.logger.Debug().Str("file", key).Msgf("creating new file")

	// wc can only be *storage.Writer or GCPStorageWriterMock as the struct,
	// but since we need to accommodate both of them, we have to pick a generalized
	// interface.
	var wc interface {
		io.Closer
		io.Writer
	}

	o := s.client.Bucket(s.bucket).Object(key)
	wc = o.NewWriter(ctx)

	// If any error happens before io.Copy returns, the
	// error will be propagated to the goroutine in exportTable(),
	// triggering forwardRead.CloseWithError(), which will allow p.out.Close() in
	// csvPipe.flush() to return with the same error. This is because `forwardRead`
	// and `p.out` are the 2 ends of a pipe. Once the read side is closed with
	// error, the same error will be propagated to the write side.
	// See also: https://go.dev/play/p/H-pHiEffcZE.

	if testingKnobs.FailedWriteToBucket.FailedAfterReadFromPipe {
		// We need a mock writer which simulates the failed upload.
		wc = &GCPStorageWriterMock{wc.(*storage.Writer)}
	}

	rows := <-numRows

	// io.Copy starts execution ONLY after p.csvWriter.Flush() is triggered.
	if _, err := io.Copy(wc, r); err != nil {
		return nil, err
	}
	// Once io.Copy finished without error, p.csvWriter.Flush() and p.out.Close()
	// will return without error.

	// If any error after io.Copy returns, the error will trigger
	// forwardRead.CloseWithError() in the goroutine in exportTable(), but it will
	// lead to "error closing write goroutine", as the pipe has been closed via
	// p.out.Close().

	if err := wc.Close(); err != nil {
		return nil, err
	}

	// Update the object to set the metadata.
	objectAttrsToUpdate := storage.ObjectAttrsToUpdate{
		Metadata: map[string]string{
			numRowsKeyGCP: fmt.Sprintf("%d", rows),
		},
	}

	if _, err := o.Update(ctx, objectAttrsToUpdate); err != nil {
		return nil, err
	}

	s.logger.Debug().Str("file", key).Int("rows", rows).Msgf("gcp file creation complete")
	return &gcpResource{
		store: s,
		key:   key,
		rows:  rows,
	}, nil
}

func (s *gcpStore) ListFromContinuationPoint(
	ctx context.Context, table dbtable.VerifiedTable, fileName string,
) ([]Resource, error) {
	key, prefix := getKeyAndPrefix(fileName, s.bucketPath, table)
	return listFromContinuationPointGCP(ctx, stiface.AdaptClient(s.client), key, prefix, s.bucket, s)
}

func listFromContinuationPointGCP(
	ctx context.Context, client stiface.Client, key, prefix, bucket string, gcpStore *gcpStore,
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
			if utils.MatchesFileConvention(attrs.Name) {
				mdNumRows, ok := attrs.Metadata[numRowsKeyGCP]
				if !ok {
					gcpStore.logger.Error().Msgf("failed to find metadata for key %s", numRowsKeyGCP)
				}

				numRows, err := strconv.Atoi(mdNumRows)
				if err != nil {
					gcpStore.logger.Err(err).Msgf("failed to convert %s to integer", mdNumRows)
				}
				// Continue even if the integer conversion or metadata get fails because
				// file is likely still fine, but metadata was not updated properly.
				// Log to let user know.

				resources = append(resources, &gcpResource{
					store: gcpStore,
					key:   attrs.Name,
					rows:  numRows,
				})
			}
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
	rows  int
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

func (r *gcpResource) Rows() int {
	return r.rows
}

func (r *gcpResource) Reader(ctx context.Context) (io.ReadCloser, error) {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).NewReader(ctx)
}

func (r *gcpResource) MarkForCleanup(ctx context.Context) error {
	return r.store.client.Bucket(r.store.bucket).Object(r.key).Delete(ctx)
}
