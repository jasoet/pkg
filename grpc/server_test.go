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
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// Mock gRPC service for testing
type mockService struct{}

func (s *mockService) TestMethod(ctx context.Context, req *mockRequest) (*mockResponse, error) {
	return &mockResponse{Message: "test response"}, nil
}

type mockRequest struct {
	Data string
}

type mockResponse struct {
	Message string
}

func TestNewServer(t *testing.T) {
	config := Config{
		GRPCPort: "8080",
		Mode:     H2CMode,
	}
	config.SetDefaults()

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config.GRPCPort != "8080" {
		t.Errorf("Expected gRPC port to be 8080, got %s", server.config.GRPCPort)
	}

	if server.healthManager == nil {
		t.Error("Expected health manager to be initialized")
	}

	if server.metricsManager == nil {
		t.Error("Expected metrics manager to be initialized")
	}

	if server.grpcServer == nil {
		t.Error("Expected gRPC server to be initialized")
	}
}

func TestNewServerWithInvalidConfig(t *testing.T) {
	config := Config{
		// Missing required GRPCPort
		Mode: H2CMode,
	}

	server, err := New(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	if server != nil {
		t.Error("Expected server to be nil for invalid config")
	}
}

func TestServerGetters(t *testing.T) {
	config := DefaultConfig()
	config.GRPCPort = "8080"

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test health manager getter
	hm := server.GetHealthManager()
	if hm == nil {
		t.Error("Expected health manager to be returned")
	}

	// Test metrics manager getter
	mm := server.GetMetricsManager()
	if mm == nil {
		t.Error("Expected metrics manager to be returned")
	}

	// Test gRPC server getter
	grpcSrv := server.GetGRPCServer()
	if grpcSrv == nil {
		t.Error("Expected gRPC server to be returned")
	}

	// Test running status
	if server.IsRunning() {
		t.Error("Expected server to not be running initially")
	}
}

func TestServerSetupGRPCServer(t *testing.T) {
	config := Config{
		GRPCPort:              "8080",
		Mode:                  H2CMode,
		EnableReflection:      true,
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      10 * time.Minute,
		MaxConnectionAgeGrace: 1 * time.Minute,
	}
	config.SetDefaults()

	// Test with custom configurer
	configurerCalled := false
	config.GRPCConfigurer = func(s *grpc.Server) {
		configurerCalled = true
	}

	// Test with service registrar
	registrarCalled := false
	config.ServiceRegistrar = func(s *grpc.Server) {
		registrarCalled = true
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if !configurerCalled {
		t.Error("Expected gRPC configurer to be called")
	}

	if !registrarCalled {
		t.Error("Expected service registrar to be called")
	}

	// Verify reflection is registered (this is harder to test directly)
	grpcSrv := server.GetGRPCServer()
	if grpcSrv == nil {
		t.Error("Expected gRPC server to be configured")
	}
}

func TestServerSetupEchoServer(t *testing.T) {
	config := Config{
		GRPCPort:          "9090",
		HTTPPort:          "9091",
		Mode:              SeparateMode,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		HealthPath:        "/health",
		MetricsPath:       "/metrics",
	}
	config.SetDefaults()

	// Test with custom Echo configurer
	configurerCalled := false
	config.EchoConfigurer = func(e *echo.Echo) {
		configurerCalled = true
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Setup Echo server
	err = server.setupEchoServer()
	if err != nil {
		t.Fatalf("Failed to setup Echo server: %v", err)
	}

	if !configurerCalled {
		t.Error("Expected Echo configurer to be called")
	}

	if server.echo == nil {
		t.Error("Expected Echo server to be configured")
	}
}

func TestServerStartStop(t *testing.T) {
	// Use a random available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	config := Config{
		GRPCPort: port,
		Mode:     H2CMode,
	}
	config.SetDefaults()

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	if !server.IsRunning() {
		t.Error("Expected server to be running")
	}

	// Stop server
	stopErr := server.Stop()
	if stopErr != nil {
		t.Errorf("Failed to stop server: %v", stopErr)
	}

	// Wait for start goroutine to complete
	wg.Wait()

	if !server.IsRunning() {
		// Server should be stopped
	} else {
		t.Error("Expected server to be stopped")
	}
}

func TestServerDoubleStart(t *testing.T) {
	config := DefaultConfig()
	config.GRPCPort = "0" // Use any available port

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	go func() {
		server.Start()
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Try to start again
	err = server.Start()
	if err == nil {
		t.Error("Expected error when starting server twice")
	}

	// Cleanup
	server.Stop()
}

func TestStartFunction(t *testing.T) {
	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
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

	if atomic.LoadInt32(&serviceRegistrarCalled) == 0 {
		t.Error("Expected service registrar to be called")
	}

	// Note: We don't have a direct way to stop the server started by Start()
	// In a real test environment, you might use a test framework that can
	// handle process cleanup
}

func TestStartH2CFunction(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
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

	if atomic.LoadInt32(&serviceRegistrarCalled) == 0 {
		t.Error("Expected service registrar to be called")
	}
}

func TestStartSeparateFunction(t *testing.T) {
	// Get two available ports
	listener1, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	grpcPort := fmt.Sprintf("%d", listener1.Addr().(*net.TCPAddr).Port)
	listener1.Close()

	listener2, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
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

	if atomic.LoadInt32(&serviceRegistrarCalled) == 0 {
		t.Error("Expected service registrar to be called")
	}
}

func TestServerWithCustomShutdown(t *testing.T) {
	config := DefaultConfig()
	config.GRPCPort = "0"

	shutdownCalled := false
	config.Shutdown = func() error {
		shutdownCalled = true
		return nil
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start and immediately stop
	go func() {
		server.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	err = server.Stop()
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	if !shutdownCalled {
		t.Error("Expected custom shutdown handler to be called")
	}
}

func TestServerModeValidation(t *testing.T) {
	tests := []struct {
		name      string
		mode      ServerMode
		grpcPort  string
		httpPort  string
		expectErr bool
	}{
		{
			name:      "valid H2C mode",
			mode:      H2CMode,
			grpcPort:  "8080",
			httpPort:  "",
			expectErr: false,
		},
		{
			name:      "valid separate mode",
			mode:      SeparateMode,
			grpcPort:  "9090",
			httpPort:  "9091",
			expectErr: false,
		},
		{
			name:      "separate mode missing HTTP port",
			mode:      SeparateMode,
			grpcPort:  "9090",
			httpPort:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Mode:     tt.mode,
				GRPCPort: tt.grpcPort,
				HTTPPort: tt.httpPort,
			}

			err := config.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected error for invalid configuration")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Only proceed to create server if validation passed
			if err == nil {
				config.SetDefaults()
				_, err := New(config)
				if err != nil {
					t.Errorf("Unexpected error creating server: %v", err)
				}
			}
		})
	}
}

// Integration test with in-memory connection
func TestServerIntegration(t *testing.T) {
	// Create buffer connection for testing
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)

	config := Config{
		GRPCPort:          "8080",
		Mode:              H2CMode,
		EnableMetrics:     true,
		EnableHealthCheck: true,
		EnableReflection:  true,
	}
	config.SetDefaults()

	// Add a simple service registrar
	config.ServiceRegistrar = func(s *grpc.Server) {
		// No need to register reflection here since EnableReflection is true
		// The server will handle reflection registration automatically
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

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
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Test that connection is not in shutdown state immediately
	state := conn.GetState()
	assert.NotEqual(t, state.String(), "SHUTDOWN", "Connection should not be shutdown immediately")

	// Cleanup
	server.grpcServer.GracefulStop()
}

func TestServerTrackUptime(t *testing.T) {
	config := DefaultConfig()
	config.GRPCPort = "8080"
	config.EnableMetrics = true

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test uptime tracking
	go server.trackUptime()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Get metrics to verify uptime is being tracked
	mm := server.GetMetricsManager()
	metricFamilies, err := mm.GetRegistry().Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Look for uptime metric
	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "grpc_server_uptime_seconds" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected uptime metric to be present")
	}
}
