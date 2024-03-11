package verify

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
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/mysqlconv"
	"github.com/cockroachdb/molt/pgconv"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/cockroachdb/molt/verify/tableverify"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

func ShardTable(
	ctx context.Context,
	truthConn dbconn.Conn,
	tbl tableverify.Result,
	reporter inconsistency.Reporter,
	numSplits int,
) ([]rowverify.TableShard, error) {
	if numSplits < 1 {
		return nil, errors.AssertionFailedf("failed to split rows: %d", numSplits)
	}
	if numSplits > 1 {
		ret := make([]rowverify.TableShard, 0, numSplits)
		// For now, be dumb and split only the first column.
		minE, err := getTableExtremes(ctx, truthConn, tbl, true)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get minimum of %s.%s", tbl.Schema, tbl.Table)
		}
		maxE, err := getTableExtremes(ctx, truthConn, tbl, false)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get maximum of %s.%s", tbl.Schema, tbl.Table)
		}
		var nextMin tree.Datums
		if !(len(minE) == 0 || len(maxE) == 0 || len(minE) != len(maxE)) {
			splittable := true
		splitLoop:
			for splitNum := 1; splitNum <= numSplits; splitNum++ {
				var nextMax tree.Datums
				if splitNum < numSplits {
					// For now, split by only first column of PK.
					switch minE[0].ResolvedType().Family() {
					case types.IntFamily:
						minVal := int64(*minE[0].(*tree.DInt))
						maxVal := int64(*maxE[0].(*tree.DInt))
						valRange := maxVal - minVal
						if valRange <= 0 {
							splittable = false
							break splitLoop
						}
						splitVal := minVal + (max((valRange/int64(numSplits)), 1) * int64(splitNum))
						nextMax = append(nextMax, tree.NewDInt(tree.DInt(splitVal)))
					case types.FloatFamily:
						minVal := float64(*minE[0].(*tree.DFloat))
						maxVal := float64(*maxE[0].(*tree.DFloat))
						valRange := maxVal - minVal
						if valRange <= 0 || math.IsNaN(valRange) || math.IsInf(valRange, 0) {
							splittable = false
							break splitLoop
						}
						splitVal := minVal + (max((valRange/float64(numSplits)), 1) * float64(splitNum))
						nextMax = append(nextMax, tree.NewDFloat(tree.DFloat(splitVal)))
					case types.UuidFamily:
						// Use the high ranges to divide.
						minVal := minE[0].(*tree.DUuid).UUID.ToUint128().Hi
						maxVal := maxE[0].(*tree.DUuid).UUID.ToUint128().Hi
						valRange := maxVal - minVal
						if valRange <= 0 {
							splittable = false
							break splitLoop
						}
						splitVal := minVal + (max((valRange/uint64(numSplits)), 1) * uint64(splitNum))
						nextMax = append(nextMax, &tree.DUuid{UUID: uuid.FromUint128(uint128.Uint128{Hi: splitVal})})
					default:
						splittable = false
						break splitLoop
					}
				}
				ret = append(ret, rowverify.TableShard{
					VerifiedTable: tbl.VerifiedTable,
					StartPKVals:   nextMin,
					EndPKVals:     nextMax,
					ShardNum:      splitNum,
					TotalShards:   numSplits,
				})
				nextMin = nextMax
			}
			if splittable {
				return ret, nil
			}
		}
	}
	ret := []rowverify.TableShard{
		{
			VerifiedTable: tbl.VerifiedTable,
			ShardNum:      1,
			TotalShards:   1,
		},
	}
	if numSplits != 1 {
		if reporter != nil {
			reporter.Report(inconsistency.StatusReport{
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
	ctx context.Context, truthConn dbconn.Conn, tbl tableverify.Result, isMin bool,
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
			rowVals, err := pgconv.ConvertRowValues(truthConn.TypeMap(), vals, tbl.ColumnOIDs[0][:len(tbl.PrimaryKeyColumns)])
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
			return mysqlconv.ScanRowDynamicTypes(rows, truthConn.TypeMap(), tbl.ColumnOIDs[0][:len(tbl.PrimaryKeyColumns)])
		}
		return nil, rows.Err()

	}
	return nil, errors.AssertionFailedf("unknown type for extremes: %T", truthConn)
}

func buildSelectForSplitPG(tbl tableverify.Result, isMin bool) *tree.Select {
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

func buildSelectForSplitMySQL(table tableverify.Result, isMin bool) *ast.SelectStmt {
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
