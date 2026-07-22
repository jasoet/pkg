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

func TestNewHTTPServer(t *testing.T) {
	operationCalled := false
	shutdownCalled := false

	operation := func(e *echo.Echo) {
		operationCalled = true
	}

	shutdown := func(e *echo.Echo) {
		shutdownCalled = true
	}

	srv, err := New(WithPort(8080), WithOperation(operation), WithShutdown(shutdown))
	require.NoError(t, err)

	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Echo())
	assert.Equal(t, 8080, srv.config.Port)
	assert.False(t, operationCalled, "Operation should not be called during initialization")
	assert.False(t, shutdownCalled, "Shutdown should not be called during initialization")
}

func TestHealthEndpoints(t *testing.T) {
	config := NewConfig(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
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

	srv, err := New(WithPort(0), WithOperation(operation), WithShutdown(func(e *echo.Echo) {}))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	select {
	case <-operationCh:
		// Operation was called
	case <-time.After(2 * time.Second):
		t.Fatal("Operation was not called within timeout")
	}

	require.NoError(t, srv.Shutdown(context.Background()))
	assert.NoError(t, <-startErr)
}

func TestShutdownExecution(t *testing.T) {
	shutdownCh := make(chan bool, 1)
	shutdown := func(e *echo.Echo) {
		shutdownCh <- true
	}

	srv, err := New(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(shutdown))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()
	waitForAddr(t, srv)

	require.NoError(t, srv.Shutdown(context.Background()))
	assert.NoError(t, <-startErr)

	select {
	case <-shutdownCh:
		// Shutdown was called
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown was not called within timeout")
	}
}

func TestNilCallbacks(t *testing.T) {
	// C9: nil Operation and Shutdown must not panic
	srv, err := New(WithPort(0), WithShutdownTimeout(5*time.Second))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()
	waitForAddr(t, srv)

	assert.NoError(t, srv.Shutdown(context.Background()))
	assert.NoError(t, <-startErr)
}

func TestBindErrorDetection(t *testing.T) {
	// C10: bind errors are now detected immediately via net.Listen
	s1, err := New(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- s1.Start() }()
	addr := waitForAddr(t, s1)

	// Get the actual port that s1 bound to
	var portInt int
	_, _ = fmt.Sscanf(addrPort(addr), "%d", &portInt)

	// Try to bind a second server on the same port — should fail immediately
	s2, err := New(WithPort(portInt), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
	require.NoError(t, err)

	err = s2.Start()
	assert.Error(t, err, "Second server should fail to bind on occupied port")
	assert.Contains(t, err.Error(), "failed to listen")

	require.NoError(t, s1.Shutdown(context.Background()))
	assert.NoError(t, <-startErr)
}

func TestCustomMiddleware(t *testing.T) {
	middlewareCalled := false
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	}

	config := NewConfig(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
	config.Middleware = []echo.MiddlewareFunc{middleware}
	e := setupEcho(config)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "Middleware should be called")
}

func TestNoHomeEndpoint(t *testing.T) {
	// I7: "/" handler was removed — library should not register opinionated routes
	config := NewConfig(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
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

	srv, err := New(WithPort(0), WithOperation(operation), WithShutdown(shutdown))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	// Operation runs before the listener is bound, so once Addr is non-empty
	// the Operation callback has completed and the address is safe to read.
	addr := waitForAddr(t, srv)
	assert.True(t, operationCalled.Load(), "Operation should be called after server start")

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:"+addrPort(addr)+"/health", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"status":"UP"}`, strings.TrimSpace(string(body)))

	err = srv.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, shutdownCalled.Load(), "Shutdown should be called after server shutdown")
	assert.NoError(t, <-startErr)
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

	srv, err := New(WithPort(0), WithOperation(operation), WithShutdown(shutdown))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	operationWg.Wait()

	err = srv.Shutdown(context.Background())
	assert.NoError(t, err)

	shutdownWg.Wait()
	assert.NoError(t, <-startErr)
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

	config := NewConfig(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
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

func TestNewServerLifecycle(t *testing.T) {
	t.Run("New with options and full lifecycle", func(t *testing.T) {
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

		srv, err := New(
			WithPort(0),
			WithOperation(operation),
			WithShutdown(shutdown),
			WithMiddleware(middleware),
		)
		require.NoError(t, err)
		assert.NotNil(t, srv)
		assert.Equal(t, 0, srv.config.Port)
		assert.Len(t, srv.config.Middleware, 1)

		startErr := make(chan error, 1)
		go func() { startErr <- srv.Start() }()
		waitForAddr(t, srv)
		assert.True(t, operationCalled.Load(), "Operation should be called")

		assert.NoError(t, srv.Shutdown(context.Background()))
		assert.True(t, shutdownCalled.Load(), "Shutdown should be called")
		assert.NoError(t, <-startErr)
	})

	t.Run("New with custom shutdown timeout", func(t *testing.T) {
		customTimeout := 15 * time.Second
		srv, err := New(WithPort(0), WithShutdownTimeout(customTimeout))
		require.NoError(t, err)
		assert.Equal(t, customTimeout, srv.config.ShutdownTimeout)
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

func TestSetupEcho_HasTimeouts(t *testing.T) {
	config := NewConfig(WithPort(0), WithOperation(func(e *echo.Echo) {}), WithShutdown(func(e *echo.Echo) {}))
	e := setupEcho(config)

	assert.Equal(t, 5*time.Second, e.Server.ReadHeaderTimeout, "ReadHeaderTimeout should be 5s")
	assert.Equal(t, 30*time.Second, e.Server.ReadTimeout, "ReadTimeout should be 30s")
	assert.Equal(t, 30*time.Second, e.Server.WriteTimeout, "WriteTimeout should be 30s")
	assert.Equal(t, 120*time.Second, e.Server.IdleTimeout, "IdleTimeout should be 120s")
}
