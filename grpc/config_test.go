package grpc

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg, err := newConfig()
	require.NoError(t, err)

	// Test default values
	assert.Equal(t, "8080", cfg.grpcPort)
	assert.Equal(t, "8081", cfg.httpPort)
	assert.Equal(t, H2CMode, cfg.mode)
	assert.Equal(t, 30*time.Second, cfg.shutdownTimeout)
	assert.Equal(t, 5*time.Second, cfg.readTimeout)
	assert.Equal(t, 10*time.Second, cfg.writeTimeout)
	assert.Equal(t, 60*time.Second, cfg.idleTimeout)
	assert.Equal(t, 15*time.Minute, cfg.maxConnectionIdle)
	assert.Equal(t, 30*time.Minute, cfg.maxConnectionAge)
	assert.Equal(t, 5*time.Second, cfg.maxConnectionAgeGrace)

	// Test feature flags
	assert.True(t, cfg.enableMetrics)
	assert.True(t, cfg.enableHealthCheck)
	assert.True(t, cfg.enableLogging)
	assert.True(t, cfg.enableReflection)
	assert.False(t, cfg.enableCORS)
	assert.False(t, cfg.enableRateLimit)

	// Test paths
	assert.Equal(t, "/metrics", cfg.metricsPath)
	assert.Equal(t, "/health", cfg.healthPath)
	assert.Equal(t, "/api/v1", cfg.gatewayBasePath)

	// Test other defaults
	assert.Equal(t, 100.0, cfg.rateLimit)
	assert.False(t, cfg.enableTLS)
	assert.Empty(t, cfg.middleware)
}

func TestWithGRPCPort(t *testing.T) {
	cfg, err := newConfig(WithGRPCPort("9090"))
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.grpcPort)
}

func TestWithHTTPPort(t *testing.T) {
	cfg, err := newConfig(WithHTTPPort("9091"))
	require.NoError(t, err)
	assert.Equal(t, "9091", cfg.httpPort)
}

func TestWithH2CMode(t *testing.T) {
	cfg, err := newConfig(WithH2CMode())
	require.NoError(t, err)
	assert.Equal(t, H2CMode, cfg.mode)
}

func TestWithSeparateMode(t *testing.T) {
	cfg, err := newConfig(WithSeparateMode("9090", "9091"))
	require.NoError(t, err)
	assert.Equal(t, SeparateMode, cfg.mode)
	assert.Equal(t, "9090", cfg.grpcPort)
	assert.Equal(t, "9091", cfg.httpPort)
}

func TestWithTimeouts(t *testing.T) {
	cfg, err := newConfig(
		WithShutdownTimeout(45*time.Second),
		WithReadTimeout(15*time.Second),
		WithWriteTimeout(20*time.Second),
		WithIdleTimeout(90*time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, 45*time.Second, cfg.shutdownTimeout)
	assert.Equal(t, 15*time.Second, cfg.readTimeout)
	assert.Equal(t, 20*time.Second, cfg.writeTimeout)
	assert.Equal(t, 90*time.Second, cfg.idleTimeout)
}

func TestWithConnectionTimeouts(t *testing.T) {
	cfg, err := newConfig(WithConnectionTimeouts(20*time.Minute, 40*time.Minute, 10*time.Second))
	require.NoError(t, err)
	assert.Equal(t, 20*time.Minute, cfg.maxConnectionIdle)
	assert.Equal(t, 40*time.Minute, cfg.maxConnectionAge)
	assert.Equal(t, 10*time.Second, cfg.maxConnectionAgeGrace)
}

func TestWithMaxConnectionIdle(t *testing.T) {
	cfg, err := newConfig(WithMaxConnectionIdle(25 * time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 25*time.Minute, cfg.maxConnectionIdle)
}

func TestWithMaxConnectionAge(t *testing.T) {
	cfg, err := newConfig(WithMaxConnectionAge(45 * time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 45*time.Minute, cfg.maxConnectionAge)
}

func TestWithMaxConnectionAgeGrace(t *testing.T) {
	cfg, err := newConfig(WithMaxConnectionAgeGrace(15 * time.Second))
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, cfg.maxConnectionAgeGrace)
}

func TestWithMetrics(t *testing.T) {
	cfg, err := newConfig(WithMetrics())
	require.NoError(t, err)
	assert.True(t, cfg.enableMetrics)
}

func TestWithoutMetrics(t *testing.T) {
	cfg, err := newConfig(WithoutMetrics())
	require.NoError(t, err)
	assert.False(t, cfg.enableMetrics)
}

func TestWithHealthCheck(t *testing.T) {
	cfg, err := newConfig(WithHealthCheck())
	require.NoError(t, err)
	assert.True(t, cfg.enableHealthCheck)
}

func TestWithoutHealthCheck(t *testing.T) {
	cfg, err := newConfig(WithoutHealthCheck())
	require.NoError(t, err)
	assert.False(t, cfg.enableHealthCheck)
}

func TestWithLogging(t *testing.T) {
	cfg, err := newConfig(WithLogging())
	require.NoError(t, err)
	assert.True(t, cfg.enableLogging)
}

func TestWithoutLogging(t *testing.T) {
	cfg, err := newConfig(WithoutLogging())
	require.NoError(t, err)
	assert.False(t, cfg.enableLogging)
}

func TestWithReflection(t *testing.T) {
	cfg, err := newConfig(WithReflection())
	require.NoError(t, err)
	assert.True(t, cfg.enableReflection)
}

func TestWithoutReflection(t *testing.T) {
	cfg, err := newConfig(WithoutReflection())
	require.NoError(t, err)
	assert.False(t, cfg.enableReflection)
}

func TestWithCORS(t *testing.T) {
	cfg, err := newConfig(WithCORS())
	require.NoError(t, err)
	assert.True(t, cfg.enableCORS)
}

func TestWithRateLimit(t *testing.T) {
	cfg, err := newConfig(WithRateLimit(250.0))
	require.NoError(t, err)
	assert.True(t, cfg.enableRateLimit)
	assert.Equal(t, 250.0, cfg.rateLimit)
}

func TestWithTLS(t *testing.T) {
	cfg, err := newConfig(WithTLS("cert.pem", "key.pem"))
	require.NoError(t, err)
	assert.True(t, cfg.enableTLS)
	assert.Equal(t, "cert.pem", cfg.certFile)
	assert.Equal(t, "key.pem", cfg.keyFile)
}

func TestWithMetricsPath(t *testing.T) {
	cfg, err := newConfig(WithMetricsPath("/custom-metrics"))
	require.NoError(t, err)
	assert.Equal(t, "/custom-metrics", cfg.metricsPath)
}

func TestWithHealthPath(t *testing.T) {
	cfg, err := newConfig(WithHealthPath("/custom-health"))
	require.NoError(t, err)
	assert.Equal(t, "/custom-health", cfg.healthPath)
}

func TestWithGatewayBasePath(t *testing.T) {
	cfg, err := newConfig(WithGatewayBasePath("/api/v2"))
	require.NoError(t, err)
	assert.Equal(t, "/api/v2", cfg.gatewayBasePath)
}

func TestWithServiceRegistrar(t *testing.T) {
	called := false
	registrar := func(s *grpc.Server) {
		called = true
	}

	cfg, err := newConfig(WithServiceRegistrar(registrar))
	require.NoError(t, err)
	assert.NotNil(t, cfg.serviceRegistrar)

	// Test the registrar works
	cfg.serviceRegistrar(nil)
	assert.True(t, called)
}

func TestWithGRPCConfigurer(t *testing.T) {
	called := false
	configurer := func(s *grpc.Server) {
		called = true
	}

	cfg, err := newConfig(WithGRPCConfigurer(configurer))
	require.NoError(t, err)
	assert.NotNil(t, cfg.grpcConfigurer)

	// Test the configurer works
	cfg.grpcConfigurer(nil)
	assert.True(t, called)
}

func TestWithEchoConfigurer(t *testing.T) {
	called := false
	configurer := func(e *echo.Echo) {
		called = true
	}

	cfg, err := newConfig(WithEchoConfigurer(configurer))
	require.NoError(t, err)
	assert.NotNil(t, cfg.echoConfigurer)

	// Test the configurer works
	cfg.echoConfigurer(nil)
	assert.True(t, called)
}

func TestWithShutdownHandler(t *testing.T) {
	called := false
	handler := func() error {
		called = true
		return nil
	}

	cfg, err := newConfig(WithShutdownHandler(handler))
	require.NoError(t, err)
	assert.NotNil(t, cfg.shutdown)

	// Test the handler works
	err = cfg.shutdown()
	require.NoError(t, err)
	assert.True(t, called)
}

func TestWithMiddleware(t *testing.T) {
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

	cfg, err := newConfig(WithMiddleware(mw1, mw2))
	require.NoError(t, err)
	assert.Len(t, cfg.middleware, 2)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			options:     []Option{},
			expectError: false,
		},
		{
			name:        "valid H2C config",
			options:     []Option{WithH2CMode(), WithGRPCPort("8080")},
			expectError: false,
		},
		{
			name:        "valid separate config",
			options:     []Option{WithSeparateMode("9090", "9091")},
			expectError: false,
		},
		{
			name:        "missing gRPC port",
			options:     []Option{WithGRPCPort("")},
			expectError: true,
			errorMsg:    "gRPC port cannot be empty",
		},
		{
			name:        "separate mode missing HTTP port",
			options:     []Option{WithSeparateMode("9090", "")},
			expectError: true,
			errorMsg:    "HTTP port cannot be empty",
		},
		{
			name:        "negative shutdown timeout",
			options:     []Option{WithShutdownTimeout(-1 * time.Second)},
			expectError: true,
			errorMsg:    "shutdown timeout cannot be negative",
		},
		{
			name:        "negative read timeout",
			options:     []Option{WithReadTimeout(-1 * time.Second)},
			expectError: true,
			errorMsg:    "read timeout cannot be negative",
		},
		{
			name:        "negative write timeout",
			options:     []Option{WithWriteTimeout(-1 * time.Second)},
			expectError: true,
			errorMsg:    "write timeout cannot be negative",
		},
		{
			name:        "negative idle timeout",
			options:     []Option{WithIdleTimeout(-1 * time.Second)},
			expectError: true,
			errorMsg:    "idle timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConfig(tt.options...)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigAddresses(t *testing.T) {
	tests := []struct {
		name         string
		options      []Option
		expectedGRPC string
		expectedHTTP string
	}{
		{
			name:         "default config",
			options:      []Option{},
			expectedGRPC: ":8080",
			expectedHTTP: ":8080", // H2C mode uses same port
		},
		{
			name:         "custom H2C port",
			options:      []Option{WithGRPCPort("9000")},
			expectedGRPC: ":9000",
			expectedHTTP: ":9000",
		},
		{
			name:         "separate mode",
			options:      []Option{WithSeparateMode("9090", "9091")},
			expectedGRPC: ":9090",
			expectedHTTP: ":9091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := newConfig(tt.options...)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedGRPC, cfg.getGRPCAddress())
			assert.Equal(t, tt.expectedHTTP, cfg.getHTTPAddress())
		})
	}
}

func TestConfigModeChecks(t *testing.T) {
	t.Run("H2C mode", func(t *testing.T) {
		cfg, err := newConfig(WithH2CMode())
		require.NoError(t, err)
		assert.True(t, cfg.isH2CMode())
		assert.False(t, cfg.isSeparateMode())
	})

	t.Run("Separate mode", func(t *testing.T) {
		cfg, err := newConfig(WithSeparateMode("9090", "9091"))
		require.NoError(t, err)
		assert.False(t, cfg.isH2CMode())
		assert.True(t, cfg.isSeparateMode())
	})
}

func TestMultipleOptions(t *testing.T) {
	cfg, err := newConfig(
		WithGRPCPort("9000"),
		WithSeparateMode("9090", "9091"),
		WithShutdownTimeout(45*time.Second),
		WithCORS(),
		WithRateLimit(200.0),
		WithMetricsPath("/custom-metrics"),
		WithHealthPath("/custom-health"),
		WithGatewayBasePath("/api/v2"),
		WithoutReflection(),
	)
	require.NoError(t, err)

	// Verify all options were applied (later options should override earlier ones)
	assert.Equal(t, SeparateMode, cfg.mode)
	assert.Equal(t, "9090", cfg.grpcPort) // Overridden by WithSeparateMode
	assert.Equal(t, "9091", cfg.httpPort)
	assert.Equal(t, 45*time.Second, cfg.shutdownTimeout)
	assert.True(t, cfg.enableCORS)
	assert.True(t, cfg.enableRateLimit)
	assert.Equal(t, 200.0, cfg.rateLimit)
	assert.Equal(t, "/custom-metrics", cfg.metricsPath)
	assert.Equal(t, "/custom-health", cfg.healthPath)
	assert.Equal(t, "/api/v2", cfg.gatewayBasePath)
	assert.False(t, cfg.enableReflection)
}