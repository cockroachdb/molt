package utils

import (
	"fmt"
	"regexp"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
)

func SchemaTableString(schema, table tree.Name) string {
	return fmt.Sprintf("%s.%s", string(schema), string(table))
}

var FileConventionRegex = regexp.MustCompile(`part_[\d+]{8}(\.csv|\.tar\.gz)`)

func MatchesFileConvention(fileName string) bool {
	return FileConventionRegex.MatchString(fileName)
}
