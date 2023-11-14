package cmdutil

import (
	"context"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/spf13/cobra"
)

var DBConnConfig = dbconn.Config{}

func RegisterDBConnFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&DBConnConfig.Source,
		"source",
		"",
		"Connection string of the source database.",
	)
	cmd.PersistentFlags().StringVar(
		&DBConnConfig.Target,
		"target",
		"",
		"Connection string of the target database.",
	)

	for _, required := range []string{"source", "target"} {
		if err := cmd.MarkPersistentFlagRequired(required); err != nil {
			panic(err)
		}
	}
}

func LoadDBConns(ctx context.Context) (dbconn.OrderedConns, error) {
	source, err := dbconn.Connect(ctx, "source", DBConnConfig.Source)
	if err != nil {
		return dbconn.OrderedConns{}, err
	}
	target, err := dbconn.Connect(ctx, "target", DBConnConfig.Target)
	if err != nil {
		return dbconn.OrderedConns{}, err
	}
	return dbconn.OrderedConns{source, target}, nil
}
