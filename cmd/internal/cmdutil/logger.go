package cmdutil

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type loggerConfig struct {
	level string
}

var loggerConfigInst = loggerConfig{
	level: zerolog.InfoLevel.String(),
}

func RegisterLoggerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&loggerConfigInst.level,
		"logging",
		loggerConfigInst.level,
		"Level to log at (maps to zerolog.Level).",
	)
}

func Logger() (zerolog.Logger, error) {
	logger := zerolog.New(zerolog.NewConsoleWriter())
	lvl, err := zerolog.ParseLevel(loggerConfigInst.level)
	if err != nil {
		return logger, err
	}
	return logger.Level(lvl), err
}
