package cmd

import (
	"errors"
	"net/url"

	"github.com/spf13/cobra"
)

func EscapePasswordCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "escape-password",
		Short: "Utility to help escape passwords with special characters.",
		Long:  `Utility to help percent-encode passwords that have special characters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("received empty password string")
			}
			passwordStr := args[0]
			cmd.Printf("Substitute the following encoded password in your original connection url string:\n%s\n", url.QueryEscape(passwordStr))
			return nil
		},
	}
}
