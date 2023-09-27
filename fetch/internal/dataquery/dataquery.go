package dataquery

import (
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/rowiterator"
)

func NewPGCopyTo(table dbtable.VerifiedTable) string {
	copyFrom := &tree.CopyTo{
		Statement: rowiterator.NewPGBaseSelectClause(rowiterator.Table{
			Name:              table.Name,
			ColumnNames:       table.Columns,
			PrimaryKeyColumns: table.PrimaryKeyColumns,
		}),
		Options: tree.CopyOptions{
			CopyFormat: tree.CopyFormatCSV,
			HasFormat:  true,
		},
	}
	f := tree.NewFmtCtx(tree.FmtParsableNumerics)
	f.FormatNode(copyFrom)
	return f.CloseAndGetString()
}

func ImportInto(table dbtable.VerifiedTable, locs []string, opts tree.KVOptions) string {
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
	f.FormatNode(importInto)
	return f.CloseAndGetString()
}

func CopyFrom(table dbtable.VerifiedTable) string {
	copyFrom := &tree.CopyFrom{
		Table:   table.MakeTableName(),
		Columns: table.Columns,
		Stdin:   true,
		Options: tree.CopyOptions{
			CopyFormat: tree.CopyFormatCSV,
			HasFormat:  true,
		},
	}
	f := tree.NewFmtCtx(tree.FmtParsableNumerics)
	f.FormatNode(copyFrom)
	// Temporary hack for v22.2- compat. Remove when we use 23.1 in CI.
	return strings.ReplaceAll(f.CloseAndGetString(), "STDIN WITH (FORMAT CSV)", "STDIN CSV")
}
