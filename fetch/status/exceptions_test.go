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
			StartedAt:     time.Now(),
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
			SQLState: "1000",
			Command:  "SELECT VERSION()",
			Time:     time.Now(),
		}
		err = e.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, e.ID)
		require.Equal(t, StageDataLoad, e.Stage)
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
			SQLState: "1000",
			Command:  "SELECT VERSION()",
			Time:     time.Now(),
		}
		err = e.CreateEntry(ctx, pgConn, StageDataLoad)
		require.EqualError(t, err, "ERROR: insert on table \"_molt_fetch_exception\" violates foreign key constraint \"_molt_fetch_exception_fetch_id_fkey\" (SQLSTATE 23503)")
	})
}

func TestExtractFileNameFromErr(t *testing.T) {
	type args struct {
		errString string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "found file name in error string",
			args: args{
				errString: "error importing data: ERROR: http://192.168.0.207:9005/public.employees/part_00000001.csv: error parsing row 1: expected 9 fields, got 16 (row: e8400-e29b-41d4-a716-446655440000,Employee_1,2023-11-03 09:00:00+00,2023-11-03,t,24,5000.00,100.252,550e8400-e29b-41d4-a716-446655440000,Employee_2,2023-11-03 09:00:00+00,2023-11-03,t,24,5000.00,100.25) (SQLSTATE XXUUU)exit status 1",
			},
			want: "part_00000001.csv",
		},
		{
			name: "file name not found",
			args: args{
				errString: "error importing data: ERROR: http://192.168.0.207:9005/public.employees: error parsing row 1: expected 9 fields, got 16 (row: e8400-e29b-41d4-a716-446655440000,Employee_1,2023-11-03 09:00:00+00,2023-11-03,t,24,5000.00,100.252,550e8400-e29b-41d4-a716-446655440000,Employee_2,2023-11-03 09:00:00+00,2023-11-03,t,24,5000.00,100.25) (SQLSTATE XXUUU)exit status 1",
			},
			want: "",
		},
		{
			name: "empty string",
			args: args{
				errString: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := extractFileNameFromErr(tt.args.errString)
			require.Equal(t, tt.want, actual)
		})
	}
}
