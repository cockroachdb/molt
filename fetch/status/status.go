package status

import (
	"context"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	StatusInProgress = "IN PROGRESS"
	StatusFailed     = "FAILED"
	StatusSucceeded  = "SUCCEEDED"
)

const createStatusTable = `CREATE TABLE IF NOT EXISTS _molt_fetch_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name STRING,
    status STRING,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    source_dialect STRING
);
`

type FetchStatus struct {
	ID            uuid.UUID
	Name          string
	Status        string
	StartedAt     time.Time
	FinishedAt    time.Time
	SourceDialect string
}

func (s *FetchStatus) CreateEntry(ctx context.Context, conn *pgx.Conn) error {
	startTime := time.Now().UTC()
	query := `INSERT INTO _molt_fetch_status (name, status, started_at, source_dialect) VALUES(@name, @status, @started_at, @source_dialect) RETURNING id, status`
	args := pgx.NamedArgs{
		"name":           s.Name,
		"source_dialect": s.SourceDialect,
		"status":         StatusInProgress,
		"started_at":     startTime,
	}
	row := conn.QueryRow(ctx, query, args)

	if err := row.Scan(&s.ID, &s.Status); err != nil {
		return err
	}

	s.StartedAt = startTime
	return nil
}

func (s *FetchStatus) markComplete(ctx context.Context, conn *pgx.Conn, status string) error {
	endTime := time.Now().UTC()
	query := `UPDATE _molt_fetch_status SET status=@status, finished_at=@finished_at WHERE id=@id`
	args := pgx.NamedArgs{
		"id":          s.ID,
		"status":      status,
		"finished_at": endTime,
	}

	if _, err := conn.Exec(ctx, query, args); err != nil {
		return err
	}

	s.Status = status
	s.FinishedAt = endTime
	return nil
}

func (s *FetchStatus) MarkSuccessful(ctx context.Context, conn *pgx.Conn) error {
	return s.markComplete(ctx, conn, StatusSucceeded)
}

func (s *FetchStatus) MarkFailed(ctx context.Context, conn *pgx.Conn) error {
	return s.markComplete(ctx, conn, StatusFailed)
}
