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
		localPath               string
		localPathListenAddr     string
		localPathCRDBAccessAddr string
		directCRDBCopy          bool
		cfg                     fetch.Config
	)
	cmd := &cobra.Command{
		Use:  "fetch",
		Long: `Fetches data from source to directly import into target.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			logger, err := cmdutil.Logger()
			if err != nil {
				return err
			}
			cmdutil.RunMetricsServer(logger)

			if cfg.Live {
				if cfg.Compression == compression.GZIP {
					return errors.New("cannot run live mode with compression")
				} else if cfg.Compression <= compression.Default {
					logger.Info().Msgf("default compression to none")
					cfg.Compression = compression.None
				}
			} else {
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
				src = datablobstorage.NewGCPStore(logger, gcpClient, creds, gcpBucket)
			case s3Bucket != "":
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				creds, err := sess.Config.Credentials.Get()
				if err != nil {
					return err
				}
				src = datablobstorage.NewS3Store(logger, sess, creds, s3Bucket)
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

	cmd.PersistentFlags().BoolVar(
		&directCRDBCopy,
		"direct-copy",
		false,
		"whether to use direct copy mode",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Cleanup,
		"cleanup",
		false,
		"whether any file resources created should be deleted",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Live,
		"live",
		false,
		"whether the table must be queriable during load import",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.FlushSize,
		"flush-size",
		0,
		"if set, size (in bytes) before the data source is flushed",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.FlushRows,
		"flush-rows",
		0,
		"if set, number of rows before the data source is flushed",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.Concurrency,
		"concurrency",
		4,
		"number of tables to move data with at a time",
	)
	cmd.PersistentFlags().StringVar(
		&s3Bucket,
		"s3-bucket",
		"",
		"s3 bucket",
	)
	cmd.PersistentFlags().StringVar(
		&gcpBucket,
		"gcp-bucket",
		"",
		"gcp bucket",
	)
	cmd.PersistentFlags().StringVar(
		&localPath,
		"local-path",
		"",
		"path to upload files to locally",
	)
	cmd.PersistentFlags().StringVar(
		&localPathListenAddr,
		"local-path-listen-addr",
		"",
		"local address to listen to for traffic",
	)
	cmd.PersistentFlags().StringVar(
		&localPathCRDBAccessAddr,
		"local-path-crdb-access-addr",
		"",
		"address CockroachDB can access to connect to the --local-path-listen-addr",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.Truncate,
		"truncate",
		false,
		"whether to truncate the table being imported to",
	)
	cmd.PersistentFlags().IntVar(
		&cfg.ExportSettings.RowBatchSize,
		"row-batch-size",
		100_000,
		"how many rows to select at a time for the database",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ExportSettings.PG.SlotName,
		"pg-logical-replication-slot-name",
		"",
		"if set, the name of a replication slot that should be created before taking a snapshot of data",
	)
	cmd.PersistentFlags().StringVar(
		&cfg.ExportSettings.PG.Plugin,
		"pg-logical-replication-slot-plugin",
		"pgoutput",
		"if set, the output plugin used for logical replication under pg-logical-replication-slot-name",
	)
	cmd.PersistentFlags().BoolVar(
		&cfg.ExportSettings.PG.DropIfExists,
		"pg-logical-replication-slot-drop-if-exists",
		false,
		"if set, drops the replication slot if it exists",
	)
	cmd.PersistentFlags().Var(
		enumflag.New(
			&cfg.Compression,
			"compression",
			compression.CompressionStringRepresentations,
			enumflag.EnumCaseInsensitive,
		),
		"compression",
		"compression to use (default/gzip/none)",
	)
	cmdutil.RegisterDBConnFlags(cmd)
	cmdutil.RegisterLoggerFlags(cmd)
	cmdutil.RegisterNameFilterFlags(cmd)
	cmdutil.RegisterMetricsFlags(cmd)
	return cmd
}
