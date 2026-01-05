package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusAdapter provides Prometheus metrics collection
type PrometheusAdapter struct {
	// HTTP Metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// File Operations Metrics
	fileUploadsTotal    *prometheus.CounterVec
	fileDownloadsTotal  *prometheus.CounterVec
	fileDeletesTotal    *prometheus.CounterVec
	fileOperationSize   *prometheus.HistogramVec
	fileOperationErrors *prometheus.CounterVec

	// Bucket Metrics
	bucketsTotal     prometheus.Gauge
	bucketOperations *prometheus.CounterVec

	// Storage Metrics
	storageUsedBytes  *prometheus.GaugeVec
	storageObjects    *prometheus.GaugeVec
	storageOperations *prometheus.CounterVec

	// Cache Metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec
	cacheErrors *prometheus.CounterVec

	// Database Metrics
	dbQueriesTotal    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec
	dbConnectionsOpen prometheus.Gauge
	dbConnectionsIdle prometheus.Gauge

	// Multipart Upload Metrics
	multipartUploadsActive   prometheus.Gauge
	multipartUploadsTotal    *prometheus.CounterVec
	multipartPartsTotal      *prometheus.CounterVec
	multipartUploadsDuration *prometheus.HistogramVec
}

// NewPrometheusAdapter creates a new Prometheus metrics adapter
func NewPrometheusAdapter(namespace string) *PrometheusAdapter {
	if namespace == "" {
		namespace = "s3_service"
	}

	adapter := &PrometheusAdapter{
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
		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		// File Operations
		fileUploadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_uploads_total",
				Help:      "Total number of file uploads",
			},
			[]string{"bucket_id", "status"},
		),
		fileDownloadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_downloads_total",
				Help:      "Total number of file downloads",
			},
			[]string{"bucket_id", "status"},
		),
		fileDeletesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_deletes_total",
				Help:      "Total number of file deletions",
			},
			[]string{"bucket_id", "status"},
		),
		fileOperationSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "file_operation_size_bytes",
				Help:      "Size of file operations in bytes",
				Buckets:   prometheus.ExponentialBuckets(1024, 10, 8), // 1KB to ~10GB
			},
			[]string{"operation", "bucket_id"},
		),
		fileOperationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_operation_errors_total",
				Help:      "Total number of file operation errors",
			},
			[]string{"operation", "error_type"},
		),

		// Bucket Metrics
		bucketsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "buckets_total",
				Help:      "Total number of buckets",
			},
		),
		bucketOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bucket_operations_total",
				Help:      "Total number of bucket operations",
			},
			[]string{"operation", "status"},
		),

		// Storage Metrics
		storageUsedBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "storage_used_bytes",
				Help:      "Total storage used in bytes",
			},
			[]string{"bucket_id"},
		),
		storageObjects: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "storage_objects_total",
				Help:      "Total number of stored objects",
			},
			[]string{"bucket_id"},
		),
		storageOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "storage_operations_total",
				Help:      "Total number of storage operations",
			},
			[]string{"operation", "status"},
		),

		// Cache Metrics
		cacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		cacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		cacheErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_errors_total",
				Help:      "Total number of cache errors",
			},
			[]string{"cache_type", "error_type"},
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

		// Multipart Upload Metrics
		multipartUploadsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "multipart_uploads_active",
				Help:      "Number of active multipart uploads",
			},
		),
		multipartUploadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "multipart_uploads_total",
				Help:      "Total number of multipart uploads",
			},
			[]string{"status"},
		),
		multipartPartsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "multipart_parts_total",
				Help:      "Total number of multipart parts uploaded",
			},
			[]string{"status"},
		),
		multipartUploadsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "multipart_upload_duration_seconds",
				Help:      "Multipart upload duration in seconds",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"status"},
		),
	}

	return adapter
}

// HTTP Metrics Methods
func (p *PrometheusAdapter) RecordHTTPRequest(method, endpoint, status string) {
	p.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

func (p *PrometheusAdapter) RecordHTTPDuration(method, endpoint string, duration time.Duration) {
	p.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (p *PrometheusAdapter) RecordHTTPRequestSize(method, endpoint string, size int64) {
	p.httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

func (p *PrometheusAdapter) RecordHTTPResponseSize(method, endpoint string, size int64) {
	p.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// File Operations Methods
func (p *PrometheusAdapter) RecordFileUpload(bucketID, status string) {
	p.fileUploadsTotal.WithLabelValues(bucketID, status).Inc()
}

func (p *PrometheusAdapter) RecordFileDownload(bucketID, status string) {
	p.fileDownloadsTotal.WithLabelValues(bucketID, status).Inc()
}

func (p *PrometheusAdapter) RecordFileDelete(bucketID, status string) {
	p.fileDeletesTotal.WithLabelValues(bucketID, status).Inc()
}

func (p *PrometheusAdapter) RecordFileOperationSize(operation, bucketID string, size int64) {
	p.fileOperationSize.WithLabelValues(operation, bucketID).Observe(float64(size))
}

func (p *PrometheusAdapter) RecordFileOperationError(operation, errorType string) {
	p.fileOperationErrors.WithLabelValues(operation, errorType).Inc()
}

// Bucket Methods
func (p *PrometheusAdapter) SetBucketsTotal(count float64) {
	p.bucketsTotal.Set(count)
}

func (p *PrometheusAdapter) RecordBucketOperation(operation, status string) {
	p.bucketOperations.WithLabelValues(operation, status).Inc()
}

// Storage Methods
func (p *PrometheusAdapter) SetStorageUsed(bucketID string, bytes int64) {
	p.storageUsedBytes.WithLabelValues(bucketID).Set(float64(bytes))
}

func (p *PrometheusAdapter) SetStorageObjects(bucketID string, count int64) {
	p.storageObjects.WithLabelValues(bucketID).Set(float64(count))
}

func (p *PrometheusAdapter) RecordStorageOperation(operation, status string) {
	p.storageOperations.WithLabelValues(operation, status).Inc()
}

// Cache Methods
func (p *PrometheusAdapter) RecordCacheHit(cacheType string) {
	p.cacheHits.WithLabelValues(cacheType).Inc()
}

func (p *PrometheusAdapter) RecordCacheMiss(cacheType string) {
	p.cacheMisses.WithLabelValues(cacheType).Inc()
}

func (p *PrometheusAdapter) RecordCacheError(cacheType, errorType string) {
	p.cacheErrors.WithLabelValues(cacheType, errorType).Inc()
}

// Database Methods
func (p *PrometheusAdapter) RecordDBQuery(queryType, status string) {
	p.dbQueriesTotal.WithLabelValues(queryType, status).Inc()
}

func (p *PrometheusAdapter) RecordDBQueryDuration(queryType string, duration time.Duration) {
	p.dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func (p *PrometheusAdapter) SetDBConnectionsOpen(count int) {
	p.dbConnectionsOpen.Set(float64(count))
}

func (p *PrometheusAdapter) SetDBConnectionsIdle(count int) {
	p.dbConnectionsIdle.Set(float64(count))
}

// Multipart Upload Methods
func (p *PrometheusAdapter) SetMultipartUploadsActive(count int) {
	p.multipartUploadsActive.Set(float64(count))
}

func (p *PrometheusAdapter) RecordMultipartUpload(status string) {
	p.multipartUploadsTotal.WithLabelValues(status).Inc()
}

func (p *PrometheusAdapter) RecordMultipartPart(status string) {
	p.multipartPartsTotal.WithLabelValues(status).Inc()
}

func (p *PrometheusAdapter) RecordMultipartDuration(status string, duration time.Duration) {
	p.multipartUploadsDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// Handler returns the Prometheus HTTP handler for /metrics endpoint
func (p *PrometheusAdapter) Handler() http.Handler {
	return promhttp.Handler()
}
