package fetchmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "molt"
	Subsystem = "fetch"
)

var (
	ImportedRows = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "rows_imported",
		Help:      "Number of rows that have been imported in.",
	}, []string{"table"})
)
