package cmdutil

import (
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/spf13/cobra"
)

var tableFilter = dbverify.DefaultFilterConfig()

func RegisterNameFilterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&tableFilter.TableFilter,
		"table-filter",
		tableFilter.TableFilter,
		"POSIX regexp filter for tables to action on.",
	)
	cmd.PersistentFlags().StringVar(
		&tableFilter.SchemaFilter,
		"schema-filter",
		tableFilter.SchemaFilter,
		"POSIX regexp filter for schemas to action on.",
	)
}

func TableFilter() dbverify.FilterConfig {
	return tableFilter
}
