package grpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otellog "go.opentelemetry.io/otel/log"
	logembedded "go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// ============================================================================
// Capturing logger provider: records the span context of every emitted log so
// tests can assert log-trace correlation (trace_id/span_id present on logs).
// ============================================================================

type recordedLog struct {
	spanContext trace.SpanContext
	body        string
}

type logSink struct {
	mu      sync.Mutex
	records []recordedLog
}

func (s *logSink) all() []recordedLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedLog, len(s.records))
	copy(out, s.records)
	return out
}

type capturingLogger struct {
	logembedded.Logger
	sink *logSink
}

func (l *capturingLogger) Emit(ctx context.Context, record otellog.Record) {
	l.sink.mu.Lock()
	defer l.sink.mu.Unlock()
	l.sink.records = append(l.sink.records, recordedLog{
		spanContext: trace.SpanContextFromContext(ctx),
		body:        record.Body().AsString(),
	})
}

func (l *capturingLogger) Enabled(context.Context, otellog.EnabledParameters) bool { return true }

type capturingLoggerProvider struct {
	logembedded.LoggerProvider
	sink *logSink
}

func (p *capturingLoggerProvider) Logger(string, ...otellog.LoggerOption) otellog.Logger {
	return &capturingLogger{sink: p.sink}
}

// remoteParent builds an incoming context carrying a W3C traceparent for a
// known remote trace/span, mirroring what an upstream caller would send.
func remoteParent(t *testing.T) (context.Context, trace.TraceID, trace.SpanID) {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)

	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	md := metadata.MD{}
	prop.Inject(trace.ContextWithSpanContext(context.Background(), remote), metadataCarrier(md))
	return metadata.NewIncomingContext(context.Background(), md), traceID, spanID
}

// TestGRPCTracingInterceptorExtractsRemoteParent asserts that the unary tracing
// interceptor extracts the incoming W3C traceparent and starts the server span
// as a child of that remote parent (same trace id) rather than a new root.
func TestGRPCTracingInterceptorExtractsRemoteParent(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	cfg := pkgotel.NewConfig("test", pkgotel.WithTracerProvider(tp))
	interceptor := createGRPCTracingInterceptor(cfg)

	ctx, traceID, spanID := remoteParent(t)

	var handlerTraceID trace.TraceID
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerTraceID = trace.SpanFromContext(ctx).SpanContext().TraceID()
		return "ok", nil
	}

	_, err := interceptor(ctx, "req", mockUnaryInfo("/test.Service/Method"), handler)
	require.NoError(t, err)

	assert.Equal(t, traceID, handlerTraceID, "server span must inherit the incoming trace id")

	ended := sr.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, traceID, ended[0].Parent().TraceID(), "span parent must be the remote trace")
	assert.Equal(t, spanID, ended[0].Parent().SpanID(), "span parent must be the remote span")
	assert.True(t, ended[0].Parent().IsRemote(), "parent must be marked remote")
}

// TestGRPCStreamTracingInterceptorExtractsRemoteParent is the stream-interceptor
// counterpart: the stream tracing interceptor must exist and link the server
// span to the incoming trace, and expose it via the wrapped stream context.
func TestGRPCStreamTracingInterceptorExtractsRemoteParent(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	cfg := pkgotel.NewConfig("test", pkgotel.WithTracerProvider(tp))
	interceptor := createGRPCStreamTracingInterceptor(cfg)

	ctx, traceID, spanID := remoteParent(t)

	var streamTraceID trace.TraceID
	handler := func(_ interface{}, ss grpc.ServerStream) error {
		streamTraceID = trace.SpanFromContext(ss.Context()).SpanContext().TraceID()
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream", IsServerStream: true}
	err := interceptor(nil, &fakeServerStream{ctx: ctx}, info, handler)
	require.NoError(t, err)

	assert.Equal(t, traceID, streamTraceID, "stream span must inherit the incoming trace id")

	ended := sr.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, traceID, ended[0].Parent().TraceID())
	assert.Equal(t, spanID, ended[0].Parent().SpanID())
	assert.True(t, ended[0].Parent().IsRemote())
}

// fakeServerStream is a minimal grpc.ServerStream whose Context returns a fixed
// context, used to drive stream interceptors in tests.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeServerStream) Context() context.Context { return s.ctx }

// TestGRPCLoggingInterceptorCarriesTraceID pins the unary chain ordering:
// tracing must run before logging so the access log emitted by the logging
// interceptor carries the trace id of the active span.
func TestGRPCLoggingInterceptorCarriesTraceID(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	sink := &logSink{}
	lp := &capturingLoggerProvider{sink: sink}
	cfg := pkgotel.NewConfig("test",
		pkgotel.WithTracerProvider(tp),
		pkgotel.WithLoggerProvider(lp))

	tracing := createGRPCTracingInterceptor(cfg)
	logging := createGRPCLoggingInterceptor(cfg)
	info := mockUnaryInfo("/test.Service/Method")
	handler := mockUnaryHandler("ok", nil)

	// tracing (outer) -> logging (inner) -> handler, matching server wiring.
	_, err := tracing(context.Background(), "req", info,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return logging(ctx, req, info, handler)
		})
	require.NoError(t, err)

	logs := sink.all()
	require.Len(t, logs, 1)
	assert.True(t, logs[0].spanContext.IsValid(), "access log must carry a valid span context")

	ended := sr.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, ended[0].SpanContext().TraceID(), logs[0].spanContext.TraceID(),
		"access log trace id must match the server span")
	assert.Equal(t, ended[0].SpanContext().SpanID(), logs[0].spanContext.SpanID(),
		"access log span id must match the server span")
}

// TestEchoLoggingMiddlewareCarriesTraceID pins the Echo chain ordering plus the
// context re-read: the tracing middleware runs first and installs the span, and
// the logging middleware re-reads the request context after next() so the
// access log carries the trace id.
func TestEchoLoggingMiddlewareCarriesTraceID(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	sink := &logSink{}
	lp := &capturingLoggerProvider{sink: sink}
	cfg := pkgotel.NewConfig("test",
		pkgotel.WithTracerProvider(tp),
		pkgotel.WithLoggerProvider(lp))

	tracing := createHTTPGatewayTracingMiddleware(cfg)
	logging := createHTTPGatewayLoggingMiddleware(cfg)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/thing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/thing")

	handler := func(c echo.Context) error { return c.String(http.StatusOK, "OK") }

	// tracing (outer) -> logging (inner) -> handler, matching server wiring.
	wrapped := tracing(logging(handler))
	require.NoError(t, wrapped(c))

	logs := sink.all()
	require.Len(t, logs, 1)
	assert.True(t, logs[0].spanContext.IsValid(), "HTTP access log must carry a valid span context")

	ended := sr.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, ended[0].SpanContext().TraceID(), logs[0].spanContext.TraceID(),
		"HTTP access log trace id must match the HTTP server span")
}
