package cmd

import (
	"fmt"
	"os"

	"github.com/cockroachdb/molt/cmd/fetch"
	"github.com/cockroachdb/molt/cmd/verify"
	"github.com/spf13/cobra"
)

const rootCmdUse = "molt"

var rootCmd = &cobra.Command{
	Use:   "molt",
	Short: "Onboarding assistance for migrating to CockroachDB",
	Long:  `MOLT (Migrate Off Legacy Things) provides tooling which assists migrating off other database providers to CockroachDB.`,
}

func walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, c := range c.Commands() {
		walk(c, f)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(EscapePasswordCommand())
	rootCmd.AddCommand(verify.Command())
	rootCmd.AddCommand(fetch.Command())

	// Hide completion options, because irrelevant.
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Overrides the default help text and descriptions.
	walk(rootCmd, func(c *cobra.Command) {
		c.Flags().BoolP("help", "h", false, fmt.Sprintf("Help for %s command.", c.Name()))
	})
	rootCmd.InitDefaultHelpCmd()
	walk(rootCmd, func(c *cobra.Command) {
		if c.Name() == "help" {
			c.Short = fmt.Sprintf("Help command for %s.", rootCmdUse)
		}
	})
}
