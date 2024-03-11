package datablobstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestListFromContinuationPointGCP(t *testing.T) {
	ctx := context.Background()
	var sb strings.Builder
	gcpClient, err := storage.NewClient(ctx,
		option.WithEndpoint("http://localhost:4443/storage/v1/"),
		option.WithoutAuthentication(),
	)

	require.NoError(t, err)

	gcpStore := gcpStore{
		bucket: "fetch-test",
		logger: zerolog.New(&sb),
	}

	// Create the test bucket
	err = gcpClient.Bucket("fetch-test").Create(ctx, "", nil)
	require.NoError(t, err)

	// Seed the initial data with 8 files
	for i := 1; i <= 8; i++ {
		o := gcpClient.Bucket("fetch-test").Object(fmt.Sprintf("public.inventory/shard_%02d_part_%08d.tar.gz", 1, i))
		wc := o.NewWriter(ctx)
		_, err = io.Copy(wc, bytes.NewReader([]byte("abcde")))
		require.NoError(t, err)
		require.NoError(t, wc.Close())
	}

	// List from file 4 which should result in files 4-8 inclusive
	resources, err := listFromContinuationPointGCP(ctx, gcpClient, "public.inventory/shard_01_part_00000004.tar.gz", "public.inventory", "fetch-test", &gcpStore)
	require.NoError(t, err)
	require.Equal(t, 5, len(resources))
}

func TestListFromContinuationPointGCPMultiShard(t *testing.T) {
	ctx := context.Background()
	var sb strings.Builder
	gcpClient, err := storage.NewClient(ctx,
		option.WithEndpoint("http://localhost:4443/storage/v1/"),
		option.WithoutAuthentication(),
	)

	require.NoError(t, err)

	gcpStore := gcpStore{
		bucket: "fetch-test-shards",
		logger: zerolog.New(&sb),
	}

	// Create the test bucket
	err = gcpClient.Bucket("fetch-test-shards").Create(ctx, "", nil)
	require.NoError(t, err)

	// Seed the initial data with 8 files
	for i := 1; i <= 4; i++ {
		for j := 1; j <= 8; j++ {
			o := gcpClient.Bucket("fetch-test-shards").Object(fmt.Sprintf("public.inventory/shard_%02d_part_%08d.tar.gz", i, j))
			wc := o.NewWriter(ctx)
			_, err = io.Copy(wc, bytes.NewReader([]byte("abcde")))
			require.NoError(t, err)
			require.NoError(t, wc.Close())
		}
	}

	// List from shard 2 file 4 which should result in files from shard 2 4-8, shard 3 1-8, and shard 4 1-8
	// 5,8,8 files respetively from each shard
	resources, err := listFromContinuationPointGCP(ctx, gcpClient, "public.inventory/shard_02_part_00000004.tar.gz", "public.inventory", "fetch-test-shards", &gcpStore)
	require.NoError(t, err)
	require.Equal(t, 21, len(resources))
}
