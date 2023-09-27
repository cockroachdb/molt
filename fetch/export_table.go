package fetch

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type exportResult struct {
	Resources []datablobstorage.Resource
	StartTime time.Time
	EndTime   time.Time
	NumRows   int
}

func getWriter(w *io.PipeWriter, compressionType compression.Flag) io.WriteCloser {
	switch compressionType {
	case compression.GZIP:
		return newGZIPPipeWriter(w)
	}

	return w
}

func exportTable(
	ctx context.Context,
	cfg Config,
	logger zerolog.Logger,
	sqlSrc dataexport.Source,
	datasource datablobstorage.Store,
	table dbtable.VerifiedTable,
) (exportResult, error) {
	importFileExt := "csv"
	if cfg.Compression == compression.GZIP {
		importFileExt = "tar.gz"
	}

	ret := exportResult{
		StartTime: time.Now(),
	}

	cancellableCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	sqlRead, sqlWrite := io.Pipe()
	// Run the COPY TO, which feeds into the pipe, concurrently.
	copyWG, _ := errgroup.WithContext(ctx)
	copyWG.Go(func() error {
		sqlSrcConn, err := sqlSrc.Conn(ctx)
		if err != nil {
			return err
		}
		return errors.CombineErrors(
			func() error {
				if err := sqlSrcConn.Export(cancellableCtx, sqlWrite, table); err != nil {
					return errors.CombineErrors(err, sqlWrite.CloseWithError(err))
				}
				return sqlWrite.Close()
			}(),
			sqlSrcConn.Close(ctx),
		)
	})

	var resourceWG sync.WaitGroup
	itNum := 0
	// Errors must be buffered, as pipe can exit without taking the error channel.
	writerErrCh := make(chan error, 1)
	pipe := newCSVPipe(sqlRead, logger, cfg.FlushSize, cfg.FlushRows, func() io.WriteCloser {
		resourceWG.Wait()
		forwardRead, forwardWrite := io.Pipe()
		wrappedWriter := getWriter(forwardWrite, cfg.Compression)
		resourceWG.Add(1)
		go func() {
			defer resourceWG.Done()
			itNum++
			if err := func() error {
				resource, err := datasource.CreateFromReader(ctx, forwardRead, table, itNum, importFileExt)
				if err != nil {
					return err
				}
				ret.Resources = append(ret.Resources, resource)
				return nil
			}(); err != nil {
				logger.Err(err).Msgf("error during data store write")
				if err := forwardRead.CloseWithError(err); err != nil {
					logger.Err(err).Msgf("error closing write goroutine")
				}
				writerErrCh <- err
			}
		}()
		return wrappedWriter
	})

	err := pipe.Pipe(table.Name)
	// Wait for the resource wait group to complete. It may output an error
	// that is not captured in the pipe.
	resourceWG.Wait()
	// Check any errors are not left behind - this can happen if the data source
	// creation fails, but the COPY is already done.
	select {
	case werr := <-writerErrCh:
		if werr != nil {
			cancelFunc()
			err = errors.CombineErrors(err, werr)
		}
	default:
	}
	if err != nil {
		// We do not wait for COPY to complete - we're already in trouble.
		return ret, err
	}

	ret.NumRows = pipe.numRows
	ret.EndTime = time.Now()
	return ret, copyWG.Wait()
}
