package tableverify

import (
	"context"
	"database/sql"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/mysqlconv"
	"github.com/cockroachdb/molt/oracleconv"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
)

type Column struct {
	Name      tree.Name
	OID       oid.Oid
	NotNull   bool
	Collation sql.NullString
}

func GetColumns(
	ctx context.Context, conn dbconn.Conn, table dbtable.DBTable, logger zerolog.Logger,
) ([]Column, error) {
	var ret []Column

	switch conn := conn.(type) {
	case *dbconn.PGConn:
		var defaultCollation string
		if err := conn.QueryRow(
			ctx,
			`SELECT pg_database.datcollate AS current_collation
FROM pg_catalog.pg_database
WHERE pg_database.datname = pg_catalog.current_database()`,
		).Scan(&defaultCollation); err != nil {
			return ret, nil
		}
		rows, err := conn.Query(
			ctx,
			`SELECT
attname, atttypid, attnotnull, collname
FROM pg_attribute
LEFT OUTER JOIN pg_collation ON (pg_collation.oid = pg_attribute.attcollation)
WHERE attrelid = $1 AND attnum > 0
ORDER BY attnum`,
			table.OID,
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var cm Column
			if err := rows.Scan(&cm.Name, &cm.OID, &cm.NotNull, &cm.Collation); err != nil {
				return ret, errors.Wrap(err, "error decoding column metadata")
			}
			if !cm.Collation.Valid || cm.Collation.String == "default" {
				cm.Collation.String = defaultCollation
				cm.Collation.Valid = true
			}
			ret = append(ret, cm)
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting column metadata")
		}
		rows.Close()
	case *dbconn.OracleConn:
		rows, err := conn.QueryContext(
			ctx,
			`select column_name, data_type, nullable, character_set_name, data_precision, data_scale
FROM all_tab_columns
WHERE table_name = :1`,
			string(table.Table),
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var cn string
			var dt string
			var isNullable string
			var collation sql.NullString
			var precision sql.NullInt64
			var scale sql.NullInt64
			if err := rows.Scan(&cn, &dt, &isNullable, &collation, &precision, &scale); err != nil {
				return ret, errors.Wrap(err, "error decoding column metadata")
			}

			var cm Column
			// TODO: we shouldn't lowercase everything
			cm.Name = tree.Name(tree.Name(cn).Normalize())
			var acceptedType bool
			cm.OID, acceptedType = oracleconv.DataTypeToOID(dt, precision, scale)
			if !acceptedType {
				// For now, we just make the column text. We should do something more later.
				cm.OID = oid.T_text
			}
			cm.NotNull = isNullable == "N"
			cm.Collation = collation
			ret = append(ret, cm)
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting column metadata")
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	case *dbconn.MySQLConn:
		rows, err := conn.QueryContext(
			ctx,
			`SELECT
column_name, data_type, column_type, is_nullable, collation_name
FROM information_schema.columns
WHERE table_schema = database() AND table_name = ?
ORDER BY ordinal_position`,
			string(table.Table),
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var cn string
			var ct string
			var dt string
			var isNullable string
			var collation sql.NullString
			if err := rows.Scan(&cn, &dt, &ct, &isNullable, &collation); err != nil {
				return ret, errors.Wrap(err, "error decoding column metadata")
			}
			var cm Column
			cm.Name = tree.Name(strings.ToLower(cn))
			typeOid, err := mysqlconv.DataTypeToOID(dt, logger)
			if err != nil {
				return ret, err
			}
			cm.OID = typeOid
			cm.NotNull = isNullable == "NO"
			cm.Collation = collation
			ret = append(ret, cm)
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting column metadata")
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	default:
		return nil, errors.Newf("connection %T not supported", conn)
	}
	return ret, nil
}

func getColumnsForTables(
	ctx context.Context, conns dbconn.OrderedConns, logger zerolog.Logger, tbls [2]dbtable.DBTable,
) ([2][]Column, error) {
	var ret [2][]Column
	for i, conn := range conns {
		var err error
		ret[i], err = GetColumns(ctx, conn, tbls[i], logger)
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func getPrimaryKeysForTables(
	ctx context.Context, conns dbconn.OrderedConns, tbls [2]dbtable.DBTable,
) ([2][]tree.Name, error) {
	var ret [2][]tree.Name
	for i, conn := range conns {
		var err error
		ret[i], err = getPrimaryKey(ctx, conn, tbls[i])
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func getPrimaryKey(
	ctx context.Context, conn dbconn.Conn, table dbtable.DBTable,
) ([]tree.Name, error) {
	var ret []tree.Name

	switch conn := conn.(type) {
	case *dbconn.PGConn:
		rows, err := conn.Query(
			ctx,
			`
SELECT
    a.attname AS column_name
FROM
    pg_index i
JOIN
    pg_attribute a ON a.attrelid = i.indrelid
AND
    a.attnum = ANY(i.indkey)
WHERE
    i.indrelid = $1
AND
    i.indisprimary
ORDER BY
    array_position(i.indkey, a.attnum);
`,
			table.OID,
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var c tree.Name
			if err := rows.Scan(&c); err != nil {
				return ret, errors.Wrap(err, "error decoding column name")
			}
			ret = append(ret, c)
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting primary key")
		}
		rows.Close()
	case *dbconn.MySQLConn:
		rows, err := conn.QueryContext(
			ctx,
			`SELECT k.column_name
FROM information_schema.table_constraints t
JOIN information_schema.key_column_usage k
USING(constraint_name,table_schema,table_name)
WHERE t.constraint_type = 'PRIMARY KEY'
  AND t.table_schema = database()
  AND t.table_name = ?
  ORDER BY k.ordinal_position`,
			string(table.Table),
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				return ret, errors.Wrap(err, "error decoding column name")
			}
			ret = append(ret, tree.Name(strings.ToLower(c)))
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting primary key")
		}
		if err := rows.Close(); err != nil {
			return ret, err
		}
	case *dbconn.OracleConn:
		rows, err := conn.QueryContext(
			ctx,
			`SELECT column_name
FROM all_cons_columns
WHERE constraint_name = (
	SELECT constraint_name FROM user_constraints WHERE table_name = :1
    AND CONSTRAINT_TYPE = 'P'
) ORDER BY position`,
			string(table.Table),
		)
		if err != nil {
			return ret, err
		}

		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				return ret, errors.Wrap(err, "error decoding column name")
			}
			ret = append(ret, tree.Name(strings.ToLower(c)))
		}
		if rows.Err() != nil {
			return ret, errors.Wrap(err, "error collecting primary key")
		}
		if err := rows.Close(); err != nil {
			return ret, err
		}
	default:
		return nil, errors.AssertionFailedf("unhandled database connection: %T", conn)
	}
	return ret, nil
}

func mapColumns(cols []Column) map[tree.Name]Column {
	ret := make(map[tree.Name]Column, len(cols))
	for _, col := range cols {
		ret[col.Name] = col
	}
	return ret
}
