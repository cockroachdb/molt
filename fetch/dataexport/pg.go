package dataexport

import (
	"context"
	"fmt"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
	"github.com/cockroachdb/molt/rowiterator"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgSource struct {
	tx         pgx.Tx
	conn       dbconn.Conn
	settings   Settings
	snapshotID string
	cdcCursor  string
}

type PGReplicationSlotSettings struct {
	SlotName     string
	Plugin       string
	DropIfExists bool
}

func NewPGSource(ctx context.Context, settings Settings, conn *dbconn.PGConn) (*pgSource, error) {
	var cdcCursor string
	if settings.PG.SlotName != "" {
		if settings.PG.DropIfExists {
			if _, err := conn.Exec(ctx, "SELECT pg_drop_replication_slot($1)", settings.PG.SlotName); err != nil {
				if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code != "42704" {
					return nil, errors.Wrap(err, "error attempting to drop replication slot")
				}
			}
		}
		if err := conn.QueryRow(
			ctx,
			"SELECT lsn FROM pg_create_logical_replication_slot($1, $2)",
			settings.PG.SlotName,
			settings.PG.Plugin,
		).Scan(&cdcCursor); err != nil {
			return nil, errors.Wrap(err, "error creating replication slot")
		}
	} else {
		if err := conn.QueryRow(ctx, "SELECT pg_current_wal_insert_lsn()").Scan(&cdcCursor); err != nil {
			return nil, errors.Wrap(err, "failed to export wal LSN")
		}
	}
	// Keep tx with snapshot open to establish a consistent snapshot.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, err
	}
	var snapshotID string
	if err := func() error {
		if err := tx.QueryRow(ctx, "SELECT pg_export_snapshot()").Scan(&snapshotID); err != nil {
			return errors.Wrap(err, "failed to export snapshot")
		}
		return nil
	}(); err != nil {
		return nil, errors.CombineErrors(err, tx.Rollback(ctx))
	}
	return &pgSource{
		snapshotID: snapshotID,
		cdcCursor:  cdcCursor,
		settings:   settings,
		tx:         tx,
		conn:       conn,
	}, nil
}

func (p *pgSource) Close(ctx context.Context) error {
	return p.tx.Rollback(ctx)
}

func (p *pgSource) CDCCursor() string {
	return p.cdcCursor
}

func (p *pgSource) Conn(ctx context.Context) (SourceConn, error) {
	conn, err := p.conn.Clone(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := conn.(*dbconn.PGConn).BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, errors.CombineErrors(err, conn.Close(ctx))
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET TRANSACTION SNAPSHOT '%s'", p.snapshotID)); err != nil {
		return nil, errors.CombineErrors(err, conn.Close(ctx))
	}
	return &pgSourceConn{
		conn: conn,
		tx:   tx,
		src:  p,
	}, nil
}

type pgSourceConn struct {
	conn dbconn.Conn
	tx   pgx.Tx
	src  *pgSource
}

func (p *pgSourceConn) Export(
	ctx context.Context, writer io.Writer, table dbtable.VerifiedTable, shard rowverify.TableShard,
) error {

	expMode := ctx.Value("exportmode")
	if expMode == nil {
		return errors.New("the export mode is not found")
	}

	switch expMode {
	case testutils.ExportWithSelect:
		fmt.Println("select mode on!!")
		return scanWithRowIterator(ctx, p.src.settings, p.conn, writer, rowiterator.ScanTable{
			Table: rowiterator.Table{
				Name:              table.Name,
				ColumnNames:       table.Columns,
				ColumnOIDs:        table.ColumnOIDs[0],
				PrimaryKeyColumns: table.PrimaryKeyColumns,
			},
			StartPKVals: shard.StartPKVals,
			EndPKVals:   shard.EndPKVals,
		})
	case testutils.ExportWithCopy:
		fmt.Println("copy mode on!!")
		if _, err := p.tx.Conn().PgConn().CopyTo(
			ctx,
			writer,
			dataquery.NewPGCopyTo(table, shard.StartPKVals, shard.EndPKVals),
		); err != nil {
			return err
		}
		return nil
	default:
		return errors.Newf("unknown export mode: %d", expMode)
	}

}

func (p *pgSourceConn) Close(ctx context.Context) error {
	return p.conn.Close(ctx)
}
