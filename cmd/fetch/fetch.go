package fetch

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/cmd/internal/cmdutil"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"
	"golang.org/x/oauth2/google"
)

func Command() *cobra.Command {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			logger, err := cmdutil.Logger(logFile)
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
			} else if !isCopyMode && cfg.Compression == compression.Default {
				logger.Info().Msgf("default compression to gzip")
				cfg.Compression = compression.GZIP
			}

			conns, err := cmdutil.LoadDBConns(ctx)
			if err != nil {
				return err
			}
			if !conns[1].IsCockroach() {
				return errors.AssertionFailedf("target must be cockroach")
			}

			var src datablobstorage.Store
			switch {
			case directCRDBCopy:
				src = datablobstorage.NewCopyCRDBDirect(logger, conns[1].(*dbconn.PGConn).Conn)
			case gcpBucket != "":
				creds, err := google.FindDefaultCredentials(ctx)
				if err != nil {
					return err
				}
				gcpClient, err := storage.NewClient(context.Background())
				if err != nil {
					return err
				}
				src = datablobstorage.NewGCPStore(logger, gcpClient, creds, gcpBucket, bucketPath)
			case s3Bucket != "":
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				creds, err := sess.Config.Credentials.Get()
				if err != nil {
					return err
				}
				src = datablobstorage.NewS3Store(logger, sess, creds, s3Bucket, bucketPath)
			case localPath != "":
				src, err = datablobstorage.NewLocalStore(logger, localPath, localPathListenAddr, localPathCRDBAccessAddr)
				if err != nil {
					return err
				}
			default:
				return errors.AssertionFailedf("data source must be configured (--s3-bucket, --gcp-bucket, --direct-copy)")
			}
			return fetch.Fetch(
				ctx,
				cfg,
				logger,
				conns,
				src,
				cmdutil.TableFilter(),
			)
		},
	}

	cmd.PersistentFlags().StringVar(
		&logFile,
		"log-file",
		"",
		"If set, writes to the log file specified. Otherwise, only writes to stdout.",
	)
	cmd.PersistentFlags().BoolVar(
		&directCRDBCopy,
		"direct-copy",
		false,
		"Enables direct copy mode, which copies data directly from source to target without using an intermediate store.",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Cleanup,
		"cleanup",
		false,
		"Whether any created resources should be deleted.",
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
	cmdutil.RegisterDBConnFlags(cmd)
	cmdutil.RegisterLoggerFlags(cmd)
	cmdutil.RegisterNameFilterFlags(cmd)
	cmdutil.RegisterMetricsFlags(cmd)
	return cmd
}
