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

func TestGetExceptionLogByToken(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_get_exception_log_by_token"

	t.Run("successfully retrieved exception log by token", func(t *testing.T) {
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

		curTime := time.Time{}
		curTime = curTime.Add(time.Minute)

		e := &ExceptionLog{
			FetchID:  s.ID,
			FileName: "test.log",
			Table:    "employees",
			Schema:   "public",
			Message:  "this all failed",
			SQLState: "1000",
			Command:  "SELECT VERSION()",
			Time:     curTime,
		}
		err = e.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)

		retE, err := GetExceptionLogByToken(ctx, pgConn, e.ID.String())
		require.NoError(t, err)
		require.Equal(t, e, retE)
	})

	t.Run("failed because exception ID is invalid", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn
		// Setup the tables that we need to write for status.
		require.NoError(t, CreateStatusAndExceptionTables(ctx, pgConn))

		s := &FetchStatus{
			Name:          "run 1",
			StartedAt:     time.Now(),
			SourceDialect: "postgres",
		}
		// Create entry first.
		err = s.CreateEntry(ctx, pgConn)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, s.ID)

		e := &ExceptionLog{
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

		retE, err := GetExceptionLogByToken(ctx, pgConn, "")
		require.EqualError(t, err, `ERROR: error in argument for $1: could not parse "" as type uuid: could not parse "" as type uuid: uuid: incorrect UUID length:  (SQLSTATE 22P02)`)
		require.Nil(t, retE)
	})
}

func TestGetAllExceptionLogsByFetchID(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_get_exception_log_by_fid"

	t.Run("successfully retrieved exception logs by fetch ID", func(t *testing.T) {
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

		curTime := time.Time{}
		curTime = curTime.Add(time.Minute)
		e1 := &ExceptionLog{
			FetchID:  s.ID,
			FileName: "test.log",
			Table:    "employees",
			Schema:   "public",
			Message:  "this all failed",
			SQLState: "1000",
			Command:  "SELECT VERSION()",
			Time:     curTime,
		}
		err = e1.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)

		// Creating this entry to verify that the first one doesn't
		// get included, but only the most recent one for a table/schema combo.
		e2 := &ExceptionLog{
			FetchID:  s.ID,
			FileName: "test2.log",
			Table:    "employees",
			Schema:   "public",
			Message:  "i'm failing again",
			SQLState: "500",
			Command:  "SELECT VERSION()",
			Time:     curTime.Add(time.Minute),
		}
		err = e2.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)

		e3 := &ExceptionLog{
			FetchID:  s.ID,
			FileName: "test3.log",
			Table:    "salary",
			Schema:   "public",
			Message:  "wrong salary",
			SQLState: "1000",
			Command:  "SELECT VERSION()",
			Time:     curTime,
		}
		err = e3.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)

		e4 := &ExceptionLog{
			FetchID:  s.ID,
			FileName: "test4.log",
			Table:    "taxes",
			Schema:   "public",
			Message:  "tax offset wrong",
			SQLState: "188800",
			Command:  "SELECT VERSION()",
			Time:     curTime,
		}
		err = e4.CreateEntry(ctx, pgConn, StageDataLoad)
		require.NoError(t, err)

		exceptions, err := GetAllExceptionLogsByFetchID(ctx, pgConn, s.ID.String())
		require.NoError(t, err)
		require.Equal(t, []*ExceptionLog{e2, e3, e4}, exceptions)
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

func TestGetTableSchemaToExceptionLog(t *testing.T) {
	el1 := &ExceptionLog{
		Message: "failed 1",
		Schema:  "public",
		Table:   "table1",
	}
	el2 := &ExceptionLog{
		Message: "failed 2",
		Schema:  "public",
		Table:   "table2",
	}
	el3 := &ExceptionLog{
		Message: "failed 3",
		Schema:  "public",
		Table:   "table3",
	}

	type args struct {
		el []*ExceptionLog
	}
	tests := []struct {
		name string
		args args
		want map[string]*ExceptionLog
	}{
		{
			name: "only 1 item in exception log list",
			args: args{
				el: []*ExceptionLog{el1},
			},
			want: map[string]*ExceptionLog{
				"public.table1": el1,
			},
		},
		{
			name: "multiple items in exception log list",
			args: args{
				el: []*ExceptionLog{el1, el2, el3},
			},
			want: map[string]*ExceptionLog{
				"public.table1": el1,
				"public.table2": el2,
				"public.table3": el3,
			},
		},
		{
			name: "no items in exception log list",
			args: args{
				el: []*ExceptionLog{},
			},
			want: map[string]*ExceptionLog{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetTableSchemaToExceptionLog(tt.args.el)
			require.Equal(t, tt.want, actual)
		})
	}
}
