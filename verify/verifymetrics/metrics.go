package verifymetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "molt"
	Subsystem = "verify"
)

var (
	// Overall task
	NumShards = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "shards_running",
		Help:      "Number of verification shards that are running.",
	})
	NumTablesProcessed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "num_tables",
		Help:      "Number of tables per category.",
	}, []string{"category"})
	NumRowFixups = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "num_row_fixups",
		Help:      "Number of fixups per category.",
	}, []string{"category", "table"})
	OverallDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "duration_seconds",
		Help:      "Duration (in seconds) for the verify run.",
	})

	// Reverifier
	LivePrimaryKeys = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "live_primary_keys",
		Help:      "Number of primary keys that are being reverified.",
	})
	LiveBatches = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "live_batches",
		Help:      "Number of batches that are in the queue to be reverified.",
	})
	LiveRows = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "live_reverified_rows",
		Help:      "Number of rows that require reverification by the live reverifier.",
	})
	LiveRemainingPKs = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "live_queued_pks",
		Help:      "Number of rows that are queued by the live reverifier.",
	})
	LiveRemainingBatches = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "live_queued_batches",
		Help:      "Number of batches of rows that require the live reverifier.",
	})

	// Row verification.
	RowStatus = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "row_verification_status",
		Help:      "Status of rows that have been verified.",
	}, []string{"status", "table"})
	RowsRead = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "rows_read",
		Help:      "Rate of rows that are being read from source database.",
	}, []string{"table"})

	rowStatusCategories = []string{"extraneous", "missing", "mismatching", "mismatching_column", "success", "conditional_success"}
	tableCategories     = []string{"verified", "missing", "extraneous"}
	fixupCategories     = []string{"mismatching", "missing", "extraneous"}
)

func init() {
	// Initialize the label for the number of tables processed.
	for _, t := range tableCategories {
		NumTablesProcessed.WithLabelValues(t)
	}

	for _, f := range fixupCategories {
		NumRowFixups.WithLabelValues(f, "overall")
	}

	for _, r := range rowStatusCategories {
		RowStatus.WithLabelValues(r, "overall")
	}
}
