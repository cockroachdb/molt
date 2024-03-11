package datablobstorage

import (
	"bytes"
	"context"
	"fmt"
	"sort"
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
			Key:      aws.String(fmt.Sprintf("public.inventory/shard_%02d_part_%08d.tar.gz", 1, i)),
			Body:     bytes.NewReader([]byte("abcde")),
			Bucket:   aws.String("fetch-test"),
			Metadata: map[string]*string{numRowKeysAWS: aws.String("5")},
		})
		require.NoError(t, err)
	}

	// List from file 4 which should result in files 4-8 inclusive
	resources, err := listFromContinuationPointAWS(ctx, s3Cli, "public.inventory/shard_01_part_00000004.tar.gz", "public.inventory", &s3Store, 10)
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
			Key:      aws.String(fmt.Sprintf("public.inventory/shard_%02d_part_%08d.tar.gz", 1, i)),
			Body:     bytes.NewReader([]byte("abcde")),
			Bucket:   aws.String("fetch-test-paginate"),
			Metadata: map[string]*string{numRowKeysAWS: aws.String("5")},
		})
		require.NoError(t, err)
	}

	// List from file 13 and ensure pagination worked as expected
	resources, err := listFromContinuationPointAWS(ctx, s3Cli, "public.inventory/shard_01_part_00000013.tar.gz", "public.inventory", &s3Store, 5)
	require.NoError(t, err)
	require.Equal(t, 8, len(resources))
}

func TestListFromContinuationPointAWSMultiShard(t *testing.T) {
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
		bucket: "fetch-test-shards",
		logger: zerolog.New(&sb),
	}

	// Create the test bucket
	_, err = s3Cli.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("fetch-test-shards"),
	})
	require.NoError(t, err)

	// Seed the initial data with 8 files
	for i := 1; i <= 4; i++ {
		for j := 1; j <= 8; j++ {
			_, err := s3Cli.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Key:      aws.String(fmt.Sprintf("public.inventory/shard_%02d_part_%08d.tar.gz", i, j)),
				Body:     bytes.NewReader([]byte("abcde")),
				Bucket:   aws.String("fetch-test-shards"),
				Metadata: map[string]*string{numRowKeysAWS: aws.String("5")},
			})
			require.NoError(t, err)
		}
	}

	// List from shard 2 file 4 which should result in files from shard 2 4-8, shard 3 1-8, and shard 4 1-8
	// 5,8,8 files respetively from each shard
	resources, err := listFromContinuationPointAWS(ctx, s3Cli, "public.inventory/shard_02_part_00000004.tar.gz", "public.inventory", &s3Store, 10)
	require.NoError(t, err)
	require.Equal(t, 21, len(resources))

	// Check the ordering of the returned slice is correct.
	// Lexicographically each successive file should be greater
	// that the previous one
	require.True(t, sort.SliceIsSorted(resources, func(i, j int) bool {
		l, _ := resources[i].Key()
		r, _ := resources[j].Key()
		return l < r
	}))
}
