package verify

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type connArg struct {
	id      dbconn.ID
	connStr string
}

func TestVerifyOpts_rateLimit(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		opts     verifyOpts
		expected rate.Limit
	}{
		{
			desc: "no rate limit",
			opts: verifyOpts{
				rowBatchSize: 10000,
			},
			expected: rate.Inf,
		},
		{
			desc: "multiple batches per second",
			opts: verifyOpts{
				rowBatchSize:  10000,
				rowsPerSecond: 20000,
			},
			expected: rate.Every(time.Second / 2),
		},
		{
			desc: "multiple seconds",
			opts: verifyOpts{
				rowBatchSize:  10000,
				rowsPerSecond: 5000,
			},
			expected: rate.Every(time.Second * 2),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.opts.rateLimit())
		})
	}
}

func TestDataDrivenPG(t *testing.T) {
	datadriven.Walk(
		t,
		"testdata/datadriven/pg",
		func(t *testing.T, path string) {
			testDataDriven(t, path, []connArg{
				{id: "pg", connStr: testutils.PGConnStr()},
				{id: "crdb", connStr: testutils.CRDBConnStr()},
			})
		},
	)
}

func TestDataDrivenMySQL(t *testing.T) {
	datadriven.Walk(
		t,
		"testdata/datadriven/mysql",
		func(t *testing.T, path string) {
			testDataDriven(t, path, []connArg{
				{id: "mysql", connStr: testutils.MySQLConnStr()},
				{id: "crdb", connStr: testutils.CRDBConnStr()},
			})
		},
	)
}

func testDataDriven(t *testing.T, path string, connArgs []connArg) {
	ctx := context.Background()

	var conns dbconn.OrderedConns
	for i, args := range connArgs {
		cleanConn, err := dbconn.TestOnlyCleanDatabase(ctx, args.id, args.connStr, "dd_test")
		require.NoError(t, err)
		conns[i] = cleanConn
	}

	defer func() {
		for _, conn := range conns {
			_ = conn.Close(ctx)
		}
	}()

	datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
		var sb strings.Builder
		switch d.Cmd {
		case "exec":
			return testutils.ExecConnTestdata(t, d, conns)
		case "query":
			return testutils.QueryConnCommand(t, d, conns)
		case "verify":
			numSplits := 1
			for _, arg := range d.CmdArgs {
				switch arg.Key {
				case "splits":
					var err error
					numSplits, err = strconv.Atoi(arg.Vals[0])
					require.NoError(t, err)
				}
			}
			reporter := &inconsistency.LogReporter{
				Logger: zerolog.New(&sb),
			}
			// Use 1 concurrency / splitting to ensure deterministic results.
			err := Verify(
				ctx,
				conns,
				zerolog.Nop(),
				reporter,
				WithConcurrency(1),
				WithRowBatchSize(2),
				WithTableSplits(numSplits),
			)
			if err != nil {
				sb.WriteString(fmt.Sprintf("error: %s\n", err.Error()))
			}
		default:
			t.Fatalf("unknown command: %s", d.Cmd)
		}
		return sb.String()
	})
}
