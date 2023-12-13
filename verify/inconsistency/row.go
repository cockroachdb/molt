package inconsistency

import (
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/molt/dbtable"
)

type ReportableObject interface{}

type MissingRow struct {
	dbtable.Name

	PrimaryKeyColumns []tree.Name
	PrimaryKeyValues  tree.Datums

	Columns []tree.Name
	Values  tree.Datums
}

type ExtraneousRow struct {
	dbtable.Name

	PrimaryKeyColumns []tree.Name
	PrimaryKeyValues  tree.Datums
}

type MismatchingRow struct {
	dbtable.Name

	PrimaryKeyColumns []tree.Name
	PrimaryKeyValues  tree.Datums

	MismatchingColumns []tree.Name
	TruthVals          tree.Datums
	TargetVals         tree.Datums
}

type MismatchingColumn struct {
	dbtable.Name

	PrimaryKeyColumns []tree.Name
	PrimaryKeyValues  tree.Datums

	MismatchingColumns []tree.Name
	TruthVals          tree.Datums
	TargetVals         tree.Datums
	Info               []string
}
