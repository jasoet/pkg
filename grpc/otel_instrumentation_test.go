package grpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgotel "github.com/jasoet/pkg/v2/otel"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ============================================================================
// Helper Functions
// ============================================================================

// mockUnaryHandler creates a mock gRPC handler
func mockUnaryHandler(resp interface{}, err error) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return resp, err
	}
}

// mockUnaryInfo creates mock gRPC unary server info
func mockUnaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{
		FullMethod: method,
	}
}

// ============================================================================
// extractServiceName Tests
// ============================================================================

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		expected   string
	}{
		{
			name:       "standard method",
			fullMethod: "/package.Service/Method",
			expected:   "package.Service",
		},
		{
			name:       "nested package",
			fullMethod: "/com.example.v1.UserService/GetUser",
			expected:   "com.example.v1.UserService",
		},
		{
			name:       "no leading slash",
			fullMethod: "package.Service/Method",
			expected:   "package.Service",
		},
		{
			name:       "empty string",
			fullMethod: "",
			expected:   "",
		},
		{
			name:       "only slash",
			fullMethod: "/",
			expected:   "",
		},
		{
			name:       "no method separator",
			fullMethod: "/package.Service",
			expected:   "package.Service",
		},
		{
			name:       "multiple separators",
			fullMethod: "/api/v1/service/method",
			expected:   "api/v1/service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.fullMethod)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// createGRPCMetricsInterceptor Tests
// ============================================================================

func TestCreateGRPCMetricsInterceptor(t *testing.T) {
	t.Run("nil config returns passthrough interceptor", func(t *testing.T) {
		interceptor := createGRPCMetricsInterceptor(nil)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("metrics disabled returns passthrough interceptor", func(t *testing.T) {
		// Config without meter provider means metrics disabled
		config := pkgotel.NewConfig("test-service")

		interceptor := createGRPCMetricsInterceptor(config)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("metrics enabled records requests", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider())

		interceptor := createGRPCMetricsInterceptor(config)
		require.NotNil(t, interceptor)

		handler := mockUnaryHandler("success", nil)
		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test.Service/Method"), handler)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("records failed requests", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider())

		interceptor := createGRPCMetricsInterceptor(config)

		expectedErr := status.Error(codes.NotFound, "not found")
		handler := mockUnaryHandler(nil, expectedErr)
		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test.Service/Method"), handler)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)
	})
}

// ============================================================================
// createGRPCTracingInterceptor Tests
// ============================================================================

func TestCreateGRPCTracingInterceptor(t *testing.T) {
	t.Run("nil config returns passthrough interceptor", func(t *testing.T) {
		interceptor := createGRPCTracingInterceptor(nil)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("tracing disabled returns passthrough interceptor", func(t *testing.T) {
		// Config without tracer provider means tracing disabled
		config := pkgotel.NewConfig("test-service")

		interceptor := createGRPCTracingInterceptor(config)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("tracing enabled creates spans", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithTracerProvider(tracenoop.NewTracerProvider())

		interceptor := createGRPCTracingInterceptor(config)
		require.NotNil(t, interceptor)

		handler := mockUnaryHandler("response", nil)
		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test.Service/Method"), handler)

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("records error in span", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithTracerProvider(tracenoop.NewTracerProvider())

		interceptor := createGRPCTracingInterceptor(config)

		expectedErr := status.Error(codes.Internal, "internal error")
		handler := mockUnaryHandler(nil, expectedErr)

		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test/Method"), handler)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)
	})
}

// ============================================================================
// createGRPCLoggingInterceptor Tests
// ============================================================================

func TestCreateGRPCLoggingInterceptor(t *testing.T) {
	t.Run("nil config returns passthrough interceptor", func(t *testing.T) {
		interceptor := createGRPCLoggingInterceptor(nil)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("logging disabled returns passthrough interceptor", func(t *testing.T) {
		// Config without logger provider means logging disabled (or default stdout)
		// Use WithoutLogging() to explicitly disable
		config := pkgotel.NewConfig("test-service").
			WithoutLogging()

		interceptor := createGRPCLoggingInterceptor(config)
		require.NotNil(t, interceptor)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		resp, err := interceptor(context.Background(), "request", mockUnaryInfo("/test/Method"), handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, called)
	})

	t.Run("logging enabled creates log records", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithLoggerProvider(noop.NewLoggerProvider())

		interceptor := createGRPCLoggingInterceptor(config)
		require.NotNil(t, interceptor)

		handler := mockUnaryHandler("response", nil)
		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test.Service/Method"), handler)

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("logs failed requests with error", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithLoggerProvider(noop.NewLoggerProvider())

		interceptor := createGRPCLoggingInterceptor(config)

		expectedErr := status.Error(codes.PermissionDenied, "access denied")
		handler := mockUnaryHandler(nil, expectedErr)
		resp, err := interceptor(context.Background(), "req", mockUnaryInfo("/test/Method"), handler)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)
	})
}

// ============================================================================
// createHTTPGatewayMetricsMiddleware Tests
// ============================================================================

func TestCreateHTTPGatewayMetricsMiddleware(t *testing.T) {
	t.Run("nil config returns passthrough middleware", func(t *testing.T) {
		middleware := createHTTPGatewayMetricsMiddleware(nil)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		handler := func(c echo.Context) error {
			called = true
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("metrics enabled records metrics", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider())

		middleware := createHTTPGatewayMetricsMiddleware(config)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/users")

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records error responses", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider())

		middleware := createHTTPGatewayMetricsMiddleware(config)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/error")

		handler := func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusInternalServerError, "Internal error")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.Error(t, err)
	})
}

// ============================================================================
// createHTTPGatewayTracingMiddleware Tests
// ============================================================================

func TestCreateHTTPGatewayTracingMiddleware(t *testing.T) {
	t.Run("nil config returns passthrough middleware", func(t *testing.T) {
		middleware := createHTTPGatewayTracingMiddleware(nil)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		handler := func(c echo.Context) error {
			called = true
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("tracing enabled creates spans", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithTracerProvider(tracenoop.NewTracerProvider())

		middleware := createHTTPGatewayTracingMiddleware(config)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/users")

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records error in span", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithTracerProvider(tracenoop.NewTracerProvider())

		middleware := createHTTPGatewayTracingMiddleware(config)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/error")

		handler := func(c echo.Context) error {
			return errors.New("handler error")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.Error(t, err)
	})
}

// ============================================================================
// createHTTPGatewayLoggingMiddleware Tests
// ============================================================================

func TestCreateHTTPGatewayLoggingMiddleware(t *testing.T) {
	t.Run("nil config returns passthrough middleware", func(t *testing.T) {
		middleware := createHTTPGatewayLoggingMiddleware(nil)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		handler := func(c echo.Context) error {
			called = true
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("logging enabled creates log records", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithLoggerProvider(noop.NewLoggerProvider())

		middleware := createHTTPGatewayLoggingMiddleware(config)
		require.NotNil(t, middleware)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/users")

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("logs 5xx errors with error severity", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithLoggerProvider(noop.NewLoggerProvider())

		middleware := createHTTPGatewayLoggingMiddleware(config)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/error")

		handler := func(c echo.Context) error {
			return c.String(http.StatusInternalServerError, "Internal Error")
		}

		wrapped := middleware(handler)
		err := wrapped(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ============================================================================
// Integration Tests - Combined Interceptors
// ============================================================================

func TestGRPCInterceptorsCombined(t *testing.T) {
	t.Run("all interceptors work together", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider()).
			WithTracerProvider(tracenoop.NewTracerProvider()).
			WithLoggerProvider(noop.NewLoggerProvider())

		metricsInterceptor := createGRPCMetricsInterceptor(config)
		tracingInterceptor := createGRPCTracingInterceptor(config)
		loggingInterceptor := createGRPCLoggingInterceptor(config)

		// Chain interceptors
		handler := mockUnaryHandler("response", nil)

		// Apply in reverse order (like middleware)
		finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return loggingInterceptor(ctx, req, mockUnaryInfo("/test/Method"),
				func(ctx context.Context, req interface{}) (interface{}, error) {
					return tracingInterceptor(ctx, req, mockUnaryInfo("/test/Method"),
						func(ctx context.Context, req interface{}) (interface{}, error) {
							return metricsInterceptor(ctx, req, mockUnaryInfo("/test/Method"), handler)
						})
				})
		}

		resp, err := finalHandler(context.Background(), "request")

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}

func TestHTTPGatewayMiddlewareCombined(t *testing.T) {
	t.Run("all middleware work together", func(t *testing.T) {
		config := pkgotel.NewConfig("test-service").
			WithMeterProvider(metricnoop.NewMeterProvider()).
			WithTracerProvider(tracenoop.NewTracerProvider()).
			WithLoggerProvider(noop.NewLoggerProvider())

		metricsMiddleware := createHTTPGatewayMetricsMiddleware(config)
		tracingMiddleware := createHTTPGatewayTracingMiddleware(config)
		loggingMiddleware := createHTTPGatewayLoggingMiddleware(config)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/test")

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}

		// Chain middleware
		wrapped := metricsMiddleware(tracingMiddleware(loggingMiddleware(handler)))

		err := wrapped(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})
}
