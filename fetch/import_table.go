package fetch

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/fetchmetrics"
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/testutils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

type importResult struct {
	StartTime time.Time
	EndTime   time.Time
}

type importProgress struct {
	Description       string
	Started           time.Time
	FractionCompleted float64 `db:"fraction_completed"`
}

type PGIface interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

const (
	pattern     = `%`
	replacement = `\%`
	batchSize   = 10
)

var re = regexp.MustCompile(pattern)

func getShowJobsQuery(table dbtable.VerifiedTable, curTime string) string {
	schema := strings.Trim(re.ReplaceAllLiteralString(table.Schema.String(), replacement), `"`)
	tableName := strings.Trim(re.ReplaceAllLiteralString(table.Table.String(), replacement), `"`)
	return fmt.Sprintf(`WITH x as (SHOW JOBS)
SELECT description, started, fraction_completed
FROM x
WHERE job_type='IMPORT'
    AND description LIKE '%%%s.%s(%%'
    AND started > '%s'
ORDER BY created DESC`,
		schema, tableName, curTime)
}

func reportImportTableProgress(
	ctx context.Context,
	baseConn dbconn.Conn,
	logger zerolog.Logger,
	table dbtable.VerifiedTable,
	curTime time.Time,
	testing bool,
) error {
	dataLogger := moltlogger.GetDataLogger(logger)
	curTimeUTC := curTime.UTC().Format("2006-01-02T15:04:05")
	r, err := retry.NewRetry(retry.Settings{
		InitialBackoff: 10 * time.Second,
		Multiplier:     1,
		MaxRetries:     math.MaxInt64,
	})
	if err != nil {
		return err
	}

	pgConn, ok := baseConn.(*dbconn.PGConn)
	if !ok {
		return errors.Newf("expected pgx conn, got %T", baseConn)
	}

	conn, err := pgx.ConnectConfig(ctx, pgConn.Config())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	prevVal := 0.0

	if err := r.Do(func() error {
		query := getShowJobsQuery(table, curTimeUTC)
		rows, err := conn.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		p, err := pgx.CollectRows(rows, pgx.RowToStructByName[importProgress])
		if err != nil {
			return err
		} else if len(p) == 0 {
			return errors.New("retrying because no rows found")
		} else if p[0].FractionCompleted != 1 {
			frac := p[0].FractionCompleted
			if frac != 0.0 && prevVal != frac {
				dataLogger.Info().Str("completion", fmt.Sprintf("%.2f%%", frac*100)).Msgf("progress")
				fetchmetrics.CompletionPercentage.WithLabelValues(table.SafeString()).Set(frac * 100)
			}

			prevVal = p[0].FractionCompleted
			return errors.New("retrying because job not finished yet")
		}

		if testing {
			logger.Info().Msgf("%.2f%% completed (%s.%s)", p[0].FractionCompleted*100, table.Schema.String(), table.Table.String())
		}

		return nil
	}, func(err error) {}); err != nil {
		return err
	}

	return err
}

func importTable(
	ctx context.Context,
	cfg Config,
	baseConn dbconn.Conn,
	logger zerolog.Logger,
	table dbtable.VerifiedTable,
	resources []datablobstorage.Resource,
	testingKnobs testutils.FetchTestingKnobs,
	isLocal bool,
) (importResult, error) {
	ret := importResult{
		StartTime: time.Now(),
	}

	var locs []string
	var numRows []int
	for _, resource := range resources {
		u, err := resource.ImportURL()
		if err != nil {
			return importResult{}, err
		}
		locs = append(locs, u)
		numRows = append(numRows, resource.Rows())
	}
	conn := baseConn.(*dbconn.PGConn)

	kvOptions := tree.KVOptions{}
	if cfg.Compression == compression.GZIP {
		kvOptions = append(kvOptions, tree.KVOption{
			Key:   "decompress",
			Value: tree.NewStrVal("gzip"),
		})
	}

	// In local mode, we skip the header row which would contain the number of rows.
	if isLocal {
		kvOptions = append(kvOptions, tree.KVOption{
			Key:   "skip",
			Value: tree.NewStrVal("1"),
		})
	}

	for i := 0; i < len(locs); i += batchSize {
		end := i + batchSize
		// necessary to prevent going over len
		if end > len(locs) {
			end = len(locs)
		}
		locBatch := locs[i:end]
		totalRows := sumSlice(numRows[i:end])

		file, err := importWithBisect(ctx, kvOptions, table, logger, conn, locBatch)
		if err != nil {
			fileName := status.ExtractFileNameFromErr(file)
			pgErr := status.MaybeReportException(ctx, logger, baseConn.(*dbconn.PGConn).Conn, table.Name, err, fileName, status.StageDataLoad)
			return ret, errors.Wrap(pgErr, "error importing data")
		}

		logger.Info().Msgf("imported %d rows for batch for files %d to %d", totalRows, i+1, end)
		fetchmetrics.ImportedRows.WithLabelValues(table.SafeString()).Add(float64(totalRows))
	}
	ret.EndTime = time.Now()
	return ret, nil
}

func sumSlice(input []int) int {
	output := 0
	for _, val := range input {
		output += val
	}
	return output
}

// importWithBisect handles the logic of trying to find the
// broken file in a batch of files that were sent to import.
// We are using a batch size of 10 files and the way the algorithm
// breaks up the files is as follows.
// [1,2,3,4,5,6,7,8,9,10]
// [1,2,3,4,5][6,7,8,9,10]
// [1,2][3,4,5][6,7][8,9,10]
// [1][2][3][4,5][6][7][8][9,10]
// [4][5][9][10]
// Since the pushes to the stack are in FIFO order, if there are multiple
// files with errors, there is a guarantee that the lowest file part is
// returned first to preserve import order. So, in the example above, if file
// 3 and 5 had errors, 3 would be returned first since the row showing the
// breakdown [1][2][3][4,5] shows that 3 is being processed before 5 in the
// stack.
func importWithBisect(
	ctx context.Context,
	kvOptions tree.KVOptions,
	table dbtable.VerifiedTable,
	logger zerolog.Logger,
	conn PGIface,
	locs []string,
) (string, error) {
	stack := [][]string{locs}
	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		importQuery, redactedQuery := dataquery.ImportInto(table, curr, kvOptions)
		logger.Debug().Msgf("running import query: %q", redactedQuery)
		_, err := conn.Exec(ctx, importQuery)

		// If the import query returns an error, then we need to bisect.
		// Otherwise do nothing. If the len is more than 1, we have
		// not bottomed out yet on a leaf node.
		if err != nil && len(curr) > 1 {
			mid := len(curr) / 2
			stack = append(stack, curr[mid:], curr[:mid])
			// If there is an error and the current batch is of
			// length 1, then we reached a leaf and found the issue.
		} else if err != nil && len(curr) == 1 {
			return curr[0], err
		}
		// No else case needed since we are just pruning all successful files
		// and don't want to push to the stack.
	}
	return "", nil
}
