package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Connection metrics
	connectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gowsoos_connections_total",
			Help: "Total number of connections",
		},
		[]string{"type", "status"},
	)

	connectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gowsoos_connections_active",
			Help: "Number of active connections",
		},
	)

	// Traffic metrics
	bytesTransferred = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gowsoos_bytes_transferred_total",
			Help: "Total bytes transferred",
		},
		[]string{"direction"},
	)

	// Duration metrics
	connectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gowsoos_connection_duration_seconds",
			Help:    "Connection duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// Error metrics
	errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gowsoos_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "error"},
	)
)

// Metrics holds the metrics collector
type Metrics struct {
	enabled bool
	logger  *slog.Logger
}

// NewMetrics creates a new metrics collector
func NewMetrics(enabled bool, logger *slog.Logger) *Metrics {
	m := &Metrics{
		enabled: enabled,
		logger:  logger,
	}

	if enabled {
		// Register metrics with Prometheus
		prometheus.MustRegister(connectionsTotal)
		prometheus.MustRegister(connectionsActive)
		prometheus.MustRegister(bytesTransferred)
		prometheus.MustRegister(connectionDuration)
		prometheus.MustRegister(errorsTotal)

		logger.Info("Metrics enabled")
	}

	return m
}

// RecordConnection records a connection event
func (m *Metrics) RecordConnection(connType, status string) {
	if !m.enabled {
		return
	}
	connectionsTotal.WithLabelValues(connType, status).Inc()
	connectionsActive.Inc()
}

// RecordConnectionClosed records a connection closure
func (m *Metrics) RecordConnectionClosed() {
	if !m.enabled {
		return
	}
	connectionsActive.Dec()
}

// RecordBytesTransferred records bytes transferred
func (m *Metrics) RecordBytesTransferred(direction string, bytes int64) {
	if !m.enabled {
		return
	}
	bytesTransferred.WithLabelValues(direction).Add(float64(bytes))
}

// RecordConnectionDuration records connection duration
func (m *Metrics) RecordConnectionDuration(connType string, duration float64) {
	if !m.enabled {
		return
	}
	connectionDuration.WithLabelValues(connType).Observe(duration)
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, errorMsg string) {
	if !m.enabled {
		return
	}
	errorsTotal.WithLabelValues(errorType, errorMsg).Inc()
}

// StartMetricsServer starts the Prometheus metrics server
func (m *Metrics) StartMetricsServer(address string) error {
	if !m.enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	m.logger.Info("Starting metrics server", "address", address)
	return server.ListenAndServe()
}