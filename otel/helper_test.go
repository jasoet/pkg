package otel

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/log/noop"
)

func TestNewLogHelper(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel config", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		if helper == nil {
			t.Fatal("expected helper to be created")
		}
		if helper.otelLogger != nil {
			t.Error("expected otelLogger to be nil when config is nil")
		}
		if helper.function != "test.Function" {
			t.Errorf("expected function to be 'test.Function', got '%s'", helper.function)
		}
	})

	t.Run("with OTel config but logging disabled", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "test-service",
			// LoggerProvider is nil, so logging is disabled
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		if helper == nil {
			t.Fatal("expected helper to be created")
		}
		if helper.otelLogger != nil {
			t.Error("expected otelLogger to be nil when logging is disabled")
		}
	})

	t.Run("with OTel config and logging enabled", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		if helper == nil {
			t.Fatal("expected helper to be created")
		}
		if helper.otelLogger == nil {
			t.Error("expected otelLogger to be set when logging is enabled")
		}
		if helper.function != "test.Function" {
			t.Errorf("expected function to be 'test.Function', got '%s'", helper.function)
		}
	})
}

func TestLogHelper_Debug(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Debug("debug message")
		helper.Debug("debug message with fields", "key", "value", "count", 42)
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Debug("debug message")
		helper.Debug("debug message with fields", "key", "value", "count", 42)
	})
}

func TestLogHelper_Info(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Info("info message")
		helper.Info("info message with fields", "key", "value", "enabled", true)
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Info("info message")
		helper.Info("info message with fields", "key", "value", "enabled", true)
	})
}

func TestLogHelper_Warn(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Warn("warning message")
		helper.Warn("warning message with fields", "key", "value", "ratio", 0.75)
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Warn("warning message")
		helper.Warn("warning message with fields", "key", "value", "ratio", 0.75)
	})
}

func TestLogHelper_Error(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Error(testErr, "error message")
		helper.Error(testErr, "error message with fields", "key", "value", "code", 500)
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Error(testErr, "error message")
		helper.Error(testErr, "error message with fields", "key", "value", "code", 500)
	})
}

func TestLogHelper_MixedTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("various data types without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		helper.Info("mixed types",
			"string", "value",
			"int", 123,
			"int64", int64(456),
			"bool", true,
			"float64", 3.14,
		)
	})

	t.Run("various data types with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		helper.Info("mixed types",
			"string", "value",
			"int", 123,
			"int64", int64(456),
			"bool", true,
			"float64", 3.14,
		)
	})
}

func TestLogHelper_InvalidKeyValuePairs(t *testing.T) {
	ctx := context.Background()

	t.Run("odd number of arguments without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic with odd number of arguments
		helper.Info("message", "key")
	})

	t.Run("odd number of arguments with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic with odd number of arguments
		helper.Info("message", "key")
	})

	t.Run("non-string keys without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic with non-string keys
		helper.Info("message", 123, "value")
	})

	t.Run("non-string keys with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic with non-string keys
		helper.Info("message", 123, "value")
	})
}
