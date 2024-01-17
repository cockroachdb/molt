package fetch

import (
	"context"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/testutils"
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
	testingKnobs testutils.FetchTestingKnobs,
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

	resourceWG, _ := errgroup.WithContext(ctx)
	resourceWG.SetLimit(1)

	itNum := 0
	// Errors must be buffered, as pipe can exit without taking the error channel.
	pipe := newCSVPipe(sqlRead, logger, cfg.FlushSize, cfg.FlushRows, func(numRowsCh chan int) (io.WriteCloser, error) {
		if err := resourceWG.Wait(); err != nil {
			// We need to check if the last iteration saw any error when creating
			// resource from reader. If so, just exit the current iteration.
			// Otherwise, with the error from the last iteration congesting writerErrCh,
			// the current iteration, upon seeing the same error, will hang at
			// writerErrCh <- err.
			return nil, err
		}
		forwardRead, forwardWrite := io.Pipe()
		wrappedWriter := getWriter(forwardWrite, cfg.Compression)
		resourceWG.Go(func() error {
			itNum++
			if err := func() error {
				resource, err := datasource.CreateFromReader(ctx, forwardRead, table, itNum, importFileExt, numRowsCh)
				if err != nil {
					return err
				}
				ret.Resources = append(ret.Resources, resource)
				return nil
			}(); err != nil {
				logger.Err(err).Msgf("error during data store write")
				if closeReadErr := forwardRead.CloseWithError(err); closeReadErr != nil {
					logger.Err(closeReadErr).Msgf("error closing write goroutine")
				}
				return err
			}
			return nil
		})
		return wrappedWriter, nil
	})

	// This is so we can simulate corrupted CSVs for testing.
	pipe.testingKnobs = testingKnobs
	err := pipe.Pipe(table.Name)
	if err != nil {
		return ret, err
	}
	// Wait for the resource wait group to complete. It may output an error
	// that is not captured in the pipe.
	// This is still needed though we also check the resourceWG.wait() in the
	// newWriter(), because if the error happened at the _last_ iteration,
	// we won't call newWriter() again, and hence won't reach that error check.
	// This check here is for this edge case, and is tested with single-row table
	// in TestFailedWriteToStore.
	// Note that wg.Wait() is idempotent and returns the same error if there's any,
	// see https://go.dev/play/p/dLL5v6MqZel.
	if dataStoreWriteErr := resourceWG.Wait(); dataStoreWriteErr != nil {
		return ret, dataStoreWriteErr
	}

	ret.NumRows = pipe.numRows
	ret.EndTime = time.Now()
	return ret, copyWG.Wait()
}
