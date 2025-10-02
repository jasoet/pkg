package rest

import (
	"context"
	"fmt"

	pkgotel "github.com/jasoet/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

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

	// Start a new span for the HTTP request
	ctx, span := m.tracer.Start(ctx, fmt.Sprintf("%s %s", method, url),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(method),
			semconv.URLFullKey.String(url),
			semconv.HTTPRequestBodySizeKey.Int(len(body)),
		),
	)

	// Inject trace context into HTTP headers for distributed tracing
	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, propagation.MapCarrier(headers))

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

	// Record response attributes
	span.SetAttributes(
		semconv.HTTPResponseStatusCodeKey.Int(info.StatusCode),
		semconv.HTTPResponseBodySizeKey.Int(len(info.Response)),
		attribute.Int64("http.request.duration_ms", info.Duration.Milliseconds()),
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

	// Create metrics instruments
	// Note: errors are intentionally ignored as they only occur with nil meter (checked by GetMeter)
	requestCounter, _ := meter.Int64Counter( //nolint:errcheck
		"http.client.request.count",
		metric.WithDescription("Total number of HTTP client requests"),
		metric.WithUnit("{request}"),
	)

	requestDuration, _ := meter.Float64Histogram( //nolint:errcheck
		"http.client.request.duration",
		metric.WithDescription("HTTP client request duration"),
		metric.WithUnit("ms"),
	)

	requestSize, _ := meter.Int64Histogram( //nolint:errcheck
		"http.client.request.size",
		metric.WithDescription("HTTP client request body size"),
		metric.WithUnit("By"),
	)

	responseSize, _ := meter.Int64Histogram( //nolint:errcheck
		"http.client.response.size",
		metric.WithDescription("HTTP client response body size"),
		metric.WithUnit("By"),
	)

	retryCounter, _ := meter.Int64Counter( //nolint:errcheck
		"http.client.retry.count",
		metric.WithDescription("Total number of HTTP client retries"),
		metric.WithUnit("{retry}"),
	)

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

	// Record metrics
	m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.requestDuration.Record(ctx, float64(info.Duration.Milliseconds()), metric.WithAttributes(attrs...))

	if len(info.Response) > 0 {
		m.responseSize.Record(ctx, int64(len(info.Response)), metric.WithAttributes(attrs...))
	}
}

// RecordRetry records a retry attempt (to be called by retry logic)
func (m *OTelMetricsMiddleware) RecordRetry(ctx context.Context, method string, attempt int) {
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
		otellog.String("http.url", info.URL),
		otellog.Int("http.response.status_code", info.StatusCode),
		otellog.Int64("http.request.duration_ms", info.Duration.Milliseconds()),
		otellog.Int("http.request.body.size", len(info.Body)),
		otellog.Int("http.response.body.size", len(info.Response)),
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
