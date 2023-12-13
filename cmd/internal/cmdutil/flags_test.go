package cmdutil

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCheckFlagDependency(t *testing.T) {
	cmd := &cobra.Command{}
	// Mix in persistent and no persistent flags to properly test for mixed case.
	cmd.PersistentFlags().Bool("dependency", false, "")
	cmd.Flags().Bool("updated-dependency", false, "")
	require.NoError(t, cmd.Flags().Set("updated-dependency", "true"))

	cmd.Flags().Bool("not-updated-dependent", false, "")
	cmd.PersistentFlags().Bool("updated-dependent", false, "")
	require.NoError(t, cmd.PersistentFlags().Set("updated-dependent", "true"))

	type args struct {
		cmd            *cobra.Command
		dependencyFlag string
		dependentFlags []string
	}
	tests := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "dependency flag does not exist",
			args: args{
				cmd:            cmd,
				dependencyFlag: "non-existent",
			},
			expectedErr: errors.New(`Flag "non-existent" not found`),
		},
		{
			name: "dependent flags do not exist",
			args: args{
				cmd:            cmd,
				dependencyFlag: "dependency",
				dependentFlags: []string{"missing-dependent-flag"},
			},
			expectedErr: errors.New(`Flag "missing-dependent-flag" not found`),
		},
		{
			name: "neither dependent nor dependency flag set by user",
			args: args{
				cmd:            cmd,
				dependencyFlag: "dependency",
				dependentFlags: []string{"not-updated-dependent"},
			},
			expectedErr: nil,
		},
		{
			name: "neither dependent nor dependency flag set by user",
			args: args{
				cmd:            cmd,
				dependencyFlag: "dependency",
				dependentFlags: []string{"not-updated-dependent"},
			},
			expectedErr: nil,
		},
		{
			name: "both dependent and dependency flag set by user",
			args: args{
				cmd:            cmd,
				dependencyFlag: "updated-dependency",
				dependentFlags: []string{"updated-dependent"},
			},
			expectedErr: nil,
		},
		{
			name: "dependent flag not set but dependency flag set by user",
			args: args{
				cmd:            cmd,
				dependencyFlag: "dependency",
				dependentFlags: []string{"updated-dependent"},
			},
			expectedErr: errors.New(`Flag "updated-dependent" set without explicitly setting dependency "dependency"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualErr := CheckFlagDependency(tt.args.cmd, tt.args.dependencyFlag, tt.args.dependentFlags)
			if tt.expectedErr != nil {
				require.EqualError(t, tt.expectedErr, actualErr.Error())
			} else {
				require.Nil(t, actualErr)
			}
		})
	}
}
