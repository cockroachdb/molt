package datablobstorage

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/cockroachdb/errors"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"
)

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

// GCPStorageWriterMock is to mock a gcp storage that always fail to upload to
// the bucket. We use it to simulate a disastrous edge case and ensure that
// the error in this case would be properly propagated.
type GCPStorageWriterMock struct {
	*storage.Writer
}

const GCPWriterMockErrMsg = "forced error for gcp storage writer"

func (w *GCPStorageWriterMock) Write(p []byte) (n int, err error) {
	return 0, errors.Newf(GCPWriterMockErrMsg)
}
