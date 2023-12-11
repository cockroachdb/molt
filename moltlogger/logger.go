package moltlogger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	LoggerTypeKey = "type"
	// Summary logs are ones that report on either task or step summaries
	// (i.e. 500 tables processed; task: "10 minutes")
	LoggerTypeSummary = "summary"
	// Data logs are ones that report on row or individual level operations
	// and quantitiative measures (i.e. 10000 rows completed; failed to verify item with PK 1000)
	LoggerTypeData = "data"
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

// The default logger returned without a `type` attribute is considered the task logger.
// We only mark a log differently if it relates to data and summary fields because
// those are the minority case and we want people to know that this information
// is relevant to the performance / operation of their data load.
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

func GetDataLogger(logger zerolog.Logger) zerolog.Logger {
	return logger.With().Str(LoggerTypeKey, LoggerTypeData).Logger()
}

func GetSummaryLogger(logger zerolog.Logger) zerolog.Logger {
	return logger.With().Str(LoggerTypeKey, LoggerTypeSummary).Logger()
}
