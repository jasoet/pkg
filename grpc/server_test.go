package grpc

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// sendSelfSIGTERM delivers SIGTERM to the current process so a running Start*
// convenience function (which installs its own SIGTERM handler) shuts down
// gracefully. The signal is suppressed from terminating the process because a
// handler is registered.
func sendSelfSIGTERM(t *testing.T) {
	t.Helper()
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGTERM))
}

func TestNewServer(t *testing.T) {
	server, err := New(
		WithGRPCPort("8080"),
		WithH2CMode(),
	)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, "8080", server.config.grpcPort)
	assert.NotNil(t, server.healthManager)
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
		WithHealthPath("/health"),
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
	port := freePort(t)

	server, err := New(
		WithGRPCPort(port),
		WithH2CMode(),
		WithShutdownTimeout(5*time.Second),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()
	t.Cleanup(func() { _ = server.Stop() })

	waitForPort(t, port, 5*time.Second)
	assert.True(t, server.IsRunning())

	require.NoError(t, server.Stop())
	assert.NoError(t, recvWithTimeout(t, startErr, 10*time.Second))
	assert.False(t, server.IsRunning())
}

func TestServerDoubleStart(t *testing.T) {
	port := freePort(t)
	server, err := New(WithGRPCPort(port), WithShutdownTimeout(5*time.Second))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()
	t.Cleanup(func() { _ = server.Stop() })

	waitForPort(t, port, 5*time.Second)

	// A second Start while running must fail.
	err = server.Start()
	assert.Error(t, err, "Expected error when starting server twice")

	require.NoError(t, server.Stop())
	assert.NoError(t, recvWithTimeout(t, startErr, 10*time.Second))
}

// runConvenienceStart starts one of the fire-and-forget Start* helpers in a
// goroutine, waits deterministically for it to listen, asserts the registrar
// ran, then triggers graceful shutdown via SIGTERM and waits for the helper to
// return (which also exercises the signal.Stop cleanup path).
func runConvenienceStart(t *testing.T, port string, called *int32, start func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		start()
		close(done)
	}()

	waitForPort(t, port, 5*time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(called), "Expected service registrar to be called")

	sendSelfSIGTERM(t)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Start* did not return after SIGTERM")
	}
}

func TestStartFunction(t *testing.T) {
	port := freePort(t)
	var called int32
	registrar := func(s *grpc.Server) { atomic.StoreInt32(&called, 1) }
	runConvenienceStart(t, port, &called, func() { _ = Start(port, registrar) })
}

func TestStartH2CFunction(t *testing.T) {
	port := freePort(t)
	var called int32
	registrar := func(s *grpc.Server) { atomic.StoreInt32(&called, 1) }
	runConvenienceStart(t, port, &called, func() { _ = StartH2C(port, registrar) })
}

func TestStartSeparateFunction(t *testing.T) {
	grpcPort := freePort(t)
	httpPort := freePort(t)
	var called int32
	registrar := func(s *grpc.Server) { atomic.StoreInt32(&called, 1) }
	runConvenienceStart(t, grpcPort, &called, func() { _ = StartSeparate(grpcPort, httpPort, registrar) })
}

func TestStartWithOptions(t *testing.T) {
	port := freePort(t)
	var called int32
	registrar := func(s *grpc.Server) { atomic.StoreInt32(&called, 1) }
	runConvenienceStart(t, port, &called, func() {
		_ = Start(port, registrar, WithCORS(), WithRateLimit(200.0), WithoutReflection())
	})
}

func TestServerWithCustomShutdown(t *testing.T) {
	shutdownCalled := false
	port := freePort(t)

	server, err := New(
		WithGRPCPort(port),
		WithShutdownTimeout(5*time.Second),
		WithShutdownHandler(func() error {
			shutdownCalled = true
			return nil
		}),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()
	t.Cleanup(func() { _ = server.Stop() })

	waitForPort(t, port, 5*time.Second)

	require.NoError(t, server.Stop())
	assert.NoError(t, recvWithTimeout(t, startErr, 10*time.Second))
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
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Test that connection is not in shutdown state immediately
	state := conn.GetState()
	assert.NotEqual(t, state.String(), "SHUTDOWN", "Connection should not be shutdown immediately")

	// Cleanup
	server.grpcServer.GracefulStop()
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
		WithHealthCheck(),
		WithReflection(),
		WithCORS(),
		WithRateLimit(250.0),
		WithHealthPath("/custom-health"),
		WithGatewayBasePath("/api/v2"),
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
	assert.True(t, server.config.enableHealthCheck)
	assert.True(t, server.config.enableReflection)
	assert.True(t, server.config.enableCORS)
	assert.True(t, server.config.enableRateLimit)
	assert.Equal(t, 250.0, server.config.rateLimit)
	assert.Equal(t, "/custom-health", server.config.healthPath)
	assert.Equal(t, "/api/v2", server.config.gatewayBasePath)
}
