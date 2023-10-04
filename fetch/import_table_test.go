package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/testutils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

const listenAddr = "0.0.0.0:4041"
const filePath = "./testdata/csv/"
const createTableStmt = "CREATE TABLE IF NOT EXISTS teams (id INT PRIMARY KEY, name STRING, role STRING)"
const dropTableStmt = "DROP TABLE teams"

func getTestFileServer() *http.Server {
	return &http.Server{
		Addr:    listenAddr,
		Handler: http.FileServer(http.Dir(filePath)),
	}
}

func verifyValidServerResponse(t *testing.T, URL string) error {
	r, err := retry.NewRetry(retry.Settings{
		InitialBackoff: 1 * time.Second,
		Multiplier:     1,
		MaxRetries:     5,
	})
	require.NoError(t, err)

	if err := r.Do(func() error {
		response, err := http.Get(URL)
		if err != nil {
			return err
		}

		responseData, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		t.Log(string(responseData))
		return err
	}, func(err error) {
		t.Log(err)
	}); err != nil {
		return err
	}

	return nil
}

func TestReportImportTableProgress(t *testing.T) {
	t.Run("successfully reports import progress", func(t *testing.T) {
		curTime := time.Now()
		ctx := context.Background()
		srv := getTestFileServer()
		go func() {
			t.Logf("serving HTTP at %s", listenAddr)
			if err := srv.ListenAndServe(); err != nil {
				t.Logf("error starting server: %s", err)
			}
		}()
		defer func() {
			err := srv.Shutdown(ctx)
			require.NoError(t, err)
		}()

		path := fmt.Sprintf("http://%s/import_basic_000.csv", listenAddr)
		err := verifyValidServerResponse(t, path)
		require.NoError(t, err)

		dbName := "fetch_test_report_import"
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)

		pgConn, ok := conn.(*dbconn.PGConn)
		require.Equal(t, true, ok)

		_, err = pgConn.Exec(ctx, createTableStmt)
		defer func() {
			_, err := pgConn.Exec(ctx, dropTableStmt)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		importStmt := fmt.Sprintf(`IMPORT INTO teams(id, name, role) CSV DATA ('http://%s/import_basic_000.csv')`, listenAddr)
		_, err = pgConn.Exec(ctx, importStmt)
		require.NoError(t, err)
		var b bytes.Buffer
		logger := zerolog.New(&b)

		err = reportImportTableProgress(ctx, conn, logger, dbtable.VerifiedTable{
			Name: dbtable.Name{
				Schema: tree.Name("public"),
				Table:  tree.Name("teams"),
			},
		}, curTime, true /*testing*/)
		require.NoError(t, err)
		require.Equal(t, `{"level":"info","message":"100.00% completed (public.teams)"}
`, b.String())
	})

	t.Run("wrong connection type passed in", func(t *testing.T) {
		curTime := time.Now()
		ctx := context.Background()
		dbName := "fetch_test_report_import"
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.MySQLConnStr(), dbName)
		require.NoError(t, err)

		logger := zerolog.New(zerolog.NewConsoleWriter())
		err = reportImportTableProgress(ctx, conn, logger, dbtable.VerifiedTable{
			Name: dbtable.Name{
				Schema: tree.Name("public"),
				Table:  tree.Name("teams"),
			},
		}, curTime, true /*testing*/)
		require.EqualError(t, err, "expected pgx conn, got *dbconn.MySQLConn")
	})
}

func TestGetShowJobsQuery(t *testing.T) {
	for _, tc := range []struct {
		name     string
		table    dbtable.VerifiedTable
		curTime  string
		expected string
	}{
		{
			name: "normal schema and table",
			table: dbtable.VerifiedTable{
				Name: dbtable.Name{
					Schema: tree.Name("public"),
					Table:  tree.Name("test1"),
				},
			},
			curTime: "2006-01-02T15:04:05",
			expected: `WITH x as (SHOW JOBS)
SELECT description, started, fraction_completed
FROM x
WHERE job_type='IMPORT'
    AND description LIKE '%public.test1(%'
    AND started > '2006-01-02T15:04:05'
ORDER BY created DESC`,
		},
		{
			name: "escaped percent symbol",
			table: dbtable.VerifiedTable{
				Name: dbtable.Name{
					Schema: tree.Name("public"),
					Table:  tree.Name("test%1"),
				},
			},
			curTime: "2006-01-02T15:04:05",
			expected: `WITH x as (SHOW JOBS)
SELECT description, started, fraction_completed
FROM x
WHERE job_type='IMPORT'
    AND description LIKE '%public.test\%1(%'
    AND started > '2006-01-02T15:04:05'
ORDER BY created DESC`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := getShowJobsQuery(tc.table, tc.curTime)
			require.Equal(t, tc.expected, query)
		})
	}
}
