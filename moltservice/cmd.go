package main

import (
	"context"
	"os"

	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/moltservice/api"
	"github.com/spf13/cobra"
)

const (
	artifactsDir = "artifacts"

	defaultListenPort        = 4500
	defaultMetricsListenPort = 4499

	flagDebug        = "debug"
	flagListenPort   = "port"
	flagMetricsPort  = "metrics-port"
	flagShowDocsHTML = "show-docs-html"
)

var cfg = &api.ServerConfig{}

// orchestratorCmd represents the orchestrator cmd.
var moltServiceCmd = &cobra.Command{
	Use:   "molt-service",
	Short: "Service that clients can interact with in order to intiate actions for MOLT tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := moltlogger.Logger("")
		if err != nil {
			return err
		}

		// Setup artifacts directory.
		err = os.MkdirAll(artifactsDir, os.ModePerm)
		if err != nil {
			return err
		}

		ctx := context.Background()
		cfg.Logger = logger
		server, err := api.NewServer(ctx, cfg)
		if err != nil {
			return err
		}
		serviceSvr := server.ServiceServer
		metricsSvr := server.HealthMetricsServer

		go func() {
			logger.Info().Msgf("Listening and serving metrics HTTP on: %s", metricsSvr.Addr)
			if err := metricsSvr.ListenAndServe(); err != nil {
				logger.Err(err).Msg("failed to start metrics server")
			}
		}()

		// TODO: do TLS later.
		logger.Info().Msgf("Listening and serving HTTP on: %s", serviceSvr.Addr)
		if err := serviceSvr.ListenAndServe(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	moltServiceCmd.PersistentFlags().BoolVar(
		&cfg.DebugMode,
		"debug",
		true,
		"whether to enable debug logging.",
	)

	moltServiceCmd.PersistentFlags().IntVar(
		&cfg.ListenPort,
		flagListenPort,
		defaultListenPort,
		"the port to serve the API from",
	)

	moltServiceCmd.PersistentFlags().IntVar(
		&cfg.MetricsListenPort,
		flagMetricsPort,
		defaultMetricsListenPort,
		"the port to serve the API from",
	)

	moltServiceCmd.PersistentFlags().BoolVar(
		&cfg.ShowDocsHTML,
		flagShowDocsHTML,
		true,
		"whether or not to show the docs.html interactive page",
	)
}
