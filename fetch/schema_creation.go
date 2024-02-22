package fetch

import (
	"context"
	"fmt"
	"github.com/cockroachdb/molt/mysqlconv"
	"regexp"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	crdbtypes "github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
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
		colDef, err := col.CRDBColDef(includePkForEachCol)
		if err != nil {
			return "", err
		}
		res.Defs = append(res.Defs, colDef)
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
	schemaName    string
	tableName     string
	columnName    string
	dataType      string
	columnType    string
	typeOid       oid.Oid
	nullable      bool
	isPrimaryKey  bool
	udtName       string
	udtDefinition string
}

func (t *columnWithType) CRDBColDef(includePk bool) (*tree.ColumnTableDef, error) {
	var colType tree.ResolvableTypeReference
	var err error
	if t.udtDefinition != "" {
		if t.udtName == "" {
			// This should not happen, but as a sanity check.
			return nil, errors.AssertionFailedf("user defined type definition %q is not null, but the type name is null", t.udtDefinition)
		}
		colType, err = tree.NewUnresolvedObjectName(1 /* numParts */, [3]string{t.udtName, "", ""}, tree.NoAnnotation /* annotation idx */)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse the type name %q", t.udtName)
		}
	} else {
		colType = crdbtypes.OidToType[t.typeOid]
	}

	res := &tree.ColumnTableDef{
		Name: tree.Name(t.columnName),
		Type: colType,
	}
	if t.nullable {
		res.Nullable.Nullability = parser.NULL
	}
	if t.isPrimaryKey && includePk {
		res.PrimaryKey.IsPrimaryKey = true
	}
	return res, nil
}

func (t *columnWithType) Name() string {
	return fmt.Sprintf("%s.%s.%s", t.schemaName, t.tableName, t.columnName)
}

func (t *columnWithType) String() string {
	return fmt.Sprintf("schema:%q, table:%q, column:%q, type:%q, typeoid: %d, nullable:%t, pk:%t\n",
		t.schemaName, t.tableName, t.columnName, t.dataType, t.typeOid, t.nullable, t.isPrimaryKey)
}

func GetColumnTypes(
	ctx context.Context, logger zerolog.Logger, conn dbconn.Conn, table dbtable.DBTable, testOnly bool,
) (columnsWithType, error) {
	const (
		pgQuery = `SELECT
    t1.schema_name,
    t1.table_name,
    t1.column_name,
    t1.data_type,
    t1.type_oid,
    t1.nullable,
    t1.is_primary_key,
    COALESCE(t2.udt_name, '') AS enum_type,
    COALESCE(t2.udt_def, '') AS enum_type_definition
FROM (
    SELECT
        c.relnamespace::regnamespace::text AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name,
        format_type(a.atttypid, a.atttypmod) AS data_type,
        a.atttypid AS type_oid,
        NOT a.attnotnull AS nullable,
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
    JOIN pg_catalog.pg_attribute a ON c.oid = a.attrelid
    LEFT JOIN pg_catalog.pg_index ix ON c.oid = ix.indrelid AND a.attnum = ANY(ix.indkey)
    WHERE
        c.relkind = 'r'  -- 'r' indicates a table (relation)
        AND a.attnum > 0 -- Exclude system columns
        AND c.relname = $1
        AND c.relnamespace::regnamespace::text = $2
) t1
LEFT JOIN (
    SELECT
        c.column_name,
        c.table_name,
        c.table_schema,
        c.udt_name,
        t.definition AS udt_def,
        c.ordinal_position
    FROM
        information_schema.columns c
    LEFT JOIN (
        SELECT
            'CREATE TYPE IF NOT EXISTS ' || t.typname || ' AS ENUM ' ||
            '(' || string_agg(quote_literal(e.enumlabel), ', ' ORDER BY e.enumsortorder) || ');' AS definition,
            t.typname
        FROM
            pg_type t
        JOIN pg_enum e ON t.oid = e.enumtypid
        GROUP BY
            t.typname
    ) t ON c.udt_name = t.typname
    WHERE
        c.table_name = $1 AND c.table_schema = $2
) t2 ON t1.column_name = t2.column_name
    AND t1.table_name = t2.table_name
    AND t1.schema_name = t2.table_schema
ORDER BY
    t1.schema_name,
    t1.table_name,
    t2.ordinal_position;
`
		mysqlQuery = `SELECT 
    TABLE_SCHEMA, 
    TABLE_NAME, 
    COLUMN_NAME, 
    DATA_TYPE,
    COLUMN_TYPE, 
    CASE 
        WHEN IS_NULLABLE = 'YES' THEN 'TRUE'
        ELSE 'FALSE' 
    END AS NULLABLE,
    CASE 
        WHEN COLUMN_KEY = 'PRI' THEN 'TRUE'
        ELSE 'FALSE'
    END AS IS_PRIMARY_KEY
FROM information_schema.COLUMNS 
WHERE 
    TABLE_SCHEMA = '%s' 
    AND TABLE_NAME = '%s'
ORDER BY 
    TABLE_SCHEMA, 
    TABLE_NAME,  
    ORDINAL_POSITION;
`
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
			if err := rows.Scan(&newCol.schemaName, &newCol.tableName, &newCol.columnName, &newCol.dataType, &newCol.typeOid, &newCol.nullable, &newCol.isPrimaryKey, &newCol.udtName, &newCol.udtDefinition); err != nil {
				return nil, errors.Wrap(err, "failed to scan query result to a columnWithType object")
			}
			logger.Debug().Msgf("collect column:%s", newCol.String())
			res = append(res, newCol)
		}
	case *dbconn.MySQLConn:
		q := fmt.Sprintf(mysqlQuery, table.Schema, table.Table)
		rows, err := conn.Query(q)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			newCol := columnWithType{}
			if err := rows.Scan(&newCol.schemaName, &newCol.tableName, &newCol.columnName, &newCol.dataType, &newCol.columnType, &newCol.nullable, &newCol.isPrimaryKey); err != nil {
				return nil, errors.Wrap(err, "failed to scan query result to a columnWithType object")
			}
			logger.Debug().Msgf("collect column:%s", newCol.String())
			pgOid, err := mysqlconv.DataTypeToOID(newCol.dataType, newCol.columnType)
			if err != nil && !testOnly {
				return nil, err
			}
			newCol.typeOid = pgOid
			if pgOid == oid.T_anyenum {
				udtDefinition, udtName, getUdtErr := convertMySQLEnum(newCol)
				if getUdtErr != nil {
					return nil, getUdtErr
				}
				newCol.udtDefinition = udtDefinition
				newCol.udtName = udtName
			}
			res = append(res, newCol)
		}
	default:
		return nil, errors.New("not supported conn type")
	}

	logger.Info().Msgf("finished getting column types for table: %s", table.String())
	return res, nil
}

func GetDropTableStmt(table dbtable.DBTable) (string, error) {
	tName, err := parser.ParseQualifiedTableName(table.Table.String())
	if err != nil {
		return "", err
	}
	res := tree.DropTable{
		Names:    tree.TableNames{*tName},
		IfExists: true,
	}

	return res.String(), nil
}

func GetCreateTableStmt(
	ctx context.Context, logger zerolog.Logger, conn dbconn.Conn, table dbtable.DBTable, testOnly bool,
) (string, error) {
	newCols, err := GetColumnTypes(ctx, logger, conn, table, testOnly)
	if err != nil {
		return "", errors.Wrapf(err, "failed get columns for target table: %s", table.String())
	}

	var res string
	for _, col := range newCols {
		if col.udtDefinition != "" {
			logger.Warn().Msgf("the original schema contains enum type %q, a tentative enum type creation is triggered via %q", col.udtName, col.udtDefinition)
			res = strings.Join([]string{res, col.udtDefinition}, " ")
		}
	}
	createTableStmt, err := newCols.CRDBCreateTableStmt()
	if err != nil {
		return "", err
	}

	if res != "" {
		return strings.Join([]string{res, createTableStmt}, " "), nil
	}
	return createTableStmt, nil
}

func convertMySQLEnum(newCol columnWithType) (createEnumStmt string, enumTypeName string, err error) {
	if newCol.columnType == "" {
		return "", "", errors.Newf("original type is enum but with empty column type definition")
	}
	enumTypeName = fmt.Sprintf("%s_%s_%s_enum", newCol.schemaName, newCol.tableName, newCol.columnName)
	// Define the regular expression pattern to match enum strings
	pattern := regexp.MustCompile(`enum(\(.+\))`)

	createEnumStmt = newCol.columnType
	// Find all matches in the input string
	matches := pattern.FindAllStringSubmatch(newCol.columnType, -1)

	if len(matches) == 0 {
		return "", "", errors.Newf("cannot extract enum values from the original enum expression: %q", newCol.columnType)
	}
	// Iterate over the matches and construct the desired output
	for _, match := range matches {
		if len(match) < 2 {
			return "", "", errors.Newf("cannot extract enum values from matched enum expression: %q", match)
		}
		enumValues := match[1]
		output := fmt.Sprintf("CREATE TYPE IF NOT EXISTS %s AS ENUM %s", enumTypeName, enumValues)
		createEnumStmt = pattern.ReplaceAllString(createEnumStmt, output)
	}

	return createEnumStmt, enumTypeName, nil
}
