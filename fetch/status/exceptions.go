package status

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/jackc/pgx/v5"
)

const createExceptionsTable = `CREATE TABLE IF NOT EXISTS _molt_fetch_exception (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	fetch_id UUID NOT NULL REFERENCES _molt_fetch_status (id),
    table_name STRING,
    schema_name STRING,
    message STRING,
    sql_state INT,
    file_name STRING,
    command STRING,
    time TIMESTAMP,
	INDEX(fetch_id, sql_state)
);
`

type ExceptionLog struct {
	ID       uuid.UUID
	FetchID  uuid.UUID
	Table    string
	Schema   string
	Message  string
	SQLState int
	FileName string
	Command  string
	Time     time.Time
}

func (e *ExceptionLog) CreateEntry(ctx context.Context, conn *pgx.Conn) error {
	curTime := time.Now().UTC()
	query := `INSERT INTO _molt_fetch_exception (fetch_id, table_name, schema_name, message, sql_state, file_name, command, time) VALUES(@fetch_id, @table_name, @schema_name, @message, @sql_state, @file_name, @command, @time) RETURNING id`
	args := pgx.NamedArgs{
		"fetch_id":    e.FetchID,
		"table_name":  e.Table,
		"schema_name": e.Schema,
		"message":     e.Message,
		"command":     e.Command,
		"time":        curTime,
	}

	if e.SQLState > 0 {
		args["sql_state"] = e.SQLState
	}

	if strings.TrimSpace(e.FileName) != "" {
		args["file_name"] = e.FileName
	}

	row := conn.QueryRow(ctx, query, args)

	if err := row.Scan(&e.ID); err != nil {
		return err
	}
	e.Time = curTime

	return nil
}
