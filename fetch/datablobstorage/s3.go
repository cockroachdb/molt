package datablobstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// AWS capitalizes the first letter of the string and lowercases the rest, hence
// the naming discrepancy between this and GCP.
const numRowKeysAWS = "Numrows"

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
	utils.RedactedQueryParams = map[string]struct{}{utils.AWSSecretAccessKey: {}}
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
	testingKnobs testutils.FetchTestingKnobs,
) (Resource, error) {
	key := fmt.Sprintf("%s/part_%08d.%s", table.SafeString(), iteration, fileExt)
	if s.bucketPath != "" {
		key = fmt.Sprintf("%s/%s/part_%08d.%s", s.bucketPath, table.SafeString(), iteration, fileExt)
	}

	s.logger.Debug().Str("file", key).Msgf("creating new file")

	bucketName := s.bucket

	var uploader s3manageriface.UploaderAPI

	if testingKnobs.FailedWriteToBucket.FailedAfterReadFromPipe {
		uploader = &s3UploaderMock{}
	} else {
		uploader = s3manager.NewUploader(s.session)
	}

	rows := <-numRows
	numRowStr := fmt.Sprintf("%d", rows)

	if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		Body:     r,
		Metadata: map[string]*string{numRowKeysAWS: &numRowStr},
	}); err != nil {
		return nil, err
	}

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

	g, _ := errgroup.WithContext(ctx)
	resources := make([]Resource, len(s3Objects.Contents))
	for i, obj := range s3Objects.Contents {
		curI := i
		curObj := obj
		// Find the key we want to start at. Because we name the files
		// in a specific pattern, we can guarantee lexicographical ordering
		// based on the guarantee of return order from the S3 API.
		// eg. If key = fetch/public.inventory/part_00000004.tar.gz,
		// fetch/public.inventory/part_00000005.tar.gz is >= to key meaning,
		// it is a file we need to include.
		g.Go(func() error {
			objResp, err := s3Client.GetObject(&s3.GetObjectInput{
				Bucket: aws.String(s3Store.bucket),
				Key:    aws.String(*curObj.Key),
			})
			if err != nil {
				return err
			}

			numRows := 0
			mdNumRows, ok := objResp.Metadata[numRowKeysAWS]
			if !ok {
				s3Store.logger.Error().Msgf("failed to find metadata for key %s", numRowKeysAWS)
			} else {
				numRows, err = strconv.Atoi(*mdNumRows)
				if err != nil {
					s3Store.logger.Err(err).Msgf("failed to convert %s to integer", *mdNumRows)
				}
			}

			// Continue even if the integer conversion or metadata get fails because
			// file is likely still fine, but metadata was not updated properly.
			// Log to let user know.

			if aws.StringValue(curObj.Key) >= key && utils.MatchesFileConvention(aws.StringValue(curObj.Key)) {
				resources[curI] = &s3Resource{
					key:     aws.StringValue(curObj.Key),
					session: s3Store.session,
					store:   s3Store,
					rows:    numRows,
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return removeNilResources(resources), nil
}

func removeNilResources(input []Resource) []Resource {
	output := []Resource{}
	for _, res := range input {
		if res != nil {
			output = append(output, res)
		}
	}

	return output
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
