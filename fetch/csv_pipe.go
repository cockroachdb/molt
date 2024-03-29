package fetch

import (
	"encoding/csv"
	"io"

	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/fetchmetrics"
	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/testutils"
	"github.com/rs/zerolog"
)

type csvPipe struct {
	in io.Reader

	csvWriter *csv.Writer
	out       io.WriteCloser
	logger    zerolog.Logger

	flushSize int
	flushRows int
	currSize  int
	currRows  int
	numRows   int
	shardNum  int
	numRowsCh chan int
	newWriter func(numRowsCh chan int) (io.WriteCloser, error)

	testingKnobs testutils.FetchTestingKnobs
}

func newCSVPipe(
	in io.Reader,
	logger zerolog.Logger,
	flushSize int,
	flushRows int,
	shardNum int,
	newWriter func(numRowsCh chan int) (io.WriteCloser, error),
) *csvPipe {
	return &csvPipe{
		in:        in,
		logger:    logger,
		flushSize: flushSize,
		flushRows: flushRows,
		shardNum:  shardNum,
		numRowsCh: make(chan int, 1),
		newWriter: newWriter,
	}
}

// Pipe is responsible for reading data from the SQL Pipe
// and creating a CSV file. The CSV data is either flushed
// when max rows or max file size is reached. The output
// is written to another pipe which is used as an io.Reader.
func (p *csvPipe) Pipe(tn dbtable.Name) error {
	r := csv.NewReader(p.in)
	r.ReuseRecord = true
	m := fetchmetrics.ExportedRows.WithLabelValues(tn.SafeString())
	dataLogger := moltlogger.GetDataLogger(p.logger)
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return p.flush()
			}
			return err
		}
		if err := p.maybeInitWriter(); err != nil {
			return err
		}
		p.currRows++
		p.numRows++
		m.Inc()
		if p.numRows%100000 == 0 {
			dataLogger.Info().
				Str("table", tn.SafeString()).
				Int("num_rows", p.numRows).
				Int("shard", p.shardNum).
				Msgf("row import status")
		}
		for _, s := range record {
			p.currSize += len(s) + 1
		}
		if err := p.csvWriter.Write(record); err != nil {
			return err
		}

		if p.testingKnobs.TriggerCorruptCSVFile {
			if err := p.csvWriter.Write([]string{"this", "should", "lead", "to", "an", "error"}); err != nil {
				return err
			}
		}

		if p.currSize > p.flushSize || (p.flushRows > 0 && p.currRows >= p.flushRows) {
			if err := p.flush(); err != nil {
				return err
			}
		}
	}
}

// Flush flushes the current csv files when either
// we reached the end of the file or our file limits.
// It also sends the current number of rows processed
// to a channel to be processed by the data storage
// backend.
func (p *csvPipe) flush() error {
	if p.csvWriter != nil {
		p.numRowsCh <- p.currRows
		p.csvWriter.Flush()
		if err := p.out.Close(); err != nil {
			return err
		}
	}
	p.currSize = 0
	p.currRows = 0
	p.out = nil
	p.csvWriter = nil
	return nil
}

func (p *csvPipe) maybeInitWriter() error {
	if p.csvWriter == nil {
		out, err := p.newWriter(p.numRowsCh)
		if err != nil {
			return err
		}
		p.out = out
		p.csvWriter = csv.NewWriter(p.out)
	}
	return nil
}
