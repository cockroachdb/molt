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

func TestCreateStatusEntry(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_status"

	s := &FetchStatus{
		Name:          "run 1",
		StartedAt:     time.Now(),
		SourceDialect: "postgres",
	}
	conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
	require.NoError(t, err)
	pgConn := conn.(*dbconn.PGConn).Conn
	// Setup the tables that we need to write for status.
	require.NoError(t, CreateStatusAndExceptionTables(ctx, pgConn))

	// Verify we can create one entry.
	err = s.CreateEntry(ctx, pgConn)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, s.ID)

	// Verify we can create the entry after the first one.
	err = s.CreateEntry(ctx, pgConn)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, s.ID)

	err = s.CreateEntry(ctx, pgConn)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, s.ID)
}
