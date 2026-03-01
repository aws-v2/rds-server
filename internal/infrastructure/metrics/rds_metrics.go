package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RDSMetrics provides Prometheus metrics collection for RDS service
type RDSMetrics struct {
	// HTTP Metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// Instance Metrics
	instancesTotal            prometheus.Gauge
	instanceOperationsTotal   *prometheus.CounterVec
	instanceOperationDuration *prometheus.HistogramVec

	// Container Metrics
	containerStartsTotal   *prometheus.CounterVec
	containerStopsTotal    *prometheus.CounterVec
	containerFailuresTotal *prometheus.CounterVec

	// Database Metrics
	dbQueriesTotal    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec
	dbConnectionsOpen prometheus.Gauge
	dbConnectionsIdle prometheus.Gauge
}

// NewRDSMetrics creates a new RDS metrics adapter
func NewRDSMetrics(namespace string) *RDSMetrics {
	if namespace == "" {
		namespace = "rds_service"
	}

	return &RDSMetrics{
		// HTTP Metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		// Instance Metrics
		instancesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instances_total",
				Help:      "Total number of database instances by status",
			},
		),
		instanceOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "instance_operations_total",
				Help:      "Total number of instance operations",
			},
			[]string{"operation", "status"},
		),
		instanceOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "instance_operation_duration_seconds",
				Help:      "Instance operation duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		// Container Metrics
		containerStartsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "container_starts_total",
				Help:      "Total number of container starts",
			},
			[]string{"status"},
		),
		containerStopsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "container_stops_total",
				Help:      "Total number of container stops",
			},
			[]string{"status"},
		),
		containerFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "container_failures_total",
				Help:      "Total number of container failures",
			},
			[]string{"operation"},
		),

		// Database Metrics
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"query_type", "status"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"query_type"},
		),
		dbConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Number of open database connections",
			},
		),
		dbConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Number of idle database connections",
			},
		),
	}
}

// HTTP Metrics Methods
func (m *RDSMetrics) RecordHTTPRequest(method, endpoint, status string) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

func (m *RDSMetrics) RecordHTTPDuration(method, endpoint string, duration time.Duration) {
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// Instance Metrics Methods
func (m *RDSMetrics) SetInstancesTotal(count float64) {
	m.instancesTotal.Set(count)
}

func (m *RDSMetrics) RecordInstanceOperation(operation, status string) {
	m.instanceOperationsTotal.WithLabelValues(operation, status).Inc()
}

func (m *RDSMetrics) RecordInstanceOperationDuration(operation string, duration time.Duration) {
	m.instanceOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// Container Metrics Methods
func (m *RDSMetrics) RecordContainerStart(status string) {
	m.containerStartsTotal.WithLabelValues(status).Inc()
}

func (m *RDSMetrics) RecordContainerStop(status string) {
	m.containerStopsTotal.WithLabelValues(status).Inc()
}

func (m *RDSMetrics) RecordContainerFailure(operation string) {
	m.containerFailuresTotal.WithLabelValues(operation).Inc()
}

// Database Metrics Methods
func (m *RDSMetrics) RecordDBQuery(queryType, status string) {
	m.dbQueriesTotal.WithLabelValues(queryType, status).Inc()
}

func (m *RDSMetrics) RecordDBQueryDuration(queryType string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func (m *RDSMetrics) SetDBConnectionsOpen(count int) {
	m.dbConnectionsOpen.Set(float64(count))
}

func (m *RDSMetrics) SetDBConnectionsIdle(count int) {
	m.dbConnectionsIdle.Set(float64(count))
}

// Handler returns the Prometheus HTTP handler for /metrics endpoint
func (m *RDSMetrics) Handler() http.Handler {
	return promhttp.Handler()
}
