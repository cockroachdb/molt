package fetch

import (
	"context"

	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/utils"
	"github.com/jackc/pgx/v5"
)

func ListContinuationTokens(
	ctx context.Context, testOnly bool, targetPgxConn *pgx.Conn, numResults int,
) (string, error) {
	exceptionLogs, err := status.GetAllExceptionLogs(ctx, targetPgxConn, numResults)
	if err != nil {
		return "", err
	}

	if len(exceptionLogs) == 0 {
		return "No continuation tokens found.\n", nil
	}

	// Loop through the exception log results.
	outputFormat := []utils.OutputFormat{}
	for _, item := range exceptionLogs {
		item.ID = utils.MaybeFormatID(testOnly, item.ID)
		item.FetchID = utils.MaybeFormatID(testOnly, item.FetchID)
		outputFormat = append(outputFormat, item)
	}

	tableStr, err := utils.BuildTable(outputFormat)
	if err != nil {
		return "", err
	}

	return tableStr, nil
}
