package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/jasoet/pkg/v3/otel"
)

// attrValue finds an attribute by key in a recorded span's attribute set.
func attrValue(attrs []attribute.KeyValue, key string) (attribute.Value, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value, true
		}
	}
	return attribute.Value{}, false
}

// newRecordingTracer returns a real SDK tracer provider wired to an in-memory
// span recorder so tests can assert on real spans, attributes, and injected
// propagation headers (never a noop provider).
func newRecordingTracer() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	return tp, sr
}

// TestOTelTracing_NilHeaders_NoPanic reproduces the critical production crash:
// with a real SDK tracer, BeforeRequest injects a traceparent header. If
// doRequest hands the caller's (nil) map to the middleware chain, the injection
// writes into a nil map and panics. The fixed client normalizes headers to a
// non-nil copy, so nil headers must be safe AND traceparent must reach the wire.
func TestOTelTracing_NilHeaders_NoPanic(t *testing.T) {
	var gotTraceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	tp, sr := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	client := NewClient(WithOTelConfig(cfg))

	require.NotPanics(t, func() {
		resp, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	assert.NotEmpty(t, gotTraceparent, "traceparent must be injected into the request the server received")
	require.Len(t, sr.Ended(), 1, "exactly one client span must be recorded")
}

// TestOTelTracing_CallerMapUntouched proves the middleware chain no longer
// mutates the caller-supplied headers map: the injected traceparent must reach
// the wire but must NOT appear in the caller's map (which would race under
// concurrent use of a shared map).
func TestOTelTracing_CallerMapUntouched(t *testing.T) {
	var gotTraceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tp, _ := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	client := NewClient(WithOTelConfig(cfg))

	headers := map[string]string{"X-Custom": "v"}
	_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", headers)
	require.NoError(t, err)

	assert.NotEmpty(t, gotTraceparent, "server must still receive the injected traceparent")
	_, hasTP := headers["traceparent"]
	assert.False(t, hasTP, "caller map must not be polluted with traceparent")
	assert.Len(t, headers, 1, "caller map must be untouched")
	assert.Equal(t, "v", headers["X-Custom"])
}

// TestOTelTracing_RealSpanAttributes asserts on real recorded span content
// (kind, method, status code, span status) using an in-memory exporter rather
// than a noop provider.
func TestOTelTracing_RealSpanAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	tp, sr := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	client := NewClient(WithOTelConfig(cfg))

	_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, trace.SpanKindClient, span.SpanKind())
	assert.Equal(t, http.MethodGet, span.Name())

	method, ok := attrValue(span.Attributes(), "http.request.method")
	require.True(t, ok, "http.request.method attribute must be present")
	assert.Equal(t, http.MethodGet, method.AsString())

	status, ok := attrValue(span.Attributes(), "http.response.status_code")
	require.True(t, ok, "http.response.status_code attribute must be present")
	assert.Equal(t, int64(http.StatusOK), status.AsInt64())

	full, ok := attrValue(span.Attributes(), "url.full")
	require.True(t, ok, "url.full attribute must be present")
	assert.Equal(t, server.URL, full.AsString())
}

// TestOTelTracing_URLRedaction verifies embedded credentials in the URL are
// stripped from the recorded url.full attribute.
func TestOTelTracing_URLRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tp, sr := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	client := NewClient(WithOTelConfig(cfg))

	// Inject userinfo into the test-server URL: http://user:secret@host:port
	creds := strings.Replace(server.URL, "http://", "http://user:secret@", 1)

	_, err := client.MakeRequest(context.Background(), http.MethodGet, creds, "", nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	full, ok := attrValue(spans[0].Attributes(), "url.full")
	require.True(t, ok)
	assert.NotContains(t, full.AsString(), "secret", "userinfo secret must be redacted from url.full")
	assert.NotContains(t, full.AsString(), "user:", "userinfo must be redacted from url.full")
}

// TestOTelTracing_DurationFractionalMillis asserts sub-millisecond durations are
// recorded as a fractional millisecond value rather than truncated to 0.
func TestOTelTracing_DurationFractionalMillis(t *testing.T) {
	tp, sr := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	mw := NewOTelTracingMiddleware(cfg)
	require.NotNil(t, mw)

	ctx := mw.BeforeRequest(context.Background(), http.MethodGet, "http://example.com", "", map[string]string{})
	mw.AfterRequest(ctx, RequestInfo{
		Method:     http.MethodGet,
		URL:        "http://example.com",
		StatusCode: 200,
		Duration:   500 * time.Microsecond, // 0.5ms
	})

	spans := sr.Ended()
	require.Len(t, spans, 1)
	dur, ok := attrValue(spans[0].Attributes(), "http.request.duration_ms")
	require.True(t, ok, "duration attribute must be present")
	assert.InDelta(t, 0.5, dur.AsFloat64(), 0.0001, "0.5ms must not truncate to 0")
}

// TestOTelTracing_ResponseSizeNotTruncated verifies the recorded response body
// size is the true transport size, not the length of the log-truncated body.
func TestOTelTracing_ResponseSizeNotTruncated(t *testing.T) {
	const bodyLen = 5000
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("x", bodyLen)))
	}))
	defer server.Close()

	tp, sr := newRecordingTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	restCfg := DefaultRestConfig()
	restCfg.MaxResponseBodyLog = 10 // aggressively truncate the logged body
	client := NewClient(WithRestConfig(*restCfg), WithOTelConfig(cfg))

	_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	size, ok := attrValue(spans[0].Attributes(), "http.response.body.size")
	require.True(t, ok)
	assert.Equal(t, int64(bodyLen), size.AsInt64(), "response size must be the full body size, not the truncated length")
}

// TestOTelMetrics_ResponseSizeNotTruncated asserts the metrics middleware also
// records the true response size via an in-memory ManualReader.
func TestOTelMetrics_ResponseSizeNotTruncated(t *testing.T) {
	const bodyLen = 4096
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("y", bodyLen)))
	}))
	defer server.Close()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() { _ = mp.Shutdown(context.Background()) }()

	cfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))
	restCfg := DefaultRestConfig()
	restCfg.MaxResponseBodyLog = 8
	client := NewClient(WithRestConfig(*restCfg), WithOTelConfig(cfg))

	_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	var maxRecorded int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.client.response.size" {
				continue
			}
			found = true
			hist, ok := m.Data.(metricdata.Histogram[int64])
			require.True(t, ok, "expected Histogram[int64], got %T", m.Data)
			for _, dp := range hist.DataPoints {
				if v, ok := dp.Max.Value(); ok && v > maxRecorded {
					maxRecorded = v
				}
			}
		}
	}
	require.True(t, found, "http.client.response.size metric must be recorded")
	assert.Equal(t, int64(bodyLen), maxRecorded, "recorded response size must be the full body size")
}

// TestOptionOrderIndependence_OTelPreserved verifies WithRestConfig placed after
// WithOTelConfig no longer discards the OTel config: telemetry must remain
// active regardless of option order.
func TestOptionOrderIndependence_OTelPreserved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	run := func(t *testing.T, opts ...ClientOption) {
		t.Helper()
		client := NewClient(opts...)
		require.NotNil(t, client.GetRestConfig().OTelConfig, "OTel config must survive option merge")

		_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
		require.NoError(t, err)
	}

	t.Run("OTel then RestConfig", func(t *testing.T) {
		tp, sr := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		run(t, WithOTelConfig(cfg), WithRestConfig(*DefaultRestConfig()))
		require.Len(t, sr.Ended(), 1, "a span must be recorded, proving OTel stayed active")
	})

	t.Run("RestConfig then OTel", func(t *testing.T) {
		tp, sr := newRecordingTracer()
		defer func() { _ = tp.Shutdown(context.Background()) }()
		cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
		run(t, WithRestConfig(*DefaultRestConfig()), WithOTelConfig(cfg))
		require.Len(t, sr.Ended(), 1, "a span must be recorded, proving OTel stayed active")
	})
}
