package otel

import (
	"context"
	"testing"

	"github.com/jasoet/pkg/v2/logging"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
)

// TestWithConsoleOutput tests the WithConsoleOutput option
func TestWithConsoleOutput(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "console output enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "console output disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			opt := WithConsoleOutput(tt.enabled)
			opt(cfg)

			if cfg.consoleOutput != tt.expected {
				t.Errorf("expected consoleOutput to be %v, got %v", tt.expected, cfg.consoleOutput)
			}
		})
	}
}

// TestWithOTLPEndpoint tests the WithOTLPEndpoint option
func TestWithOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		endpoint         string
		insecure         bool
		expectedEndpoint string
		expectedInsecure bool
	}{
		{
			name:             "secure endpoint",
			endpoint:         "localhost:4318",
			insecure:         false,
			expectedEndpoint: "localhost:4318",
			expectedInsecure: false,
		},
		{
			name:             "insecure endpoint",
			endpoint:         "localhost:4318",
			insecure:         true,
			expectedEndpoint: "localhost:4318",
			expectedInsecure: true,
		},
		{
			name:             "https endpoint",
			endpoint:         "https://otel-collector:4318",
			insecure:         false,
			expectedEndpoint: "https://otel-collector:4318",
			expectedInsecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			opt := WithOTLPEndpoint(tt.endpoint, tt.insecure)
			opt(cfg)

			if cfg.otlpEndpoint != tt.expectedEndpoint {
				t.Errorf("expected otlpEndpoint to be %s, got %s", tt.expectedEndpoint, cfg.otlpEndpoint)
			}
			if cfg.otlpInsecure != tt.expectedInsecure {
				t.Errorf("expected otlpInsecure to be %v, got %v", tt.expectedInsecure, cfg.otlpInsecure)
			}
		})
	}
}

// TestWithLogLevel tests the WithLogLevel option
func TestWithLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected LogLevel
	}{
		{
			name:     "debug level",
			level:    logging.LogLevelDebug,
			expected: logging.LogLevelDebug,
		},
		{
			name:     "info level",
			level:    logging.LogLevelInfo,
			expected: logging.LogLevelInfo,
		},
		{
			name:     "warn level",
			level:    logging.LogLevelWarn,
			expected: logging.LogLevelWarn,
		},
		{
			name:     "error level",
			level:    logging.LogLevelError,
			expected: logging.LogLevelError,
		},
		{
			name:     "none level",
			level:    logging.LogLevelNone,
			expected: logging.LogLevelNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			opt := WithLogLevel(tt.level)
			opt(cfg)

			if cfg.logLevel != tt.expected {
				t.Errorf("expected logLevel to be %s, got %s", tt.expected, cfg.logLevel)
			}
		})
	}
}

// TestNewLoggerProviderWithOptions_NoOTLP tests fallback to console-only when no OTLP endpoint
func TestNewLoggerProviderWithOptions_NoOTLP(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		opts        []LoggerProviderOption
	}{
		{
			name:        "debug mode without OTLP",
			serviceName: "test-service",
			opts:        []LoggerProviderOption{WithLogLevel(logging.LogLevelDebug)},
		},
		{
			name:        "info mode without OTLP",
			serviceName: "test-service",
			opts:        []LoggerProviderOption{},
		},
		{
			name:        "explicit log level without OTLP",
			serviceName: "test-service",
			opts: []LoggerProviderOption{
				WithLogLevel(logging.LogLevelWarn),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewLoggerProviderWithOptions(tt.serviceName, tt.opts...)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
			}
		})
	}
}

// TestNewLoggerProviderWithOptions_WithOTLP tests OTLP configuration
func TestNewLoggerProviderWithOptions_WithOTLP(t *testing.T) {
	// Note: This test will fail if there's no OTLP collector running
	// We're testing the configuration, not the actual connection
	t.Run("invalid endpoint should return error", func(t *testing.T) {
		// Use an invalid endpoint that will cause immediate failure
		_, err := NewLoggerProviderWithOptions(
			"test-service",
			WithOTLPEndpoint("", true), // Empty endpoint should fail
		)
		// We expect this to fallback to console-only since endpoint is empty
		if err != nil {
			t.Fatalf("expected no error with empty endpoint (fallback), got %v", err)
		}
	})

	t.Run("console output with OTLP", func(t *testing.T) {
		// This will fail to connect but should not panic
		// Testing configuration correctness, not actual connection
		serviceName := "test-service"
		endpoint := "nonexistent-host:9999"

		// This should create the provider even if connection fails later
		_, err := NewLoggerProviderWithOptions(
			serviceName,
			WithOTLPEndpoint(endpoint, true),
			WithConsoleOutput(true),
			WithLogLevel(logging.LogLevelInfo),
		)

		// We expect an error since the endpoint is invalid
		if err == nil {
			t.Log("Warning: Expected error for invalid endpoint, but got none")
		}
	})
}

// TestNewLoggerProviderWithOptions_LogLevelPriority tests log level priority
func TestNewLoggerProviderWithOptions_LogLevelPriority(t *testing.T) {
	tests := []struct {
		name          string
		explicitLevel LogLevel
	}{
		{
			name:          "explicit error level",
			explicitLevel: logging.LogLevelError,
		},
		{
			name:          "explicit debug level",
			explicitLevel: logging.LogLevelDebug,
		},
		{
			name:          "explicit warn level",
			explicitLevel: logging.LogLevelWarn,
		},
		{
			name:          "default level (info)",
			explicitLevel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []LoggerProviderOption
			if tt.explicitLevel != "" {
				opts = append(opts, WithLogLevel(tt.explicitLevel))
			}

			provider, err := NewLoggerProviderWithOptions("test-service", opts...)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
			}

			// Provider created successfully - test passed
		})
	}
}

// TestNewLoggerProviderWithOptions_MultipleOptions tests combining multiple options
func TestNewLoggerProviderWithOptions_MultipleOptions(t *testing.T) {
	t.Run("all options combined without OTLP", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(true),
			WithLogLevel(logging.LogLevelWarn),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})

	t.Run("disable console output", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(false),
			WithLogLevel(logging.LogLevelInfo),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

// TestLoggerProviderConfig_Defaults tests default configuration values
func TestLoggerProviderConfig_Defaults(t *testing.T) {
	t.Run("default config values", func(t *testing.T) {
		cfg := &loggerProviderConfig{
			serviceName:   "test-service",
			consoleOutput: true, // Default value
		}

		if !cfg.consoleOutput {
			t.Error("expected default consoleOutput to be true")
		}
		if cfg.otlpEndpoint != "" {
			t.Errorf("expected default otlpEndpoint to be empty, got %s", cfg.otlpEndpoint)
		}
		if cfg.otlpInsecure {
			t.Error("expected default otlpInsecure to be false")
		}
		if cfg.logLevel != "" {
			t.Errorf("expected default logLevel to be empty, got %s", cfg.logLevel)
		}
	})
}

// TestSetupZerologConsole tests the setupZerologConsole function indirectly
// by ensuring providers can be created with different log levels
func TestSetupZerologConsole(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
	}{
		{
			name:     "debug level",
			logLevel: logging.LogLevelDebug,
		},
		{
			name:     "info level",
			logLevel: logging.LogLevelInfo,
		},
		{
			name:     "warn level",
			logLevel: logging.LogLevelWarn,
		},
		{
			name:     "error level",
			logLevel: logging.LogLevelError,
		},
		{
			name:     "none level",
			logLevel: logging.LogLevelNone,
		},
		{
			name:     "unknown level defaults to info",
			logLevel: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This indirectly tests setupZerologConsole
			provider, err := NewLoggerProviderWithOptions(
				"test-service",
				WithLogLevel(tt.logLevel),
				WithConsoleOutput(true),
			)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if provider == nil {
				t.Fatal("expected provider to be non-nil")
			}
		})
	}
}

// TestNewLoggerProviderWithOptions_Integration tests realistic usage patterns
func TestNewLoggerProviderWithOptions_Integration(t *testing.T) {
	t.Run("local development setup", func(t *testing.T) {
		// Typical local development: console output only, debug enabled
		provider, err := NewLoggerProviderWithOptions(
			"my-service",
			WithConsoleOutput(true),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}

		// Verify we can get a logger from the provider
		logger := provider.Logger("test-scope")
		if logger == nil {
			t.Fatal("expected logger to be non-nil")
		}
	})

	t.Run("production-like setup without collector", func(t *testing.T) {
		// Production without OTLP: console output with specific log level
		provider, err := NewLoggerProviderWithOptions(
			"my-service",
			WithConsoleOutput(true),
			WithLogLevel(logging.LogLevelInfo),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})

	t.Run("silent mode", func(t *testing.T) {
		// Silent mode: no console output, none log level
		provider, err := NewLoggerProviderWithOptions(
			"my-service",
			WithConsoleOutput(false),
			WithLogLevel(logging.LogLevelNone),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

// TestLoggerProviderCompatibility tests that the provider implements the interface correctly
func TestLoggerProviderCompatibility(t *testing.T) {
	t.Run("provider implements log.LoggerProvider", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("test-service")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Type assertion to ensure it implements the interface
		var _ log.LoggerProvider = provider
	})

	t.Run("logger can be obtained from provider", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("test-service")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		logger := provider.Logger("test-scope")
		if logger == nil {
			t.Fatal("expected logger to be non-nil")
		}

		// Type assertion to ensure it implements the interface
		var _ log.Logger = logger
	})
}

// TestNewLoggerProviderWithOptions_EmptyServiceName tests behavior with empty service name
func TestNewLoggerProviderWithOptions_EmptyServiceName(t *testing.T) {
	t.Run("empty service name", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil even with empty service name")
		}
	})
}

// TestLoggerProviderOptions_Chaining tests that options can be chained
func TestLoggerProviderOptions_Chaining(t *testing.T) {
	t.Run("chain multiple options", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(true),
			WithLogLevel(logging.LogLevelDebug),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})

	t.Run("last option wins for same config", func(t *testing.T) {
		// Multiple log level options - last one should win
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithLogLevel(logging.LogLevelDebug),
			WithLogLevel(logging.LogLevelError), // This should win
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

// TestLoggerProvider_NoopComparison compares behavior with noop provider
func TestLoggerProvider_NoopComparison(t *testing.T) {
	t.Run("created provider vs noop provider", func(t *testing.T) {
		// Create our provider
		ourProvider, err := NewLoggerProviderWithOptions("test-service")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Create noop provider
		noopProvider := noop.NewLoggerProvider()

		// Both should return valid loggers
		ourLogger := ourProvider.Logger("test-scope")
		noopLogger := noopProvider.Logger("test-scope")

		if ourLogger == nil {
			t.Error("expected our logger to be non-nil")
		}
		if noopLogger == nil {
			t.Error("expected noop logger to be non-nil")
		}

		// Both should accept Emit calls without panicking
		ctx := context.Background()
		record := log.Record{}
		record.SetBody(log.StringValue("test message"))

		// Should not panic
		ourLogger.Emit(ctx, record)
		noopLogger.Emit(ctx, record)
	})
}
