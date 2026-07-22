package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

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
