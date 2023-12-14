package rowverify

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/comparectx"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestLiveReverifier(t *testing.T) {
	const schema = `CREATE TABLE test_table (id INT4 PRIMARY KEY, text TEXT)`
	tbl := dbtable.VerifiedTable{
		Name: dbtable.Name{
			Schema: "public",
			Table:  "test_table",
		},
		PrimaryKeyColumns: []tree.Name{"id"},
		Columns:           []tree.Name{"id", "text"},
		ColumnOIDs: [2][]oid.Oid{
			{oid.T_int4, oid.T_text},
			{oid.T_int4, oid.T_text},
		},
	}

	defaultRetrySettings := retry.Settings{
		InitialBackoff: time.Millisecond,
		Multiplier:     1,
		MaxBackoff:     time.Second,
		MaxRetries:     3,
	}

	now := time.Now()
	ctx := context.Background()
	for testIdx, tc := range []struct {
		desc string

		expectedBeforeScans [][]tree.Datums
		beforeIteration     map[int]func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums)
		push                []liveRetryItem

		expectedExtraneous  []inconsistency.ExtraneousRow
		expectedMissing     []inconsistency.MissingRow
		expectedMismatching []inconsistency.MismatchingRow
	}{
		{
			desc: "no items ever pop up on queue",
		},
		{
			desc: "items get fixed before iteration 1",
			push: []liveRetryItem{
				{
					PrimaryKeys: []tree.Datums{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
					Retry:       retry.MustRetry(defaultRetrySettings),
				},
			},

			expectedBeforeScans: [][]tree.Datums{
				{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
			},
		},
		{
			desc: "items get fixed before iteration 2",
			push: []liveRetryItem{
				{
					PrimaryKeys: []tree.Datums{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
					Retry:       retry.MustRetry(defaultRetrySettings),
				},
			},
			beforeIteration: map[int]func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums){
				1: func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums) {
					_, err := conns[0].(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (2, 'a')")
					require.NoError(t, err)
				},
				2: func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums) {
					_, err := conns[1].(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (2, 'a')")
					require.NoError(t, err)
				},
			},
			expectedBeforeScans: [][]tree.Datums{
				{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
				{{tree.NewDInt(2)}},
			},
		},
		{
			desc: "items never get restored",
			push: []liveRetryItem{
				{
					PrimaryKeys: []tree.Datums{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
					Retry:       retry.MustRetry(defaultRetrySettings),
				},
			},
			beforeIteration: map[int]func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums){
				1: func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums) {
					_, err := conns[0].(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (2, 'a')")
					require.NoError(t, err)
				},
			},
			expectedBeforeScans: [][]tree.Datums{
				{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
				{{tree.NewDInt(2)}},
				{{tree.NewDInt(2)}},
			},
			expectedMissing: []inconsistency.MissingRow{
				{
					Name: dbtable.Name{
						Schema: "public",
						Table:  "test_table",
					},
					PrimaryKeyColumns: []tree.Name{"id"},
					PrimaryKeyValues:  tree.Datums{tree.NewDInt(2)},
					Columns:           []tree.Name{"id", "text"},
					Values:            tree.Datums{tree.NewDInt(2), tree.NewDString("a")},
				},
			},
		},
		{
			desc: "two items in queue, 1 gets fixed on iteration 1 and one on iteration 2",
			push: []liveRetryItem{
				{
					PrimaryKeys: []tree.Datums{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
					Retry:       retry.MustRetryWithTime(now, defaultRetrySettings),
				},
				{
					PrimaryKeys: []tree.Datums{{tree.NewDInt(3)}},
					Retry:       retry.MustRetryWithTime(now.Add(time.Microsecond), defaultRetrySettings),
				},
			},
			beforeIteration: map[int]func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums){
				1: func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums) {
					if pks[0][0].Compare(comparectx.CompareContext, tree.NewDInt(1)) == 0 {
						_, err := conns[0].(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (2, 'a')")
						require.NoError(t, err)
					}
					if pks[0][0].Compare(comparectx.CompareContext, tree.NewDInt(3)) == 0 {
						for _, conn := range conns {
							_, err := conn.(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (3, 'a')")
							require.NoError(t, err)
						}
					}
				},
				2: func(t *testing.T, conns dbconn.OrderedConns, pks []tree.Datums) {
					_, err := conns[1].(*dbconn.PGConn).Exec(ctx, "INSERT INTO test_table VALUES (2, 'a')")
					require.NoError(t, err)
				},
			},
			expectedBeforeScans: [][]tree.Datums{
				{{tree.NewDInt(1)}, {tree.NewDInt(2)}},
				{{tree.NewDInt(3)}},
				{{tree.NewDInt(2)}},
			},
		},
	} {
		testIdx := testIdx
		t.Run(tc.desc, func(t *testing.T) {
			var conns dbconn.OrderedConns
			for i, args := range []struct {
				id      dbconn.ID
				connStr string
			}{
				{id: "pg", connStr: testutils.PGConnStr()},
				{id: "crdb", connStr: testutils.CRDBConnStr()},
			} {
				cleanConn, err := dbconn.TestOnlyCleanDatabase(ctx, args.id, args.connStr, fmt.Sprintf("live_reverifier_test_%d", testIdx))
				require.NoError(t, err)
				defer func() {
					require.NoError(t, cleanConn.Close(ctx))
				}()
				conns[i] = cleanConn

				_, err = cleanConn.(*dbconn.PGConn).Exec(ctx, schema)
				require.NoError(t, err)
			}

			evl := &mockRowEventListener{}
			lvr, err := newLiveReverifier(
				ctx,
				zerolog.New(os.Stderr),
				conns,
				TableShard{VerifiedTable: tbl},
				evl,
				rate.NewLimiter(rate.Inf, 1),
			)
			require.NoError(t, err)

			var beforeScans [][]tree.Datums
			lvr.testingKnobs.beforeScan = func(r *retry.Retry, pks []tree.Datums) {
				if f, ok := tc.beforeIteration[r.Iteration]; ok {
					f(t, conns, pks)
				}
				ret := make([]tree.Datums, len(pks))
				copy(ret, pks)
				beforeScans = append(beforeScans, ret)
			}

			lvr.testingKnobs.blockScan.Store(true)
			for i := range tc.push {
				lvr.Push(&tc.push[i])
			}
			lvr.testingKnobs.blockScan.Store(false)

			lvr.ScanComplete()
			lvr.WaitForDone()

			require.Equal(t, tc.expectedBeforeScans, beforeScans)
			require.Equal(t, tc.expectedExtraneous, evl.extraneous)
			require.Equal(t, tc.expectedMissing, evl.missing)
			require.Equal(t, tc.expectedMismatching, evl.mismatching)
		})
	}
}

type mockRowEventListener struct {
	extraneous  []inconsistency.ExtraneousRow
	missing     []inconsistency.MissingRow
	mismatching []inconsistency.MismatchingRow
}

func (m *mockRowEventListener) OnExtraneousRow(row inconsistency.ExtraneousRow) {
	m.extraneous = append(m.extraneous, row)
}

func (m *mockRowEventListener) OnMissingRow(row inconsistency.MissingRow) {
	m.missing = append(m.missing, row)
}

func (m *mockRowEventListener) OnMismatchingRow(row inconsistency.MismatchingRow) {
	m.mismatching = append(m.mismatching, row)
}

func (m *mockRowEventListener) OnMatch() {}

func (m *mockRowEventListener) OnColumnMismatchNoOtherIssues(
	row inconsistency.MismatchingColumn, reportLog bool,
) {
}

func (m *mockRowEventListener) OnRowScan() {}
