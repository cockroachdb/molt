package dbverify

import (
	"testing"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/utils"
	"github.com/stretchr/testify/require"
)

func TestTableCompare(t *testing.T) {
	table1 := dbtable.DBTable{
		OID: 1,
		Name: dbtable.Name{
			Schema: "1",
			Table:  "1",
		},
	}
	table2 := dbtable.DBTable{
		OID: 2,
		Name: dbtable.Name{
			Schema: "2",
			Table:  "2",
		},
	}
	table3 := dbtable.DBTable{
		OID: 3,
		Name: dbtable.Name{
			Schema: "3",
			Table:  "3",
		},
	}
	table3DifferentOID := dbtable.DBTable{
		OID: 33,
		Name: dbtable.Name{
			Schema: "3",
			Table:  "3",
		},
	}
	conn1 := dbconn.MakeFakeConn("1")
	conn2 := dbconn.MakeFakeConn("1")
	for _, tc := range []struct {
		desc     string
		its      [2]tableVerificationIterator
		expected utils.Result
	}{
		{
			desc: "exactly the same",
			its: [2]tableVerificationIterator{
				{
					tables: connWithTables{
						Conn:          conn1,
						tableMetadata: []dbtable.DBTable{table1, table2, table3},
					},
				},
				{
					tables: connWithTables{
						Conn:          conn2,
						tableMetadata: []dbtable.DBTable{table1, table2, table3DifferentOID},
					},
				},
			},
			expected: utils.Result{
				Verified: [][2]dbtable.DBTable{
					{table1, table1},
					{table2, table2},
					{table3, table3DifferentOID},
				},
			},
		},
		{
			desc: "everything missing",
			its: [2]tableVerificationIterator{
				{
					tables: connWithTables{
						Conn:          conn1,
						tableMetadata: []dbtable.DBTable{table1, table2, table3},
					},
				},
				{
					tables: connWithTables{
						Conn: conn2,
					},
				},
			},
			expected: utils.Result{
				MissingTables: []utils.MissingTable{
					{DBTable: table1},
					{DBTable: table2},
					{DBTable: table3},
				},
			},
		},
		{
			desc: "everything extraneous",
			its: [2]tableVerificationIterator{
				{
					tables: connWithTables{
						Conn: conn1,
					},
				},
				{
					tables: connWithTables{
						Conn:          conn2,
						tableMetadata: []dbtable.DBTable{table1, table2, table3},
					},
				},
			},
			expected: utils.Result{
				ExtraneousTables: []utils.ExtraneousTable{
					{DBTable: table1},
					{DBTable: table2},
					{DBTable: table3},
				},
			},
		},
		{
			desc: "only table 2 in common",
			its: [2]tableVerificationIterator{
				{
					tables: connWithTables{
						Conn:          conn1,
						tableMetadata: []dbtable.DBTable{table1, table2},
					},
				},
				{
					tables: connWithTables{
						Conn:          conn2,
						tableMetadata: []dbtable.DBTable{table2, table3},
					},
				},
			},
			expected: utils.Result{
				Verified: [][2]dbtable.DBTable{
					{table2, table2},
				},
				MissingTables: []utils.MissingTable{
					{DBTable: table1},
				},
				ExtraneousTables: []utils.ExtraneousTable{
					{DBTable: table3},
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.expected, compare(tc.its))
		})
	}
}
