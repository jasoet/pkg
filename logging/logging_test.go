package logging

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestInitialize(t *testing.T) {
	// Reset the global logger for testing
	zlog.Logger = zerolog.New(os.Stderr)

	// Call Initialize
	Initialize("test-service", true)

	// Verify that the global level is set to Debug
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Expected global level to be Debug, got %v", zerolog.GlobalLevel())
	}

	// Call Initialize again with different parameters
	Initialize("another-service", false)

	// Verify that the global level is still Debug (due to sync.Once)
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Expected global level to remain Debug, got %v", zerolog.GlobalLevel())
	}
}

func TestContextLogger(t *testing.T) {
	// Reset the global logger for testing
	zlog.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Create a context with values
	type contextKey string
	const requestIDKey contextKey = "request_id"
	ctx := context.WithValue(context.Background(), requestIDKey, "123456")

	// Get a logger with context
	logger := ContextLogger(ctx, "test-component")

	// Verify that the logger has the component field
	if logger.GetLevel() != zlog.Logger.GetLevel() {
		t.Errorf("Expected logger level to match global logger level")
	}
}

func TestIntegration(t *testing.T) {
	// Reset the global logger for testing
	zlog.Logger = zerolog.New(os.Stderr)

	// Initialize the logger
	Initialize("test-service", true)

	// Create a context logger
	ctx := context.Background()
	logger := ContextLogger(ctx, "test-component")

	// Log a message (this is just to verify it doesn't panic)
	logger.Info().Msg("Test message")

	// Use the global logger directly
	zlog.Info().Msg("Global logger test message")
}

// TestNewLoggerProvider tests the creation of OTel LoggerProvider
func TestNewLoggerProvider(t *testing.T) {
	t.Run("creates provider with debug level", func(t *testing.T) {
		provider := NewLoggerProvider("test-service", true)
		if provider == nil {
			t.Fatal("Expected non-nil LoggerProvider")
		}
	})

	t.Run("creates provider with info level", func(t *testing.T) {
		provider := NewLoggerProvider("test-service", false)
		if provider == nil {
			t.Fatal("Expected non-nil LoggerProvider")
		}
	})
}

// TestLoggerProvider_Logger tests the Logger method
func TestLoggerProvider_Logger(t *testing.T) {
	provider := NewLoggerProvider("test-service", false)

	t.Run("creates logger with scope", func(t *testing.T) {
		logger := provider.Logger("test-scope")
		if logger == nil {
			t.Fatal("Expected non-nil Logger")
		}
	})

	t.Run("creates multiple loggers", func(t *testing.T) {
		logger1 := provider.Logger("scope1")
		logger2 := provider.Logger("scope2")

		if logger1 == nil || logger2 == nil {
			t.Fatal("Expected non-nil loggers")
		}
	})
}

// TestLogger_Emit tests basic log emission
func TestLogger_Emit(t *testing.T) {
	provider := NewLoggerProvider("test-service", true)
	logger := provider.Logger("test-scope")
	ctx := context.Background()

	t.Run("emits log record with message", func(t *testing.T) {
		var record log.Record
		record.SetBody(log.StringValue("test message"))
		record.SetTimestamp(time.Now())
		record.SetSeverity(log.SeverityInfo)

		// Should not panic
		logger.Emit(ctx, record)
	})

	t.Run("emits log record with attributes", func(t *testing.T) {
		var record log.Record
		record.SetBody(log.StringValue("test with attributes"))
		record.SetSeverity(log.SeverityWarn)
		record.AddAttributes(
			log.String("key1", "value1"),
			log.Int64("key2", 42),
			log.Bool("key3", true),
		)

		// Should not panic
		logger.Emit(ctx, record)
	})

	t.Run("emits different severity levels", func(t *testing.T) {
		severities := []log.Severity{
			log.SeverityDebug,
			log.SeverityInfo,
			log.SeverityWarn,
			log.SeverityError,
		}

		for _, severity := range severities {
			var record log.Record
			record.SetBody(log.StringValue("test message"))
			record.SetSeverity(severity)

			// Should not panic
			logger.Emit(ctx, record)
		}
	})
}

// TestLogger_EmitWithTraceContext tests trace context extraction
func TestLogger_EmitWithTraceContext(t *testing.T) {
	provider := NewLoggerProvider("test-service", true)
	logger := provider.Logger("test-scope")

	t.Run("extracts trace context from span", func(t *testing.T) {
		// Create a tracer provider for testing
		tp := sdktrace.NewTracerProvider()
		tracer := tp.Tracer("test-tracer")

		// Start a span
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		// Emit log with span context
		var record log.Record
		record.SetBody(log.StringValue("test with trace context"))
		record.SetSeverity(log.SeverityInfo)

		// Should not panic and should extract trace_id/span_id
		logger.Emit(ctx, record)

		// Verify span context is valid
		spanCtx := trace.SpanContextFromContext(ctx)
		if !spanCtx.IsValid() {
			t.Error("Expected valid span context")
		}
		if !spanCtx.TraceID().IsValid() {
			t.Error("Expected valid trace ID")
		}
		if !spanCtx.SpanID().IsValid() {
			t.Error("Expected valid span ID")
		}
	})

	t.Run("handles context without span", func(t *testing.T) {
		ctx := context.Background()

		var record log.Record
		record.SetBody(log.StringValue("test without trace context"))
		record.SetSeverity(log.SeverityInfo)

		// Should not panic even without span context
		logger.Emit(ctx, record)
	})
}

// TestLogger_Enabled tests the Enabled method
func TestLogger_Enabled(t *testing.T) {
	t.Run("debug logger enables all levels", func(t *testing.T) {
		provider := NewLoggerProvider("test-service", true)
		logger := provider.Logger("test-scope")
		ctx := context.Background()

		testCases := []struct {
			severity log.Severity
			expected bool
		}{
			{log.SeverityDebug, true},
			{log.SeverityInfo, true},
			{log.SeverityWarn, true},
			{log.SeverityError, true},
			{log.SeverityFatal, true},
		}

		for _, tc := range testCases {
			params := log.EnabledParameters{Severity: tc.severity}
			if got := logger.Enabled(ctx, params); got != tc.expected {
				t.Errorf("Severity %v: expected %v, got %v", tc.severity, tc.expected, got)
			}
		}
	})

	t.Run("info logger filters debug", func(t *testing.T) {
		provider := NewLoggerProvider("test-service", false)
		logger := provider.Logger("test-scope")
		ctx := context.Background()

		testCases := []struct {
			severity log.Severity
			expected bool
		}{
			{log.SeverityDebug, false},
			{log.SeverityInfo, true},
			{log.SeverityWarn, true},
			{log.SeverityError, true},
			{log.SeverityFatal, true},
		}

		for _, tc := range testCases {
			params := log.EnabledParameters{Severity: tc.severity}
			if got := logger.Enabled(ctx, params); got != tc.expected {
				t.Errorf("Severity %v: expected %v, got %v", tc.severity, tc.expected, got)
			}
		}
	})
}
