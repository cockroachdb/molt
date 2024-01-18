package status

import (
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/stretchr/testify/require"
)

func TestCreateExceptionEntry(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_status"

	t.Run("succesful create", func(t *testing.T) {
		s := &FetchStatus{
			Name:          "run 1",
			Status:        "IN PROGRESS",
			StartedAt:     time.Now(),
			FinishedAt:    time.Now(),
			SourceDialect: "postgres",
		}
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn
		// Setup the tables that we need to write for status.
		require.NoError(t, CreateStatusAndExceptionTables(ctx, pgConn))

		// Create entry first.
		err = s.CreateEntry(ctx, pgConn)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, s.ID)

		e := ExceptionLog{
			FetchID:  s.ID,
			FileName: "test.log",
			Table:    "employees",
			Schema:   "public",
			Message:  "this all failed",
			SQLState: 1000,
			Command:  "SELECT VERSION()",
			Time:     time.Now(),
		}
		err = e.CreateEntry(ctx, pgConn)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, e.ID)
	})

	t.Run("failed because fetch ID invalid", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn
		// Setup the tables that we need to write for status.
		require.NoError(t, CreateStatusAndExceptionTables(ctx, pgConn))

		e := ExceptionLog{
			FetchID:  uuid.Nil,
			FileName: "test.log",
			Table:    "employees",
			Schema:   "public",
			Message:  "this all failed",
			SQLState: 1000,
			Command:  "SELECT VERSION()",
			Time:     time.Now(),
		}
		err = e.CreateEntry(ctx, pgConn)
		require.EqualError(t, err, "ERROR: insert on table \"_molt_fetch_exception\" violates foreign key constraint \"_molt_fetch_exception_fetch_id_fkey\" (SQLSTATE 23503)")
	})
}
