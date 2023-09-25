package fetch

import (
	"encoding/csv"
	"io"

	"github.com/cockroachdb/molt/dbtable"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	newWriter func() io.WriteCloser
}

var (
	importedRows = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "molt",
		Subsystem: "fetch",
		Name:      "rows_imported",
		Help:      "Number of rows that have been imported in",
	}, []string{"table"})
)

func newCSVPipe(
	in io.Reader,
	logger zerolog.Logger,
	flushSize int,
	flushRows int,
	newWriter func() io.WriteCloser,
) *csvPipe {
	return &csvPipe{
		in:        in,
		logger:    logger,
		flushSize: flushSize,
		flushRows: flushRows,
		newWriter: newWriter,
	}
}

func (p *csvPipe) Pipe(tn dbtable.Name) error {
	r := csv.NewReader(p.in)
	r.ReuseRecord = true
	m := importedRows.WithLabelValues(tn.SafeString())
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return p.flush()
			}
			return err
		}
		p.maybeInitWriter()
		p.currRows++
		p.numRows++
		m.Inc()
		if p.numRows%100000 == 0 {
			p.logger.Info().Int("num_rows", p.numRows).Msgf("row import status")
		}
		for _, s := range record {
			p.currSize += len(s) + 1
		}
		if err := p.csvWriter.Write(record); err != nil {
			return err
		}

		if p.currSize > p.flushSize || (p.flushRows > 0 && p.currRows >= p.flushRows) {
			if err := p.flush(); err != nil {
				return err
			}
		}
	}
}

func (p *csvPipe) flush() error {
	if p.csvWriter != nil {
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

func (p *csvPipe) maybeInitWriter() {
	if p.csvWriter == nil {
		p.out = p.newWriter()
		p.csvWriter = csv.NewWriter(p.out)
	}
}
