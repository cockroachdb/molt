package datablobstorage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestListFromContinuationPointAWS(t *testing.T) {
	ctx := context.Background()
	var sb strings.Builder
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String("http://s3.localhost.localstack.cloud:4566"),
		Region:           aws.String("us-east-1"),
	})
	require.NoError(t, err)
	s3Cli := s3.New(sess)

	s3Store := s3Store{
		bucket: "fetch-test",
		logger: zerolog.New(&sb),
	}

	// Create the test bucket
	_, err = s3Cli.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("fetch-test"),
	})
	require.NoError(t, err)

	// Seed the initial data with 8 files
	for i := 1; i <= 8; i++ {
		_, err := s3Cli.PutObjectWithContext(ctx, &s3.PutObjectInput{
			Key:      aws.String(fmt.Sprintf("public.inventory/part_0000000%d.tar.gz", i)),
			Body:     bytes.NewReader([]byte("abcde")),
			Bucket:   aws.String("fetch-test"),
			Metadata: map[string]*string{numRowKeysAWS: aws.String("5")},
		})
		require.NoError(t, err)
	}

	// List from file 4 which should result in files 4-8 inclusive
	resources, err := listFromContinuationPointAWS(ctx, s3Cli, "public.inventory/part_00000004.tar.gz", "public.inventory", &s3Store, 10)
	require.NoError(t, err)
	require.Equal(t, 5, len(resources))
}

func TestListFromContinuationPointAWSPagination(t *testing.T) {
	ctx := context.Background()
	var sb strings.Builder
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String("http://s3.localhost.localstack.cloud:4566"),
		Region:           aws.String("us-east-1"),
	})
	require.NoError(t, err)
	s3Cli := s3.New(sess)

	s3Store := s3Store{
		bucket: "fetch-test-paginate",
		logger: zerolog.New(&sb),
	}

	// Create the test bucket
	_, err = s3Cli.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("fetch-test-paginate"),
	})
	require.NoError(t, err)

	// Seed the initial data with 20 files
	for i := 1; i <= 20; i++ {
		_, err := s3Cli.PutObjectWithContext(ctx, &s3.PutObjectInput{
			Key:      aws.String(fmt.Sprintf("public.inventory/part_%08d.tar.gz", i)),
			Body:     bytes.NewReader([]byte("abcde")),
			Bucket:   aws.String("fetch-test-paginate"),
			Metadata: map[string]*string{numRowKeysAWS: aws.String("5")},
		})
		require.NoError(t, err)
	}

	// List from file 13 and ensure pagination worked as expected
	resources, err := listFromContinuationPointAWS(ctx, s3Cli, "public.inventory/part_00000013.tar.gz", "public.inventory", &s3Store, 5)
	require.NoError(t, err)
	require.Equal(t, 8, len(resources))
}
