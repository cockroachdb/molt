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
	"github.com/cockroachdb/molt/fetch/internal/dataquery"
	"github.com/cockroachdb/molt/retry"
	"github.com/jackc/pgx/v5"
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

const (
	pattern     = `%`
	replacement = `\%`
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
				logger.Info().Str("completion", fmt.Sprintf("%.2f%%", frac*100)).Msgf("progress")
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
) (importResult, error) {
	ret := importResult{
		StartTime: time.Now(),
	}

	var locs []string
	for _, resource := range resources {
		u, err := resource.ImportURL()
		if err != nil {
			return importResult{}, err
		}
		locs = append(locs, u)
	}
	conn := baseConn.(*dbconn.PGConn)
	r, err := retry.NewRetry(retry.Settings{
		InitialBackoff: time.Second,
		Multiplier:     2,
		MaxRetries:     4,
	})
	if err != nil {
		return ret, err
	}
	if err := r.Do(func() error {
		kvOptions := tree.KVOptions{}
		if cfg.Compression == compression.GZIP {
			kvOptions = append(kvOptions, tree.KVOption{
				Key:   "decompress",
				Value: tree.NewStrVal("gzip"),
			})
		}

		if _, err := conn.Exec(
			ctx,
			dataquery.ImportInto(table, locs, kvOptions),
		); err != nil {
			return errors.Wrap(err, "error importing data")
		}
		return nil
	}, func(err error) {
		logger.Err(err).Msgf("error importing data, retrying")
	}); err != nil {
		return ret, err
	}
	ret.EndTime = time.Now()
	return ret, nil
}
