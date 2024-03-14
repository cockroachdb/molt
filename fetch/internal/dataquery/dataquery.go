package dataquery

import (
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/rowiterator"
	"github.com/cockroachdb/molt/utils"
)

func NewPGCopyTo(
	table dbtable.VerifiedTable, startPKVals []tree.Datum, endPKVals []tree.Datum,
) string {
	selectStmt := rowiterator.NewPGBaseSelectClause(rowiterator.Table{
		Name:              table.Name,
		ColumnNames:       table.Columns,
		PrimaryKeyColumns: table.PrimaryKeyColumns,
	})
	rowiterator.AppendSelectWithPKRanges(selectStmt, table.Columns, startPKVals, endPKVals)
	copyFrom := &tree.CopyTo{
		Statement: selectStmt,
		Options: tree.CopyOptions{
			CopyFormat: tree.CopyFormatCSV,
			HasFormat:  true,
		},
	}
	f := tree.NewFmtCtx(tree.FmtParsableNumerics)
	f.FormatNode(copyFrom)
	return f.CloseAndGetString()
}

func ImportInto(table dbtable.VerifiedTable, locs []string, opts tree.KVOptions) (string, string) {
	importInto := &tree.Import{
		Into:       true,
		Table:      table.NewTableName(),
		FileFormat: "CSV",
		IntoCols:   table.Columns,
	}

	// If we default set the Options parameter when there are no KVOptions,
	// an incorrect 'WITH' gets appended to the statement.
	if len(opts) > 0 {
		importInto.Options = opts
	}

	for _, loc := range locs {
		importInto.Files = append(
			importInto.Files,
			tree.NewStrVal(loc),
		)
	}
	f := tree.NewFmtCtx(tree.FmtParsableNumerics)
	// Skipping the error check since all the URL's created are
	// safe formatted by us internally so there should be no parse
	// errors. If it fails, redacted will just be "".

	// TODO: Don't redact if using local store.
	// Right now it will just always return err but we
	// skip the error anyways
	redacted, _ := redactImportQuery(importInto, locs)
	f.FormatNode(importInto)
	return f.CloseAndGetString(), redacted
}

func CopyFrom(table dbtable.VerifiedTable, skipHeader bool) string {
	copyFrom := &tree.CopyFrom{
		Table:   table.MakeTableName(),
		Columns: table.Columns,
		Stdin:   true,
		Options: tree.CopyOptions{
			CopyFormat: tree.CopyFormatCSV,
			HasFormat:  true,
			Header:     skipHeader,
			HasHeader:  skipHeader,
		},
	}
	f := tree.NewFmtCtx(tree.FmtParsableNumerics)
	f.FormatNode(copyFrom)
	// Temporary hack for v22.2- compat. Remove when we use 23.1 in CI.
	return strings.ReplaceAll(f.CloseAndGetString(), "STDIN WITH (FORMAT CSV)", "STDIN CSV")
}

func redactImportQuery(orig *tree.Import, files []string) (string, error) {
	stmt := *orig
	stmt.Files = nil
	for _, file := range files {
		clean, err := utils.SanitizeExternalStorageURI(file, nil /* extraParams */)
		if err != nil {
			return "", err
		}
		stmt.Files = append(stmt.Files, tree.NewDString(clean))
	}
	// Don't log the IntoCols in case of sensitive column names.
	stmt.IntoCols = nil
	return tree.AsString(&stmt), nil
}
