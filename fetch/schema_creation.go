package fetch

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	crdbtypes "github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/utils/typeconv"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
)

type columnsWithType []columnWithType

// CRDBCreateTableStmt returns a create table statement string with columnsWithType
// as the column clause.
func (cs columnsWithType) CRDBCreateTableStmt(logger zerolog.Logger) (string, error) {
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
		colDef, err := col.CRDBColDef(includePkForEachCol, logger)
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

	return res.String(), nil
}

type columnWithType struct {
	// The following fields are shared information for columns from all dialects.
	schemaName      string
	tableName       string
	columnName      string
	dataType        string
	columnType      string
	typeOid         oid.Oid
	nullable        bool
	isPrimaryKey    bool
	udtName         string
	udtDefinition   string
	ordinalPosition int

	// mysqlMeta stores the mysql column dedicated information.
	mysqlMeta *typeconv.MySQLColumnMeta
}

func (t *columnWithType) CRDBColDef(
	includePk bool, logger zerolog.Logger,
) (*tree.ColumnTableDef, error) {
	var colType tree.ResolvableTypeReference
	var err error
	var scs []*typeconv.TypeConvError
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
		if t.mysqlMeta == nil {
			// If this is from a PG source.
			colType = crdbtypes.OidToType[t.typeOid]
		} else {
			// If this is from a mysql source.
			colType, scs = t.mysqlMeta.ToDefaultCRDBType(t.dataType, t.columnName)
			for _, sc := range scs {
				if sc.Blocking {
					err = errors.Newf(
						"failed to get crdb type from mysql type %s for column %s.%s: %s",
						t.dataType,
						t.tableName,
						t.columnName,
						sc.ShortDescription,
					)
					logger.Err(err)
					return nil, err
				}
				logger.Warn().Msgf("mysql type %s for column %s.%s: %s", t.dataType, t.tableName, t.columnName, sc.ShortDescription)
			}
		}
	}

	res := &tree.ColumnTableDef{
		Name: tree.Name(t.columnName),
		Type: colType,
	}
	if t.nullable {
		res.Nullable.Nullability = tree.SilentNull
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
	ctx context.Context,
	logger zerolog.Logger,
	conn dbconn.Conn,
	table dbtable.DBTable,
	skipUnsupportedTypeErr bool,
) (columnsWithType, error) {
	const (
		pgQuery = `SELECT DISTINCT
    t1.schema_name,
    t1.table_name,
    t1.column_name,
    t1.data_type,
    t1.type_oid,
    t1.nullable,
    t1.is_primary_key,
    COALESCE(t2.udt_name, '') AS enum_type,
    COALESCE(t2.udt_def, '') AS enum_type_definition,
    t2.ordinal_position
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
    c.TABLE_SCHEMA, 
    c.TABLE_NAME, 
    c.COLUMN_NAME, 
    c.DATA_TYPE,
    COALESCE(c.CHARACTER_MAXIMUM_LENGTH, '-1') AS CHARACTER_MAXIMUM_LENGTH,
    COALESCE(c.CHARACTER_OCTET_LENGTH,'-1') AS CHARACTER_OCTET_LENGTH,
    COALESCE(c.COLUMN_TYPE, '') AS COLUMN_TYPE,
    COALESCE(c.NUMERIC_PRECISION,'-1') AS NUMERIC_PRECISION,
    COALESCE(c.NUMERIC_SCALE,'-1') AS NUMERIC_SCALE,
    COALESCE(c.DATETIME_PRECISION,'-1') AS DATETIME_PRECISION,
    COALESCE(c.CHARACTER_SET_NAME,'') AS CHARACTER_SET_NAME,
    COALESCE(c.COLLATION_NAME,'') AS COLLATION_NAME,
    COALESCE(c.COLUMN_DEFAULT, '') AS COLUMN_DEFAULT,
    CASE 
        WHEN c.IS_NULLABLE = 'YES' THEN 'TRUE'
        ELSE 'FALSE' 
    END AS NULLABLE,
    CASE 
        WHEN c.COLUMN_KEY = 'PRI' THEN 'TRUE'
        ELSE 'FALSE'
    END AS IS_PRIMARY_KEY
FROM 
    information_schema.COLUMNS c
JOIN 
    information_schema.TABLES t ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
WHERE 
    c.TABLE_SCHEMA = DATABASE() 
    AND t.TABLE_TYPE = 'BASE TABLE'
    AND c.TABLE_NAME = '%s'
ORDER BY 
    c.TABLE_SCHEMA, 
    c.TABLE_NAME,  
    c.ORDINAL_POSITION;
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
			if err := rows.Scan(&newCol.schemaName, &newCol.tableName, &newCol.columnName, &newCol.dataType, &newCol.typeOid, &newCol.nullable, &newCol.isPrimaryKey, &newCol.udtName, &newCol.udtDefinition, &newCol.ordinalPosition); err != nil {
				return nil, errors.Wrap(err, "failed to scan query result to a columnWithType object")
			}
			logger.Debug().Msgf("collected column:%s", newCol.String())
			res = append(res, newCol)
		}
	case *dbconn.MySQLConn:
		q := fmt.Sprintf(mysqlQuery, table.Table)
		rows, err := conn.Query(q)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			mysqlMt := &typeconv.MySQLColumnMeta{}
			newCol := columnWithType{mysqlMeta: mysqlMt}
			if err := rows.Scan(
				&newCol.schemaName,
				&newCol.tableName,
				&newCol.columnName,
				&newCol.dataType,
				&mysqlMt.CharMaxLen,
				&mysqlMt.CharOctetLen,
				&mysqlMt.ColumnType,
				&mysqlMt.NumPrecision,
				&mysqlMt.NumScale,
				&mysqlMt.DatetimePrecision,
				&mysqlMt.CharSetName,
				&mysqlMt.CollationName,
				&mysqlMt.ColumnDefault,
				&newCol.nullable, &newCol.isPrimaryKey); err != nil {
				return nil, errors.Wrap(err, "failed to scan query result to a columnWithType object")
			}
			logger.Debug().Msgf("collected column:%s", newCol.String())

			if strings.ToLower(newCol.dataType) == enumTypeStr {
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
	ctx context.Context, logger zerolog.Logger, conn dbconn.Conn, table dbtable.DBTable,
) (string, error) {
	newCols, err := GetColumnTypes(ctx, logger, conn, table, false /* skipUnsupportedTypeErr */)
	if err != nil {
		return "", errors.Wrapf(err, "failed get columns for target table: %s", table.String())
	}

	var res string
	for _, col := range newCols {
		if col.udtDefinition != "" {
			logger.Info().Msgf("the original schema contains enum type %q. A tentative enum type will be created as %q", col.udtName, col.udtDefinition)
			res = strings.Join([]string{res, col.udtDefinition}, " ")
		}
	}
	createTableStmt, err := newCols.CRDBCreateTableStmt(logger)
	if err != nil {
		return "", err
	}

	if res != "" {
		return strings.TrimPrefix(strings.Join([]string{res, createTableStmt}, " "), " "), nil
	}
	return createTableStmt, nil
}

const enumTypeStr = "enum"

func convertMySQLEnum(
	newCol columnWithType,
) (createEnumStmt string, enumTypeName string, err error) {
	if newCol.mysqlMeta == nil || newCol.mysqlMeta.ColumnType == "" {
		return "", "", errors.Newf("original type is enum but with empty column type definition")
	}
	enumTypeName = fmt.Sprintf("%s_%s_%s_enum", newCol.schemaName, newCol.tableName, newCol.columnName)
	pattern := regexp.MustCompile(`enum(\(.+\))`)
	createEnumStmt = newCol.mysqlMeta.ColumnType
	matches := pattern.FindAllStringSubmatch(createEnumStmt, -1)
	if len(matches) == 0 {
		return "", "", errors.Newf("cannot extract enum values from the original enum expression: %q", newCol.columnType)
	}
	for _, match := range matches {
		if len(match) < 2 {
			return "", "", errors.Newf("cannot extract enum values from matched enum expression: %q", match)
		}
		enumValues := match[1]
		output := fmt.Sprintf("CREATE TYPE IF NOT EXISTS %s AS ENUM %s;", enumTypeName, enumValues)
		createEnumStmt = pattern.ReplaceAllString(createEnumStmt, output)
	}
	return createEnumStmt, enumTypeName, nil
}

type constraints []string

type constraintsWithTable struct {
	table dbtable.DBTable
	cons  constraints
}

func (ct *constraintsWithTable) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("table: %s,", ct.table))
	for i, con := range ct.cons {
		b.WriteString(fmt.Sprintf("%q", con))
		if i != len(ct.cons)-1 {
			b.WriteString(",")
		}
	}
	return b.String()
}

func GetConstraints(
	ctx context.Context, logger zerolog.Logger, conn dbconn.Conn, table dbtable.DBTable,
) ([]string, error) {
	const (
		pgQuery = `SELECT         
        pg_catalog.pg_get_constraintdef(c.oid) AS constraint_def
        FROM pg_catalog.pg_class s
        JOIN pg_catalog.pg_constraint c ON (s.oid = c.conrelid)
        WHERE conparentid = 0 
          AND s.relkind = 'r' -- 'r' indicates a table (relation)
          AND c.contype != 'p' -- 'p' indicates a primary key constraint
          AND s.relname= $1
          AND s.relnamespace::regnamespace::text = $2 
        ORDER BY conrelid, conname;`
		mysqlQuery = `SHOW CREATE TABLE %s`
	)

	var res []string
	switch conn := conn.(type) {
	case *dbconn.PGConn:
		rows, err := conn.Query(ctx, pgQuery, table.Table, table.Schema)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get the constraints for table %s", table.Table)
		}
		for rows.Next() {
			var constraintStmt string
			if err := rows.Scan(&constraintStmt); err != nil {
				return nil, err
			}
			res = append(res, constraintStmt)
		}
	case *dbconn.MySQLConn:
		rows, err := conn.Query(fmt.Sprintf(mysqlQuery, table.Table))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get the constraints for table %s", table.Table)
		}
		var tableName string
		var createTableStmt string
		for rows.Next() {
			if err := rows.Scan(&tableName, &createTableStmt); err != nil {
				return nil, errors.Wrapf(err, "failed to scan results to get the constraints for table %s", table.Table)
			}
			res = append(res, formatMySQLConstraints(createTableStmt)...)
		}
	}
	return res, nil
}

func formatMySQLConstraints(createTableStmt string) []string {
	var res []string
	const (
		uniqueKeyMySQLRegex = `UNIQUE KEY [^\n]+`
		fkMySQLRegex        = `CONSTRAINT \S+ FOREIGN KEY [^\n]+`
		checkMySQLRegex     = `CONSTRAINT \S+ CHECK [^\n]+`
	)

	for _, rx := range []string{
		uniqueKeyMySQLRegex,
		fkMySQLRegex,
		checkMySQLRegex,
	} {
		ks := regexp.MustCompile(rx).FindAllStringSubmatch(createTableStmt, -1)
		if len(ks) > 0 {
			for _, kgroup := range ks {
				res = append(res, strings.TrimSuffix(kgroup[0], ","))
			}
		}
	}
	return res
}
