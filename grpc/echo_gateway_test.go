package grpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestMountGatewayOnEcho(t *testing.T) {
	e := echo.New()
	mux := runtime.NewServeMux()

	MountGatewayOnEcho(e, mux, "/api/v1")

	// Verify that routes were registered
	routes := e.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/*" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected gateway route to be registered")
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

// TestWithGatewayRegistrar verifies that the function passed via
// WithGatewayRegistrar is invoked with the server's gateway mux during setup,
// and that routes registered through it are served under the gateway base path.
func TestWithGatewayRegistrar(t *testing.T) {
	registrarCalled := false
	var gotMux *runtime.ServeMux

	server, err := New(
		WithServiceRegistrar(func(s *grpc.Server) {}),
		WithGatewayRegistrar(func(mux *runtime.ServeMux) {
			registrarCalled = true
			gotMux = mux
			err := mux.HandlePath(http.MethodGet, "/api/v1/ping", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
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

// TestWithGatewayRegistrarNotInvokedWithoutServiceRegistrar pins the current
// behavior that the gateway (and therefore the gateway registrar) is only set
// up when a service registrar is configured.
func TestWithGatewayRegistrarNotInvokedWithoutServiceRegistrar(t *testing.T) {
	called := false
	server, err := New(
		WithGatewayRegistrar(func(mux *runtime.ServeMux) { called = true }),
	)
	require.NoError(t, err)

	require.NoError(t, server.setupEchoServer())

	assert.False(t, called, "gateway setup only runs when a service registrar is configured")
	assert.Nil(t, server.gatewayMux)
}
