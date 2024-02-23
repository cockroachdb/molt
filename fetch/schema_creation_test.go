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
						typeOid:      oid.T_int4,
					},
					"smallint_col": {
						dataType: "smallint",
						nullable: true,
						typeOid:  oid.T_int2,
					},
					"bigint_col": {
						dataType: "bigint",
						typeOid:  oid.T_int8,
					},
					"decimal_col": {
						dataType: "decimal",
						nullable: true,
						typeOid:  oid.T_numeric,
					},
					"float_col": {
						dataType: "float",
						nullable: true,
						typeOid:  oid.T_float4,
					},
					"double_col": {
						dataType: "double",
						nullable: true,
						typeOid:  oid.T_float8,
					},
					"bit_col": {
						dataType: "bit",
						typeOid:  oid.T_varbit,
					},
					"date_col": {
						dataType: "date",
						nullable: true,
						typeOid:  oid.T_date,
					},
					"datetime_col": {
						dataType: "datetime",
						nullable: true,
						typeOid:  oid.T_timestamp,
					},
					"timestamp_col": {
						dataType: "timestamp",
						nullable: true,
						typeOid:  oid.T_timestamptz,
					},
					"time_col": {
						dataType: "time",
						typeOid:  oid.T_time,
					},
					"char_col": {
						dataType: "char",
						nullable: true,
						typeOid:  oid.T_varchar,
					},
					"varchar_col": {
						dataType: "varchar",
						nullable: true,
						typeOid:  oid.T_varchar,
					},
					"binary_col": {
						dataType: "binary",
						nullable: true,
						typeOid:  oid.T_bytea,
					},
					"varbinary_col": {
						dataType: "varbinary",
						nullable: true,
						typeOid:  oid.T_bytea,
					},
					"blob_col": {
						dataType: "blob",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"text_col": {
						dataType: "text",
						typeOid:  oid.T_text,
					},
					"mediumtext_col": {
						dataType: "mediumtext",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"longtext_col": {
						dataType: "longtext",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"json_col": {
						dataType: "json",
						nullable: true,
						typeOid:  oid.T_jsonb,
					},
					"enum_col": {
						dataType:      "enum",
						nullable:      true,
						typeOid:       oid.T_anyenum,
						udtName:       `get_column_types_test_table_enum_col_enum`,
						udtDefinition: `CREATE TYPE IF NOT EXISTS get_column_types_test_table_enum_col_enum AS ENUM ('value1','value2','value3')`,
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
						typeOid:      oid.T_int4,
					},
					"smallint_col": {
						dataType:     "smallint",
						isPrimaryKey: true,
						typeOid:      oid.T_int2,
					},
					"bigint_col": {
						dataType: "bigint",
						nullable: true,
						typeOid:  oid.T_int8,
					},
					"decimal_col": {
						dataType: "decimal",
						nullable: true,
						typeOid:  oid.T_numeric,
					},
					"float_col": {
						dataType: "float",
						nullable: true,
						typeOid:  oid.T_float4,
					},
					"double_col": {
						dataType: "double",
						nullable: true,
						typeOid:  oid.T_float8,
					},
					"bit_col": {
						dataType: "bit",
						nullable: true,
						typeOid:  oid.T_varbit,
					},
					"date_col": {
						dataType: "date",
						nullable: true,
						typeOid:  oid.T_date,
					},
					"datetime_col": {
						dataType: "datetime",
						nullable: true,
						typeOid:  oid.T_timestamp,
					},
					"timestamp_col": {
						dataType: "timestamp",
						nullable: true,
						typeOid:  oid.T_timestamptz,
					},
					"time_col": {
						dataType: "time",
						nullable: true,
						typeOid:  oid.T_time,
					},
					"char_col": {
						dataType: "char",
						nullable: true,
						typeOid:  oid.T_varchar,
					},
					"varchar_col": {
						dataType: "varchar",
						nullable: true,
						typeOid:  oid.T_varchar,
					},
					"binary_col": {
						dataType: "binary",
						nullable: true,
						typeOid:  oid.T_bytea,
					},
					"varbinary_col": {
						dataType: "varbinary",
						nullable: true,
						typeOid:  oid.T_bytea,
					},
					"blob_col": {
						dataType: "blob",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"text_col": {
						dataType: "text",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"mediumtext_col": {
						dataType: "mediumtext",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"longtext_col": {
						dataType: "longtext",
						nullable: true,
						typeOid:  oid.T_text,
					},
					"json_col": {
						dataType: "json",
						nullable: true,
						typeOid:  oid.T_jsonb,
					},
					"enum_col": {
						dataType:      "enum",
						nullable:      true,
						typeOid:       oid.T_anyenum,
						udtName:       `get_column_types_test_table_multi_pk_enum_col_enum`,
						udtDefinition: `CREATE TYPE IF NOT EXISTS get_column_types_test_table_multi_pk_enum_col_enum AS ENUM ('value1','value2','value3')`,
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

func TestCreateTableStatement(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stderr)

	type testcase struct {
		dialect                  testutils.Dialect
		desc                     string
		createTableStatements    []string
		tableFilter              utils.FilterConfig
		expectedCreateTableStmts []string
		expectedErr              string
	}

	const dbName = "create_new_schema"
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
			expectedCreateTableStmts: []string{
				`CREATE TABLE employees (id INT4 NOT NULL PRIMARY KEY, unique_id UUID NOT NULL, name VARCHAR NOT NULL, created_at TIMESTAMPTZ, updated_at DATE, is_hired BOOL, age INT2, salary DECIMAL, bonus FLOAT4)`,
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
			expectedCreateTableStmts: []string{
				`CREATE TABLE employees (id INT4 NOT NULL, unique_id UUID NOT NULL, name VARCHAR NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at DATE, is_hired BOOL, age INT2, salary DECIMAL, bonus FLOAT4, CONSTRAINT "primary" PRIMARY KEY (id, unique_id, created_at))`,
			},
		},
		{
			dialect: testutils.PostgresDialect,
			desc:    "foreign key is ignored",
			createTableStatements: []string{`
				CREATE TABLE department (
				   department_id SERIAL PRIMARY KEY,
				   department_name VARCHAR(50) NOT NULL
				);
		
				`,
				`
				CREATE TABLE employee (
				   employee_id SERIAL PRIMARY KEY,
				   employee_name VARCHAR(50) NOT NULL,
				   department_id INT REFERENCES department(department_id) ON DELETE CASCADE
				);
				`},
			tableFilter: utils.FilterConfig{TableFilter: `employee`},
			expectedCreateTableStmts: []string{
				`CREATE TABLE employee (employee_id INT4 NOT NULL PRIMARY KEY, employee_name VARCHAR NOT NULL, department_id INT4)`,
			},
		},
		{
			dialect: testutils.PostgresDialect,
			desc:    "unique, check and 2nd index are ignored",
			createTableStatements: []string{`
				CREATE TABLE employee (
				   id SERIAL PRIMARY KEY,
				   name VARCHAR(50) UNIQUE NOT NULL,
				   age INTEGER,
				   address VARCHAR(50) NOT NULL,
				   start_date DATE,
				   end_date DATE,
				   CONSTRAINT check_dates CHECK (start_date <= end_date),  -- Check Constraint
				   CONSTRAINT unique_constraint_name UNIQUE (start_date)  -- Secondary index
				);
		
				`,
				`
				CREATE UNIQUE INDEX my_unique_idx ON employee(age);
				`},
			tableFilter: utils.FilterConfig{TableFilter: `employee`},
			expectedCreateTableStmts: []string{
				`CREATE TABLE employee (id INT4 NOT NULL PRIMARY KEY, name VARCHAR NOT NULL, age INT4, address VARCHAR NOT NULL, start_date DATE, end_date DATE)`,
			},
		},
		{
			dialect: testutils.PostgresDialect,
			desc:    "enum column",
			createTableStatements: []string{`
		CREATE TYPE my_enum_type AS ENUM ('value1', 'value2', 'value3');
		`, `
		CREATE TABLE enum_table (
		   id INT NOT NULL PRIMARY KEY,
		   enum_column my_enum_type,
		   other_column1 TEXT
		);
		`,
			},
			tableFilter: utils.FilterConfig{TableFilter: `enum_table`},
			expectedCreateTableStmts: []string{
				` CREATE TYPE IF NOT EXISTS my_enum_type AS ENUM ('value1', 'value2', 'value3'); CREATE TABLE enum_table (id INT4 NOT NULL PRIMARY KEY, enum_column my_enum_type, other_column1 STRING)`},
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
    enum_col ENUM('value1', 'value2', 'value3') DEFAULT 'value2'
);
`,
			},
			tableFilter: utils.FilterConfig{TableFilter: `test_table`},
			expectedCreateTableStmts: []string{
				` CREATE TYPE IF NOT EXISTS create_new_schema_test_table_enum_col_enum AS ENUM ('value1','value2','value3'); CREATE TABLE test_table (integer_col INT4 NOT NULL PRIMARY KEY, smallint_col INT2, bigint_col INT8 NOT NULL, decimal_col DECIMAL, float_col FLOAT4, double_col FLOAT8, bit_col VARBIT NOT NULL, date_col DATE, datetime_col TIMESTAMP, timestamp_col TIMESTAMPTZ, time_col TIME NOT NULL, char_col VARCHAR, varchar_col VARCHAR, binary_col BYTES, varbinary_col BYTES, blob_col STRING, text_col STRING NOT NULL, mediumtext_col STRING, longtext_col STRING, json_col JSONB, enum_col create_new_schema_test_table_enum_col_enum)`},
		},
		{
			dialect: testutils.MySQLDialect,
			desc:    "multiple pk",
			createTableStatements: []string{`
CREATE TABLE test_table_multi_pk (
    integer_col INT,
    smallint_col SMALLINT,
    bigint_col BIGINT,
    decimal_col DECIMAL(10,2) CHECK (decimal_col <= 10),
    float_col FLOAT UNIQUE,
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
    PRIMARY KEY (integer_col, smallint_col)
);
`,
			},
			tableFilter: utils.FilterConfig{TableFilter: `test_table_multi_pk`},
			expectedCreateTableStmts: []string{
				` CREATE TYPE IF NOT EXISTS create_new_schema_test_table_multi_pk_enum_col_enum AS ENUM ('value1','value2','value3'); CREATE TABLE test_table_multi_pk (integer_col INT4 NOT NULL, smallint_col INT2 NOT NULL, bigint_col INT8, decimal_col DECIMAL, float_col FLOAT4, double_col FLOAT8, bit_col VARBIT, date_col DATE, datetime_col TIMESTAMP, timestamp_col TIMESTAMPTZ, time_col TIME, char_col VARCHAR, varchar_col VARCHAR, binary_col BYTES, varbinary_col BYTES, blob_col STRING, text_col STRING, mediumtext_col STRING, longtext_col STRING, json_col JSONB, enum_col create_new_schema_test_table_multi_pk_enum_col_enum, CONSTRAINT "primary" PRIMARY KEY (integer_col, smallint_col))`},
		}, {
			dialect: testutils.MySQLDialect,
			desc:    "unsupported type",
			createTableStatements: []string{`
CREATE TABLE test_table_set_col (
    integer_col INT PRIMARY KEY,
    set_col SET('value1', 'value2', 'value3')
);
`,
			},
			tableFilter: utils.FilterConfig{TableFilter: `test_table_set_col`},
			expectedErr: "set not yet handled",
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

			for _, stmt := range tc.createTableStatements {
				_, err = testutils.ExecConnQuery(ctx, stmt, conns[0])
				require.NoError(t, err)
			}

			missingTables, err := getFilteredMissingTables(ctx, conns, tc.tableFilter)
			require.NoError(t, err)

			if tc.expectedErr == "" {
				require.Equal(t, len(tc.expectedCreateTableStmts), len(missingTables))
			}

			for i, missingTable := range missingTables {
				actualCreateTableStmt, err := GetCreateTableStmt(ctx, logger, conns[0], missingTable.DBTable)
				if tc.expectedErr != "" {
					require.ErrorContains(t, err, tc.expectedErr)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expectedCreateTableStmts[i], actualCreateTableStmt)
				}
			}

			t.Logf("test passed!")
		})
	}

}
