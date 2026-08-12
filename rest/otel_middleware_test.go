package rest

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/jasoet/pkg/v3/otel"
)

// newRecordingMeter returns a real SDK meter provider backed by an in-memory
// ManualReader (never a noop provider) so metrics can be asserted.
func newRecordingMeter() (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	return mp, reader
}

// sumCounter collects a named Int64 counter metric and returns its total.
func sumCounter(t *testing.T, reader sdkmetric.Reader, name string) (int64, bool) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	var total int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			found = true
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "expected Sum[int64] for %s, got %T", name, m.Data)
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
		}
	}
	return total, found
}

// logRecorder is an in-memory sdklog.Exporter capturing emitted severities and
// bodies so OTel logging middleware behavior can be asserted with a real
// LoggerProvider (never a noop).
type logRecorder struct {
	mu         sync.Mutex
	severities []otellog.Severity
	bodies     []string
}

func (r *logRecorder) Export(_ context.Context, records []sdklog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range records {
		r.severities = append(r.severities, records[i].Severity())
		r.bodies = append(r.bodies, records[i].Body().AsString())
	}
	return nil
}
func (r *logRecorder) Shutdown(context.Context) error   { return nil }
func (r *logRecorder) ForceFlush(context.Context) error { return nil }

func (r *logRecorder) lastSeverity() otellog.Severity {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.severities) == 0 {
		return otellog.SeverityUndefined
	}
	return r.severities[len(r.severities)-1]
}

func newRecordingLogger() (*sdklog.LoggerProvider, *logRecorder) {
	rec := &logRecorder{}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(rec)))
	return lp, rec
}

// ============================================================================
// OTelTracingMiddleware Tests
// ============================================================================

func TestNewOTelTracingMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		assert.Nil(t, NewOTelTracingMiddleware(nil))
	})

	t.Run("returns nil when tracing is not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service") // no tracer provider
		assert.Nil(t, NewOTelTracingMiddleware(cfg))
	})

	t.Run("creates middleware when tracing is enabled", func(t *testing.T) {
		tp, _ := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))

		middleware := NewOTelTracingMiddleware(cfg)
		require.NotNil(t, middleware)
		assert.Same(t, cfg, middleware.cfg)
		assert.NotNil(t, middleware.tracer)
	})
}

func TestOTelTracingMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelTracingMiddleware
		ctx := context.Background()
		result := middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", map[string]string{})
		assert.Equal(t, ctx, result)
	})

	t.Run("starts span and injects real traceparent into headers", func(t *testing.T) {
		tp, _ := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := context.Background()
		headers := map[string]string{}
		resultCtx := middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com/api", "test body", headers)

		assert.NotEqual(t, ctx, resultCtx, "context should carry the span")
		require.NotNil(t, spanFromContext(resultCtx))

		// With a real SDK tracer the propagator injects a valid traceparent.
		assert.NotEmpty(t, headers["traceparent"], "real tracer must inject a traceparent")
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		tp, _ := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
			ctx := context.Background()
			resultCtx := middleware.BeforeRequest(ctx, method, "http://example.com", "", map[string]string{})
			assert.NotEqual(t, ctx, resultCtx, "method %s", method)
		}
	})
}

func TestOTelTracingMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelTracingMiddleware
		require.NotPanics(t, func() {
			middleware.AfterRequest(context.Background(), RequestInfo{Method: http.MethodGet, StatusCode: 200})
		})
	})

	t.Run("does nothing when span not in context", func(t *testing.T) {
		tp, _ := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		require.NotPanics(t, func() {
			middleware.AfterRequest(context.Background(), RequestInfo{Method: http.MethodGet, StatusCode: 200})
		})
	})

	t.Run("records successful response status Ok", func(t *testing.T) {
		tp, sr := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := middleware.BeforeRequest(context.Background(), http.MethodGet, "http://example.com", "", map[string]string{})
		middleware.AfterRequest(ctx, RequestInfo{
			Method: http.MethodGet, URL: "http://example.com", StatusCode: 200,
			ResponseSize: 13, Duration: 100 * time.Millisecond,
		})

		spans := sr.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, codes.Ok, spans[0].Status().Code)
	})

	t.Run("records error status when request failed", func(t *testing.T) {
		tp, sr := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := middleware.BeforeRequest(context.Background(), http.MethodPost, "http://example.com", "", map[string]string{})
		middleware.AfterRequest(ctx, RequestInfo{
			Method: http.MethodPost, URL: "http://example.com", StatusCode: 500,
			Error: errors.New("server error"), Duration: 50 * time.Millisecond,
		})

		spans := sr.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, codes.Error, spans[0].Status().Code)
	})

	t.Run("records error status for 4xx", func(t *testing.T) {
		tp, sr := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		middleware := NewOTelTracingMiddleware(cfg)

		ctx := middleware.BeforeRequest(context.Background(), http.MethodGet, "http://example.com", "", map[string]string{})
		middleware.AfterRequest(ctx, RequestInfo{
			Method: http.MethodGet, URL: "http://example.com", StatusCode: 404,
			Duration: 30 * time.Millisecond,
		})

		spans := sr.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, codes.Error, spans[0].Status().Code)
	})
}

// ============================================================================
// OTelMetricsMiddleware Tests
// ============================================================================

func TestNewOTelMetricsMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		assert.Nil(t, NewOTelMetricsMiddleware(nil))
	})

	t.Run("returns nil when metrics are not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service") // no meter provider
		assert.Nil(t, NewOTelMetricsMiddleware(cfg))
	})

	t.Run("creates middleware when metrics are enabled", func(t *testing.T) {
		mp, _ := newRecordingMeter()
		defer func() { _ = mp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))

		middleware := NewOTelMetricsMiddleware(cfg)
		require.NotNil(t, middleware)
		assert.Same(t, cfg, middleware.cfg)
		assert.NotNil(t, middleware.requestCounter)
		assert.NotNil(t, middleware.requestDuration)
		assert.NotNil(t, middleware.requestSize)
		assert.NotNil(t, middleware.responseSize)
		assert.NotNil(t, middleware.retryCounter)
	})
}

func TestOTelMetricsMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		ctx := context.Background()
		assert.Equal(t, ctx, middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", map[string]string{}))
	})

	t.Run("returns context unchanged with and without body", func(t *testing.T) {
		mp, _ := newRecordingMeter()
		defer func() { _ = mp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))
		middleware := NewOTelMetricsMiddleware(cfg)

		ctx := context.Background()
		assert.Equal(t, ctx, middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com", "body", map[string]string{}))
		assert.Equal(t, ctx, middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", map[string]string{}))
	})
}

func TestOTelMetricsMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		require.NotPanics(t, func() {
			middleware.AfterRequest(context.Background(), RequestInfo{Method: http.MethodGet, StatusCode: 200})
		})
	})

	t.Run("increments request counter", func(t *testing.T) {
		mp, reader := newRecordingMeter()
		defer func() { _ = mp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))
		middleware := NewOTelMetricsMiddleware(cfg)

		for _, code := range []int{200, 201, 400, 404, 500} {
			middleware.AfterRequest(context.Background(), RequestInfo{
				Method: http.MethodPost, URL: "http://example.com", StatusCode: code,
				Duration: 50 * time.Millisecond,
			})
		}

		total, found := sumCounter(t, reader, "http.client.request.count")
		require.True(t, found)
		assert.Equal(t, int64(5), total)
	})
}

func TestOTelMetricsMiddleware_recordRetry(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelMetricsMiddleware
		require.NotPanics(t, func() { middleware.recordRetry(context.Background(), http.MethodGet, 1) })
	})

	t.Run("records retry attempts", func(t *testing.T) {
		mp, reader := newRecordingMeter()
		defer func() { _ = mp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))
		middleware := NewOTelMetricsMiddleware(cfg)

		middleware.recordRetry(context.Background(), http.MethodPost, 1)
		middleware.recordRetry(context.Background(), http.MethodPost, 2)
		middleware.recordRetry(context.Background(), http.MethodPost, 3)

		total, found := sumCounter(t, reader, "http.client.retry.count")
		require.True(t, found)
		assert.Equal(t, int64(3), total)
	})
}

// ============================================================================
// OTelLoggingMiddleware Tests
// ============================================================================

func TestNewOTelLoggingMiddleware(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		assert.Nil(t, NewOTelLoggingMiddleware(nil))
	})

	t.Run("returns nil when logging is not enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service", otel.WithoutLogging())
		assert.Nil(t, NewOTelLoggingMiddleware(cfg))
	})

	t.Run("creates middleware when logging is enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service") // default has logging enabled
		middleware := NewOTelLoggingMiddleware(cfg)
		require.NotNil(t, middleware)
		assert.Same(t, cfg, middleware.cfg)
		assert.NotNil(t, middleware.logger)
	})
}

func TestOTelLoggingMiddleware_BeforeRequest(t *testing.T) {
	t.Run("returns context unchanged when middleware is nil", func(t *testing.T) {
		var middleware *OTelLoggingMiddleware
		ctx := context.Background()
		assert.Equal(t, ctx, middleware.BeforeRequest(ctx, http.MethodGet, "http://example.com", "", map[string]string{}))
	})

	t.Run("returns context unchanged", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		middleware := NewOTelLoggingMiddleware(cfg)
		ctx := context.Background()
		assert.Equal(t, ctx, middleware.BeforeRequest(ctx, http.MethodPost, "http://example.com", "body", map[string]string{}))
	})
}

func TestOTelLoggingMiddleware_AfterRequest(t *testing.T) {
	t.Run("does nothing when middleware is nil", func(t *testing.T) {
		var middleware *OTelLoggingMiddleware
		require.NotPanics(t, func() {
			middleware.AfterRequest(context.Background(), RequestInfo{Method: http.MethodGet, StatusCode: 200})
		})
	})

	// Severity mapping asserted against a real in-memory log exporter.
	cases := []struct {
		name     string
		info     RequestInfo
		expected otellog.Severity
	}{
		{
			name:     "2xx maps to Info",
			info:     RequestInfo{Method: http.MethodGet, URL: "http://example.com", StatusCode: 200, StartTime: time.Now()},
			expected: otellog.SeverityInfo,
		},
		{
			name:     "4xx maps to Warn",
			info:     RequestInfo{Method: http.MethodGet, URL: "http://example.com", StatusCode: 404, StartTime: time.Now()},
			expected: otellog.SeverityWarn,
		},
		{
			name:     "5xx maps to Error",
			info:     RequestInfo{Method: http.MethodPost, URL: "http://example.com", StatusCode: 500, StartTime: time.Now()},
			expected: otellog.SeverityError,
		},
		{
			name:     "transport error maps to Error",
			info:     RequestInfo{Method: http.MethodGet, URL: "http://example.com", StatusCode: 0, Error: errors.New("timeout"), StartTime: time.Now()},
			expected: otellog.SeverityError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lp, rec := newRecordingLogger()
			defer func() { _ = lp.Shutdown(context.Background()) }()
			cfg := otel.NewConfig("test-service", otel.WithLoggerProvider(lp))
			middleware := NewOTelLoggingMiddleware(cfg)
			require.NotNil(t, middleware)

			middleware.AfterRequest(context.Background(), tc.info)
			assert.Equal(t, tc.expected, rec.lastSeverity())
		})
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestContextWithSpan(t *testing.T) {
	t.Run("stores and retrieves span from context", func(t *testing.T) {
		tp, _ := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		ctx, span := tp.Tracer("test").Start(context.Background(), "test-span")
		defer span.End()

		ctxWithSpan := contextWithSpan(ctx, span)
		assert.NotNil(t, spanFromContext(ctxWithSpan))
	})

	t.Run("returns nil when span not in context", func(t *testing.T) {
		assert.Nil(t, spanFromContext(context.Background()))
	})
}

func TestSanitizeURL(t *testing.T) {
	assert.Equal(t, "http://example.com/path", sanitizeURL("http://user:secret@example.com/path"))
	assert.Equal(t, "https://api.example.com/v1", sanitizeURL("https://api.example.com/v1"))
	// Unparseable / bare paths are returned unchanged.
	assert.Equal(t, "/relative/path", sanitizeURL("/relative/path"))
}

func TestDurationMillis(t *testing.T) {
	assert.InDelta(t, 0.5, durationMillis(500*time.Microsecond), 1e-9)
	assert.InDelta(t, 1500.0, durationMillis(1500*time.Millisecond), 1e-9)
	assert.InDelta(t, 0.0, durationMillis(0), 1e-9)
}
