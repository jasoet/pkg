package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewServer(t *testing.T) {
	server, err := New(
		WithGRPCPort("8080"),
		WithH2CMode(),
	)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, "8080", server.config.grpcPort)
	assert.NotNil(t, server.healthManager)
	assert.NotNil(t, server.metricsManager)
	assert.NotNil(t, server.grpcServer)
}

func TestNewServerWithInvalidConfig(t *testing.T) {
	server, err := New(
		WithGRPCPort(""), // Empty port
	)
	assert.Error(t, err)
	assert.Nil(t, server)
}

func TestServerGetters(t *testing.T) {
	server, err := New(WithGRPCPort("8080"))
	require.NoError(t, err)

	// Test health manager getter
	hm := server.GetHealthManager()
	assert.NotNil(t, hm)

	// Test metrics manager getter
	mm := server.GetMetricsManager()
	assert.NotNil(t, mm)

	// Test gRPC server getter
	grpcSrv := server.GetGRPCServer()
	assert.NotNil(t, grpcSrv)

	// Test running status
	assert.False(t, server.IsRunning())
}

func TestServerSetupGRPCServer(t *testing.T) {
	// Test with custom configurer
	configurerCalled := false
	registrarCalled := false

	server, err := New(
		WithGRPCPort("8080"),
		WithH2CMode(),
		WithReflection(),
		WithConnectionTimeouts(5*time.Minute, 10*time.Minute, 1*time.Minute),
		WithGRPCConfigurer(func(s *grpc.Server) {
			configurerCalled = true
		}),
		WithServiceRegistrar(func(s *grpc.Server) {
			registrarCalled = true
		}),
	)
	require.NoError(t, err)

	assert.True(t, configurerCalled, "Expected gRPC configurer to be called")
	assert.True(t, registrarCalled, "Expected service registrar to be called")

	grpcSrv := server.GetGRPCServer()
	assert.NotNil(t, grpcSrv)
}

func TestServerSetupEchoServer(t *testing.T) {
	configurerCalled := false

	server, err := New(
		WithSeparateMode("9090", "9091"),
		WithHealthCheck(),
		WithMetrics(),
		WithHealthPath("/health"),
		WithMetricsPath("/metrics"),
		WithEchoConfigurer(func(e *echo.Echo) {
			configurerCalled = true
			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "test")
			})
		}),
	)
	require.NoError(t, err)

	// Setup Echo server
	err = server.setupEchoServer()
	require.NoError(t, err)

	assert.True(t, configurerCalled, "Expected Echo configurer to be called")
	assert.NotNil(t, server.echo)
}

func TestServerStartStop(t *testing.T) {
	// Use a random available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	server, err := New(
		WithGRPCPort(port),
		WithH2CMode(),
	)
	require.NoError(t, err)

	// Start server in goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	assert.True(t, server.IsRunning())

	// Stop server
	stopErr := server.Stop()
	assert.NoError(t, stopErr)

	// Wait for start goroutine to complete
	wg.Wait()

	// Server should be stopped
	assert.False(t, server.IsRunning())
}

func TestServerDoubleStart(t *testing.T) {
	server, err := New(WithGRPCPort("0")) // Use any available port
	require.NoError(t, err)

	// Start server in goroutine
	go func() {
		server.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Try to start again
	err = server.Start()
	assert.Error(t, err, "Expected error when starting server twice")

	// Cleanup
	server.Stop()
}

func TestStartFunction(t *testing.T) {
	// Get available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	// Test the convenience Start function
	var serviceRegistrarCalled int32
	serviceRegistrar := func(s *grpc.Server) {
		atomic.StoreInt32(&serviceRegistrarCalled, 1)
	}

	// Start in goroutine
	go func() {
		Start(port, serviceRegistrar)
	}()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&serviceRegistrarCalled), "Expected service registrar to be called")
}

func TestStartH2CFunction(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	var serviceRegistrarCalled int32
	serviceRegistrar := func(s *grpc.Server) {
		atomic.StoreInt32(&serviceRegistrarCalled, 1)
	}

	// Test StartH2C in goroutine
	go func() {
		StartH2C(port, serviceRegistrar)
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&serviceRegistrarCalled), "Expected service registrar to be called")
}

func TestStartSeparateFunction(t *testing.T) {
	// Get two available ports
	listener1, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	grpcPort := fmt.Sprintf("%d", listener1.Addr().(*net.TCPAddr).Port)
	listener1.Close()

	listener2, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	httpPort := fmt.Sprintf("%d", listener2.Addr().(*net.TCPAddr).Port)
	listener2.Close()

	var serviceRegistrarCalled int32
	serviceRegistrar := func(s *grpc.Server) {
		atomic.StoreInt32(&serviceRegistrarCalled, 1)
	}

	// Test StartSeparate in goroutine
	go func() {
		StartSeparate(grpcPort, httpPort, serviceRegistrar)
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&serviceRegistrarCalled), "Expected service registrar to be called")
}

func TestStartWithOptions(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	var serviceRegistrarCalled int32
	serviceRegistrar := func(s *grpc.Server) {
		atomic.StoreInt32(&serviceRegistrarCalled, 1)
	}

	// Test Start with additional options
	go func() {
		Start(port, serviceRegistrar,
			WithCORS(),
			WithRateLimit(200.0),
			WithoutReflection(),
		)
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&serviceRegistrarCalled), "Expected service registrar to be called")
}

func TestServerWithCustomShutdown(t *testing.T) {
	shutdownCalled := false

	server, err := New(
		WithGRPCPort("0"),
		WithShutdownHandler(func() error {
			shutdownCalled = true
			return nil
		}),
	)
	require.NoError(t, err)

	// Start and immediately stop
	go func() {
		server.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	err = server.Stop()
	assert.NoError(t, err)
	assert.True(t, shutdownCalled, "Expected custom shutdown handler to be called")
}

func TestServerModeValidation(t *testing.T) {
	tests := []struct {
		name      string
		options   []Option
		expectErr bool
	}{
		{
			name: "valid H2C mode",
			options: []Option{
				WithH2CMode(),
				WithGRPCPort("8080"),
			},
			expectErr: false,
		},
		{
			name: "valid separate mode",
			options: []Option{
				WithSeparateMode("9090", "9091"),
			},
			expectErr: false,
		},
		{
			name: "separate mode missing HTTP port",
			options: []Option{
				WithSeparateMode("9090", ""),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.options...)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Integration test with in-memory connection
func TestServerIntegration(t *testing.T) {
	// Create buffer connection for testing
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)

	server, err := New(
		WithGRPCPort("8080"),
		WithH2CMode(),
		WithMetrics(),
		WithHealthCheck(),
		WithReflection(),
		WithServiceRegistrar(func(s *grpc.Server) {
			// Service registrar called
		}),
	)
	require.NoError(t, err)

	// Start server with buffer listener
	go func() {
		server.grpcServer.Serve(lis)
	}()

	// Create client connection
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Test that connection is not in shutdown state immediately
	state := conn.GetState()
	assert.NotEqual(t, state.String(), "SHUTDOWN", "Connection should not be shutdown immediately")

	// Cleanup
	server.grpcServer.GracefulStop()
}

func TestServerTrackUptime(t *testing.T) {
	server, err := New(
		WithGRPCPort("8080"),
		WithMetrics(),
	)
	require.NoError(t, err)

	// Test uptime tracking
	go server.trackUptime()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Get metrics to verify uptime is being tracked
	mm := server.GetMetricsManager()
	metricFamilies, err := mm.GetRegistry().Gather()
	require.NoError(t, err)

	// Look for uptime metric
	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "grpc_server_uptime_seconds" {
			found = true
			break
		}
	}

	assert.True(t, found, "Expected uptime metric to be present")
}

func TestServerWithMiddleware(t *testing.T) {
	mw1 := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	mw2 := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	server, err := New(
		WithGRPCPort("8080"),
		WithMiddleware(mw1, mw2),
	)
	require.NoError(t, err)

	// Setup Echo server to trigger middleware attachment
	err = server.setupEchoServer()
	require.NoError(t, err)

	assert.NotNil(t, server.echo)
	assert.Len(t, server.config.middleware, 2)
}

func TestServerWithAllOptions(t *testing.T) {
	server, err := New(
		WithSeparateMode("9090", "9091"),
		WithShutdownTimeout(45*time.Second),
		WithReadTimeout(15*time.Second),
		WithWriteTimeout(20*time.Second),
		WithIdleTimeout(90*time.Second),
		WithConnectionTimeouts(20*time.Minute, 40*time.Minute, 10*time.Second),
		WithMetrics(),
		WithHealthCheck(),
		WithLogging(),
		WithReflection(),
		WithCORS(),
		WithRateLimit(250.0),
		WithMetricsPath("/custom-metrics"),
		WithHealthPath("/custom-health"),
		WithGatewayBasePath("/api/v2"),
		WithTLS("cert.pem", "key.pem"),
	)
	require.NoError(t, err)

	// Verify configuration
	assert.Equal(t, SeparateMode, server.config.mode)
	assert.Equal(t, "9090", server.config.grpcPort)
	assert.Equal(t, "9091", server.config.httpPort)
	assert.Equal(t, 45*time.Second, server.config.shutdownTimeout)
	assert.Equal(t, 15*time.Second, server.config.readTimeout)
	assert.Equal(t, 20*time.Second, server.config.writeTimeout)
	assert.Equal(t, 90*time.Second, server.config.idleTimeout)
	assert.Equal(t, 20*time.Minute, server.config.maxConnectionIdle)
	assert.Equal(t, 40*time.Minute, server.config.maxConnectionAge)
	assert.Equal(t, 10*time.Second, server.config.maxConnectionAgeGrace)
	assert.True(t, server.config.enableMetrics)
	assert.True(t, server.config.enableHealthCheck)
	assert.True(t, server.config.enableLogging)
	assert.True(t, server.config.enableReflection)
	assert.True(t, server.config.enableCORS)
	assert.True(t, server.config.enableRateLimit)
	assert.Equal(t, 250.0, server.config.rateLimit)
	assert.Equal(t, "/custom-metrics", server.config.metricsPath)
	assert.Equal(t, "/custom-health", server.config.healthPath)
	assert.Equal(t, "/api/v2", server.config.gatewayBasePath)
	assert.True(t, server.config.enableTLS)
	assert.Equal(t, "cert.pem", server.config.certFile)
	assert.Equal(t, "key.pem", server.config.keyFile)
}
