package mysqlconv

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestDataType(t *testing.T) {
	ctx := context.Background()

	datadriven.Walk(t, "testdata/datatype", func(t *testing.T, path string) {
		dbConn, err := dbconn.TestOnlyCleanDatabase(ctx, "mysql", testutils.MySQLConnStr(), "datatype_test")
		require.NoError(t, err)
		defer func() { _ = dbConn.Close(ctx) }()

		conn := dbConn.(*dbconn.MySQLConn)
		datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
			switch d.Cmd {
			case "from_create_table":
				p := parser.New()
				nodes, _, err := p.Parse(d.Input, "", "")
				require.NoError(t, err)

				stmt, ok := nodes[0].(*ast.CreateTableStmt)
				require.Truef(t, ok, "got %T", nodes[0])

				_, err = conn.ExecContext(ctx, d.Input)
				require.NoError(t, err)

				rows, err := conn.QueryContext(
					ctx,
					`SELECT column_name, data_type, column_type FROM information_schema.columns
WHERE table_schema = database() AND table_name = ? ORDER BY ORDINAL_POSITION`,
					stmt.Table.Name.String(),
				)
				require.NoError(t, err)

				var sb strings.Builder
				for rows.Next() {
					var ct string
					var dt string
					var cn string
					require.NoError(t, rows.Scan(&cn, &dt, &ct))
					oid, err := DataTypeToOID(dt, zerolog.Nop())
					require.NoError(t, err)
					str := fmt.Sprintf("(non-standard) %d", oid)
					if typ, ok := types.OidToType[oid]; ok {
						str = typ.String()
					}
					sb.WriteString(fmt.Sprintf("%s: %s\n", cn, str))
				}
				require.NoError(t, err)
				return sb.String()
			default:
				t.Fatalf("unknown command %s", d.Cmd)
			}
			return ""
		})
	})
}
