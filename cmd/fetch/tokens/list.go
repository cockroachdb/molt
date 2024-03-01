package tokens

import (
	"context"
	"errors"

	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var connString string
	var numResults int
	var testOnly bool

	cmd := &cobra.Command{
		Use:   "tokens list",
		Short: "List details about each continuation token.",
		Long:  `List details about each continuation token.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			conn, err := dbconn.Connect(ctx, "target", connString)
			if err != nil {
				return err
			}

			targetPgConn, valid := conn.(*dbconn.PGConn)
			if !valid {
				return errors.New("failed to assert conn as a pgconn")
			}
			targetPgxConn := targetPgConn.Conn

			tableStr, err := fetch.ListContinuationTokens(ctx, testOnly, targetPgxConn, numResults)
			if err != nil {
				return err
			}

			_, err = cmd.OutOrStdout().Write([]byte(tableStr))
			return err
		},
	}

	cmd.PersistentFlags().StringVar(
		&connString,
		"conn-string",
		"",
		"Connection string of the database which has the _molt_fetch metadata.",
	)
	cmd.PersistentFlags().IntVarP(
		&numResults,
		"num-results",
		"n",
		0,
		"Number of results to return",
	)
	cmd.PersistentFlags().BoolVarP(
		&testOnly,
		"test-only",
		"t",
		false,
		"If set, runs in test mode with deterministic data.",
	)

	if err := cmd.MarkPersistentFlagRequired("conn-string"); err != nil {
		panic(err)
	}

	return cmd
}
