package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus instruments.
type Metrics struct {
	RequestDuration *prometheus.HistogramVec
	ScenarioHits    *prometheus.CounterVec
	ErrorTotal      *prometheus.CounterVec
}

// New registers and returns application metrics.
func New() *Metrics {
	return &Metrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mock_server_request_duration_seconds",
				Help:    "HTTP request latency distribution.",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"method", "path", "status"},
		),
		ScenarioHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mock_scenario_executions_total",
				Help: "Number of times each mock scenario was triggered.",
			},
			[]string{"scenario", "endpoint"},
		),
		ErrorTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mock_server_errors_total",
				Help: "Total number of errors by type.",
			},
			[]string{"type"},
		),
	}
}
