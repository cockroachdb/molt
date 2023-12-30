package main

import (
	"context"

	"github.com/cockroachdb/molt/moltlogger"
	"github.com/cockroachdb/molt/moltservice/api"
	"github.com/spf13/cobra"
)

// orchestratorCmd represents the orchestrator cmd.
var moltServiceCmd = &cobra.Command{
	Use:   "molt-service",
	Short: "Service that clients can interact with in order to intiate actions for MOLT tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := moltlogger.Logger("")
		if err != nil {
			return err
		}

		ctx := context.Background()
		server, err := api.NewServer(ctx, &api.ServerConfig{
			Logger:            logger,
			ListenPort:        4500,
			MetricsListenPort: 4499,
			ShowDocsHTML:      true,
		})
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

}
