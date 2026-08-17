package grpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func TestMountGatewayOnEcho(t *testing.T) {
	e := echo.New()
	mux := runtime.NewServeMux()

	MountGatewayOnEcho(e, mux, "/api/v1")

	// Verify that routes were registered
	routes := e.Routes()
	foundWildcard := false
	foundBare := false
	for _, route := range routes {
		if route.Path == "/api/v1/*" {
			foundWildcard = true
		}
		if route.Path == "/api/v1" {
			foundBare = true
		}
	}
	assert.True(t, foundWildcard, "Expected gateway wildcard route to be registered")
	assert.True(t, foundBare, "Expected bare base-path route to be registered")
}

func TestCreateGatewayMux(t *testing.T) {
	mux := CreateGatewayMux()
	assert.NotNil(t, mux)

	// The mux should be properly configured
	// We can't easily test the internal configuration, but we can verify it was created
}

func TestCreateGatewayMuxMetadata(t *testing.T) {
	// Test that CreateGatewayMux creates a mux with metadata forwarding
	// We can't easily test the internal metadata function directly,
	// but we can verify the mux is created
	mux := CreateGatewayMux()
	assert.NotNil(t, mux)
}

// TestGatewayMetadataAnnotatorForwardsTraceparentHeader verifies that an inbound
// W3C traceparent header is forwarded as gRPC metadata so the backend keeps the
// trace instead of starting a new root.
func TestGatewayMetadataAnnotatorForwardsTraceparentHeader(t *testing.T) {
	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thing", nil)
	req.Header.Set("traceparent", traceparent)
	req.Header.Set("User-Agent", "test-agent")

	md := gatewayMetadataAnnotator(context.Background(), req)

	require.Equal(t, []string{traceparent}, md.Get("traceparent"),
		"annotator must forward the inbound traceparent header")
	assert.Equal(t, []string{"test-agent"}, md.Get("user-agent"))
}

// TestGatewayMetadataAnnotatorInjectsActiveSpan verifies that when a span is
// active in the request context (as after the Echo tracing middleware), the
// annotator injects a traceparent for it, linking the HTTP and gRPC spans.
func TestGatewayMetadataAnnotatorInjectsActiveSpan(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thing", nil)
	req = req.WithContext(trace.ContextWithSpanContext(req.Context(), sc))

	md := gatewayMetadataAnnotator(context.Background(), req)

	// The injected traceparent must carry the active span's trace/span ids.
	extracted := propagation.TraceContext{}.Extract(context.Background(), metadataCarrier(md))
	got := trace.SpanContextFromContext(extracted)
	assert.Equal(t, traceID, got.TraceID(), "annotator must inject the active span's trace id")
	assert.Equal(t, spanID, got.SpanID(), "annotator must inject the active span's span id")
}

// TestWithGatewayRegistrar verifies that the function passed via
// WithGatewayRegistrar is invoked with the server's gateway mux during setup,
// and that routes registered through it are served under the gateway base path.
// The mount strips the base path, so mux patterns are proto http-rule style
// (e.g. "/ping"), while clients GET "/api/v1/ping".
func TestWithGatewayRegistrar(t *testing.T) {
	registrarCalled := false
	var gotMux *runtime.ServeMux

	server, err := New(
		WithServiceRegistrar(func(s *grpc.Server) {}),
		WithGatewayRegistrar(func(mux *runtime.ServeMux) {
			registrarCalled = true
			gotMux = mux
			err := mux.HandlePath(http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("pong"))
			})
			assert.NoError(t, err)
		}),
	)
	require.NoError(t, err)

	// setupEchoServer runs the gateway integration, same as Start does.
	require.NoError(t, server.setupEchoServer())

	assert.True(t, registrarCalled, "expected gateway registrar to be invoked during gateway setup")
	assert.Same(t, server.gatewayMux, gotMux, "registrar must receive the server's gateway mux")

	// The registered route must be reachable through Echo under the gateway base path.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

// TestWithGatewayRegistrarInvokedWithoutServiceRegistrar verifies that the
// gateway is mounted and the gateway registrar is invoked even when no
// service registrar is configured.
func TestWithGatewayRegistrarInvokedWithoutServiceRegistrar(t *testing.T) {
	called := false
	server, err := New(
		WithGatewayRegistrar(func(mux *runtime.ServeMux) {
			called = true
			err := mux.HandlePath(http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				_, _ = w.Write([]byte("pong"))
			})
			assert.NoError(t, err)
		}),
	)
	require.NoError(t, err)

	require.NoError(t, server.setupEchoServer())

	assert.True(t, called, "gateway registrar must be invoked even without a service registrar")
	assert.NotNil(t, server.gatewayMux, "gateway mux must be set up even without a service registrar")

	// The gateway is mounted: the route registered on the mux is reachable
	// through Echo under the gateway base path.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}
