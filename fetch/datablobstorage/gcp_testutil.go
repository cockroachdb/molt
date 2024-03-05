package datablobstorage

import (
	"cloud.google.com/go/storage"
	"github.com/cockroachdb/errors"
)

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
