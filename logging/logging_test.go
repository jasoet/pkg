package logging

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func TestInitialize(t *testing.T) {
	t.Run("sets debug level when debug is true", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr)

		// Call Initialize with debug=true
		Initialize("test-service", true)

		// Verify that the global level is set to Debug
		if zerolog.GlobalLevel() != zerolog.DebugLevel {
			t.Errorf("Expected global level to be Debug, got %v", zerolog.GlobalLevel())
		}

		// Verify global logger is not nil
		if zlog.Logger.GetLevel() != zerolog.DebugLevel {
			t.Errorf("Expected logger level to be Debug, got %v", zlog.Logger.GetLevel())
		}
	})

	t.Run("sets info level when debug is false", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr)

		// Call Initialize with debug=false
		Initialize("prod-service", false)

		// Verify that the global level is set to Info
		if zerolog.GlobalLevel() != zerolog.InfoLevel {
			t.Errorf("Expected global level to be Info, got %v", zerolog.GlobalLevel())
		}

		// Verify global logger is not nil
		if zlog.Logger.GetLevel() != zerolog.InfoLevel {
			t.Errorf("Expected logger level to be Info, got %v", zlog.Logger.GetLevel())
		}
	})
}

func TestContextLogger(t *testing.T) {
	t.Run("creates logger with component field", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

		// Create a context
		ctx := context.Background()

		// Get a logger with context
		logger := ContextLogger(ctx, "test-component")

		globalLogger := zlog.Logger
		// Verify that the logger level matches global logger level
		if logger.GetLevel() != globalLogger.GetLevel() {
			t.Errorf("Expected logger level to match global logger level")
		}

		// Verify logger is not nil and can log without panic
		logger.Info().Msg("test message")
	})

	t.Run("inherits level from global logger", func(t *testing.T) {
		// Set global logger with specific level
		zlog.Logger = zerolog.New(os.Stderr).Level(zerolog.WarnLevel)

		ctx := context.Background()
		logger := ContextLogger(ctx, "warn-component")

		// Verify logger inherits warn level
		if logger.GetLevel() != zerolog.WarnLevel {
			t.Errorf("Expected logger level to be Warn, got %v", logger.GetLevel())
		}
	})

	t.Run("works with context values", func(t *testing.T) {
		// Reset the global logger
		zlog.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

		// Create a context with values
		type contextKey string
		const requestIDKey contextKey = "request_id"
		ctx := context.WithValue(context.Background(), requestIDKey, "123456")

		// Get a logger with context - should not panic
		logger := ContextLogger(ctx, "ctx-component")

		// Verify logger can be used
		logger.Info().Msg("message with context")
	})
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

	globalLogger := zlog.Logger
	globalLogger.Info().Msg("Global logger test message")
}

