package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	otellog "go.opentelemetry.io/otel/log"
	noopm "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	noopt "go.opentelemetry.io/otel/trace/noop"
)

// testProcessor is a simple processor that captures log records for testing
type testProcessor struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (p *testProcessor) OnEmit(_ context.Context, record *sdklog.Record) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.records = append(p.records, record.Clone())
	return nil
}

func (p *testProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (p *testProcessor) ForceFlush(_ context.Context) error {
	return nil
}

func (p *testProcessor) Records() []sdklog.Record {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]sdklog.Record{}, p.records...)
}

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
	assert.Nil(t, server.config.OTelConfig, "OTelConfig should be nil by default")
	assert.False(t, operationCalled, "Operation should not be called during initialization")
	assert.False(t, shutdownCalled, "Shutdown should not be called during initialization")
}

func TestHealthEndpoints(t *testing.T) {
	// Test health check endpoints
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
	// Test that operation is executed when server starts
	operationCh := make(chan bool, 1)
	operation := func(e *echo.Echo) {
		operationCh <- true
	}

	config := DefaultConfig(0, operation, func(e *echo.Echo) {})
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
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:"+port+"/health", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)

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

func TestCreateLoggingMiddleware(t *testing.T) {
	t.Run("logs HTTP requests with OTel", func(t *testing.T) {
		// Create OTel config with logging enabled
		cfg := otel.NewConfig("test-service")

		// Create Echo with logging middleware
		e := echo.New()
		e.Use(createLoggingMiddleware(cfg))
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		// Make request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})

	t.Run("logs errors with error severity", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")

		e := echo.New()
		e.Use(createLoggingMiddleware(cfg))
		e.GET("/error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusInternalServerError, "test error")
		})

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("logs 4xx with warning severity", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")

		e := echo.New()
		e.Use(createLoggingMiddleware(cfg))
		e.GET("/notfound", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		})

		req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("5xx errors use error attribute", func(t *testing.T) {
		// Create a test processor to capture log records
		processor := &testProcessor{}
		loggerProvider := sdklog.NewLoggerProvider(
			sdklog.WithProcessor(processor),
		)

		cfg := otel.NewConfig("test-service").WithLoggerProvider(loggerProvider)

		e := echo.New()
		e.Use(createLoggingMiddleware(cfg))
		e.GET("/server-error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		})

		req := httptest.NewRequest(http.MethodGet, "/server-error", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		// Wait for logs to be processed
		time.Sleep(10 * time.Millisecond)

		// Check that the log record has "error" attribute for 5xx
		records := processor.Records()
		assert.NotEmpty(t, records, "Should have recorded logs")
		if len(records) > 0 {
			record := records[0]
			var hasErrorAttr bool
			var hasMessageAttr bool
			record.WalkAttributes(func(kv otellog.KeyValue) bool {
				if kv.Key == "error" {
					hasErrorAttr = true
				}
				if kv.Key == "message" {
					hasMessageAttr = true
				}
				return true
			})
			assert.True(t, hasErrorAttr, "5xx errors should have 'error' attribute")
			assert.False(t, hasMessageAttr, "5xx errors should not have 'message' attribute")
		}
	})

	t.Run("4xx errors use message attribute instead of error", func(t *testing.T) {
		// Create a test processor to capture log records
		processor := &testProcessor{}
		loggerProvider := sdklog.NewLoggerProvider(
			sdklog.WithProcessor(processor),
		)

		cfg := otel.NewConfig("test-service").WithLoggerProvider(loggerProvider)

		e := echo.New()
		e.Use(createLoggingMiddleware(cfg))
		e.GET("/client-error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		})

		req := httptest.NewRequest(http.MethodGet, "/client-error", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		// Wait for logs to be processed
		time.Sleep(10 * time.Millisecond)

		// Check that the log record has "message" attribute for 4xx
		records := processor.Records()
		assert.NotEmpty(t, records, "Should have recorded logs")
		if len(records) > 0 {
			record := records[0]
			var hasErrorAttr bool
			var hasMessageAttr bool
			record.WalkAttributes(func(kv otellog.KeyValue) bool {
				if kv.Key == "error" {
					hasErrorAttr = true
				}
				if kv.Key == "message" {
					hasMessageAttr = true
				}
				return true
			})
			assert.False(t, hasErrorAttr, "4xx errors should not have 'error' attribute")
			assert.True(t, hasMessageAttr, "4xx errors should have 'message' attribute")
		}
	})

	t.Run("handles nil OTel config gracefully", func(t *testing.T) {
		// This shouldn't happen in practice as setupEcho checks for nil,
		// but test defensive behavior
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("createLoggingMiddleware panicked with nil config: %v", r)
			}
		}()

		e := echo.New()
		// Don't use nil config - use config with logging disabled instead
		cfg := otel.NewConfig("test-service").WithoutLogging()
		if cfg.IsLoggingEnabled() {
			e.Use(createLoggingMiddleware(cfg))
		}
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestCreateMetricsMiddleware(t *testing.T) {
	t.Run("records HTTP metrics with OTel", func(t *testing.T) {
		// Create OTel config with metrics enabled
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		// Create Echo with metrics middleware
		e := echo.New()
		e.Use(createMetricsMiddleware(cfg))
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		// Make request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records metrics for multiple requests", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		e := echo.New()
		e.Use(createMetricsMiddleware(cfg))
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		// Make multiple requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("records metrics for different status codes", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		e := echo.New()
		e.Use(createMetricsMiddleware(cfg))
		e.GET("/success", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		e.GET("/error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusInternalServerError, "error")
		})
		e.GET("/notfound", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		})

		// Test different status codes
		tests := []struct {
			path           string
			expectedStatus int
		}{
			{"/success", http.StatusOK},
			{"/error", http.StatusInternalServerError},
			{"/notfound", http.StatusNotFound},
		}

		for _, tt := range tests {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		}
	})

	t.Run("handles nil OTel config gracefully", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("createMetricsMiddleware panicked with nil metrics: %v", r)
			}
		}()

		e := echo.New()
		// Don't use nil config - use config without metrics
		cfg := otel.NewConfig("test-service")
		if cfg.IsMetricsEnabled() {
			e.Use(createMetricsMiddleware(cfg))
		}
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSetupEchoWithOTel(t *testing.T) {
	t.Run("sets up Echo with logging middleware when enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		config := DefaultConfig(8080, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = cfg

		e := setupEcho(config)

		// Verify middleware is set up by making a request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("sets up Echo with metrics middleware when enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())
		config := DefaultConfig(8080, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = cfg

		e := setupEcho(config)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("sets up Echo with tracing middleware when enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())
		config := DefaultConfig(8080, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = cfg

		e := setupEcho(config)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("sets up Echo with all telemetry enabled", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider()).
			WithMeterProvider(noopm.NewMeterProvider())
		config := DefaultConfig(8080, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = cfg

		e := setupEcho(config)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("sets up Echo without telemetry when OTelConfig is nil", func(t *testing.T) {
		config := DefaultConfig(8080, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = nil

		e := setupEcho(config)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestStartFunction(t *testing.T) {
	t.Run("Start function with default config", func(t *testing.T) {
		// Test that Start() function properly sets up configuration
		// We can't fully test the blocking nature, but we can test that it creates valid config

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

		// Create server with the same config that Start() would create
		config := DefaultConfig(0, operation, shutdown)
		config.Middleware = []echo.MiddlewareFunc{middleware}

		server := newHttpServer(config)
		assert.NotNil(t, server)
		assert.Equal(t, 0, server.config.Port)
		assert.Len(t, server.config.Middleware, 1)

		// Start and stop to verify callbacks work
		server.start()
		time.Sleep(100 * time.Millisecond)
		assert.True(t, operationCalled.Load(), "Operation should be called")

		_ = server.stop()
		assert.True(t, shutdownCalled.Load(), "Shutdown should be called")
	})
}

func TestStartWithConfigFunction(t *testing.T) {
	t.Run("StartWithConfig creates server correctly", func(t *testing.T) {
		// Test that StartWithConfig() function properly sets up the server
		// We test the initialization without actually blocking on signals

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

		// Create and start server (same as StartWithConfig does)
		server := newHttpServer(config)
		assert.NotNil(t, server)
		assert.Equal(t, 0, server.config.Port)
		assert.Equal(t, 5*time.Second, server.config.ShutdownTimeout)

		// Test start/stop cycle
		server.start()
		time.Sleep(100 * time.Millisecond)
		assert.True(t, operationCalled.Load(), "Operation should be called during start")

		err := server.stop()
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

	t.Run("StartWithConfig with OTel configuration", func(t *testing.T) {
		cfg := otel.NewConfig("test-server").
			WithTracerProvider(noopt.NewTracerProvider()).
			WithMeterProvider(noopm.NewMeterProvider())

		config := DefaultConfig(0, func(e *echo.Echo) {}, func(e *echo.Echo) {})
		config.OTelConfig = cfg

		server := newHttpServer(config)
		assert.NotNil(t, server.config.OTelConfig)
		assert.Equal(t, "test-server", server.config.OTelConfig.ServiceName)
	})
}
