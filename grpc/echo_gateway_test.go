package grpc

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func TestMountGatewayWithStripPrefix(t *testing.T) {
	e := echo.New()
	mux := runtime.NewServeMux()

	MountGatewayWithStripPrefix(e, mux, "/api/*", "/api")

	// Verify that routes were registered
	routes := e.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/*" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected gateway route with strip prefix to be registered")
}

func TestSetupGatewayForH2C(t *testing.T) {
	ctx := context.Background()
	mux := runtime.NewServeMux()
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	serviceRegistrar := func(s *grpc.Server) {}

	err := SetupGatewayForH2C(ctx, mux, serviceRegistrar, grpcServer)
	require.NoError(t, err)
}

func TestSetupGatewayForSeparate(t *testing.T) {
	t.Run("server available", func(t *testing.T) {
		// Start a real gRPC server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		grpcServer := grpc.NewServer()
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		defer grpcServer.Stop()

		// Wait a bit for server to start
		time.Sleep(100 * time.Millisecond)

		ctx := context.Background()
		mux := runtime.NewServeMux()

		err = SetupGatewayForSeparate(ctx, mux, lis.Addr().String())
		assert.NoError(t, err)
	})
}

func TestWaitForGRPCServer(t *testing.T) {
	t.Run("server becomes available", func(t *testing.T) {
		// Start a real gRPC server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		grpcServer := grpc.NewServer()
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		defer grpcServer.Stop()

		// Wait a bit for server to start
		time.Sleep(100 * time.Millisecond)

		ctx := context.Background()
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		mux := runtime.NewServeMux()

		err = waitForGRPCServer(ctx, lis.Addr().String(), opts, mux)
		assert.NoError(t, err)
	})
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

func TestGatewayHealthMiddleware(t *testing.T) {
	e := echo.New()

	// Add the middleware
	middleware := GatewayHealthMiddleware()
	e.Use(middleware)

	// Create a test handler
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Verify headers were added
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "grpc-gateway/v2", rec.Header().Get("X-Gateway-Version"))
	assert.Equal(t, "grpc-gateway", rec.Header().Get("X-Server-Type"))
}

func TestLogGatewayRoutes(t *testing.T) {
	// This function just logs, so we just verify it doesn't panic
	services := []string{"UserService", "ProductService", "OrderService"}

	LogGatewayRoutes("/api/v1", services)

	// If we get here without panic, the function works
}

func TestLogGatewayRoutesEmpty(t *testing.T) {
	// Test with empty services list
	LogGatewayRoutes("/api/v1", []string{})

	// If we get here without panic, the function works
}
