package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewHttpServer(t *testing.T) {
	// Test server initialization with default config
	operationCalled := false
	shutdownCalled := false

	operation := func(e *echo.Echo) {
		operationCalled = true
	}

	shutdown := func(e *echo.Echo) {
		shutdownCalled = true
	}

	config := DefaultConfig(8080, operation, shutdown)
	server := newHttpServer(config)

	assert.NotNil(t, server)
	assert.NotNil(t, server.echo)
	assert.Equal(t, config.Port, server.config.Port)
	assert.Equal(t, config.EnableMetrics, server.config.EnableMetrics)
	assert.False(t, operationCalled, "Operation should not be called during initialization")
	assert.False(t, shutdownCalled, "Shutdown should not be called during initialization")
}

func TestHealthEndpoints(t *testing.T) {
	// Test health check endpoints
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.MetricsSubsystem = "testHealthEndpoints"
	e := setupEcho(config)

	// Test /health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"UP"}`, strings.TrimSpace(rec.Body.String()))

	// Test /health/ready endpoint
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"READY"}`, strings.TrimSpace(rec.Body.String()))

	// Test /health/live endpoint
	req = httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"ALIVE"}`, strings.TrimSpace(rec.Body.String()))
}

func TestMetricsEndpoint(t *testing.T) {
	// Test metrics endpoint with metrics enabled
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.MetricsSubsystem = "TestMetricsEndpoint"
	config.EnableMetrics = true
	e := setupEcho(config)

	// Test /metrics endpoint
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "# HELP")

	// Test metrics endpoint with metrics disabled
	config.EnableMetrics = false
	e = setupEcho(config)

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestOperationExecution(t *testing.T) {
	// Test that operation is executed when server starts
	operationCh := make(chan bool, 1)
	operation := func(e *echo.Echo) {
		operationCh <- true
	}

	config := DefaultConfig(0, operation, func(e *echo.Echo) {})
	config.MetricsSubsystem = "TestOperationExecution"
	server := newHttpServer(config)

	// Start the server
	server.start()

	// Wait for operation to be called or timeout
	select {
	case <-operationCh:
		// Operation was called
	case <-time.After(2 * time.Second):
		t.Fatal("Operation was not called within timeout")
	}

	// Stop the server
	_ = server.stop()
}

func TestShutdownExecution(t *testing.T) {
	// Test that shutdown is executed when server stops
	shutdownCh := make(chan bool, 1)
	shutdown := func(e *echo.Echo) {
		shutdownCh <- true
	}

	config := DefaultConfig(0, func(e *echo.Echo) {}, shutdown)
	config.MetricsSubsystem = "TestShutdownExecution"
	server := newHttpServer(config)

	// Start the server
	server.start()

	// Stop the server
	_ = server.stop()

	// Wait for shutdown to be called or timeout
	select {
	case <-shutdownCh:
		// Shutdown was called
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown was not called within timeout")
	}
}

func TestHomeEndpoint(t *testing.T) {
	// Test home endpoint
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.MetricsSubsystem = "TestHomeEndpoint"
	e := setupEcho(config)

	// Test / endpoint
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Home", rec.Body.String())
}

func TestCustomMiddleware(t *testing.T) {
	// Test custom middleware
	middlewareCalled := false
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	}

	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.MetricsSubsystem = "TestCustomMiddleware"
	config.Middleware = []echo.MiddlewareFunc{middleware}
	e := setupEcho(config)

	// Test any endpoint to trigger middleware
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "Middleware should be called")
}

func TestIntegration(t *testing.T) {
	t.Skip("Skipping integration test due to race condition in Echo framework")
	// Integration test that simulates a real server lifecycle
	var operationCalled atomic.Bool
	var shutdownCalled atomic.Bool
	serverReady := make(chan string, 1)

	operation := func(e *echo.Echo) {
		operationCalled.Store(true)
		// Wait for server to be fully ready
		go func() {
			for i := 0; i < 20; i++ {
				if e.Listener != nil {
					serverReady <- e.Listener.Addr().String()
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
			serverReady <- ""
		}()
	}

	shutdown := func(e *echo.Echo) {
		shutdownCalled.Store(true)
	}

	// Create a server with a random port
	config := DefaultConfig(0, operation, shutdown)
	config.MetricsSubsystem = "TestIntegration"
	server := newHttpServer(config)

	// Start the server
	server.start()

	// Wait for server to be ready
	select {
	case addr := <-serverReady:
		if addr == "" {
			t.Fatal("Server listener not ready")
		}

		assert.True(t, operationCalled.Load(), "Operation should be called after server start")

		// Make a request to the server
		client := &http.Client{
			Timeout: 1 * time.Second,
		}

		// Get the actual port that was assigned
		port := strings.Split(addr, ":")[1]
		resp, err := client.Get("http://localhost:" + port + "/health")

		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, `{"status":"UP"}`, strings.TrimSpace(string(body)))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for server to be ready")
	}

	// Stop the server
	err := server.stop()
	assert.NoError(t, err)
	assert.True(t, shutdownCalled.Load(), "Shutdown should be called after server stopFunc")
}

func TestServerStartStop(t *testing.T) {
	// Test server start and stopFunc
	var operationWg sync.WaitGroup
	operationWg.Add(1)

	var shutdownWg sync.WaitGroup
	shutdownWg.Add(1)

	operation := func(e *echo.Echo) {
		operationWg.Done()
	}

	shutdown := func(e *echo.Echo) {
		shutdownWg.Done()
	}

	config := DefaultConfig(0, operation, shutdown) // Use port 0 to get a random available port
	config.MetricsSubsystem = "TestServerStartStop"
	server := newHttpServer(config)

	// Start the server
	server.start()

	// Wait for operation to be called
	operationWg.Wait()

	// Stop the server
	err := server.stop()
	assert.NoError(t, err)

	// Wait for shutdown to be called
	shutdownWg.Wait()
}

func TestEchoConfigurer(t *testing.T) {
	// Test that EchoConfigurer is called and can modify the Echo instance
	var configurerCalled bool
	var customErrorHandlerCalled bool

	// Custom error handler for testing
	customErrorHandler := func(err error, c echo.Context) {
		customErrorHandlerCalled = true
		_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create a configurer that sets a custom error handler
	configurer := func(e *echo.Echo) {
		configurerCalled = true
		e.HTTPErrorHandler = customErrorHandler
	}

	// Create a config with the configurer
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.MetricsSubsystem = "TestEchoConfigurer"
	config.EchoConfigurer = configurer

	// Setup Echo with the config
	e := setupEcho(config)

	// Verify that the configurer was called
	assert.True(t, configurerCalled, "EchoConfigurer should be called during setupEcho")

	// Test that the custom error handler is used
	req := httptest.NewRequest(http.MethodGet, "/non-existent-path", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Verify that the custom error handler was called
	assert.True(t, customErrorHandlerCalled, "Custom error handler should be called for non-existent path")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "error")
}
