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
