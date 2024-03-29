package dataexport

import (
	"context"
	"encoding/csv"
	"io"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/rowiterator"
	"github.com/cockroachdb/molt/verify/rowverify"
)

type Source interface {
	CDCCursor() string
	Conn(ctx context.Context) (SourceConn, error)
	Close(ctx context.Context) error
}

type SourceConn interface {
	Export(ctx context.Context, writer io.Writer, table dbtable.VerifiedTable, shard rowverify.TableShard) error
	Close(ctx context.Context) error
}

type Settings struct {
	RowBatchSize int

	PG PGReplicationSlotSettings
}

func InferExportSource(ctx context.Context, settings Settings, conn dbconn.Conn) (Source, error) {
	switch conn := conn.(type) {
	case *dbconn.PGConn:
		if conn.IsCockroach() {
			return NewCRDBSource(ctx, settings, conn)
		}
		return NewPGSource(ctx, settings, conn)
	case *dbconn.MySQLConn:
		return NewMySQLSource(ctx, settings, conn)
	}
	return nil, errors.AssertionFailedf("unknown conn type: %T", conn)
}

func scanWithRowIterator(
	ctx context.Context,
	settings Settings,
	c dbconn.Conn,
	writer io.Writer,
	table rowiterator.ScanTable,
) error {
	cw := csv.NewWriter(writer)
	it, err := rowiterator.NewScanIterator(
		ctx,
		c,
		table,
		settings.RowBatchSize,
		nil,
	)
	if err != nil {
		return err
	}
	strings := make([]string, 0, len(table.ColumnNames))
	for it.HasNext(ctx) {
		strings = strings[:0]
		datums := it.Next(ctx)
		var fmtFlags tree.FmtFlags
		for _, d := range datums {
			// FmtPgwireText is needed so that null columns get written as "" instead of string NULL
			// which happens inside f.FormatNode for type dNull datums.
			switch d.(type) {
			case *tree.DFloat:
				// With tree.FmtParsableNumerics, negative value will be bracketed, making it unable to be imported from
				// csv.
				fmtFlags = tree.FmtExport | tree.FmtPgwireText
			default:
				fmtFlags = tree.FmtExport | tree.FmtParsableNumerics | tree.FmtPgwireText
			}
			f := tree.NewFmtCtx(fmtFlags)
			f.FormatNode(d)
			strings = append(strings, f.CloseAndGetString())
		}
		if err := cw.Write(strings); err != nil {
			return err
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	cw.Flush()
	return nil
}
