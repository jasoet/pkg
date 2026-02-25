package server

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/require"
)

func TestNewHttpServer(t *testing.T) {
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
	assert.False(t, operationCalled, "Operation should not be called during initialization")
	assert.False(t, shutdownCalled, "Shutdown should not be called during initialization")
}

func TestHealthEndpoints(t *testing.T) {
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
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

func TestOperationExecution(t *testing.T) {
	operationCh := make(chan bool, 1)
	operation := func(e *echo.Echo) {
		operationCh <- true
	}

	config := DefaultConfig(0, operation, func(e *echo.Echo) {})
	server := newHttpServer(config)

	err := server.start()
	require.NoError(t, err)

	select {
	case <-operationCh:
		// Operation was called
	case <-time.After(2 * time.Second):
		t.Fatal("Operation was not called within timeout")
	}

	_ = server.stop()
}

func TestShutdownExecution(t *testing.T) {
	shutdownCh := make(chan bool, 1)
	shutdown := func(e *echo.Echo) {
		shutdownCh <- true
	}

	config := DefaultConfig(0, func(e *echo.Echo) {}, shutdown)
	server := newHttpServer(config)

	err := server.start()
	require.NoError(t, err)

	_ = server.stop()

	select {
	case <-shutdownCh:
		// Shutdown was called
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown was not called within timeout")
	}
}

func TestNilCallbacks(t *testing.T) {
	// C9: nil Operation and Shutdown must not panic
	config := Config{
		Port:            0,
		ShutdownTimeout: 5 * time.Second,
	}
	server := newHttpServer(config)

	err := server.start()
	require.NoError(t, err)

	err = server.stop()
	assert.NoError(t, err)
}

func TestBindErrorDetection(t *testing.T) {
	// C10: bind errors are now detected immediately via net.Listen
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	s1 := newHttpServer(config)
	err := s1.start()
	require.NoError(t, err)

	// Get the actual port that s1 bound to
	addr := s1.echo.Listener.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	// Try to bind a second server on the same port — should fail immediately
	config2 := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	// Parse port string to int
	var portInt int
	_, _ = fmt.Sscanf(port, "%d", &portInt)
	config2.Port = portInt
	s2 := newHttpServer(config2)

	err = s2.start()
	assert.Error(t, err, "Second server should fail to bind on occupied port")
	assert.Contains(t, err.Error(), "failed to listen")

	_ = s1.stop()
}

func TestCustomMiddleware(t *testing.T) {
	middlewareCalled := false
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	}

	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.Middleware = []echo.MiddlewareFunc{middleware}
	e := setupEcho(config)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "Middleware should be called")
}

func TestNoHomeEndpoint(t *testing.T) {
	// I7: "/" handler was removed — library should not register opinionated routes
	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	e := setupEcho(config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Echo returns 404/405 for unregistered routes
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestIntegration(t *testing.T) {
	var operationCalled atomic.Bool
	var shutdownCalled atomic.Bool

	operation := func(e *echo.Echo) {
		operationCalled.Store(true)
	}

	shutdown := func(e *echo.Echo) {
		shutdownCalled.Store(true)
	}

	config := DefaultConfig(0, operation, shutdown)
	server := newHttpServer(config)

	err := server.start()
	require.NoError(t, err)

	assert.True(t, operationCalled.Load(), "Operation should be called after server start")

	// The listener is set immediately so we can read the address without polling.
	addr := server.echo.Listener.Addr().String()

	client := &http.Client{Timeout: 1 * time.Second}
	port := strings.Split(addr, ":")[1]
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:"+port+"/health", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)

	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"status":"UP"}`, strings.TrimSpace(string(body)))
	}

	err = server.stop()
	assert.NoError(t, err)
	assert.True(t, shutdownCalled.Load(), "Shutdown should be called after server stop")
}

func TestServerStartStop(t *testing.T) {
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

	config := DefaultConfig(0, operation, shutdown)
	server := newHttpServer(config)

	err := server.start()
	require.NoError(t, err)

	operationWg.Wait()

	err = server.stop()
	assert.NoError(t, err)

	shutdownWg.Wait()
}

func TestEchoConfigurer(t *testing.T) {
	var configurerCalled bool
	var customErrorHandlerCalled bool

	customErrorHandler := func(err error, c echo.Context) {
		customErrorHandlerCalled = true
		_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	configurer := func(e *echo.Echo) {
		configurerCalled = true
		e.HTTPErrorHandler = customErrorHandler
	}

	config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
	config.EchoConfigurer = configurer

	e := setupEcho(config)

	assert.True(t, configurerCalled, "EchoConfigurer should be called during setupEcho")

	req := httptest.NewRequest(http.MethodGet, "/non-existent-path", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, customErrorHandlerCalled, "Custom error handler should be called for non-existent path")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "error")
}

func TestStartFunction(t *testing.T) {
	t.Run("Start function with default config", func(t *testing.T) {
		var operationCalled atomic.Bool
		operation := func(e *echo.Echo) {
			operationCalled.Store(true)
		}

		var shutdownCalled atomic.Bool
		shutdown := func(e *echo.Echo) {
			shutdownCalled.Store(true)
		}

		middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}

		config := DefaultConfig(0, operation, shutdown)
		config.Middleware = []echo.MiddlewareFunc{middleware}

		server := newHttpServer(config)
		assert.NotNil(t, server)
		assert.Equal(t, 0, server.config.Port)
		assert.Len(t, server.config.Middleware, 1)

		err := server.start()
		require.NoError(t, err)
		assert.True(t, operationCalled.Load(), "Operation should be called")

		_ = server.stop()
		assert.True(t, shutdownCalled.Load(), "Shutdown should be called")
	})
}

func TestStartWithConfigFunction(t *testing.T) {
	t.Run("StartWithConfig creates server correctly", func(t *testing.T) {
		var operationCalled atomic.Bool
		operation := func(e *echo.Echo) {
			operationCalled.Store(true)
		}

		var shutdownCalled atomic.Bool
		shutdown := func(e *echo.Echo) {
			shutdownCalled.Store(true)
		}

		config := DefaultConfig(0, operation, shutdown)
		config.ShutdownTimeout = 5 * time.Second

		server := newHttpServer(config)
		assert.NotNil(t, server)
		assert.Equal(t, 0, server.config.Port)
		assert.Equal(t, 5*time.Second, server.config.ShutdownTimeout)

		err := server.start()
		require.NoError(t, err)
		assert.True(t, operationCalled.Load(), "Operation should be called during start")

		err = server.stop()
		assert.NoError(t, err, "Stop should not error")
		assert.True(t, shutdownCalled.Load(), "Shutdown should be called during stop")
	})

	t.Run("StartWithConfig with custom shutdown timeout", func(t *testing.T) {
		customTimeout := 15 * time.Second
		config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.ShutdownTimeout = customTimeout

		server := newHttpServer(config)
		assert.Equal(t, customTimeout, server.config.ShutdownTimeout)
	})
}

func TestNewConfig(t *testing.T) {
	var opCalled bool
	cfg := NewConfig(
		WithPort(9090),
		WithOperation(func(e *echo.Echo) { opCalled = true }),
		WithShutdownTimeout(20*time.Second),
	)

	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, 20*time.Second, cfg.ShutdownTimeout)
	assert.NotNil(t, cfg.Operation)
	cfg.Operation(nil) // invoke to verify it was set
	assert.True(t, opCalled)
	assert.Nil(t, cfg.Shutdown, "Shutdown should be nil when not set")
}

func TestWithOptions(t *testing.T) {
	mw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { return next(c) }
	}

	cfg := NewConfig(
		WithMiddleware(mw),
		WithEchoConfigurer(func(e *echo.Echo) {}),
	)

	assert.Len(t, cfg.Middleware, 1)
	assert.NotNil(t, cfg.EchoConfigurer)
}
