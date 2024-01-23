package status

import (
	"context"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/jackc/pgx/v5"
)

const createStatusTable = `CREATE TABLE IF NOT EXISTS _molt_fetch_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name STRING,
    started_at TIMESTAMP,
    source_dialect STRING
);
`

type FetchStatus struct {
	ID            uuid.UUID
	Name          string
	StartedAt     time.Time
	SourceDialect string
}

func (s *FetchStatus) CreateEntry(ctx context.Context, conn *pgx.Conn) error {
	startTime := time.Now().UTC()
	query := `INSERT INTO _molt_fetch_status (name, started_at, source_dialect) VALUES(@name, @started_at, @source_dialect) RETURNING id`
	args := pgx.NamedArgs{
		"name":           s.Name,
		"source_dialect": s.SourceDialect,
		"started_at":     startTime,
	}
	row := conn.QueryRow(ctx, query, args)

	if err := row.Scan(&s.ID); err != nil {
		return err
	}

	s.StartedAt = startTime
	return nil
}
