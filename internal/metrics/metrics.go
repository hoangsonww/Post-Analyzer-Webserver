package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Application metrics
	postsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "posts_total",
			Help: "Total number of posts in the system",
		},
	)

	postsFetched = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "posts_fetched_total",
			Help: "Total number of posts fetched from external API",
		},
	)

	postsAdded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "posts_added_total",
			Help: "Total number of posts added by users",
		},
	)

	analysisOperations = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analysis_operations_total",
			Help: "Total number of character analysis operations",
		},
	)

	analysisDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "analysis_duration_seconds",
			Help:    "Character analysis operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	dbOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)

	dbOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

// Middleware creates a middleware for recording HTTP metrics
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status and size
		rw := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Record request size
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(rw.bytesWritten))
	})
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := mrw.ResponseWriter.Write(b)
	mrw.bytesWritten += n
	return n, err
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordPostsTotal records the total number of posts
func RecordPostsTotal(count int) {
	postsTotal.Set(float64(count))
}

// RecordPostsFetched increments the posts fetched counter
func RecordPostsFetched(count int) {
	postsFetched.Add(float64(count))
}

// RecordPostAdded increments the posts added counter
func RecordPostAdded() {
	postsAdded.Inc()
}

// RecordAnalysisOperation records a character analysis operation
func RecordAnalysisOperation(duration time.Duration) {
	analysisOperations.Inc()
	analysisDuration.Observe(duration.Seconds())
}

// RecordDBOperation records a database operation
func RecordDBOperation(operation, status string, duration time.Duration) {
	dbOperations.WithLabelValues(operation, status).Inc()
	dbOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
