package testutils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/stretchr/testify/require"
)

func PGConnStr() string {
	pgInstanceURL := "postgres://postgres:postgres@127.0.0.1:5432/defaultdb"
	if override, ok := os.LookupEnv("POSTGRES_URL"); ok {
		pgInstanceURL = override
	}
	return pgInstanceURL
}

func CRDBConnStr() string {
	crdbInstanceURL := "postgres://root@127.0.0.1:26257/defaultdb?sslmode=disable"
	if override, ok := os.LookupEnv("COCKROACH_URL"); ok {
		crdbInstanceURL = override
	}
	return crdbInstanceURL
}

func CRDBTargetConnStr() string {
	crdbInstanceURL := "postgres://root@127.0.0.1:26258/defaultdb?sslmode=disable"
	if override, ok := os.LookupEnv("COCKROACH_TARGET_URL"); ok {
		crdbInstanceURL = override
	}
	return crdbInstanceURL
}

func MySQLConnStr() string {
	mysqlInstanceURL := "jdbc:mysql://root@tcp(localhost:3306)/defaultdb"
	if override, ok := os.LookupEnv("MYSQL_URL"); ok {
		mysqlInstanceURL = override
	}
	return mysqlInstanceURL
}

func ExecConnQuery(ctx context.Context, q string, conn dbconn.Conn) (res string, err error) {
	switch ct := conn.(type) {
	case *dbconn.PGConn:
		tag, err := ct.Exec(ctx, q)
		if err != nil {
			return "", err
		}
		return tag.String(), err
	case *dbconn.MySQLConn:
		tag, err := ct.ExecContext(ctx, q)
		if err != nil {
			return "", err
		}
		r, err := tag.RowsAffected()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%s] %d rows affected\n", conn.ID(), r), nil
	default:
		return "", errors.AssertionFailedf("unhandled Conn type: %T", conn)
	}
}

func ExecConnTestdata(t *testing.T, d *datadriven.TestData, conns dbconn.OrderedConns) string {
	ctx := context.Background()
	var sb strings.Builder

	var connIdxs []int
	for _, arg := range d.CmdArgs {
		switch arg.Key {
		case "source":
			connIdxs = append(connIdxs, 0)
		case "target":
			connIdxs = append(connIdxs, 1)
		case "all":
			for connIdx := range conns {
				connIdxs = append(connIdxs, connIdx)
			}
		}
	}
	require.NotEmpty(t, connIdxs, "destination sql must be defined")
	for _, connIdx := range connIdxs {
		switch conn := conns[connIdx].(type) {
		case *dbconn.PGConn:
			tag, err := conn.Exec(ctx, d.Input)
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error: %s\n", conn.ID(), err.Error()))
				continue
			}
			sb.WriteString(fmt.Sprintf("[%s] %s\n", conn.ID(), tag.String()))
		case *dbconn.MySQLConn:
			tag, err := conn.ExecContext(ctx, d.Input)
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error: %s\n", conn.ID(), err.Error()))
				continue
			}
			r, err := tag.RowsAffected()
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error getting rows affected: %s\n", conn.ID(), err.Error()))
				continue
			}
			sb.WriteString(fmt.Sprintf("[%s] %d rows affected\n", conn.ID(), r))
		default:
			t.Fatalf("unhandled Conn type: %T", conn)
		}
	}

	// Deallocate caches - otherwise the plans may stick around.
	for _, conn := range conns {
		switch conn := conn.(type) {
		case *dbconn.PGConn:
			require.NoError(t, conn.DeallocateAll(ctx))
		}
	}
	return sb.String()
}

func QueryConnCommand(t *testing.T, d *datadriven.TestData, conns dbconn.OrderedConns) string {
	ctx := context.Background()
	var sb strings.Builder

	var connIdxs []int
	for _, arg := range d.CmdArgs {
		switch arg.Key {
		case "source":
			connIdxs = append(connIdxs, 0)
		case "target":
			connIdxs = append(connIdxs, 1)
		case "all":
			for connIdx := range conns {
				connIdxs = append(connIdxs, connIdx)
			}
		}
	}
	require.NotEmpty(t, connIdxs, "destination sql must be defined")
	for _, connIdx := range connIdxs {
		switch conn := conns[connIdx].(type) {
		case *dbconn.PGConn:
			rows, err := conn.Query(ctx, d.Input)
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error: %s\n", conn.ID(), err.Error()))
				continue
			}
			sb.WriteString(fmt.Sprintf("[%s]:\n", conn.ID()))
			for i, fd := range rows.FieldDescriptions() {
				if i > 0 {
					sb.WriteString("\t")
				}
				sb.WriteString(fd.Name)
			}
			sb.WriteString("\n")
			for rows.Next() {
				vals, err := rows.Values()
				require.NoError(t, err)
				for i, val := range vals {
					if i > 0 {
						sb.WriteString("\t")
					}
					sb.WriteString(fmt.Sprintf("%v", val))
				}
				sb.WriteString("\n")
			}
			if rows.Err() != nil {
				sb.WriteString(fmt.Sprintf("[%s] error: %s\n", conn.ID(), rows.Err()))
				continue
			}
			sb.WriteString(fmt.Sprintf("tag: %s\n", rows.CommandTag()))
		case *dbconn.MySQLConn:
			tag, err := conn.ExecContext(ctx, d.Input)
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error: %s\n", conn.ID(), err.Error()))
				continue
			}
			r, err := tag.RowsAffected()
			if err != nil {
				sb.WriteString(fmt.Sprintf("[%s] error getting rows affected: %s\n", conn.ID(), err.Error()))
				continue
			}
			sb.WriteString(fmt.Sprintf("[%s] %d rows affected\n", conn.ID(), r))
		default:
			t.Fatalf("unhandled Conn type: %T", conn)
		}
	}

	// Deallocate caches - otherwise the plans may stick around.
	for _, conn := range conns {
		switch conn := conn.(type) {
		case *dbconn.PGConn:
			require.NoError(t, conn.DeallocateAll(ctx))
		}
	}
	return sb.String()
}
