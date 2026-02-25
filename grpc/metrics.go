package grpc

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsManager manages Prometheus metrics for the gRPC server
type MetricsManager struct {
	registry  *prometheus.Registry
	startTime time.Time

	// gRPC Metrics
	grpcRequestsTotal     *prometheus.CounterVec
	grpcRequestDuration   *prometheus.HistogramVec
	grpcRequestSize       *prometheus.HistogramVec
	grpcResponseSize      *prometheus.HistogramVec
	grpcActiveConnections prometheus.Gauge

	// HTTP Gateway Metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpActiveRequests  prometheus.Gauge

	// Server Metrics
	serverUptime    prometheus.Gauge
	serverStartTime prometheus.Gauge
}

// NewMetricsManager creates a new metrics manager
func NewMetricsManager(namespace string) *MetricsManager {
	if namespace == "" {
		namespace = "grpc_server"
	}

	registry := prometheus.NewRegistry()

	mm := &MetricsManager{
		registry:  registry,
		startTime: time.Now(),

		// gRPC Metrics
		grpcRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "grpc",
				Name:      "requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "status_code"},
		),

		grpcRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "grpc",
				Name:      "request_duration_seconds",
				Help:      "Duration of gRPC requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method"},
		),

		grpcRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "grpc",
				Name:      "request_size_bytes",
				Help:      "Size of gRPC request payloads in bytes",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"method"},
		),

		grpcResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "grpc",
				Name:      "response_size_bytes",
				Help:      "Size of gRPC response payloads in bytes",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"method"},
		),

		grpcActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "grpc",
				Name:      "active_connections",
				Help:      "Number of active gRPC connections",
			},
		),

		// HTTP Gateway Metrics
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),

		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_size_bytes",
				Help:      "Size of HTTP request payloads in bytes",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"method", "path"},
		),

		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "Size of HTTP response payloads in bytes",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"method", "path"},
		),

		httpActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "active_requests",
				Help:      "Number of active HTTP requests",
			},
		),

		// Server Metrics
		serverUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "uptime_seconds",
				Help:      "Server uptime in seconds",
			},
		),

		serverStartTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "start_time_seconds",
				Help:      "Server start time as Unix timestamp",
			},
		),
	}

	// Register all metrics
	mm.registerMetrics()

	// Set initial values
	mm.serverStartTime.Set(float64(mm.startTime.Unix()))

	return mm
}

// registerMetrics registers all metrics with the registry
func (mm *MetricsManager) registerMetrics() {
	mm.registry.MustRegister(
		// gRPC metrics
		mm.grpcRequestsTotal,
		mm.grpcRequestDuration,
		mm.grpcRequestSize,
		mm.grpcResponseSize,
		mm.grpcActiveConnections,

		// HTTP metrics
		mm.httpRequestsTotal,
		mm.httpRequestDuration,
		mm.httpRequestSize,
		mm.httpResponseSize,
		mm.httpActiveRequests,

		// Server metrics
		mm.serverUptime,
		mm.serverStartTime,

		// Standard process metrics
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
}

// GetRegistry returns the prometheus registry
func (mm *MetricsManager) GetRegistry() *prometheus.Registry {
	return mm.registry
}

// CreateMetricsHandler creates an HTTP handler for the metrics endpoint
func (mm *MetricsManager) CreateMetricsHandler() http.Handler {
	return promhttp.HandlerFor(mm.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          mm.registry,
	})
}

// RecordGRPCRequest records metrics for a gRPC request
func (mm *MetricsManager) RecordGRPCRequest(method string, statusCode string, duration time.Duration, requestSize, responseSize int) {
	mm.grpcRequestsTotal.WithLabelValues(method, statusCode).Inc()
	mm.grpcRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
	if requestSize > 0 {
		mm.grpcRequestSize.WithLabelValues(method).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		mm.grpcResponseSize.WithLabelValues(method).Observe(float64(responseSize))
	}
}

// RecordHTTPRequest records metrics for an HTTP request
func (mm *MetricsManager) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, requestSize, responseSize int) {
	statusStr := strconv.Itoa(statusCode)
	mm.httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	mm.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	if requestSize > 0 {
		mm.httpRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		mm.httpResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
	}
}

// IncrementGRPCConnections increments the active gRPC connections counter
func (mm *MetricsManager) IncrementGRPCConnections() {
	mm.grpcActiveConnections.Inc()
}

// DecrementGRPCConnections decrements the active gRPC connections counter
func (mm *MetricsManager) DecrementGRPCConnections() {
	mm.grpcActiveConnections.Dec()
}

// IncrementHTTPRequests increments the active HTTP requests counter
func (mm *MetricsManager) IncrementHTTPRequests() {
	mm.httpActiveRequests.Inc()
}

// DecrementHTTPRequests decrements the active HTTP requests counter
func (mm *MetricsManager) DecrementHTTPRequests() {
	mm.httpActiveRequests.Dec()
}

// UpdateUptime updates the server uptime metric
func (mm *MetricsManager) UpdateUptime() {
	uptime := time.Since(mm.startTime).Seconds()
	mm.serverUptime.Set(uptime)
}

// HTTPMetricsMiddleware creates middleware for recording HTTP metrics
func (mm *MetricsManager) HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mm.IncrementHTTPRequests()
		defer mm.DecrementHTTPRequests()

		// Wrap the response writer to capture status and size
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		requestSize := int(r.ContentLength)
		responseSize := wrapped.size

		mm.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration, requestSize, responseSize)
	})
}

// responseWriter wraps http.ResponseWriter to capture response metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Flush implements http.Flusher so streaming responses work through the metrics middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// EchoMetricsMiddleware returns Echo middleware that records HTTP metrics
func (m *MetricsManager) EchoMetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Get path for metrics (we'll need it for both start and end metrics)
			req := c.Request()
			path := c.Path() // Echo provides clean path template
			if path == "" {
				path = c.Request().URL.Path
			}

			// Record request size if available
			if req.ContentLength > 0 {
				m.httpRequestSize.WithLabelValues(req.Method, path).Observe(float64(req.ContentLength))
			}

			// Increment active requests
			m.httpActiveRequests.Inc()

			// Process the request
			err := next(c)

			// Record metrics after request completion
			duration := time.Since(start).Seconds()

			// Decrement active requests
			m.httpActiveRequests.Dec()

			// Get response details
			res := c.Response()
			status := strconv.Itoa(res.Status)

			// Record all metrics with correct label sets
			m.httpRequestsTotal.WithLabelValues(req.Method, path, status).Inc()             // 3 labels: method, path, status_code
			m.httpRequestDuration.WithLabelValues(req.Method, path).Observe(duration)       // 2 labels: method, path
			m.httpResponseSize.WithLabelValues(req.Method, path).Observe(float64(res.Size)) // 2 labels: method, path

			return err
		}
	}
}

// RegisterEchoMetrics registers the Prometheus metrics endpoint with Echo
// using the MetricsManager's custom registry (not the global default).
func (m *MetricsManager) RegisterEchoMetrics(e *echo.Echo, path string) {
	e.GET(path, echo.WrapHandler(m.CreateMetricsHandler()))
}

// EchoUptimeMiddleware can be used to track server uptime via Echo
func (m *MetricsManager) EchoUptimeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Update uptime metric on each request
			m.UpdateUptime()
			return next(c)
		}
	}
}
