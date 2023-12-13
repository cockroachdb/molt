package cmdutil

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
)

func CheckFlagDependency(cmd *cobra.Command, dependencyFlag string, dependentFlags []string) error {
	// Merge persistent flags.
	cmd.Flags().AddFlagSet(cmd.PersistentFlags())
	dependency := cmd.Flags().Lookup(dependencyFlag)
	if dependency == nil {
		return errors.Newf(`Flag "%s" not found`, dependencyFlag)
	}

	for _, v := range dependentFlags {
		dependent := cmd.Flags().Lookup(v)
		if dependent == nil {
			return errors.Newf(`Flag "%s" not found`, v)
		}

		if !dependency.Changed && dependent.Changed {
			return errors.Newf(`Flag "%s" set without explicitly setting dependency "%s"`, v, dependencyFlag)
		}
	}

	return nil
}
