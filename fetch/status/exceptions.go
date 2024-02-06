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

// Added logic to be able to set the time from the struct itself to simplify testing.
// For some of the tests, we must test exceptions created at multiple times in order
// to properly validate behavior. Testutils.hook for date doesn't work since all
// entries would end up having the same.
// In the actual use case (non-testing) the time will be generated as the current time
// as the logic below shows.
func (e *ExceptionLog) CreateEntry(ctx context.Context, conn *pgx.Conn, stage string) error {
	curTime := time.Now().UTC()
	if !e.Time.IsZero() {
		curTime = e.Time
	}

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

func GetExceptionLogByToken(
	ctx context.Context, conn *pgx.Conn, token string,
) (*ExceptionLog, error) {
	query := `SELECT id, fetch_id, table_name, schema_name, message, sql_state, file_name, command, stage, time 
		FROM _molt_fetch_exception 
		WHERE id=@id`
	args := pgx.NamedArgs{
		"id": token,
	}
	e := &ExceptionLog{}

	row := conn.QueryRow(ctx, query, args)
	if err := row.Scan(&e.ID, &e.FetchID, &e.Table, &e.Schema, &e.Message,
		&e.SQLState, &e.FileName, &e.Command, &e.Stage, &e.Time); err != nil {
		return nil, err
	}

	return e, nil
}

func GetAllExceptionLogsByFetchID(
	ctx context.Context, conn *pgx.Conn, fetchID string,
) ([]*ExceptionLog, error) {
	query := `SELECT DISTINCT ON (schema_name, table_name) id, fetch_id, table_name, schema_name, message, sql_state, file_name, command, stage, time 
	FROM _molt_fetch_exception 
	WHERE fetch_id=@fetch_id 
	ORDER BY schema_name, table_name,  time DESC`
	args := pgx.NamedArgs{
		"fetch_id": fetchID,
	}
	excLogs := []*ExceptionLog{}

	rows, err := conn.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := &ExceptionLog{}
		if err := rows.Scan(&e.ID, &e.FetchID, &e.Table, &e.Schema, &e.Message,
			&e.SQLState, &e.FileName, &e.Command, &e.Stage, &e.Time); err != nil {
			return nil, err
		}
		excLogs = append(excLogs, e)
	}

	return excLogs, nil
}

func GetTableSchemaToExceptionLog(el []*ExceptionLog) map[string]*ExceptionLog {
	mapping := map[string]*ExceptionLog{}

	for _, e := range el {
		key := fmt.Sprintf("%s.%s", e.Schema, e.Table)
		mapping[key] = e
	}

	return mapping
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
		fileName = ExtractFileNameFromErr(errMsg)
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

	logger.Info().
		Str("table", fmt.Sprintf("%s.%s", table.Schema.String(), table.Table.String())).
		Str("continuation_token", exceptionLog.ID.String()).Msg("created continuation token")

	return inputErr
}

var fileNameRegEx = regexp.MustCompile(`part_[\d+]{8}(\.csv|\.tar\.gz)`)

func ExtractFileNameFromErr(errString string) string {
	return fileNameRegEx.FindString(errString)
}
