package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/metric/noop"
	noopt "go.opentelemetry.io/otel/trace/noop"
)

// ============================================================================
// OTelTracingMiddleware Tests
// ============================================================================

func TestNewOTelTracingMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		middleware := NewOTelTracingMiddleware(nil)
		if middleware != nil {
			t.Error("Expected nil middleware when config is nil")
		}
	})

	t.Run("returns nil when tracing is not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		// Don't set tracer provider, so tracing is disabled
		middleware := NewOTelTracingMiddleware(cfg)
		if middleware != nil {
			t.Error("Expected nil middleware when tracing is disabled")
		}
	})

	t.Run("creates middleware when tracing is enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())

		middleware := NewOTelTracingMiddleware(cfg)
		if middleware == nil {
			t.Error("Expected non-nil middleware when tracing is enabled")
		}
		if middleware.cfg != cfg {
			t.Error("Expected middleware to store config")
		}
		if middleware.tracer == nil {
			t.Error("Expected middleware to have a tracer")
		}
	})
}

func TestOTelTracingMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelTracingMiddleware
		ctx := context.Background()
		headers := make(map[string]string)

		resultCtx := middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})

	t.Run("starts span and injects trace context into headers", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)
		body := "test body"

		resultCtx := middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com/api", body, headers)

		// Verify context is different (span was added)
		if resultCtx == ctx {
			t.Error("Expected context to be modified with span")
		}

		// Verify span is stored in context
		span := spanFromContext(resultCtx)
		if span == nil {
			t.Error("Expected span to be stored in context")
		}

		// Verify trace context headers were injected
		// TraceContext propagator injects "traceparent" header
		if _, exists := headers["traceparent"]; !exists {
			t.Log("Note: traceparent header not injected (expected with noop tracer)")
		}
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
		for _, method := range methods {
			ctx := context.Background()
			headers := make(map[string]string)

			resultCtx := middleware.BeforeRequest(ctx, method, "http://example.com", "", headers)
			if resultCtx == ctx {
				t.Errorf("Expected context to be modified for method %s", method)
			}
		}
	})
}

func TestOTelTracingMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelTracingMiddleware
		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("does nothing when span not in context", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
		}

		// Should not panic even without span
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records successful response attributes", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)
		ctx = middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)

		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
			Response:   "response body",
			Duration:   100 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records error response attributes", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)
		ctx = middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com", "", headers)

		info := RequestInfo{
			Method:     http.MethodPost,
			URL:        "http://example.com",
			StatusCode: 500,
			Error:      errors.New("server error"),
			Duration:   50 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records 4xx client error status", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)
		ctx = middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)

		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 404,
			Duration:   30 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})
}

// ============================================================================
// OTelMetricsMiddleware Tests
// ============================================================================

func TestNewOTelMetricsMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		middleware := NewOTelMetricsMiddleware(nil)
		if middleware != nil {
			t.Error("Expected nil middleware when config is nil")
		}
	})

	t.Run("returns nil when metrics are not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		// Don't set meter provider, so metrics are disabled
		middleware := NewOTelMetricsMiddleware(cfg)
		if middleware != nil {
			t.Error("Expected nil middleware when metrics are disabled")
		}
	})

	t.Run("creates middleware when metrics are enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())

		middleware := NewOTelMetricsMiddleware(cfg)
		if middleware == nil {
			t.Error("Expected non-nil middleware when metrics are enabled")
		}
		if middleware.cfg != cfg {
			t.Error("Expected middleware to store config")
		}
		if middleware.requestCounter == nil {
			t.Error("Expected request counter to be initialized")
		}
		if middleware.requestDuration == nil {
			t.Error("Expected request duration to be initialized")
		}
		if middleware.requestSize == nil {
			t.Error("Expected request size to be initialized")
		}
		if middleware.responseSize == nil {
			t.Error("Expected response size to be initialized")
		}
		if middleware.retryCounter == nil {
			t.Error("Expected retry counter to be initialized")
		}
	})
}

func TestOTelMetricsMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		ctx := context.Background()
		headers := make(map[string]string)

		resultCtx := middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})

	t.Run("records request size when body is present", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)
		body := "test request body"

		resultCtx := middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com", body, headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})

	t.Run("does not record request size when body is empty", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)

		resultCtx := middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})
}

func TestOTelMetricsMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records request metrics", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
			Duration:   150 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records response size when response is present", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
			Response:   "test response body",
			Duration:   100 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("records metrics for different status codes", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		statusCodes := []int{200, 201, 400, 404, 500}
		for _, statusCode := range statusCodes {
			ctx := context.Background()
			info := RequestInfo{
				Method:     http.MethodPost,
				URL:        "http://example.com",
				StatusCode: statusCode,
				Duration:   50 * time.Millisecond,
			}

			middleware.AfterRequest(ctx, info)
		}
	})
}

func TestOTelMetricsMiddleware_RecordRetry(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		ctx := context.Background()

		// Should not panic
		middleware.RecordRetry(ctx, http.MethodGet, 1)
	})

	t.Run("records retry attempt", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noop.NewMeterProvider())
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()

		// Should complete without panic
		middleware.RecordRetry(ctx, http.MethodPost, 1)
		middleware.RecordRetry(ctx, http.MethodPost, 2)
		middleware.RecordRetry(ctx, http.MethodPost, 3)
	})
}

// ============================================================================
// OTelLoggingMiddleware Tests
// ============================================================================

func TestNewOTelLoggingMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		middleware := NewOTelLoggingMiddleware(nil)
		if middleware != nil {
			t.Error("Expected nil middleware when config is nil")
		}
	})

	t.Run("returns nil when logging is not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").WithoutLogging()
		middleware := NewOTelLoggingMiddleware(cfg)
		if middleware != nil {
			t.Error("Expected nil middleware when logging is disabled")
		}
	})

	t.Run("creates middleware when logging is enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		// Default config has logging enabled

		middleware := NewOTelLoggingMiddleware(cfg)
		if middleware == nil {
			t.Error("Expected non-nil middleware when logging is enabled")
		}
		if middleware.cfg != cfg {
			t.Error("Expected middleware to store config")
		}
		if middleware.logger == nil {
			t.Error("Expected middleware to have a logger")
		}
	})
}

func TestOTelLoggingMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelLoggingMiddleware
		ctx := context.Background()
		headers := make(map[string]string)

		resultCtx := middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})

	t.Run("returns context unchanged", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)

		ctx := context.Background()
		headers := make(map[string]string)

		resultCtx := middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com", "body", headers)
		if resultCtx != ctx {
			t.Error("Expected same context to be returned")
		}
	})
}

func TestOTelLoggingMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelLoggingMiddleware
		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("logs successful request with info severity", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 200,
			StartTime:  time.Now(),
			Duration:   100 * time.Millisecond,
			Body:       "request body",
			Response:   "response body",
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("logs 4xx request with warning severity", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 404,
			StartTime:  time.Now(),
			Duration:   50 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("logs 5xx request with error severity", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodPost,
			URL:        "http://example.com",
			StatusCode: 500,
			StartTime:  time.Now(),
			Duration:   200 * time.Millisecond,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})

	t.Run("logs request with error", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)

		ctx := context.Background()
		info := RequestInfo{
			Method:     http.MethodGet,
			URL:        "http://example.com",
			StatusCode: 0,
			Error:      errors.New("connection timeout"),
			StartTime:  time.Now(),
			Duration:   5 * time.Second,
		}

		// Should complete without panic
		middleware.AfterRequest(ctx, info)
	})
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestContextWithSpan(t *testing.T) {
	t.Run("stores and retrieves span from context", func(t *testing.T) {
		tracer := noopt.NewTracerProvider().Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		// Store span in context
		ctxWithSpan := contextWithSpan(ctx, span)

		// Retrieve span from context
		retrievedSpan := spanFromContext(ctxWithSpan)
		if retrievedSpan == nil {
			t.Error("Expected to retrieve span from context")
		}
		// Note: Don't compare spans directly as noop.Span is not comparable
	})

	t.Run("returns nil when span not in context", func(t *testing.T) {
		ctx := context.Background()

		retrievedSpan := spanFromContext(ctx)
		if retrievedSpan != nil {
			t.Error("Expected nil span when not in context")
		}
	})
}
