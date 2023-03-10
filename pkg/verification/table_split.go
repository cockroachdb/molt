package verification

import (
	"context"
	"fmt"
	"go/constant"
	"math"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uint128"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/pkg/dbconn"
	"github.com/cockroachdb/molt/pkg/mysqlconv"
	"github.com/cockroachdb/molt/pkg/pgconv"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

func splitTable(
	ctx context.Context,
	truthConn dbconn.Conn,
	tbl verifyTableResult,
	reporter Reporter,
	numSplits int,
) ([]TableShard, error) {
	if numSplits < 1 {
		return nil, errors.AssertionFailedf("failed to split rows: %d\n", numSplits)
	}
	ret := make([]TableShard, 0, numSplits)

	// For now, be dumb and split only the first column.
	min, err := getTableExtremes(ctx, truthConn, tbl, true)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get minimum of %s.%s", tbl.Schema, tbl.Table)
	}
	max, err := getTableExtremes(ctx, truthConn, tbl, false)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get maximum of %s.%s", tbl.Schema, tbl.Table)
	}

	splittable := !(numSplits == 1 || len(min) == 0 || len(max) == 0 || len(min) != len(max))
	var nextMin tree.Datums
	if splittable {
	splitLoop:
		for splitNum := 1; splitNum <= numSplits; splitNum++ {
			var nextMax tree.Datums
			if splitNum < numSplits {
				// For now, split by only first column of PK.
				switch min[0].ResolvedType().Family() {
				case types.IntFamily:
					minVal := int64(*min[0].(*tree.DInt))
					maxVal := int64(*max[0].(*tree.DInt))
					valRange := maxVal - minVal
					if valRange <= 0 {
						splittable = false
						break splitLoop
					}
					splitVal := minVal + ((valRange / int64(numSplits)) * int64(splitNum))
					nextMax = append(nextMax, tree.NewDInt(tree.DInt(splitVal)))
				case types.FloatFamily:
					minVal := float64(*min[0].(*tree.DFloat))
					maxVal := float64(*max[0].(*tree.DFloat))
					valRange := maxVal - minVal
					if valRange <= 0 || math.IsNaN(valRange) || math.IsInf(valRange, 0) {
						splittable = false
						break splitLoop
					}
					splitVal := minVal + ((valRange / float64(numSplits)) * float64(splitNum))
					nextMax = append(nextMax, tree.NewDFloat(tree.DFloat(splitVal)))
				case types.UuidFamily:
					// Use the high ranges to divide.
					minVal := min[0].(*tree.DUuid).UUID.ToUint128().Hi
					maxVal := max[0].(*tree.DUuid).UUID.ToUint128().Hi
					valRange := maxVal - minVal
					if valRange <= 0 {
						splittable = false
						break splitLoop
					}
					splitVal := minVal + ((valRange / uint64(numSplits)) * uint64(splitNum))
					nextMax = append(nextMax, &tree.DUuid{UUID: uuid.FromUint128(uint128.Uint128{Hi: splitVal})})
				default:
					splittable = false
					break splitLoop
				}
			}
			ret = append(ret, TableShard{
				Table:                  tbl.Table,
				Schema:                 tbl.Schema,
				MatchingColumns:        tbl.MatchingColumns,
				MatchingColumnTypeOIDs: tbl.ColumnTypeOIDs,
				PrimaryKeyColumns:      tbl.PrimaryKeyColumns,
				StartPKVals:            nextMin,
				EndPKVals:              nextMax,
				ShardNum:               splitNum,
				TotalShards:            numSplits,
			})
			nextMin = nextMax
		}
	}
	if !splittable {
		ret = []TableShard{
			{
				Table:                  tbl.Table,
				Schema:                 tbl.Schema,
				MatchingColumns:        tbl.MatchingColumns,
				MatchingColumnTypeOIDs: tbl.ColumnTypeOIDs,
				PrimaryKeyColumns:      tbl.PrimaryKeyColumns,
				ShardNum:               1,
				TotalShards:            1,
			},
		}
		if numSplits != 1 {
			reporter.Report(StatusReport{
				Info: fmt.Sprintf(
					"unable to identify a split for primary key %s.%s, defaulting to a full scan",
					tbl.Schema,
					tbl.Table,
				),
			})
		}
	}
	return ret, nil
}

func getTableExtremes(
	ctx context.Context, truthConn dbconn.Conn, tbl verifyTableResult, isMin bool,
) (tree.Datums, error) {
	// Note here we use `.Query` instead of the `.QueryRow` counterpart.
	// This is because the API for `.Query` actually has other metadata from
	// the row that isn't found on `.QueryRow`.
	switch truthConn := truthConn.(type) {
	case *dbconn.PGConn:
		f := tree.NewFmtCtx(tree.FmtParsableNumerics)
		s := buildSelectForSplitPG(tbl, isMin)
		f.FormatNode(s)
		q := f.CloseAndGetString()
		rows, err := truthConn.Query(ctx, q)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting minimum value for %s.%s", tbl.Schema, tbl.Table)
		}
		defer rows.Close()
		if rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				return nil, err
			}
			rowVals, err := pgconv.ConvertRowValues(truthConn.TypeMap(), vals, tbl.ColumnTypeOIDs[0][:len(tbl.PrimaryKeyColumns)])
			if err != nil {
				return nil, err
			}
			return rowVals, nil
		}
		return nil, rows.Err()
	case *dbconn.MySQLConn:
		var sb strings.Builder
		if err := buildSelectForSplitMySQL(tbl, isMin).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
			return nil, errors.Wrap(err, "error generating MySQL statement")
		}
		q := sb.String()
		rows, err := truthConn.QueryContext(ctx, q)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting minimum value for %s.%s", tbl.Schema, tbl.Table)
		}
		defer rows.Close()
		if rows.Next() {
			return mysqlconv.ScanRowDynamicTypes(rows, truthConn.TypeMap(), tbl.ColumnTypeOIDs[0][:len(tbl.PrimaryKeyColumns)])
		}
		return nil, rows.Err()

	}
	return nil, errors.AssertionFailedf("unknown type for extremes: %T", truthConn)
}

func buildSelectForSplitPG(tbl verifyTableResult, isMin bool) *tree.Select {
	tn := tree.MakeTableNameFromPrefix(
		tree.ObjectNamePrefix{SchemaName: tbl.Schema, ExplicitSchema: true},
		tbl.Table,
	)
	selectClause := &tree.SelectClause{
		From: tree.From{
			Tables: tree.TableExprs{&tn},
		},
	}
	for _, col := range tbl.PrimaryKeyColumns {
		selectClause.Exprs = append(
			selectClause.Exprs,
			tree.SelectExpr{
				Expr: tree.NewUnresolvedName(string(col)),
			},
		)
	}
	baseSelectExpr := &tree.Select{
		Select: selectClause,
		Limit:  &tree.Limit{Count: tree.NewNumVal(constant.MakeUint64(uint64(1)), "", false)},
	}
	for _, pkCol := range tbl.PrimaryKeyColumns {
		orderClause := &tree.Order{Expr: tree.NewUnresolvedName(string(pkCol))}
		if !isMin {
			orderClause.Direction = tree.Descending
		}
		baseSelectExpr.OrderBy = append(
			baseSelectExpr.OrderBy,
			orderClause,
		)
	}
	return baseSelectExpr
}

func buildSelectForSplitMySQL(table verifyTableResult, isMin bool) *ast.SelectStmt {
	fields := &ast.FieldList{
		Fields: make([]*ast.SelectField, len(table.PrimaryKeyColumns)),
	}
	orderBy := &ast.OrderByClause{
		Items: make([]*ast.ByItem, len(table.PrimaryKeyColumns)),
	}
	for i, col := range table.PrimaryKeyColumns {
		fields.Fields[i] = &ast.SelectField{
			Expr: mysqlconv.MySQLASTColumnField(col),
		}
		orderBy.Items[i] = &ast.ByItem{
			Expr: mysqlconv.MySQLASTColumnField(col),
		}
		if !isMin {
			orderBy.Items[i].Desc = true
		}
	}
	return &ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{
			SQLCache: true,
		},
		From: &ast.TableRefsClause{
			TableRefs: &ast.Join{
				Left: &ast.TableSource{
					Source: &ast.TableName{Name: model.NewCIStr(string(table.Table))},
				},
			},
		},
		Fields:  fields,
		Kind:    ast.SelectStmtKindSelect,
		Limit:   &ast.Limit{Count: ast.NewValueExpr(1, "", "")},
		OrderBy: orderBy,
	}
}
