package datablobstorage

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
)

func TestGCPResource_ImportURL(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		r        *gcpResource
		expected string
	}{
		{
			desc: "basic test",
			r: &gcpResource{
				key: "asdf/ghjk.csv",
				store: &gcpStore{
					bucket: "nangs",
					creds: &google.Credentials{
						JSON: []byte(`{"a":b}`),
					},
				},
			},
			expected: "gs://nangs/asdf/ghjk.csv?CREDENTIALS=eyJhIjpifQ==",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := tc.r.ImportURL()
			require.NoError(t, err)
			require.Equal(t, tc.expected, u)
		})
	}
}

type gcpClientMock struct {
	stiface.Client
	mock.Mock
}
type gcpBucketMock struct {
	stiface.BucketHandle
	mock.Mock
}
type gcpObjectITMock struct {
	stiface.ObjectIterator
	i    int
	next []storage.ObjectAttrs
}

func (m *gcpClientMock) Bucket(name string) stiface.BucketHandle {
	args := m.Called(name)
	return args.Get(0).(*gcpBucketMock)
}

func (m *gcpBucketMock) Objects(ctx context.Context, q *storage.Query) (it stiface.ObjectIterator) {
	args := m.Called(ctx, q)
	return args.Get(0).(*gcpObjectITMock)
}

func (it *gcpObjectITMock) Next() (a *storage.ObjectAttrs, err error) {
	if it.i == len(it.next) {
		err = iterator.Done
		return
	}

	a = &it.next[it.i]
	it.i += 1
	return
}

func TestListFromContinuationPointGCP(t *testing.T) {
	gcpClient := &gcpClientMock{}
	bucketMock := &gcpBucketMock{}
	ctx := context.Background()

	// Set up the mock expected return
	gcpClient.On("Bucket", mock.Anything).Return(bucketMock)
	bucketMock.On("Objects", ctx, mock.Anything).
		Return(&gcpObjectITMock{
			i: 0,
			next: []storage.ObjectAttrs{
				{Name: "part_00000004.tar.gz"},
				{Name: "part_00000005.tar.gz"},
				{Name: "part_00000006.tar.gz"},
				{Name: "part_00000007.tar.gz"},
				{Name: "part_00000008.tar.gz"},
			},
		})

	gcpStore := gcpStore{
		bucket: "fetch-test",
		creds: &google.Credentials{
			JSON: []byte(`{"a":b}`),
		},
	}
	resources, err := listFromContinuationPointGCP(ctx, gcpClient, "part_00000004.tar.gz", "public.inventory", gcpStore.bucket)
	require.NoError(t, err)
	require.Len(t, resources, 5)
}
