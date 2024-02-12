package fetch

import (
	"context"
	"fmt"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	crdbtypes "github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/utils"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
)

type columnsWithType []columnWithType

// CRDBCreateTableStmt returns a create table statement string with columnsWithType
// as the column clause.
func (cs columnsWithType) CRDBCreateTableStmt() (string, error) {
	tName, err := parser.ParseQualifiedTableName(cs[0].tableName)
	if err != nil {
		return "", err
	}
	res := tree.CreateTable{
		Table: *tName,
	}

	pkList := columnsWithType{}
	for _, col := range cs {
		if col.isPrimaryKey {
			pkList = append(pkList, col)
		}
	}

	// If there is only one pk, we simply need to park this particular column as
	// pk. If there are more than one pks, we need to create a pk constraint
	// that group all the selected columns, thus result in different syntax.
	includePkForEachCol := len(pkList) <= 1
	for _, col := range cs {
		res.Defs = append(res.Defs, col.CRDBColDef(includePkForEachCol))
	}

	if !includePkForEachCol {
		pkColNode := tree.IndexElemList{}
		for _, pk := range pkList {
			pkColNode = append(pkColNode, tree.IndexElem{Column: tree.Name(pk.columnName)})
		}
		res.Defs = append(res.Defs, &tree.UniqueConstraintTableDef{
			PrimaryKey: true,
			IndexTableDef: tree.IndexTableDef{
				Name:    "primary",
				Columns: pkColNode,
			},
		})
	}

	createTableStr := res.String()
	return createTableStr, nil
}

type columnWithType struct {
	schemaName   string
	tableName    string
	columnName   string
	dataType     string
	typeOid      oid.Oid
	notNullable  bool
	isPrimaryKey bool
}

func (t *columnWithType) CRDBColDef(includePk bool) *tree.ColumnTableDef {
	res := &tree.ColumnTableDef{
		Name: tree.Name(t.columnName),
		Type: crdbtypes.OidToType[t.typeOid],
	}
	if !t.notNullable {
		res.Nullable.Nullability = parser.NULL
	}
	if t.isPrimaryKey && includePk {
		res.PrimaryKey.IsPrimaryKey = true
	}
	return res
}

func (t *columnWithType) Name() string {
	return fmt.Sprintf("%s.%s.%s", t.schemaName, t.tableName, t.columnName)
}

func (t *columnWithType) String() string {
	return fmt.Sprintf("schema:%q, table:%q, column:%q, type:%q, typeoid: %d, nullable:%t, pk:%t\n",
		t.schemaName, t.tableName, t.columnName, t.dataType, t.typeOid, t.notNullable, t.isPrimaryKey)
}

func GetColumnTypes(
	ctx context.Context, logger zerolog.Logger, conn dbconn.Conn, table utils.MissingTable,
) (columnsWithType, error) {
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
	logger.Info().Msgf("getting column types for table: %s", table.String())

	switch conn := conn.(type) {
	case *dbconn.PGConn:
		rows, err := conn.Query(ctx, pgQuery, table.Table, table.Schema)
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
		logger.Info().Msgf("finished getting column types for table: %s", table.String())
	// TODO(janexing): support mysql.
	default:
		return nil, errors.New("not supported conn type")
	}

	return res, nil
}
