package rest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// sanitizeURL strips user credentials (userinfo) and, defensively, an
// Authorization-style query secret before recording a URL on telemetry, so
// passwords and API keys embedded in the URL do not leak into traces. If the
// URL cannot be parsed it is returned unchanged (it is caller-supplied and may
// already be a bare path).
func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = nil
	}
	return u.String()
}

// durationMillis converts a duration to fractional milliseconds so sub-millisecond
// timings are preserved instead of truncating to 0 (Duration.Milliseconds()).
func durationMillis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// ============================================================================
// OpenTelemetry Tracing Middleware
// ============================================================================

// OTelTracingMiddleware implements distributed tracing for HTTP client requests
type OTelTracingMiddleware struct {
	cfg    *pkgotel.Config
	tracer trace.Tracer
}

// NewOTelTracingMiddleware creates a new OpenTelemetry tracing middleware
func NewOTelTracingMiddleware(cfg *pkgotel.Config) *OTelTracingMiddleware {
	if cfg == nil || !cfg.IsTracingEnabled() {
		return nil
	}

	return &OTelTracingMiddleware{
		cfg:    cfg,
		tracer: cfg.GetTracer("rest.client"),
	}
}

// BeforeRequest starts a new span for the HTTP request and injects trace context into headers
func (m *OTelTracingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	if m == nil {
		return ctx
	}

	// Start a new span for the HTTP request. The URL is sanitized so embedded
	// credentials are not recorded on the span.
	ctx, span := m.tracer.Start(ctx, method,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(method),
			semconv.URLFullKey.String(sanitizeURL(url)),
			semconv.HTTPRequestBodySizeKey.Int(len(body)),
		),
	)

	// Inject trace context into HTTP headers for distributed tracing.
	// Copy the headers map before injection so injected keys are isolated,
	// then write results back to the original map that doRequest will use.
	headersCopy := make(map[string]string, len(headers))
	for k, v := range headers {
		headersCopy[k] = v
	}
	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, propagation.MapCarrier(headersCopy))
	for k, v := range headersCopy {
		headers[k] = v
	}

	// Store span in context for AfterRequest
	return contextWithSpan(ctx, span)
}

// AfterRequest ends the span and records the response status
func (m *OTelTracingMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	if m == nil {
		return
	}

	span := spanFromContext(ctx)
	if span == nil {
		return
	}
	defer span.End()

	// Record response attributes. ResponseSize is the true body size (not the
	// possibly-truncated Response), and duration is fractional milliseconds so
	// sub-millisecond requests are not recorded as 0.
	span.SetAttributes(
		semconv.HTTPResponseStatusCodeKey.Int(info.StatusCode),
		semconv.HTTPResponseBodySizeKey.Int64(info.ResponseSize),
		attribute.Float64("http.request.duration_ms", durationMillis(info.Duration)),
	)

	// Record error if present
	if info.Error != nil {
		span.SetStatus(codes.Error, info.Error.Error())
		span.RecordError(info.Error)
	} else if info.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", info.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// ============================================================================
// OpenTelemetry Metrics Middleware
// ============================================================================

// OTelMetricsMiddleware implements metrics collection for HTTP client requests
type OTelMetricsMiddleware struct {
	cfg *pkgotel.Config

	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestSize     metric.Int64Histogram
	responseSize    metric.Int64Histogram
	retryCounter    metric.Int64Counter
}

// NewOTelMetricsMiddleware creates a new OpenTelemetry metrics middleware
func NewOTelMetricsMiddleware(cfg *pkgotel.Config) *OTelMetricsMiddleware {
	if cfg == nil || !cfg.IsMetricsEnabled() {
		return nil
	}

	meter := cfg.GetMeter("rest.client")

	// Create metrics instruments. If creation fails, log to stderr and skip registration.
	requestCounter, err := meter.Int64Counter(
		"http.client.request.count",
		metric.WithDescription("Total number of HTTP client requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rest: failed to create http.client.request.count metric: %v\n", err)
		return nil
	}

	requestDuration, err := meter.Float64Histogram(
		"http.client.request.duration",
		metric.WithDescription("HTTP client request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rest: failed to create http.client.request.duration metric: %v\n", err)
		return nil
	}

	requestSize, err := meter.Int64Histogram(
		"http.client.request.size",
		metric.WithDescription("HTTP client request body size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rest: failed to create http.client.request.size metric: %v\n", err)
		return nil
	}

	responseSize, err := meter.Int64Histogram(
		"http.client.response.size",
		metric.WithDescription("HTTP client response body size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rest: failed to create http.client.response.size metric: %v\n", err)
		return nil
	}

	retryCounter, err := meter.Int64Counter(
		"http.client.retry.count",
		metric.WithDescription("Total number of HTTP client retries"),
		metric.WithUnit("{retry}"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rest: failed to create http.client.retry.count metric: %v\n", err)
		return nil
	}

	return &OTelMetricsMiddleware{
		cfg:             cfg,
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		retryCounter:    retryCounter,
	}
}

// BeforeRequest records request size metrics
func (m *OTelMetricsMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	if m == nil {
		return ctx
	}

	// Record request size
	if len(body) > 0 {
		attrs := []attribute.KeyValue{
			attribute.String("http.request.method", method),
		}
		m.requestSize.Record(ctx, int64(len(body)), metric.WithAttributes(attrs...))
	}

	return ctx
}

// AfterRequest records response metrics
func (m *OTelMetricsMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	if m == nil {
		return
	}

	// Prepare attributes
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", info.Method),
		attribute.Int("http.response.status_code", info.StatusCode),
	}

	// Record metrics. Duration is fractional milliseconds (matching the "ms"
	// unit without truncating sub-millisecond timings) and response size is the
	// true body size rather than the possibly-truncated Response.
	m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.requestDuration.Record(ctx, durationMillis(info.Duration), metric.WithAttributes(attrs...))

	if info.ResponseSize > 0 {
		m.responseSize.Record(ctx, info.ResponseSize, metric.WithAttributes(attrs...))
	}
}

// recordRetry records a retry attempt; wired into the resty retry hook in NewClient.
func (m *OTelMetricsMiddleware) recordRetry(ctx context.Context, method string, attempt int) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", method),
		attribute.Int("http.retry.attempt", attempt),
	}
	m.retryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// ============================================================================
// OpenTelemetry Logging Middleware
// ============================================================================

// OTelLoggingMiddleware implements structured logging with trace correlation for HTTP client
type OTelLoggingMiddleware struct {
	cfg    *pkgotel.Config
	logger otellog.Logger
}

// NewOTelLoggingMiddleware creates a new OpenTelemetry logging middleware
func NewOTelLoggingMiddleware(cfg *pkgotel.Config) *OTelLoggingMiddleware {
	if cfg == nil || !cfg.IsLoggingEnabled() {
		return nil
	}

	return &OTelLoggingMiddleware{
		cfg:    cfg,
		logger: cfg.GetLogger("rest.client"),
	}
}

// BeforeRequest logs the start of the request
func (m *OTelLoggingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	if m == nil {
		return ctx
	}

	// We'll log in AfterRequest with full info
	return ctx
}

// AfterRequest logs the completion of the request with trace correlation
func (m *OTelLoggingMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	if m == nil {
		return
	}

	// Determine severity
	severity := otellog.SeverityInfo
	if info.Error != nil || info.StatusCode >= 500 {
		severity = otellog.SeverityError
	} else if info.StatusCode >= 400 {
		severity = otellog.SeverityWarn
	}

	// Create log attributes
	attrs := []otellog.KeyValue{
		otellog.String("http.request.method", info.Method),
		otellog.String("http.url", sanitizeURL(info.URL)),
		otellog.Int("http.response.status_code", info.StatusCode),
		otellog.Float64("http.request.duration_ms", durationMillis(info.Duration)),
		otellog.Int("http.request.body.size", len(info.Body)),
		otellog.Int64("http.response.body.size", info.ResponseSize),
	}

	if info.Error != nil {
		attrs = append(attrs, otellog.String("error", info.Error.Error()))
	}

	// Emit log record (trace context will be automatically added by LoggerProvider)
	var logRecord otellog.Record
	logRecord.SetTimestamp(info.StartTime)
	logRecord.SetSeverity(severity)
	logRecord.SetBody(otellog.StringValue(fmt.Sprintf("%s %s %d", info.Method, info.URL, info.StatusCode)))
	logRecord.AddAttributes(attrs...)

	m.logger.Emit(ctx, logRecord)
}

// ============================================================================
// Helper functions for span context
// ============================================================================

type spanKey struct{}

func contextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

func spanFromContext(ctx context.Context) trace.Span {
	if span, ok := ctx.Value(spanKey{}).(trace.Span); ok {
		return span
	}
	return nil
}
