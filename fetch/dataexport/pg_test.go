package dataexport

import (
	"context"
	"testing"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestNewPGSource(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		prerun   func(t *testing.T, conn *dbconn.PGConn)
		settings Settings
		postrun  func(t *testing.T, conn *dbconn.PGConn)
	}{
		//{
		//	desc: "do not create replication slot",
		//	settings: Settings{
		//		RowBatchSize: 10,
		//	},
		//},
		//{
		//	desc: "create a replication slot",
		//	settings: Settings{
		//		RowBatchSize: 10,
		//		PG: PGReplicationSlotSettings{
		//			SlotName: "test_slot",
		//			Plugin:   "pgoutput",
		//		},
		//	},
		//	postrun: func(t *testing.T, conn *dbconn.PGConn) {
		//		var n int
		//		require.NoError(t, conn.QueryRow(
		//			context.Background(),
		//			"SELECT COUNT(1) FROM pg_replication_slots WHERE slot_name = $1 AND plugin = $2",
		//			"test_slot",
		//			"pgoutput",
		//		).Scan(&n))
		//		require.Equal(t, n, 1)
		//	},
		//},
		//{
		//	desc: "overwrites an existing replication slot",
		//	settings: Settings{
		//		RowBatchSize: 10,
		//		PG: PGReplicationSlotSettings{
		//			SlotName:     "test_slot",
		//			Plugin:       "pgoutput",
		//			DropIfExists: true,
		//		},
		//	},
		//	prerun: func(t *testing.T, conn *dbconn.PGConn) {
		//		_, err := conn.Exec(
		//			context.Background(),
		//			"SELECT pg_create_logical_replication_slot($1, $2)",
		//			"test_slot",
		//			"test_decoding",
		//		)
		//		require.NoError(t, err)
		//	},
		//	postrun: func(t *testing.T, conn *dbconn.PGConn) {
		//		var n int
		//		require.NoError(t, conn.QueryRow(
		//			context.Background(),
		//			"SELECT COUNT(1) FROM pg_replication_slots WHERE slot_name = $1 AND plugin = $2",
		//			"test_slot",
		//			"pgoutput",
		//		).Scan(&n))
		//		require.Equal(t, n, 1)
		//	},
		//},
		{
			desc: "clone and see if txn is cloned too",
			settings: Settings{
				RowBatchSize: 3,
				PG: PGReplicationSlotSettings{
					SlotName:     "test_slot",
					Plugin:       "pgoutput",
					DropIfExists: true,
				},
			},
			prerun: func(t *testing.T, conn *dbconn.PGConn) {
				var res string
				if err := conn.QueryRow(
					context.Background(),
					"SELECT pg_export_snapshot()",
				).Scan(&res); err != nil {
					t.Fatal("failed in prerun")
				}
				t.Logf("res prerun: %s\n", res)
			},
			postrun: func(t *testing.T, conn *dbconn.PGConn) {
				ctx := context.Background()
				clonedConn, err := conn.Clone(ctx)
				require.NoError(t, err)
				defer func() { require.NoError(t, clonedConn.Close(ctx)) }()
				tx, err := clonedConn.(*dbconn.PGConn).BeginTx(ctx, pgx.TxOptions{
					IsoLevel:   pgx.RepeatableRead,
					AccessMode: pgx.ReadOnly,
				})
				require.NoError(t, err)
				var resFromTx string
				if err := tx.QueryRow(
					context.Background(),
					"SELECT pg_export_snapshot()",
				).Scan(&resFromTx); err != nil {
					t.Fatal("failed in prerun")
				}
				t.Logf("resFromTx postrun: %s\n", resFromTx)

				var resFromClonedConn string
				if err := clonedConn.(*dbconn.PGConn).QueryRow(
					context.Background(),
					"SELECT pg_export_snapshot()",
				).Scan(&resFromClonedConn); err != nil {
					t.Fatal("failed in prerun")
				}
				t.Logf("resFromClonedConn postrun: %s\n", resFromClonedConn)

				require.NoError(t, tx.Rollback(ctx))
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			connRaw, err := dbconn.TestOnlyCleanDatabase(ctx, "pg", testutils.PGConnStr(), "pg_source_test")
			require.NoError(t, err)
			defer func() {
				require.NoError(t, connRaw.Close(ctx))
			}()
			conn := connRaw.(*dbconn.PGConn)
			if tc.prerun != nil {
				tc.prerun(t, conn)
			}
			s, err := NewPGSource(ctx, tc.settings, conn)
			require.NoError(t, err)
			if tc.postrun != nil {
				tc.postrun(t, conn)
			}
			require.NoError(t, s.Close(ctx))
		})
	}
}
