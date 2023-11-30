package inconsistency

import (
	"context"
	"fmt"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree/treecmp"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/rs/zerolog"
)

type Reporter interface {
	Report(obj ReportableObject)
	Close()
}

type CombinedReporter struct {
	Reporters []Reporter
}

func (c CombinedReporter) Report(obj ReportableObject) {
	for _, r := range c.Reporters {
		r.Report(obj)
	}
}

func (c CombinedReporter) Close() {
	for _, r := range c.Reporters {
		r.Close()
	}
}

type StatusReport struct {
	Info string
}

// LogReporter reports to `zerolog`.
type LogReporter struct {
	zerolog.Logger
}

func (l LogReporter) Report(obj ReportableObject) {
	switch obj := obj.(type) {
	case MissingTable:
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Msgf("missing table detected")
	case ExtraneousTable:
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Msgf("extraneous table detected")
	case MismatchingTableDefinition:
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Str("mismatch_info", obj.Info).
			Msgf("mismatching table definition")
	case StatusReport:
		l.Info().Msg(obj.Info)
	case MismatchingRow:
		sourceValues := zerolog.Dict()
		targetVals := zerolog.Dict()
		for i, col := range obj.MismatchingColumns {
			targetVals = targetVals.Str(string(col), reportableVal(obj.TruthVals[i]))
			sourceValues = sourceValues.Str(string(col), reportableVal(obj.TargetVals[i]))
		}
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Dict("source_values", targetVals).
			Dict("target_values", sourceValues).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("mismatching row value")
	case MissingRow:
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("missing row")
	case ExtraneousRow:
		l.Warn().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("extraneous row")
	default:
		l.Error().
			Str("type", fmt.Sprintf("%T", obj)).
			Msgf("unknown object type")
	}
}

func reportableVal(d tree.Datum) string {
	f := tree.NewFmtCtx(tree.FmtExport | tree.FmtParsableNumerics)
	f.FormatNode(d)
	return f.CloseAndGetString()
}

func zipPrimaryKeysForReporting(columnVals tree.Datums) []string {
	ret := make([]string, len(columnVals))
	for i := range columnVals {
		ret[i] = reportableVal(columnVals[i])
	}
	return ret
}

func (l LogReporter) Close() {
}

type FixReporter struct {
	Conn   dbconn.Conn
	Logger zerolog.Logger
}

func (l FixReporter) Report(obj ReportableObject) {
	switch obj := obj.(type) {
	case MismatchingRow:
		l.Logger.Info().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("fixing mismatching row")
		switch conn := l.Conn.(type) {
		case *dbconn.PGConn:
			updateClause := &tree.Update{
				Table:     tree.NewUnqualifiedTableName(obj.Table),
				Where:     buildWhereClause(obj.PrimaryKeyColumns, obj.PrimaryKeyValues),
				Returning: &tree.NoReturningClause{},
				Exprs:     make(tree.UpdateExprs, len(obj.MismatchingColumns)),
			}
			for i := range obj.MismatchingColumns {
				updateClause.Exprs[i] = &tree.UpdateExpr{
					Names: []tree.Name{obj.MismatchingColumns[i]},
					Expr:  obj.TruthVals[i],
				}
			}
			fmtCtx := tree.NewFmtCtx(tree.FmtSimple)
			fmtCtx.FormatNode(updateClause)
			_, err := conn.Exec(context.Background(), fmtCtx.CloseAndGetString())
			if err != nil {
				panic(err)
			}
		}
	case MissingRow:
		l.Logger.Info().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("adding missing row")

		switch conn := l.Conn.(type) {
		case *dbconn.PGConn:
			valuesClause := tree.ValuesClause{
				Rows: []tree.Exprs{
					make([]tree.Expr, len(obj.Columns)),
				},
			}
			insertClause := &tree.Insert{
				Table:     tree.NewUnqualifiedTableName(obj.Table),
				Returning: &tree.NoReturningClause{},
				Columns:   make([]tree.Name, len(obj.Columns)),
				Rows:      &tree.Select{Select: &valuesClause},
			}
			valuesClause.Rows[0] = make([]tree.Expr, len(obj.Columns))
			for i := range obj.Columns {
				insertClause.Columns[i] = obj.Columns[i]
				valuesClause.Rows[0][i] = obj.Values[i]
			}
			fmtCtx := tree.NewFmtCtx(tree.FmtSimple)
			fmtCtx.FormatNode(insertClause)
			_, err := conn.Exec(context.Background(), fmtCtx.CloseAndGetString())
			if err != nil {
				panic(err)
			}
		}
	case ExtraneousRow:
		l.Logger.Info().
			Str("table_schema", string(obj.Schema)).
			Str("table_name", string(obj.Table)).
			Strs("primary_key", zipPrimaryKeysForReporting(obj.PrimaryKeyValues)).
			Msgf("deleting extraneous row")
		switch conn := l.Conn.(type) {
		case *dbconn.PGConn:
			deleteClause := &tree.Delete{
				Table:     tree.NewUnqualifiedTableName(obj.Table),
				Where:     buildWhereClause(obj.PrimaryKeyColumns, obj.PrimaryKeyValues),
				Returning: &tree.NoReturningClause{},
			}
			fmtCtx := tree.NewFmtCtx(tree.FmtSimple)
			fmtCtx.FormatNode(deleteClause)
			_, err := conn.Exec(context.Background(), fmtCtx.CloseAndGetString())
			if err != nil {
				panic(err)
			}
		}
	}
}

func buildWhereClause(cols []tree.Name, values []tree.Datum) *tree.Where {
	whereClause := &tree.Where{
		Type: tree.AstWhere,
	}
	for i := range values {
		op := &tree.ComparisonExpr{
			Operator: treecmp.MakeComparisonOperator(treecmp.EQ),
			Left:     tree.NewUnresolvedName(string(cols[i])),
			Right:    values[i],
		}
		if i == 0 {
			whereClause.Expr = op
			continue
		}
		whereClause.Expr = &tree.AndExpr{
			Left:  whereClause.Expr,
			Right: op,
		}
	}
	return whereClause
}

func (l FixReporter) Close() {
}
