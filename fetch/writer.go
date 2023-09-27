package fetch

import (
	"compress/gzip"
	"io"
)

// compressedPipeWriter wraps around the underlying pipe writer and
// gzip writer so that we can access both to close them.
type compressedPipeWriter struct {
	pipeWriter        *io.PipeWriter
	compressionWriter io.WriteCloser
}

// Need custom close method so that we properly close both the pipe and gzip writer.
// This prevents leaks of file descriptors and io pipes.
func (c *compressedPipeWriter) Close() error {
	if err := c.compressionWriter.Close(); err != nil {
		return err
	}

	return c.pipeWriter.Close()
}

// The gzip writer is responsible for writing the data to the compressed file,
// since it wraps around the pipeWriter.
func (c *compressedPipeWriter) Write(p []byte) (n int, err error) {
	return c.compressionWriter.Write(p)
}

func newGZIPPipeWriter(w *io.PipeWriter) *compressedPipeWriter {
	return &compressedPipeWriter{
		pipeWriter:        w,
		compressionWriter: gzip.NewWriter(w),
	}
}
