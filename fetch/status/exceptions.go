package status

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/fetchcontext"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

const (
	StageSchemaCreation = "schema_creation"
	StageDataLoad       = "data_load"
)

const createExceptionsTable = `CREATE TABLE IF NOT EXISTS _molt_fetch_exception (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	fetch_id UUID NOT NULL REFERENCES _molt_fetch_status (id),
    table_name STRING,
    schema_name STRING,
    message STRING,
    sql_state STRING,
    file_name STRING,
    command STRING,
	stage STRING,
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
	SQLState string
	FileName string
	Command  string
	Stage    string
	Time     time.Time
}

func (e *ExceptionLog) CreateEntry(ctx context.Context, conn *pgx.Conn, stage string) error {
	curTime := time.Now().UTC()
	query := `INSERT INTO _molt_fetch_exception (fetch_id, table_name, schema_name, message, sql_state, file_name, command, stage, time) VALUES(@fetch_id, @table_name, @schema_name, @message, @sql_state, @file_name, @command, @stage, @time) RETURNING id, stage`
	args := pgx.NamedArgs{
		"fetch_id":    e.FetchID,
		"table_name":  e.Table,
		"schema_name": e.Schema,
		"message":     e.Message,
		"command":     e.Command,
		"time":        curTime,
		"stage":       stage,
	}

	if e.SQLState != "" {
		args["sql_state"] = e.SQLState
	}

	if strings.TrimSpace(e.FileName) != "" {
		args["file_name"] = e.FileName
	}

	row := conn.QueryRow(ctx, query, args)

	if err := row.Scan(&e.ID, &e.Stage); err != nil {
		return err
	}
	e.Time = curTime

	return nil
}

// Stands for undefined_object.
const excludedSqlState = "42704"

func MaybeReportException(
	ctx context.Context,
	logger zerolog.Logger,
	conn *pgx.Conn,
	table dbtable.Name,
	inputErr error,
	fileName string,
	stage string,
) error {
	fetchContext := fetchcontext.GetFetchContextData(ctx)

	// Parse out error as pg.
	// If it's not a PG error or the wrong type, then we exit early
	// and do not report this.
	pgErr := (*pgconn.PgError)(nil)
	if !errors.As(inputErr, &pgErr) || pgErr.Code == excludedSqlState {
		return inputErr
	}

	errMsg := fmt.Sprintf("%s; %s", pgErr.Message, pgErr.Detail)
	sqlState := pgErr.Code
	createdAt := time.Now().UTC()

	// File name currently undetermined,
	// see if we can extract from the error message.
	if fileName == "" {
		fileName = extractFileNameFromErr(errMsg)
	}

	exceptionLog := ExceptionLog{
		FetchID:  fetchContext.RunID,
		Table:    table.Table.String(),
		Schema:   table.Schema.String(),
		Message:  errMsg,
		SQLState: sqlState,
		FileName: fileName,
		Time:     createdAt,
	}

	// TODO: figure out to deduplicate or upsert and do nothing if conflict.
	if err := exceptionLog.CreateEntry(ctx, conn, stage); err != nil {
		logger.Err(err).Send()
		return errors.CombineErrors(inputErr, err)
	}

	return inputErr
}

var fileNameRegEx = regexp.MustCompile(`(part.+csv?)`)

func extractFileNameFromErr(errString string) string {
	return fileNameRegEx.FindString(errString)
}
