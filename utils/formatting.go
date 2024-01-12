package utils

import (
	"fmt"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
)

func SchemaTableString(schema, table tree.Name) string {
	return fmt.Sprintf("%s.%s", string(schema), string(table))
}
