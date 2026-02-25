package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/jasoet/pkg/v2/logging"
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
		helper.Debug("debug message with fields", F("key", "value"), F("count", 42))
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Debug("debug message")
		helper.Debug("debug message with fields", F("key", "value"), F("count", 42))
	})
}

func TestLogHelper_Info(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Info("info message")
		helper.Info("info message with fields", F("key", "value"), F("enabled", true))
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Info("info message")
		helper.Info("info message with fields", F("key", "value"), F("enabled", true))
	})
}

func TestLogHelper_Warn(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Warn("warning message")
		helper.Warn("warning message with fields", F("key", "value"), F("ratio", 0.75))
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Warn("warning message")
		helper.Warn("warning message with fields", F("key", "value"), F("ratio", 0.75))
	})
}

func TestLogHelper_Error(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		// Should not panic
		helper.Error(testErr, "error message")
		helper.Error(testErr, "error message with fields", F("key", "value"), F("code", 500))
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		// Should not panic
		helper.Error(testErr, "error message")
		helper.Error(testErr, "error message with fields", F("key", "value"), F("code", 500))
	})
}

func TestLogHelper_MixedTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("various data types without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		helper.Info("mixed types",
			F("string", "value"),
			F("int", 123),
			F("int64", int64(456)),
			F("bool", true),
			F("float64", 3.14),
		)
	})

	t.Run("various data types with OTel", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		helper.Info("mixed types",
			F("string", "value"),
			F("int", 123),
			F("int64", int64(456)),
			F("bool", true),
			F("float64", 3.14),
		)
	})
}

// TestLogHelper_LogLevelFiltering tests that logs are filtered based on configured level
func TestLogHelper_LogLevelFiltering(t *testing.T) {
	ctx := context.Background()

	t.Run("warn level filters info and debug", func(t *testing.T) {
		// Create logger provider with WARN level
		loggerProvider, _ := NewLoggerProviderWithOptions("test-service",
			WithLogLevel(logging.LogLevelWarn))

		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: loggerProvider,
		}

		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")

		// These should be filtered (not panic, but not emit)
		helper.Debug("This debug should be filtered")
		helper.Info("This info should be filtered")

		// These should be emitted
		helper.Warn("This warning should appear")
		helper.Error(errors.New("test error"), "This error should appear")
	})

	t.Run("info level filters debug only", func(t *testing.T) {
		loggerProvider, _ := NewLoggerProviderWithOptions("test-service",
			WithLogLevel(logging.LogLevelInfo))

		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: loggerProvider,
		}

		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")

		// This should be filtered
		helper.Debug("This debug should be filtered")

		// These should be emitted
		helper.Info("This info should appear")
		helper.Warn("This warning should appear")
		helper.Error(errors.New("test error"), "This error should appear")
	})

	t.Run("error level filters all except errors", func(t *testing.T) {
		loggerProvider, _ := NewLoggerProviderWithOptions("test-service",
			WithLogLevel(logging.LogLevelError))

		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: loggerProvider,
		}

		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")

		// These should be filtered
		helper.Debug("This debug should be filtered")
		helper.Info("This info should be filtered")
		helper.Warn("This warning should be filtered")

		// This should be emitted
		helper.Error(errors.New("test error"), "This error should appear")
	})
}

func TestLogHelper_WithFields_SliceIsolation(t *testing.T) {
	ctx := context.Background()

	t.Run("sibling helpers do not share fields", func(t *testing.T) {
		parent := NewLogHelper(ctx, nil, "", "test.Function").
			WithFields(F("base", "value"))

		child1 := parent.WithFields(F("child", "one"))
		child2 := parent.WithFields(F("child", "two"))

		// Verify each helper has the correct number of fields
		if len(parent.baseFields) != 1 {
			t.Errorf("expected parent to have 1 field, got %d", len(parent.baseFields))
		}
		if len(child1.baseFields) != 2 {
			t.Errorf("expected child1 to have 2 fields, got %d", len(child1.baseFields))
		}
		if len(child2.baseFields) != 2 {
			t.Errorf("expected child2 to have 2 fields, got %d", len(child2.baseFields))
		}

		// Verify child fields don't bleed into each other
		if child1.baseFields[1].Value != "one" {
			t.Errorf("expected child1 field to be 'one', got '%v'", child1.baseFields[1].Value)
		}
		if child2.baseFields[1].Value != "two" {
			t.Errorf("expected child2 field to be 'two', got '%v'", child2.baseFields[1].Value)
		}

		// Verify parent is unchanged after creating children
		if parent.baseFields[0].Value != "value" {
			t.Errorf("expected parent field to be 'value', got '%v'", parent.baseFields[0].Value)
		}
	})

	t.Run("log calls do not mutate baseFields", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function").
			WithFields(F("base", "value"))

		originalLen := len(helper.baseFields)

		helper.Info("msg1", F("extra", "a"))
		helper.Info("msg2", F("extra", "b"))
		helper.Error(errors.New("err"), "msg3", F("extra", "c"))

		if len(helper.baseFields) != originalLen {
			t.Errorf("expected baseFields length to remain %d, got %d", originalLen, len(helper.baseFields))
		}
	})
}
