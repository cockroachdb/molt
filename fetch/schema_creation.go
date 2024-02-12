package fetch

import (
	"context"
	"fmt"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
)

type columnWithType struct {
	schemaName   string
	tableName    string
	columnName   string
	dataType     string
	typeOid      oid.Oid
	notNullable  bool
	isPrimaryKey bool
}

func (t *columnWithType) Name() string {
	return fmt.Sprintf("%s.%s.%s", t.schemaName, t.tableName, t.columnName)
}

func (t *columnWithType) String() string {
	return fmt.Sprintf("schema:%q, table:%q, column:%q, type:%q, typeoid: %d, nullable:%t, pk:%t\n",
		t.schemaName, t.tableName, t.columnName, t.dataType, t.typeOid, t.notNullable, t.isPrimaryKey)
}

func GetColumnTypes(
	ctx context.Context,
	logger zerolog.Logger,
	conn dbconn.Conn,
	tableName tree.Name,
	schemaName tree.Name,
) ([]columnWithType, error) {
	const (
		pgQuery = `SELECT
    c.relnamespace::regnamespace::text as schema_name,
        c.relname AS table_name,
    a.attname AS column_name,
    format_type(a.atttypid, a.atttypmod) AS data_type,
    a.atttypid AS type_oid,
    a.attnotnull AS not_nullable,
    CASE
        WHEN a.attname IN (
            SELECT column_name
            FROM information_schema.constraint_column_usage
            WHERE constraint_name = (
                SELECT constraint_name
                FROM information_schema.table_constraints
                WHERE table_name = c.relname
                  AND constraint_type = 'PRIMARY KEY'
            )
        ) THEN true
        ELSE false
        END AS is_primary_key
FROM
    pg_catalog.pg_class c
        JOIN
    pg_catalog.pg_attribute a ON c.oid = a.attrelid
        LEFT JOIN
    pg_catalog.pg_index ix ON c.oid = ix.indrelid AND a.attnum = ANY(ix.indkey)
WHERE
        c.relkind = 'r'  -- 'r' indicates a table (relation)
  AND a.attnum > 0 -- Exclude system columns
  AND c.relname = $1
  AND c.relnamespace::regnamespace::text = $2
ORDER BY
    table_name, a.attnum;`
	)

	res := make([]columnWithType, 0)
	logger.Info().Msgf("getting column types for table: %s.%s", schemaName, tableName)

	switch conn := conn.(type) {
	case *dbconn.PGConn:
		rows, err := conn.Query(ctx, pgQuery, tableName, schemaName)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			newCol := columnWithType{}
			if err := rows.Scan(&newCol.schemaName, &newCol.tableName, &newCol.columnName, &newCol.dataType, &newCol.typeOid, &newCol.notNullable, &newCol.isPrimaryKey); err != nil {
				return nil, errors.Wrap(err, "failed to scan query result to a columnWithType object")
			}
			logger.Debug().Msgf("collect column:%s", newCol.String())
			res = append(res, newCol)
		}
		logger.Info().Msgf("finished getting column types for table: %s.%s", schemaName, tableName)
	// TODO(janexing): support mysql.
	default:
		return nil, errors.New("not supported conn type")
	}

	return res, nil
}
