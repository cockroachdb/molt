package fetchmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "molt"
	Subsystem = "fetch"
)

// Just a note that these metrics are most useful in reporting progress
// for each table/schema combination. However, we do need to
// be mindful that there could be cardinality explosion for metrics
// if there are many table + schema combinations.
var (
	// Counts of entities in the fetch runs.
	NumTablesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "num_tables",
		Help:      "Number of tables migrated.",
	})
	ImportedRows = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "rows_imported",
		Help:      "Number of rows that have been imported by table.",
	}, []string{"table"})
	ExportedRows = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "rows_exported",
		Help:      "Number of rows that have been exported by table.",
	}, []string{"table"})

	// Data errors are ones relating to individual rows or files processed
	// for the import or copy.
	// TODO: still need to integrate data errors when we do exceptions logging.
	NumDataErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "num_data_errors",
		Help:      "Number of data level errors by table.",
	}, []string{"table"})
	NumTaskErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "num_task_errors",
		Help:      "Number of task errors.",
	})

	// Progress and duration metrics.
	CompletionPercentage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "completion_percent",
		Help:      "Completion percent by table.",
	}, []string{"table"})

	TableExportDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "table_export_duration_ms",
		Help:      "Duration (in milliseconds) for a particular table's export",
	}, []string{"table"})
	TableImportDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "table_import_duration_ms",
		Help:      "Duration (in milliseconds) for a particular table's import",
	}, []string{"table"})
	TableOverallDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "table_overall_duration_ms",
		Help:      "Duration (in milliseconds) for a particular table's fetch.",
	}, []string{"table"})
	OverallDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "overall_duration",
		Help:      "Duration (in seconds) for the overall fetch",
	})
)
