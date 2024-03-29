package fetch

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type exportResult struct {
	Resources []datablobstorage.Resource
	StartTime time.Time
	EndTime   time.Time
	NumRows   int
}

func getWriter(w *Pipe, compressionType compression.Flag) io.WriteCloser {
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
	shard rowverify.TableShard,
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
	// We use the standard io.Pipe here because during data
	// export since if we use a standard bytes.Buffer it will grow
	// to the size of the data export and cause an OOMKill.
	// Use the default size of 4096 created by bufio so that
	// the memory usage on export remains constant.
	sqlRead, sqlWrite := io.Pipe()
	// Run the COPY TO, which feeds into the pipe, concurrently.
	copyWG, _ := errgroup.WithContext(ctx)
	copyWG.Go(func() error {
		sqlSrcConn, err := sqlSrc.Conn(ctx)
		if testingKnobs.FailedEstablishSrcConnForExport {
			err = errors.Newf("forced error when establishing conn for export")
		}
		if err != nil {
			return sqlWrite.CloseWithError(err)
		}
		return errors.CombineErrors(
			func() error {
				if err := sqlSrcConn.Export(cancellableCtx, sqlWrite, table, shard); err != nil {
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
	pipe := newCSVPipe(sqlRead, logger, cfg.FlushSize, cfg.FlushRows, shard.ShardNum, func(numRowsCh chan int) (io.WriteCloser, error) {
		if err := resourceWG.Wait(); err != nil {
			// We need to check if the last iteration saw any error when creating
			// resource from reader. If so, just exit the current iteration.
			// Otherwise, with the error from the last iteration congesting writerErrCh,
			// the current iteration, upon seeing the same error, will hang at
			// writerErrCh <- err.
			return nil, err
		}
		fbuf := new(bytes.Buffer)
		fRW := NewPipe(fbuf)
		wrappedWriter := getWriter(fRW, cfg.Compression)
		resourceWG.Go(func() error {
			itNum++
			if err := func() error {
				resource, err := datasource.CreateFromReader(ctx, fRW, table, itNum, importFileExt, numRowsCh, testingKnobs, shard.ShardNum)
				if err != nil {
					return err
				}
				ret.Resources = append(ret.Resources, resource)
				return nil
			}(); err != nil {
				logger.Err(err).Msgf("error during data store write")
				if closeReadErr := fRW.CloseWithError(err); closeReadErr != nil {
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
