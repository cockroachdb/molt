package fetch

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/cmd/internal/cmdutil"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/fetchmetrics"
	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"
)

func Command() *cobra.Command {
	const (
		fetchID              = "fetch-id"
		continuationToken    = "continuation-token"
		continuationFileName = "continuation-file-name"
	)
	var (
		s3Bucket                string
		gcpBucket               string
		bucketPath              string
		localPath               string
		localPathListenAddr     string
		localPathCRDBAccessAddr string
		logFile                 string
		directCRDBCopy          bool
		cfg                     fetch.Config
	)
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Moves data from source to target.",
		Long:  `Imports data from source directly into target tables.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that if continuation-token is set that fetch-id is set
			if err := cmdutil.CheckFlagDependency(cmd, fetchID, []string{continuationToken}); err != nil {
				return err
			}
			// Ensure if continuation-file-name is set that continuation-token is set.
			if err := cmdutil.CheckFlagDependency(cmd, continuationToken, []string{continuationFileName}); err != nil {
				return err
			}

			// Ensure the continuation-file-name matches the file pattern.
			if strings.TrimSpace(cfg.ContinuationFileName) != "" && !utils.MatchesFileConvention(cfg.ContinuationFileName) {
				return errors.Newf(`continuation file name "%s" doesn't match the file convention "%s"`, cfg.ContinuationFileName, utils.FileConventionRegex.String())
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			logger, err := moltlogger.Logger(logFile)
			if err != nil {
				return err
			}
			cmdutil.RunMetricsServer(logger)

			isCopyMode := cfg.Live || directCRDBCopy
			if isCopyMode {
				if cfg.Compression == compression.GZIP {
					return errors.New("cannot run copy mode with compression")
				} else if cfg.Compression <= compression.Default {
					logger.Info().Msgf("default compression to none")
					cfg.Compression = compression.None
				}
			} else if !isCopyMode && cfg.Compression <= compression.Default {
				logger.Info().Msgf("default compression to GZIP")
				cfg.Compression = compression.GZIP
			} else {
				logger.Info().Msgf("user set compression to %s", cfg.Compression.String())
			}

			conns, err := cmdutil.LoadDBConns(ctx)
			if err != nil {
				return err
			}
			if !conns[1].IsCockroach() {
				return errors.AssertionFailedf("target must be cockroach")
			}

			var datastorePayload any

			switch {
			case directCRDBCopy:
				datastorePayload = &datablobstorage.DirectCopyPayload{
					TargetConnForCopy: conns[1].(*dbconn.PGConn).Conn,
				}
			case gcpBucket != "":
				datastorePayload = &datablobstorage.GCPPayload{
					GCPBucket:  gcpBucket,
					BucketPath: bucketPath,
				}
			case s3Bucket != "":
				datastorePayload = &datablobstorage.S3Payload{
					S3Bucket:   s3Bucket,
					BucketPath: bucketPath,
				}
			case localPath != "":
				datastorePayload = &datablobstorage.LocalPathPayload{
					LocalPath:               localPath,
					LocalPathListenAddr:     localPathListenAddr,
					LocalPathCRDBAccessAddr: localPathCRDBAccessAddr,
				}
			default:
				return errors.AssertionFailedf("data source must be configured (--s3-bucket, --gcp-bucket, --direct-copy, --local-path)")
			}

			src, err := datablobstorage.GenerateDatastore(ctx, datastorePayload, logger, false /* testFailedWriteToBucket */)
			if err != nil {
				return err
			}

			err = fetch.Fetch(
				ctx,
				cfg,
				logger,
				conns,
				src,
				cmdutil.TableFilter(),
				testutils.FetchTestingKnobs{},
			)

			if err != nil {
				fetchmetrics.NumTaskErrors.Inc()
			}

			return err
		},
	}

	cmd.PersistentFlags().StringVar(
		&logFile,
		"log-file",
		"",
		"If set, writes to the log file specified. Otherwise, only writes to stdout.",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Cleanup,
		"cleanup",
		false,
		"Whether any created resources should be deleted. Ignored if in direct-copy mode.",
	)
	cmd.PersistentFlags().BoolVar(
		&directCRDBCopy,
		"direct-copy",
		false,
		"Enables direct copy mode, which copies data directly from source to target without using an intermediate store.",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Live,
		"live",
		false,
		"Whether the table must be queryable during load import.",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.FlushSize,
		"flush-size",
		0,
		"If set, size (in bytes) before the source data is flushed to intermediate files.",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.FlushRows,
		"flush-rows",
		0,
		"If set, number of rows before the source data is flushed to intermediate files.",
	)

	cmd.PersistentFlags().IntVar(
		&cfg.Concurrency,
		"concurrency",
		4,
		"Number of tables to move at a time.",
	)
	cmd.PersistentFlags().StringVar(
		&s3Bucket,
		"s3-bucket",
		"",
		"Name of the S3 bucket.",
	)
	cmd.PersistentFlags().StringVar(
		&gcpBucket,
		"gcp-bucket",
		"",
		"Name of the GCP bucket.",
	)
	cmd.PersistentFlags().StringVar(
		&bucketPath,
		"bucket-path",
		"",
		"Path within the bucket where intermediate files are written (e.g., bucket-name/folder-name).",
	)
	cmd.PersistentFlags().StringVar(
		&localPath,
		"local-path",
		"",
		"Path to upload files to locally.",
	)
	cmd.PersistentFlags().StringVar(
		&localPathListenAddr,
		"local-path-listen-addr",
		"",
		"Address of a local store server to listen to for traffic.",
	)
	cmd.PersistentFlags().StringVar(
		&localPathCRDBAccessAddr,
		"local-path-crdb-access-addr",
		"",
		"Address of data that CockroachDB can access to import from a local store (defaults to local-path-listen-addr).",
	)
	cmd.MarkFlagsMutuallyExclusive("s3-bucket", "gcp-bucket", "local-path")

	// The test-only is for internal use only and is hidden from the usage or help prompt.
	const testOnlyFlagStr = "test-only"
	cmd.PersistentFlags().BoolVar(
		&cfg.TestOnly,
		testOnlyFlagStr,
		false,
		"Whether this fetch attempt is only for test, and hence all time/duration related stats are deterministic",
	)

	cmd.PersistentFlags().BoolVar(
		&cfg.Truncate,
		"truncate",
		false,
		"Whether to truncate the target tables before source data is imported.",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.ExportSettings.RowBatchSize,
		"row-batch-size",
		100_000,
		"Number of rows to select at a time for export from the source database.",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ExportSettings.PG.SlotName,
		"pg-logical-replication-slot-name",
		"",
		"If set, the name of a replication slot that will be created before taking a snapshot of data.",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ExportSettings.PG.Plugin,
		"pg-logical-replication-slot-plugin",
		"pgoutput",
		"If set, the output plugin used for logical replication under pg-logical-replication-slot-name.",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.ExportSettings.PG.DropIfExists,
		"pg-logical-replication-slot-drop-if-exists",
		false,
		"If set, drops the replication slot if it exists.",
	)
	cmd.PersistentFlags().Var(
		enumflag.New(
			&cfg.Compression,
			"compression",
			compression.CompressionStringRepresentations,
			enumflag.EnumCaseInsensitive,
		),
		"compression",
		"Compression type (default/gzip/none) to use (IMPORT INTO mode only).",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.FetchID,
		fetchID,
		"",
		"If set, restarts the fetch process for all failed tables of the given ID",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ContinuationToken,
		continuationToken,
		"",
		"If set, restarts the fetch process for the given continuation token for a specific table",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ContinuationFileName,
		continuationFileName,
		"",
		"If set, restarts the fetch process for at the given file name instead of recorded file in the exceptions table",
	)
	moltlogger.RegisterLoggerFlags(cmd)
	cmdutil.RegisterDBConnFlags(cmd)
	cmdutil.RegisterNameFilterFlags(cmd)
	cmdutil.RegisterMetricsFlags(cmd)

	if err := cmd.PersistentFlags().MarkHidden(testOnlyFlagStr); err != nil {
		panic(err)
	}

	return cmd
}
