package status

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func CreateStatusAndExceptionTables(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.Exec(ctx, createStatusTable); err != nil {
		return err
	}

	if _, err := conn.Exec(ctx, createExceptionsTable); err != nil {
		return err
	}

	return nil
}
