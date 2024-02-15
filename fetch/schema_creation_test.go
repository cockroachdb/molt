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

	// TODO(janexing): add crdb and mysql.
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
						notNullable:  true,
						isPrimaryKey: true,
					},
					"name": {
						dataType:    "character varying(50)",
						typeOid:     oid.T_varchar,
						notNullable: true,
					},
					"created_at": {
						dataType: "timestamp with time zone",
						typeOid:  oid.T_timestamptz,
					},
					"is_hired": {
						dataType: "boolean",
						typeOid:  oid.T_bool,
					},
					"salary": {
						dataType: "numeric(8,2)",
						typeOid:  oid.T_numeric,
					},
					"bonus": {
						dataType: "real",
						typeOid:  oid.T_float4,
					},
					"unique_id": {
						dataType:    "uuid",
						typeOid:     oid.T_uuid,
						notNullable: true,
					},
					"updated_at": {
						dataType: "date",
						typeOid:  oid.T_date,
					},
					"age": {
						dataType: "smallint",
						typeOid:  oid.T_int2,
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
						notNullable:  true,
						isPrimaryKey: true,
					},
					"name": {
						dataType:    "character varying(50)",
						typeOid:     oid.T_varchar,
						notNullable: true,
					},
					"created_at": {
						dataType:     "timestamp with time zone",
						typeOid:      oid.T_timestamptz,
						isPrimaryKey: true,
						notNullable:  true,
					},
					"is_hired": {
						dataType: "boolean",
						typeOid:  oid.T_bool,
					},
					"salary": {
						dataType: "numeric(8,2)",
						typeOid:  oid.T_numeric,
					},
					"bonus": {
						dataType: "real",
						typeOid:  oid.T_float4,
					},
					"unique_id": {
						dataType:     "uuid",
						typeOid:      oid.T_uuid,
						notNullable:  true,
						isPrimaryKey: true,
					},
					"updated_at": {
						dataType: "date",
						typeOid:  oid.T_date,
					},
					"age": {
						dataType: "smallint",
						typeOid:  oid.T_int2,
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
						notNullable:  true,
						isPrimaryKey: true,
					},
					"enum_column": {
						dataType:      "my_enum_type",
						udtName:       "my_enum_type",
						udtDefinition: "CREATE TYPE IF NOT EXISTS my_enum_type AS ENUM ('value1', 'value2', 'value3');",
					},
					"other_column1": {
						dataType: "text",
						typeOid:  oid.T_text,
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
				newCols, err := GetColumnTypes(ctx, logger, conns[0], missingTable.DBTable)
				require.NoError(t, err)
				res[missingTable.String()] = make(map[string]columnWithType)
				for _, c := range newCols {
					res[missingTable.String()][c.columnName] = c
				}
			}

			for mt, actualCols := range res {
				expectedCols := tc.expectedColumnTypes[mt]
				require.Equal(t, len(expectedCols), len(actualCols))
				for _, actualCol := range actualCols {
					require.NoError(t, checkIfColInfoEqual(actualCol, expectedCols[actualCol.columnName]))
				}
			}
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
	if actual.notNullable != expected.notNullable {
		return errors.AssertionFailedf("[%s] expected notNullable: %t, but got: %t", actual.Name(), expected.notNullable, actual.notNullable)
	}
	if actual.isPrimaryKey != expected.isPrimaryKey {
		return errors.AssertionFailedf("[%s] expected isPrimaryKey: %t, but got: %t", actual.Name(), expected.isPrimaryKey, actual.isPrimaryKey)
	}
	if expected.udtName != "" {
		if actual.udtName != expected.udtName {
			return errors.AssertionFailedf("[%s] expected udtName: %s, but got: %s", actual.Name(), expected.udtName, actual.udtName)
		}
		if actual.udtDefinition != expected.udtDefinition {
			return errors.AssertionFailedf("[%s] expected udtDefinition: %s, but got: %s", actual.Name(), expected.udtDefinition, actual.udtDefinition)
		}
	} else {
		if actual.typeOid != expected.typeOid {
			return errors.AssertionFailedf("[%s] expected typeOid: %s, but got: %s", actual.Name(), expected.typeOid, actual.typeOid)
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
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.dialect.String(), tc.desc), func(t *testing.T) {
			var conns dbconn.OrderedConns
			var err error
			switch tc.dialect {
			case testutils.PostgresDialect:
				conns[0], err = dbconn.TestOnlyCleanDatabase(ctx, "source", testutils.PGConnStr(), fmt.Sprintf("%s-%d", dbName, idx))
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

			require.Equal(t, len(tc.expectedCreateTableStmts), len(missingTables))

			for i, missingTable := range missingTables {
				actualCreateTableStmt, err := GetCreateTableStmt(ctx, logger, conns[0], missingTable.DBTable)
				require.NoError(t, err)
				require.Equal(t, tc.expectedCreateTableStmts[i], actualCreateTableStmt)
			}

			t.Logf("test passed!")
		})
	}

}
