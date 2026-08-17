package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// serve issues an arbitrary request against the server's Echo instance via
// httptest and returns the recorder.
func serve(t *testing.T, srv *Server, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	srv.Echo().ServeHTTP(rec, req)
	return rec
}

// serveHealth issues a GET /health against the server's Echo instance via
// httptest and returns the recorder.
func serveHealth(t *testing.T, srv *Server) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	return rec
}

// spanAttribute returns the value of the named attribute on a span stub.
func spanAttribute(span tracetest.SpanStub, key string) (attribute.Value, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestOTelTracingMiddleware(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))

	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)

	serveHealth(t, srv)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one span for one request")
	span := spans[0]

	assert.Equal(t, "GET /health", span.Name)
	assert.Equal(t, "http.server", span.InstrumentationScope.Name)

	method, ok := spanAttribute(span, "http.request.method")
	require.True(t, ok, "missing http.request.method attribute")
	assert.Equal(t, "GET", method.AsString())

	fullURL, ok := spanAttribute(span, "url.full")
	require.True(t, ok, "missing url.full attribute")
	assert.Equal(t, "http://example.com/health", fullURL.AsString())

	statusCode, ok := spanAttribute(span, "http.response.status_code")
	require.True(t, ok, "missing http.response.status_code attribute")
	assert.Equal(t, int64(http.StatusOK), statusCode.AsInt64())

	route, ok := spanAttribute(span, "http.route")
	require.True(t, ok, "missing http.route attribute")
	assert.Equal(t, "/health", route.AsString())
}

func TestOTelTracingMiddlewareExtractsInboundTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))

	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)

	// A caller's W3C trace context. The server span must join this trace
	// instead of starting a new root, otherwise a client->server hop shows up
	// as two disconnected traces in the backend.
	const (
		callerTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		callerSpanID  = "00f067aa0ba902b7"
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("traceparent", "00-"+callerTraceID+"-"+callerSpanID+"-01")
	rec := httptest.NewRecorder()
	srv.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one span for one request")
	span := spans[0]

	assert.Equal(t, callerTraceID, span.SpanContext.TraceID().String(),
		"server span must continue the caller's trace")
	assert.Equal(t, callerSpanID, span.Parent.SpanID().String(),
		"server span must be a child of the caller's span")
	assert.True(t, span.Parent.IsRemote(), "parent must be marked remote")
}

func TestOTelTracingMiddlewareStartsRootWithoutInboundContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))

	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)

	serveHealth(t, srv)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.False(t, spans[0].Parent.IsValid(),
		"a request without traceparent must still start a root span")
}

// scopeMetricsByName collects from the reader and indexes instruments by name
// for the given instrumentation scope.
func scopeMetricsByName(t *testing.T, reader *sdkmetric.ManualReader, scopeName string) map[string]metricdata.Metrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		if sm.Scope.Name != scopeName {
			continue
		}
		metrics := make(map[string]metricdata.Metrics, len(sm.Metrics))
		for _, m := range sm.Metrics {
			metrics[m.Name] = m
		}
		return metrics
	}
	t.Fatalf("no metrics found for scope %q", scopeName)
	return nil
}

func TestOTelMetricsMiddleware(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		assert.NoError(t, mp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithMeterProvider(mp))

	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)

	serveHealth(t, srv)

	metrics := scopeMetricsByName(t, reader, "http.server")

	count, ok := metrics["http.server.request.count"]
	require.True(t, ok, "missing http.server.request.count counter")
	countSum, ok := count.Data.(metricdata.Sum[int64])
	require.True(t, ok, "http.server.request.count should be a Sum[int64]")
	require.Len(t, countSum.DataPoints, 1)
	assert.Equal(t, int64(1), countSum.DataPoints[0].Value)

	attrs := countSum.DataPoints[0].Attributes
	method, ok := attrs.Value("http.request.method")
	require.True(t, ok, "count datapoint missing http.request.method attribute")
	assert.Equal(t, "GET", method.AsString())
	statusCode, ok := attrs.Value("http.response.status_code")
	require.True(t, ok, "count datapoint missing http.response.status_code attribute")
	assert.Equal(t, int64(http.StatusOK), statusCode.AsInt64())

	duration, ok := metrics["http.server.request.duration"]
	require.True(t, ok, "missing http.server.request.duration histogram")
	durationHist, ok := duration.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "http.server.request.duration should be a Histogram[float64]")
	require.Len(t, durationHist.DataPoints, 1)
	assert.Equal(t, uint64(1), durationHist.DataPoints[0].Count)
	// A sub-millisecond health request must still record a non-zero duration;
	// whole-Milliseconds() truncation would floor it to 0.
	assert.Positive(t, durationHist.DataPoints[0].Sum, "sub-millisecond request must record non-zero duration")
}

// TestOTelMiddleware_ErrorStatus is the coverage that was missing: it exercises
// error-returning handlers and a 404 through the middleware and asserts that the
// span/metric status is the real HTTP code (not a premature 200) and that the
// span status is Error for 5xx only.
func TestOTelMiddleware_ErrorStatus(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
		assert.NoError(t, mp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service",
		pkgotel.WithTracerProvider(tp),
		pkgotel.WithMeterProvider(mp),
	)
	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)

	srv.Echo().GET("/boom", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError, "boom")
	})
	srv.Echo().GET("/plain-error", func(c echo.Context) error {
		return errors.New("plain failure")
	})
	srv.Echo().GET("/bad", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad")
	})

	t.Run("500 HTTPError records real status and Error span status", func(t *testing.T) {
		exporter.Reset()
		rec := serve(t, srv, http.MethodGet, "/boom")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		span := spans[0]
		status, ok := spanAttribute(span, "http.response.status_code")
		require.True(t, ok)
		assert.Equal(t, int64(http.StatusInternalServerError), status.AsInt64())
		assert.Equal(t, codes.Error, span.Status.Code)
		assert.Equal(t, "GET /boom", span.Name)
	})

	t.Run("plain error maps to 500 and Error span status", func(t *testing.T) {
		exporter.Reset()
		rec := serve(t, srv, http.MethodGet, "/plain-error")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		status, ok := spanAttribute(spans[0], "http.response.status_code")
		require.True(t, ok)
		assert.Equal(t, int64(http.StatusInternalServerError), status.AsInt64())
		assert.Equal(t, codes.Error, spans[0].Status.Code)
	})

	t.Run("400 HTTPError records real status without Error span status", func(t *testing.T) {
		exporter.Reset()
		rec := serve(t, srv, http.MethodGet, "/bad")
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		status, ok := spanAttribute(spans[0], "http.response.status_code")
		require.True(t, ok)
		assert.Equal(t, int64(http.StatusBadRequest), status.AsInt64())
		assert.NotEqual(t, codes.Error, spans[0].Status.Code, "4xx must not mark a server span as error")
	})

	t.Run("404 records real status and unmatched span name", func(t *testing.T) {
		exporter.Reset()
		rec := serve(t, srv, http.MethodGet, "/no-such-route")
		assert.Equal(t, http.StatusNotFound, rec.Code)

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		span := spans[0]
		status, ok := spanAttribute(span, "http.response.status_code")
		require.True(t, ok)
		assert.Equal(t, int64(http.StatusNotFound), status.AsInt64())
		_, hasRoute := spanAttribute(span, "http.route")
		assert.False(t, hasRoute, "unmatched route must not set an empty http.route")
		assert.Equal(t, "GET unmatched", span.Name)
	})

	t.Run("metrics attribute the real status codes, never a premature 200", func(t *testing.T) {
		metrics := scopeMetricsByName(t, reader, "http.server")
		count, ok := metrics["http.server.request.count"]
		require.True(t, ok)
		countSum, ok := count.Data.(metricdata.Sum[int64])
		require.True(t, ok)

		seen := map[int64]bool{}
		for _, dp := range countSum.DataPoints {
			if sc, ok := dp.Attributes.Value("http.response.status_code"); ok {
				seen[sc.AsInt64()] = true
			}
		}
		assert.True(t, seen[http.StatusInternalServerError], "expected a 500 datapoint")
		assert.True(t, seen[http.StatusBadRequest], "expected a 400 datapoint")
		assert.True(t, seen[http.StatusNotFound], "expected a 404 datapoint")
		assert.False(t, seen[http.StatusOK], "no request returned 200; no 200 datapoint must exist")
	})
}

// TestOTelTracing_RedactsSensitiveQuery verifies that secret query-parameter
// values are redacted from the url.full span attribute.
func TestOTelTracing_RedactsSensitiveQuery(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))
	srv, err := New(WithPort(0), WithOTelConfig(cfg))
	require.NoError(t, err)
	srv.Echo().GET("/data", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	rec := serve(t, srv, http.MethodGet, "/data?access_token=supersecret&page=2")
	require.Equal(t, http.StatusOK, rec.Code)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	fullURL, ok := spanAttribute(spans[0], "url.full")
	require.True(t, ok)
	assert.NotContains(t, fullURL.AsString(), "supersecret", "secret query value must be redacted")
	assert.Contains(t, fullURL.AsString(), "access_token=REDACTED")
	assert.Contains(t, fullURL.AsString(), "page=2", "non-sensitive query params must be preserved")
}

func TestOTelNilConfig(t *testing.T) {
	t.Run("nil OTelConfig installs no middleware and does not panic", func(t *testing.T) {
		srv, err := New(WithPort(0))
		require.NoError(t, err)
		serveHealth(t, srv)
	})

	t.Run("OTelConfig without providers installs no middleware and does not panic", func(t *testing.T) {
		cfg := pkgotel.NewConfig("test-service", pkgotel.WithoutLogging())
		srv, err := New(WithPort(0), WithOTelConfig(cfg))
		require.NoError(t, err)
		serveHealth(t, srv)
	})
}
