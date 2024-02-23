package mysqlconv

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/lib/pq/oid"
	"github.com/stretchr/testify/require"
)

func getOids(cmdArgs []datadriven.CmdArg) ([]oid.Oid, error) {
	if len(cmdArgs) == 0 {
		return nil, errors.AssertionFailedf("error: no oid provided")
	}
	var res []oid.Oid
	for _, arg := range cmdArgs {
		argInt, err := strconv.Atoi(arg.Vals[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert arg %q to an oid", arg)
		}
		// Check if this is a valid oid.
		_, ok := types.OidToType[oid.Oid(uint32(argInt))]
		if !ok {
			return nil, errors.AssertionFailedf("unknown oid: %d", argInt)
		}
		res = append(res, oid.Oid(uint32(argInt)))
	}
	return res, nil
}

func TestConvertRowValue(t *testing.T) {
	ctx := context.Background()
	const dbName = "mysql_test_conv_row_val"
	dbConn, err := dbconn.TestOnlyCleanDatabase(ctx, "source", testutils.MySQLConnStr(), dbName)
	require.NoError(t, err)
	conn := dbConn.(*dbconn.MySQLConn)
	defer func() { _ = conn.Close(ctx) }()
	datadriven.Walk(t, "testdata/rowvalue", func(t *testing.T, path string) {
		datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
			var sb strings.Builder
			switch d.Cmd {
			case "convert":
				oids, err := getOids(d.CmdArgs)
				if err != nil {
					sb.WriteString(err.Error())
					return sb.String()
				}
				rows, err := conn.QueryContext(ctx, "SELECT "+d.Input)
				require.NoError(t, err)

				for rows.Next() {
					datums, err := ScanRowDynamicTypes(rows, conn.TypeMap(), oids)
					if err != nil {
						sb.WriteString(fmt.Sprintf("scan row dynamic types error: %s\n", err.Error()))
					}
					for _, datum := range datums {
						sb.WriteString(fmt.Sprintf("%T:%s;\n", datum, datum.String()))
					}
				}
				if rows.Err() != nil {
					sb.WriteString(fmt.Sprintf("query error: %s\n", rows.Err().Error()))
				}
				return sb.String()
			default:
				t.Fatalf("unknown command: %s", d.Cmd)
			}
			return sb.String()
		})
	})
}
