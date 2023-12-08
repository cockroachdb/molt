package tableverify

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq/oid"
)

type Result struct {
	RowVerifiable bool
	dbtable.VerifiedTable
	MismatchingTableDefinitions []inconsistency.MismatchingTableDefinition
}

func VerifyCommonTables(
	ctx context.Context, conns dbconn.OrderedConns, allTables [][2]dbtable.DBTable,
) ([]Result, error) {
	var ret []Result

	for _, cmpTables := range allTables {
		pkCols, err := getPrimaryKeysForTables(ctx, conns, cmpTables)
		if err != nil {
			return nil, err
		}
		columns, err := getColumnsForTables(ctx, conns, cmpTables)
		if err != nil {
			return nil, err
		}
		res, err := verifyTable(ctx, conns, cmpTables, pkCols, columns)
		if err != nil {
			return nil, err
		}
		ret = append(ret, res)
	}
	return ret, nil
}

func verifyTable(
	ctx context.Context,
	conns dbconn.OrderedConns,
	cmpTables [2]dbtable.DBTable,
	pkCols [2][]tree.Name,
	columns [2][]Column,
) (Result, error) {
	truthTbl := cmpTables[0]
	var columnMap [2]map[tree.Name]Column
	for i := range columnMap {
		columnMap[i] = make(map[tree.Name]Column)
	}
	res := Result{
		VerifiedTable: dbtable.VerifiedTable{
			Name: dbtable.Name{
				Schema: truthTbl.Schema,
				Table:  truthTbl.Table,
			},
		},
	}
	truthCols := columns[0]
	for _, truthCol := range truthCols {
		columnMap[0][truthCol.Name] = truthCol
	}

	pkSame := true
	truthPKCols := pkCols[0]
	if len(truthPKCols) == 0 {
		res.MismatchingTableDefinitions = append(
			res.MismatchingTableDefinitions,
			inconsistency.MismatchingTableDefinition{
				DBTable: truthTbl,
				Info:    "missing a PRIMARY KEY - results cannot be compared",
			},
		)
	}

	comparableColumns := mapColumns(truthCols)
	truthMappedCols := mapColumns(truthCols)
	targetTbl := cmpTables[1]
	compareColumns := columns[1]
	for _, targetCol := range compareColumns {
		columnMap[1][targetCol.Name] = targetCol
		sourceCol, ok := truthMappedCols[targetCol.Name]
		if !ok {
			res.MismatchingTableDefinitions = append(
				res.MismatchingTableDefinitions,
				inconsistency.MismatchingTableDefinition{
					DBTable: targetTbl,
					Info:    fmt.Sprintf("extraneous column %s found", targetCol.Name),
				},
			)
			continue
		}

		delete(truthMappedCols, targetCol.Name)

		if sourceCol.NotNull != targetCol.NotNull {
			res.MismatchingTableDefinitions = append(
				res.MismatchingTableDefinitions,
				inconsistency.MismatchingTableDefinition{
					DBTable: targetTbl,
					Info: fmt.Sprintf(
						"column %s NOT NULL mismatch: %t vs %t",
						targetCol.Name,
						sourceCol.NotNull,
						targetCol.NotNull,
					),
				},
			)
		}
		truthTyp, err := dbconn.GetDataType(ctx, conns[0], sourceCol.OID)
		if err != nil {
			return Result{}, err
		}
		compareTyp, err := dbconn.GetDataType(ctx, conns[1], targetCol.OID)
		if err != nil {
			return Result{}, err
		}
		if !comparableType(truthTyp, compareTyp) {
			res.MismatchingTableDefinitions = append(
				res.MismatchingTableDefinitions,
				inconsistency.MismatchingTableDefinition{
					DBTable: targetTbl,
					Info: fmt.Sprintf(
						"column type mismatch on %s: %s vs %s",
						targetCol.Name,
						truthTyp.Name,
						compareTyp.Name,
					),
				},
			)
			delete(comparableColumns, sourceCol.Name)
		}
	}
	for colName := range truthMappedCols {
		res.MismatchingTableDefinitions = append(
			res.MismatchingTableDefinitions,
			inconsistency.MismatchingTableDefinition{
				DBTable: targetTbl,
				Info: fmt.Sprintf(
					"missing column %s",
					colName,
				),
			},
		)
		delete(comparableColumns, colName)
	}

	targetPKCols := pkCols[1]

	currPKSame := len(targetPKCols) == len(truthPKCols)
	var collationMismatch bool
	if currPKSame {
		for i, col := range targetPKCols {
			if targetPKCols[i] != truthPKCols[i] {
				currPKSame = false
				break
			}
			if _, ok := comparableColumns[targetPKCols[i]]; !ok {
				currPKSame = false
				break
			}
			compareCollation := true
			for i := 0; i < 2; i++ {
				if typ, ok := types.OidToType[columnMap[i][col].OID]; !ok || typ.Family() != types.StringFamily {
					compareCollation = false
				}
			}
			if compareCollation {
				if !comparableCollation(columnMap[0][col].Collation, columnMap[1][col].Collation) {
					collationMismatch = true
					res.MismatchingTableDefinitions = append(
						res.MismatchingTableDefinitions,
						inconsistency.MismatchingTableDefinition{
							DBTable: targetTbl,
							Info: fmt.Sprintf(
								"PRIMARY KEY has a string field %s has a different collation (%s=%s, %s=%s) preventing verification",
								col,
								conns[0].ID(),
								columnMap[0][col].Collation.String,
								conns[1].ID(),
								columnMap[1][col].Collation.String,
							),
						},
					)
				}
			}
		}
	}
	if !currPKSame && len(truthPKCols) > 0 {
		pkSame = false
		res.MismatchingTableDefinitions = append(
			res.MismatchingTableDefinitions,
			inconsistency.MismatchingTableDefinition{
				DBTable: targetTbl,
				Info:    "PRIMARY KEY does not match source of truth (columns and types must match)",
			},
		)
	}

	res.PrimaryKeyColumns = truthPKCols
	// Place PK columns first.
	for _, col := range truthPKCols {
		if _, ok := comparableColumns[col]; ok {
			res.Columns = append(res.Columns, col)
			for i := 0; i < 2; i++ {
				res.ColumnOIDs[i] = append(res.ColumnOIDs[i], columnMap[i][col].OID)
			}
			delete(comparableColumns, col)
		}
	}

	// Then every other column.
	for _, col := range truthCols {
		if _, ok := comparableColumns[col.Name]; ok {
			res.Columns = append(res.Columns, col.Name)
			for i := 0; i < 2; i++ {
				res.ColumnOIDs[i] = append(res.ColumnOIDs[i], columnMap[i][col.Name].OID)
			}
		}
	}
	res.RowVerifiable = pkSame && len(truthPKCols) > 0 && !collationMismatch
	return res, nil
}

// This logic isn't 100% there, but it's good enough.
func comparableCollation(a, b sql.NullString) bool {
	if a == b {
		return true
	}
	if a.Valid != b.Valid {
		return false
	}
	// unicode specifiers after `.` can be different.
	splitA := strings.Split(a.String, ".")
	splitB := strings.Split(b.String, ".")
	if splitA[0] == splitB[0] {
		return true
	}
	return isUnicodeCollation(a.String) && isUnicodeCollation(b.String)
}

func isUnicodeCollation(a string) bool {
	// mysql
	if strings.Contains(a, "utf8mb4_") {
		return true
	}
	split := strings.Split(a, ".")
	if len(split) >= 2 {
		return strings.Contains(split[1], "UTF-8") || strings.Contains(split[1], "utf8")
	}
	return false
}

func comparableType(a, b *pgtype.Type) bool {
	if a.Name == b.Name {
		return true
	}
	if enumCodec(a) && enumCodec(b) {
		return true
	}
	if arrayCodec(a) && arrayCodec(b) {
		if enumCodec(a.Codec.(*pgtype.ArrayCodec).ElementType) && enumCodec(b.Codec.(*pgtype.ArrayCodec).ElementType) {
			return true
		}
	}
	aTyp, ok := types.OidToType[oid.Oid(a.OID)]
	if !ok {
		return false
	}
	bTyp, ok := types.OidToType[oid.Oid(b.OID)]
	if !ok {
		return false
	}
	return allowableTypes(oid.Oid(a.OID), oid.Oid(b.OID)) || aTyp.Equivalent(bTyp)
}

func enumCodec(a *pgtype.Type) bool {
	_, ok := a.Codec.(*pgtype.EnumCodec)
	return ok
}

func arrayCodec(a *pgtype.Type) bool {
	_, ok := a.Codec.(*pgtype.ArrayCodec)
	return ok
}

func allowableTypes(a oid.Oid, b oid.Oid) bool {
	if a == oid.T_timestamp && b == oid.T_timestamptz {
		return true
	}
	if a == oid.T_timestamptz && b == oid.T_timestamp {
		return true
	}
	if a == oid.T_varchar && b == oid.T_uuid {
		return true
	}
	if a == oid.T_int2 && b == oid.T_bool {
		return true
	}

	return false
}
