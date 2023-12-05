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
)

type Source interface {
	CDCCursor() string
	Conn(ctx context.Context) (SourceConn, error)
	Close(ctx context.Context) error
}

type SourceConn interface {
	Export(ctx context.Context, writer io.Writer, table dbtable.VerifiedTable) error
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
		for _, d := range datums {
			f := tree.NewFmtCtx(tree.FmtExport | tree.FmtParsableNumerics)
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
