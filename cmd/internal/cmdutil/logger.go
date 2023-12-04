package cmdutil

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type loggerConfig struct {
	level            string
	useConsoleWriter bool
}

var loggerConfigInst = loggerConfig{
	level:            zerolog.InfoLevel.String(),
	useConsoleWriter: false,
}

func RegisterLoggerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&loggerConfigInst.level,
		"logging",
		loggerConfigInst.level,
		"Level to log at (maps to zerolog.Level).",
	)
	cmd.PersistentFlags().BoolVar(
		&loggerConfigInst.useConsoleWriter,
		"use-console-writer",
		loggerConfigInst.useConsoleWriter,
		"Use the console writer, which has cleaner log output but introduces more latency (defaults to false, which logs as structured JSON).",
	)
}

func Logger(fileName string) (zerolog.Logger, error) {
	var writer io.Writer = os.Stdout
	if loggerConfigInst.useConsoleWriter {
		writer = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = time.RFC3339
		})
	}

	if fileName != "" {
		dir := filepath.Dir(fileName)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return zerolog.Logger{}, err
		}

		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return zerolog.Logger{}, err
		}
		writer = io.MultiWriter(writer, f)
	}

	logger := zerolog.New(writer)
	lvl, err := zerolog.ParseLevel(loggerConfigInst.level)
	if err != nil {
		return logger, err
	}
	return logger.Level(lvl).With().Timestamp().Logger(), err
}
