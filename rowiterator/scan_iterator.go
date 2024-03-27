package rowiterator

import (
	"context"
	"fmt"
	"go/constant"
	"strings"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree/treecmp"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/mysqlconv"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/opcode"
	"golang.org/x/time/rate"
)

type scanIterator struct {
	conn         dbconn.Conn
	table        ScanTable
	rowBatchSize int

	waitCh        chan scanIteratorResult
	cache         []tree.Datums
	pkCursor      tree.Datums
	currCacheSize int
	err           error
	scanQuery     scanQuery
	rateLimiter   *rate.Limiter
}

type scanIteratorResult struct {
	r   []tree.Datums
	err error
}

// NewScanIterator returns a row iterator which scans the given table.
func NewScanIterator(
	ctx context.Context,
	conn dbconn.Conn,
	table ScanTable,
	rowBatchSize int,
	rateLimiter *rate.Limiter,
) (Iterator, error) {
	// Initialize the type map on the connection.
	for _, typOID := range table.ColumnOIDs {
		if _, err := dbconn.GetDataType(ctx, conn, typOID); err != nil {
			return nil, errors.Wrapf(err, "Error initializing type oid %d", typOID)
		}
	}
	it := &scanIterator{
		conn:          conn,
		table:         table,
		rowBatchSize:  rowBatchSize,
		currCacheSize: rowBatchSize,
		waitCh:        make(chan scanIteratorResult, 1),
		rateLimiter:   rateLimiter,
	}
	switch conn := conn.(type) {
	case *dbconn.PGConn:
		it.scanQuery = newPGScanQuery(table, rowBatchSize)
	case *dbconn.MySQLConn:
		it.scanQuery = newMySQLScanQuery(table, rowBatchSize)
	case *dbconn.OracleConn:
		it.scanQuery = newOracleScanQuery(table, rowBatchSize)
	default:
		return nil, errors.Newf("unsupported conn type %T", conn)
	}
	it.nextPage(ctx)
	return it, nil
}

func (it *scanIterator) Conn() dbconn.Conn {
	return it.conn
}

func (it *scanIterator) HasNext(ctx context.Context) bool {
	for {
		if it.err != nil {
			return false
		}

		if len(it.cache) > 0 {
			return true
		}

		// If the last cache size was less than the row size, we're done
		// reading all the results.
		if it.currCacheSize < it.rowBatchSize {
			return false
		}

		// Wait for more results.
		res := <-it.waitCh
		if res.err != nil {
			it.err = errors.Wrap(res.err, "error getting result")
			return false
		}
		it.cache = res.r
		it.currCacheSize = len(it.cache)

		// Queue the next page immediately.
		if it.currCacheSize == it.rowBatchSize {
			it.nextPage(ctx)
		}
	}
}

// nextPage fetches rows asynchronously.
func (it *scanIterator) nextPage(ctx context.Context) {
	go func() {
		datums, err := func() ([]tree.Datums, error) {
			var currRows rows
			q, args, err := it.scanQuery.generate(it.pkCursor)
			if err != nil {
				return nil, err
			}
			if it.rateLimiter != nil {
				if err := it.rateLimiter.Wait(ctx); err != nil {
					return nil, err
				}
			}
			switch conn := it.conn.(type) {
			case *dbconn.PGConn:
				newRows, err := conn.Query(ctx, q, args...)
				if err != nil {
					return nil, errors.Wrapf(err, "error getting rows for table %s.%s from %s", it.table.Schema, it.table.Table.Name, it.conn.ID)
				}
				currRows = &pgRows{
					Rows:    newRows,
					typMap:  it.conn.TypeMap(),
					typOIDs: it.table.ColumnOIDs,
				}
			case *dbconn.MySQLConn:
				newRows, err := conn.QueryContext(ctx, q, args...)
				if err != nil {
					return nil, errors.Wrapf(err, "error getting rows for table %s in %s", it.table.Table.Name, it.conn.ID())
				}
				currRows = &mysqlRows{
					Rows:    newRows,
					typMap:  it.conn.TypeMap(),
					typOIDs: it.table.ColumnOIDs,
				}
			case *dbconn.OracleConn:
				newRows, err := conn.QueryContext(ctx, q, args...)
				if err != nil {
					return nil, errors.Wrapf(err, "error getting rows for table %s in %s", it.table.Table.Name, it.conn.ID())
				}
				currRows = &oracleRows{
					Rows:    newRows,
					typMap:  it.conn.TypeMap(),
					typOIDs: it.table.ColumnOIDs,
				}
			default:
				return nil, errors.AssertionFailedf("unhandled conn type: %T", conn)
			}
			defer func() { currRows.Close() }()

			datums := make([]tree.Datums, 0, it.rowBatchSize)
			for currRows.Next() {
				d, err := currRows.Datums()
				if err != nil {
					return nil, errors.Wrapf(err, "error getting datums")
				}
				it.pkCursor = d[:len(it.table.PrimaryKeyColumns)]
				datums = append(datums, d)
			}
			return datums, currRows.Err()
		}()
		it.waitCh <- scanIteratorResult{r: datums, err: err}
	}()
}

func (it *scanIterator) Peek(ctx context.Context) tree.Datums {
	if it.HasNext(ctx) {
		return it.cache[0]
	}
	return nil
}

func (it *scanIterator) Next(ctx context.Context) tree.Datums {
	if it.HasNext(ctx) {
		ret := it.cache[0]
		it.cache = it.cache[1:]
		return ret
	}
	return nil
}

func (it *scanIterator) Error() error {
	return it.err
}

type scanQuery struct {
	base  any
	table ScanTable
}

func newPGScanQuery(table ScanTable, rowBatchSize int) scanQuery {
	baseSelectExpr := NewPGBaseSelectClause(table.Table)
	baseSelectExpr.Limit = &tree.Limit{Count: tree.NewNumVal(constant.MakeUint64(uint64(rowBatchSize)), "", false)}
	if table.AOST != nil {
		var err error
		baseSelectExpr.Select.(*tree.SelectClause).From.AsOf.Expr, err = tree.MakeDTimestamp(*table.AOST, time.Microsecond)
		if err != nil {
			panic(err)
		}
	}
	return scanQuery{
		base:  baseSelectExpr,
		table: table,
	}
}

func NewPGBaseSelectClause(table Table) *tree.Select {
	tn := tree.MakeTableNameFromPrefix(
		tree.ObjectNamePrefix{SchemaName: table.Schema, ExplicitSchema: true},
		table.Table,
	)
	selectClause := &tree.SelectClause{
		From: tree.From{
			Tables: tree.TableExprs{&tn},
		},
	}
	for _, col := range table.ColumnNames {
		selectClause.Exprs = append(
			selectClause.Exprs,
			tree.SelectExpr{
				Expr: tree.NewUnresolvedName(string(col)),
			},
		)
	}
	baseSelectExpr := &tree.Select{
		Select: selectClause,
	}
	for _, pkCol := range table.PrimaryKeyColumns {
		baseSelectExpr.OrderBy = append(
			baseSelectExpr.OrderBy,
			&tree.Order{Expr: tree.NewUnresolvedName(string(pkCol))},
		)
	}
	return baseSelectExpr
}

type oracleStatement struct {
	rowBatchSize int
}

func newOracleScanQuery(table ScanTable, rowBatchSize int) scanQuery {
	if len(table.StartPKVals) > 0 || len(table.EndPKVals) > 0 {
		panic("sharding on oracle not yet supported")
	}
	return scanQuery{
		table: table,
		base: &oracleStatement{
			rowBatchSize: rowBatchSize,
		},
	}
}

func newMySQLScanQuery(table ScanTable, rowBatchSize int) scanQuery {
	stmt := newMySQLBaseSelectClause(table.Table)
	stmt.Limit = &ast.Limit{Count: ast.NewValueExpr(rowBatchSize, "", "")}
	return scanQuery{
		base:  stmt,
		table: table,
	}
}

func newMySQLBaseSelectClause(table Table) *ast.SelectStmt {
	fields := &ast.FieldList{
		Fields: make([]*ast.SelectField, len(table.ColumnNames)),
	}
	for i, col := range table.ColumnNames {
		fields.Fields[i] = &ast.SelectField{
			Expr: mysqlconv.MySQLASTColumnField(col),
		}
	}
	orderBy := &ast.OrderByClause{
		Items: make([]*ast.ByItem, len(table.PrimaryKeyColumns)),
	}
	for i, pkCol := range table.PrimaryKeyColumns {
		orderBy.Items[i] = &ast.ByItem{
			Expr: mysqlconv.MySQLASTColumnField(pkCol),
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
		OrderBy: orderBy,
	}
}

func AppendSelectWithPKRanges(
	stmt *tree.Select, columnNames []tree.Name, startPKVals []tree.Datum, endPKVals []tree.Datum,
) {
	andClause := &tree.AndExpr{
		Left:  tree.DBoolTrue,
		Right: tree.DBoolTrue,
	}

	if len(startPKVals) > 0 {
		andClause.Left = makePGCompareExpr(
			treecmp.MakeComparisonOperator(treecmp.GE),
			columnNames,
			startPKVals,
		)
	}

	if len(endPKVals) > 0 {
		andClause.Right = makePGCompareExpr(
			treecmp.MakeComparisonOperator(treecmp.LT),
			columnNames,
			endPKVals,
		)
	}

	stmt.Select.(*tree.SelectClause).Where = &tree.Where{
		Type: tree.AstWhere,
		Expr: andClause,
	}
}

func (sq *scanQuery) generate(pkCursor tree.Datums) (string, []any, error) {
	switch stmt := sq.base.(type) {
	case *tree.Select:
		andClause := &tree.AndExpr{
			Left:  tree.DBoolTrue,
			Right: tree.DBoolTrue,
		}
		// Use the cursor if available, otherwise not.
		if len(pkCursor) > 0 {
			andClause.Left = makePGCompareExpr(
				treecmp.MakeComparisonOperator(treecmp.GT),
				sq.table.ColumnNames,
				pkCursor,
			)
		} else if len(sq.table.StartPKVals) > 0 {
			andClause.Left = makePGCompareExpr(
				treecmp.MakeComparisonOperator(treecmp.GE),
				sq.table.ColumnNames,
				sq.table.StartPKVals,
			)
		}
		if len(sq.table.EndPKVals) > 0 {
			andClause.Right = makePGCompareExpr(
				treecmp.MakeComparisonOperator(treecmp.LT),
				sq.table.ColumnNames,
				sq.table.EndPKVals,
			)
		}
		stmt.Select.(*tree.SelectClause).Where = &tree.Where{
			Type: tree.AstWhere,
			Expr: andClause,
		}
		f := tree.NewFmtCtx(tree.FmtParsableNumerics)
		f.FormatNode(stmt)
		return f.CloseAndGetString(), nil, nil
	case *oracleStatement:
		// TODO: escaping names is not supported.
		sb := strings.Builder{}
		sb.WriteString("SELECT ")
		for i := range sq.table.ColumnNames {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(string(sq.table.ColumnNames[i]))
		}
		sb.WriteString(" FROM ")
		sb.WriteString(string(sq.table.Name.Table))

		if len(pkCursor) > 0 {
			sb.WriteString(" WHERE ")
			// TODO: support multi-pk keys
			if len(sq.table.PrimaryKeyColumns) > 1 {
				panic("unsupported")
			}
			f := tree.NewFmtCtx(tree.FmtBareStrings)
			f.FormatNode(pkCursor[0])
			// TODO: support converting data types back into string representations in oracle.
			sb.WriteString(fmt.Sprintf("%s > '%s'", sq.table.PrimaryKeyColumns[0], f.CloseAndGetString()))
		}

		sb.WriteString(" ORDER BY ")
		for i := range sq.table.PrimaryKeyColumns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(string(sq.table.PrimaryKeyColumns[i]))
		}
		sb.WriteString(fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", stmt.rowBatchSize))
		return sb.String(), nil, nil
	case *ast.SelectStmt:
		andClause := &ast.BinaryOperationExpr{
			Op: opcode.LogicAnd,
			L:  ast.NewValueExpr(1, "", ""),
			R:  ast.NewValueExpr(1, "", ""),
		}
		// Use the cursor if available, otherwise not.
		if len(pkCursor) > 0 {
			andClause.L = makeMySQLCompareExpr(
				opcode.GT,
				sq.table.ColumnNames,
				pkCursor,
			)
		} else if len(sq.table.StartPKVals) > 0 {
			andClause.L = makeMySQLCompareExpr(
				opcode.GE,
				sq.table.ColumnNames,
				sq.table.StartPKVals,
			)
		}
		if len(sq.table.EndPKVals) > 0 {
			andClause.R = makeMySQLCompareExpr(
				opcode.LT,
				sq.table.ColumnNames,
				sq.table.EndPKVals,
			)
		}
		stmt.Where = andClause
		var sb strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
			return "", nil, errors.Wrap(err, "error generating MySQL statement")
		}
		return sb.String(), nil, nil
	}
	return "", nil, errors.AssertionFailedf("unknown scan query type: %T", sq.base)
}

func makeMySQLCompareExpr(
	op opcode.Op, cols []tree.Name, vals tree.Datums,
) *ast.BinaryOperationExpr {
	cmpExpr := &ast.BinaryOperationExpr{
		Op: op,
	}
	colNames := make([]ast.ExprNode, len(vals))
	colVals := make([]ast.ExprNode, len(vals))

	if len(vals) > 1 {
		for i := range vals {
			colNames[i] = mysqlconv.MySQLASTColumnField(cols[i])
			colVals[i] = datumToMySQLValue(vals[i])
		}
		cmpExpr.L = &ast.RowExpr{Values: colNames}
		cmpExpr.R = &ast.RowExpr{Values: colVals}
	} else {
		cmpExpr.L = mysqlconv.MySQLASTColumnField(cols[0])
		f := tree.NewFmtCtx(tree.FmtParsableNumerics | tree.FmtBareStrings)
		f.FormatNode(vals[0])
		cmpExpr.R = ast.NewValueExpr(f.CloseAndGetString(), "", "")
	}
	return cmpExpr
}

func datumToMySQLValue(val tree.Datum) ast.ValueExpr {
	f := tree.NewFmtCtx(tree.FmtParsableNumerics | tree.FmtBareStrings)
	f.FormatNode(val)
	// NOTE: this may not correct for all types.
	// We shouldn't cast everything to string at the very least.
	return ast.NewValueExpr(f.CloseAndGetString(), "", "")
}

func makePGCompareExpr(
	op treecmp.ComparisonOperator, cols []tree.Name, vals tree.Datums,
) *tree.ComparisonExpr {
	cmpExpr := &tree.ComparisonExpr{
		Operator: op,
	}
	if len(vals) > 1 {
		colNames := &tree.Tuple{}
		colVals := &tree.Tuple{}
		for i := range vals {
			colNames.Exprs = append(colNames.Exprs, tree.NewUnresolvedName(string(cols[i])))
			colVals.Exprs = append(colVals.Exprs, vals[i])
		}
		cmpExpr.Left = colNames
		cmpExpr.Right = colVals
	} else {
		cmpExpr.Left = tree.NewUnresolvedName(string(cols[0]))
		cmpExpr.Right = vals[0]
	}
	return cmpExpr
}
