package fetch

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestGetColumnTypes(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stderr)

	type testcase struct {
		dialect               testutils.Dialect
		desc                  string
		createTableStatements []string
		tableFilter           utils.FilterConfig
		expectedColumnTypes   map[string]map[string]columnWithType
	}

	const dbName = "get_column_types"

	for idx, tc := range []testcase{
		{
			dialect: testutils.PostgresDialect,
			desc:    "single pk",
			createTableStatements: []string{`
				CREATE TABLE employees (
				   id INT PRIMARY KEY,
				   unique_id UUID NOT NULL,
				   name VARCHAR(50) NOT NULL,
				   created_at TIMESTAMPTZ,
				   updated_at DATE,
				   is_hired BOOLEAN,
				   age SMALLINT CHECK (age > 18),
				   salary NUMERIC(8, 2),
				   bonus REAL unique
				);
				`},
			tableFilter: utils.FilterConfig{TableFilter: `employees`},
			expectedColumnTypes: map[string]map[string]columnWithType{
				"public.employees": {
					"id": {
						dataType:     "integer",
						typeOid:      oid.T_int4,
						isPrimaryKey: true,
					},
					"name": {
						dataType: "character varying(50)",
						typeOid:  oid.T_varchar,
					},
					"created_at": {
						dataType: "timestamp with time zone",
						typeOid:  oid.T_timestamptz,
						nullable: true,
					},
					"is_hired": {
						dataType: "boolean",
						typeOid:  oid.T_bool,
						nullable: true,
					},
					"salary": {
						dataType: "numeric(8,2)",
						typeOid:  oid.T_numeric,
						nullable: true,
					},
					"bonus": {
						dataType: "real",
						typeOid:  oid.T_float4,
						nullable: true,
					},
					"unique_id": {
						dataType: "uuid",
						typeOid:  oid.T_uuid,
					},
					"updated_at": {
						dataType: "date",
						typeOid:  oid.T_date,
						nullable: true,
					},
					"age": {
						dataType: "smallint",
						typeOid:  oid.T_int2,
						nullable: true,
					},
				},
			},
		},
		{
			dialect: testutils.PostgresDialect,
			desc:    "multiple pks",
			createTableStatements: []string{`
				CREATE TABLE employees (
				   id INT NOT NULL,
				   unique_id UUID NOT NULL,
				   name VARCHAR(50) NOT NULL,
				   created_at TIMESTAMPTZ,
				   updated_at DATE,
				   is_hired BOOLEAN,
				   age SMALLINT CHECK (age > 18),
				   salary NUMERIC(8, 2),
				   bonus REAL unique,
				   CONSTRAINT "primary" PRIMARY KEY (id, unique_id, created_at)
				);
				`},
			tableFilter: utils.FilterConfig{TableFilter: `employees`},
			expectedColumnTypes: map[string]map[string]columnWithType{
				"public.employees": {
					"id": {
						dataType:     "integer",
						typeOid:      oid.T_int4,
						isPrimaryKey: true,
					},
					"name": {
						dataType: "character varying(50)",
						typeOid:  oid.T_varchar,
					},
					"created_at": {
						dataType:     "timestamp with time zone",
						typeOid:      oid.T_timestamptz,
						isPrimaryKey: true,
					},
					"is_hired": {
						dataType: "boolean",
						typeOid:  oid.T_bool,
						nullable: true,
					},
					"salary": {
						dataType: "numeric(8,2)",
						typeOid:  oid.T_numeric,
						nullable: true,
					},
					"bonus": {
						dataType: "real",
						typeOid:  oid.T_float4,
						nullable: true,
					},
					"unique_id": {
						dataType:     "uuid",
						typeOid:      oid.T_uuid,
						isPrimaryKey: true,
					},
					"updated_at": {
						dataType: "date",
						typeOid:  oid.T_date,
						nullable: true,
					},
					"age": {
						dataType: "smallint",
						typeOid:  oid.T_int2,
						nullable: true,
					},
				},
			},
		},
		{
			dialect: testutils.PostgresDialect,
			desc:    "enums",
			createTableStatements: []string{`
		CREATE TYPE my_enum_type AS ENUM ('value1', 'value2', 'value3');
		`, `
			CREATE TABLE enum_table (
			   id INT NOT NULL PRIMARY KEY,
			   enum_column my_enum_type,
			   other_column1 TEXT
			);
		`},
			tableFilter: utils.FilterConfig{TableFilter: `enum_table`},
			expectedColumnTypes: map[string]map[string]columnWithType{
				"public.enum_table": {
					"id": {
						dataType:     "integer",
						typeOid:      oid.T_int4,
						isPrimaryKey: true,
					},
					"enum_column": {
						dataType:      "my_enum_type",
						nullable:      true,
						udtName:       "my_enum_type",
						udtDefinition: "CREATE TYPE IF NOT EXISTS my_enum_type AS ENUM ('value1', 'value2', 'value3');",
					},
					"other_column1": {
						dataType: "text",
						nullable: true,
						typeOid:  oid.T_text,
					},
				},
			},
		},
		{
			dialect: testutils.MySQLDialect,
			desc:    "single pk",
			createTableStatements: []string{`
				CREATE TABLE test_table (
					integer_col INT PRIMARY KEY,
					smallint_col SMALLINT,
					bigint_col BIGINT NOT NULL,
					decimal_col DECIMAL(10,2),
					float_col FLOAT,
					double_col DOUBLE,
					bit_col BIT NOT NULL,
					date_col DATE,
					datetime_col DATETIME,
					timestamp_col TIMESTAMP,
					time_col TIME NOT NULL,
					char_col CHAR(10),
					varchar_col VARCHAR(255),
					binary_col BINARY(10) UNIQUE,
					varbinary_col VARBINARY(255),
					blob_col BLOB,
					text_col TEXT NOT NULL,
					mediumtext_col MEDIUMTEXT,
					longtext_col LONGTEXT,
					json_col JSON,
					enum_col ENUM('value1', 'value2', 'value3'),
					set_col SET('value1', 'value2', 'value3')
);
		`},
			tableFilter: utils.FilterConfig{TableFilter: `test_table`},
			expectedColumnTypes: map[string]map[string]columnWithType{
				"public.test_table": {
					"integer_col": {
						dataType:     "int",
						isPrimaryKey: true,
					},
					"smallint_col": {
						dataType: "smallint",
						nullable: true,
					},
					"bigint_col": {
						dataType: "bigint",
					},
					"decimal_col": {
						dataType: "decimal",
						nullable: true,
					},
					"float_col": {
						dataType: "float",
						nullable: true,
					},
					"double_col": {
						dataType: "double",
						nullable: true,
					},
					"bit_col": {
						dataType: "bit",
					},
					"date_col": {
						dataType: "date",
						nullable: true,
					},
					"datetime_col": {
						dataType: "datetime",
						nullable: true,
					},
					"timestamp_col": {
						dataType: "timestamp",
						nullable: true,
					},
					"time_col": {
						dataType: "time",
					},
					"char_col": {
						dataType: "char",
						nullable: true,
					},
					"varchar_col": {
						dataType: "varchar",
						nullable: true,
					},
					"binary_col": {
						dataType: "binary",
						nullable: true,
					},
					"varbinary_col": {
						dataType: "varbinary",
						nullable: true,
					},
					"blob_col": {
						dataType: "blob",
						nullable: true,
					},
					"text_col": {
						dataType: "text",
					},
					"mediumtext_col": {
						dataType: "mediumtext",
						nullable: true,
					},
					"longtext_col": {
						dataType: "longtext",
						nullable: true,
					},
					"json_col": {
						dataType: "json",
						nullable: true,
					},
					"enum_col": {
						dataType:      "enum",
						nullable:      true,
						udtName:       `get_column_types_test_table_enum_col_enum`,
						udtDefinition: `CREATE TYPE IF NOT EXISTS get_column_types_test_table_enum_col_enum AS ENUM ('value1','value2','value3');`,
					},
					"set_col": {
						dataType: "set",
						nullable: true,
					},
				},
			},
		},
		{
			dialect: testutils.MySQLDialect,
			desc:    "multiple pk",
			createTableStatements: []string{`
				CREATE TABLE test_table_multi_pk (
					integer_col INT,
					smallint_col SMALLINT,
					bigint_col BIGINT,
					decimal_col DECIMAL(10,2),
					float_col FLOAT,
					double_col DOUBLE,
					bit_col BIT,
					date_col DATE,
					datetime_col DATETIME,
					timestamp_col TIMESTAMP,
					time_col TIME,
					char_col CHAR(10),
					varchar_col VARCHAR(255),
					binary_col BINARY(10),
					varbinary_col VARBINARY(255),
					blob_col BLOB,
					text_col TEXT,
					mediumtext_col MEDIUMTEXT,
					longtext_col LONGTEXT,
					json_col JSON,
					enum_col ENUM('value1', 'value2', 'value3'),
					set_col SET('value1', 'value2', 'value3'),
					PRIMARY KEY (integer_col, smallint_col)
);
		`},
			tableFilter: utils.FilterConfig{TableFilter: `test_table_multi_pk`},
			expectedColumnTypes: map[string]map[string]columnWithType{
				"public.test_table_multi_pk": {
					"integer_col": {
						dataType:     "int",
						isPrimaryKey: true,
					},
					"smallint_col": {
						dataType:     "smallint",
						isPrimaryKey: true,
					},
					"bigint_col": {
						dataType: "bigint",
						nullable: true,
					},
					"decimal_col": {
						dataType: "decimal",
						nullable: true,
					},
					"float_col": {
						dataType: "float",
						nullable: true,
					},
					"double_col": {
						dataType: "double",
						nullable: true,
					},
					"bit_col": {
						dataType: "bit",
						nullable: true,
					},
					"date_col": {
						dataType: "date",
						nullable: true,
					},
					"datetime_col": {
						dataType: "datetime",
						nullable: true,
					},
					"timestamp_col": {
						dataType: "timestamp",
						nullable: true,
					},
					"time_col": {
						dataType: "time",
						nullable: true,
					},
					"char_col": {
						dataType: "char",
						nullable: true,
					},
					"varchar_col": {
						dataType: "varchar",
						nullable: true,
					},
					"binary_col": {
						dataType: "binary",
						nullable: true,
					},
					"varbinary_col": {
						dataType: "varbinary",
						nullable: true,
					},
					"blob_col": {
						dataType: "blob",
						nullable: true,
					},
					"text_col": {
						dataType: "text",
						nullable: true,
					},
					"mediumtext_col": {
						dataType: "mediumtext",
						nullable: true,
					},
					"longtext_col": {
						dataType: "longtext",
						nullable: true,
					},
					"json_col": {
						dataType: "json",
						nullable: true,
					},
					"enum_col": {
						dataType:      "enum",
						nullable:      true,
						udtName:       `get_column_types_test_table_multi_pk_enum_col_enum`,
						udtDefinition: `CREATE TYPE IF NOT EXISTS get_column_types_test_table_multi_pk_enum_col_enum AS ENUM ('value1','value2','value3');`,
					},
					"set_col": {
						dataType: "set",
						nullable: true,
					},
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.dialect.String(), tc.desc), func(t *testing.T) {
			var conns dbconn.OrderedConns
			var err error
			switch tc.dialect {
			case testutils.PostgresDialect:
				conns[0], err = dbconn.TestOnlyCleanDatabase(ctx, "source", testutils.PGConnStr(), fmt.Sprintf("%s-%d", dbName, idx))
				require.NoError(t, err)
			case testutils.MySQLDialect:
				conns[0], err = dbconn.TestOnlyCleanDatabase(ctx, "source", testutils.MySQLConnStr(), dbName)
				require.NoError(t, err)
			default:
				t.Fatalf("unsupported dialect: %s", tc.dialect.String())
			}

			conns[1], err = dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), fmt.Sprintf("%s-%d", dbName, idx))
			require.NoError(t, err)

			// Check the 2 dbs are up.
			for _, c := range conns {
				_, err := testutils.ExecConnQuery(ctx, "SELECT 1", c)
				require.NoError(t, err)
			}

			defer func() {
				require.NoError(t, conns[0].Close(ctx))
				require.NoError(t, conns[1].Close(ctx))
			}()

			for _, stmt := range tc.createTableStatements {
				_, err = testutils.ExecConnQuery(ctx, stmt, conns[0])
				require.NoError(t, err)
			}

			missingTables, err := getFilteredMissingTables(ctx, conns, tc.tableFilter)
			require.NoError(t, err)

			res := make(map[string]map[string]columnWithType)

			for _, missingTable := range missingTables {
				newCols, err := GetColumnTypes(ctx, logger, conns[0], missingTable.DBTable, true /* skipUnsupportedTypeErr */)
				require.NoError(t, err)
				res[missingTable.String()] = make(map[string]columnWithType)
				for _, c := range newCols {
					res[missingTable.String()][c.columnName] = c
				}
			}

			var err1 error
			for mt, actualCols := range res {
				expectedCols := tc.expectedColumnTypes[mt]
				require.Equal(t, len(expectedCols), len(actualCols))
				for _, actualCol := range actualCols {
					if err = checkIfColInfoEqual(actualCol, expectedCols[actualCol.columnName]); err != nil {
						err1 = err
						t.Log(err)
					}
				}
			}
			require.NoError(t, err1)
			t.Logf("test passed!")
		})
	}
}

func getFilteredMissingTables(
	ctx context.Context, conns dbconn.OrderedConns, filter utils.FilterConfig,
) ([]utils.MissingTable, error) {
	dbTables, err := dbverify.Verify(ctx, conns)
	if err != nil {
		return nil, err
	}
	if dbTables, err = utils.FilterResult(filter, dbTables); err != nil {
		return nil, err
	}
	return dbTables.MissingTables, nil
}

func checkIfColInfoEqual(actual, expected columnWithType) error {
	if actual.dataType != expected.dataType {
		return errors.AssertionFailedf("[%s] expected datatype: %s, but got: %s", actual.Name(), expected.dataType, actual.dataType)
	}
	if actual.nullable != expected.nullable {
		return errors.AssertionFailedf("[%s] expected nullable: %t, but got: %t", actual.Name(), expected.nullable, actual.nullable)
	}
	if actual.isPrimaryKey != expected.isPrimaryKey {
		return errors.AssertionFailedf("[%s] expected isPrimaryKey: %t, but got: %t", actual.Name(), expected.isPrimaryKey, actual.isPrimaryKey)
	}
	if expected.typeOid != 0 && actual.typeOid != expected.typeOid {
		return errors.AssertionFailedf("[%s] expected typeOid: %s, but got: %s", actual.Name(), expected.typeOid, actual.typeOid)
	}
	if expected.udtName != "" {
		if actual.udtName != expected.udtName {
			return errors.AssertionFailedf("[%s] expected udtName: %s, but got: %s", actual.Name(), expected.udtName, actual.udtName)
		}
		if actual.udtDefinition != expected.udtDefinition {
			return errors.AssertionFailedf("[%s] expected udtDefinition: %s, but got: %s", actual.Name(), expected.udtDefinition, actual.udtDefinition)
		}
	}
	return nil
}
