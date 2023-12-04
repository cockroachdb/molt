package verify

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/cmd/internal/cmdutil"
	"github.com/cockroachdb/molt/retry"
	"github.com/cockroachdb/molt/verify"
	"github.com/cockroachdb/molt/verify/inconsistency"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	// TODO: sanity check bounds.
	var (
		verifyConcurrency              int
		verifyTableSplits              int
		verifyRowBatchSize             int
		verifyFixup                    bool
		verifyContinuousPause          time.Duration
		verifyContinuous               bool
		verifyLive                     bool
		verifyLiveVerificationSettings = rowverify.LiveReverificationSettings{
			MaxBatchSize:  1000,
			FlushInterval: time.Second,
			RetrySettings: retry.Settings{
				InitialBackoff: 250 * time.Millisecond,
				Multiplier:     2,
				MaxBackoff:     time.Second,
				MaxRetries:     5,
			},
			RunsPerSecond: 0,
		}
		verifyLimitRowsPerSecond int
		verifyRows               bool
		verifyLogFile            string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify table schemas and row data align.",
		Long:  `Verify ensure table schemas and row data between the two databases are aligned.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := cmdutil.Logger(verifyLogFile)
			if err != nil {
				return err
			}
			cmdutil.RunMetricsServer(logger)

			reporter := inconsistency.CombinedReporter{}
			reporter.Reporters = append(reporter.Reporters, &inconsistency.LogReporter{Logger: logger})
			defer reporter.Close()

			ctx := context.Background()
			conns, err := cmdutil.LoadDBConns(ctx)
			if err != nil {
				return err
			}
			if verifyFixup {
				fixupConn, err := conns[1].Clone(ctx)
				if err != nil {
					panic(err)
				}
				reporter.Reporters = append(reporter.Reporters, &inconsistency.FixReporter{
					Conn:   fixupConn,
					Logger: logger,
				})
			}

			reporter.Report(inconsistency.StatusReport{Info: "verification in progress"})
			if err := verify.Verify(
				ctx,
				conns,
				logger,
				reporter,
				verify.WithConcurrency(verifyConcurrency),
				verify.WithTableSplits(verifyTableSplits),
				verify.WithRowBatchSize(verifyRowBatchSize),
				verify.WithContinuous(verifyContinuous, verifyContinuousPause),
				verify.WithLive(verifyLive, verifyLiveVerificationSettings),
				verify.WithDBFilter(cmdutil.TableFilter()),
				verify.WithRowsPerSecond(verifyLimitRowsPerSecond),
				verify.WithRows(verifyRows),
			); err != nil {
				return errors.Wrapf(err, "error verifying")
			}
			reporter.Report(inconsistency.StatusReport{Info: "verification complete"})
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(
		&verifyLogFile,
		"log-file",
		"",
		"If set, writes to the log file specified. Otherwise, only writes to stdout.",
	)
	cmd.PersistentFlags().IntVar(
		&verifyConcurrency,
		"concurrency",
		0,
		"Number of tables to process at a time (defaults to number of CPUs).",
	)
	cmd.PersistentFlags().IntVar(
		&verifyTableSplits,
		"table-splits",
		1,
		"Number of shards to break down each table into while doing row-based verification.",
	)
	cmd.PersistentFlags().IntVar(
		&verifyRowBatchSize,
		"row-batch-size",
		20000,
		"Number of source/target rows to scan at a time.",
	)
	cmd.PersistentFlags().IntVar(
		&verifyLimitRowsPerSecond,
		"rows-per-second",
		0,
		"If set, maximum number of rows to read per second on each shard.",
	)
	cmd.PersistentFlags().BoolVar(
		&verifyFixup,
		"fixup",
		false,
		"Whether to fix up inconsistencies found during row verification.",
	)
	cmd.PersistentFlags().DurationVar(
		&verifyContinuousPause,
		"continuous-pause-between-runs",
		0,
		"Amount of time to pause between continuous runs (e.g. 1h, 2m).",
	)
	cmd.PersistentFlags().BoolVar(
		&verifyContinuous,
		"continuous",
		false,
		"Whether verification should continuously run on each shard.",
	)
	cmd.PersistentFlags().BoolVar(
		&verifyLive,
		"live",
		false,
		"Enable live mode, which attempts to account for rows that can change in value by retrying them before marking them as inconsistent.",
	)
	cmd.PersistentFlags().BoolVar(
		&verifyRows,
		"rows",
		true,
		"If true, verify both the schema (columns, types) and row data. If false, verify only the schema.",
	)
	cmd.PersistentFlags().IntVar(
		&verifyLiveVerificationSettings.RunsPerSecond,
		"live-runs-per-second",
		verifyLiveVerificationSettings.RunsPerSecond,
		"Maximum number of retry attempts per second (live mode only).",
	)
	cmd.PersistentFlags().IntVar(
		&verifyLiveVerificationSettings.MaxBatchSize,
		"live-max-batch-size",
		verifyLiveVerificationSettings.MaxBatchSize,
		"Maximum number of rows to retry at a time (live mode only).",
	)
	cmd.PersistentFlags().DurationVar(
		&verifyLiveVerificationSettings.FlushInterval,
		"live-flush-interval",
		verifyLiveVerificationSettings.FlushInterval,
		"Maximum amount of time to wait before retrying rows (live mode only).",
	)
	cmd.PersistentFlags().IntVar(
		&verifyLiveVerificationSettings.RetrySettings.MaxRetries,
		"live-retries-max-iterations",
		verifyLiveVerificationSettings.RetrySettings.MaxRetries,
		"Maximum number of retries before marking rows as inconsistent (live mode only).",
	)
	cmd.PersistentFlags().DurationVar(
		&verifyLiveVerificationSettings.RetrySettings.MaxBackoff,
		"live-retry-max-backoff",
		verifyLiveVerificationSettings.RetrySettings.MaxBackoff,
		"Maximum amount of time a retry attempt should take before retrying again (live mode only).",
	)
	cmd.PersistentFlags().DurationVar(
		&verifyLiveVerificationSettings.RetrySettings.InitialBackoff,
		"live-retry-initial-backoff",
		verifyLiveVerificationSettings.RetrySettings.InitialBackoff,
		"Amount of time live verification should initially backoff for before retrying.",
	)
	cmd.PersistentFlags().IntVar(
		&verifyLiveVerificationSettings.RetrySettings.Multiplier,
		"live-retry-multiplier",
		verifyLiveVerificationSettings.RetrySettings.Multiplier,
		"Multiplier to apply to backoff duration after each failed row verification (live mode only).",
	)
	for _, hidden := range []string{"fixup", "table-splits"} {
		if err := cmd.PersistentFlags().MarkHidden(hidden); err != nil {
			panic(err)
		}
	}
	cmdutil.RegisterDBConnFlags(cmd)
	cmdutil.RegisterLoggerFlags(cmd)
	cmdutil.RegisterNameFilterFlags(cmd)
	cmdutil.RegisterMetricsFlags(cmd)
	return cmd
}
