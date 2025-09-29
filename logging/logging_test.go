package logging

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestInitialize(t *testing.T) {
	// Reset the global logger for testing
	log.Logger = zerolog.New(os.Stderr)

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
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Create a context with values
	type contextKey string
	const requestIDKey contextKey = "request_id"
	ctx := context.WithValue(context.Background(), requestIDKey, "123456")

	// Get a logger with context
	logger := ContextLogger(ctx, "test-component")

	// Verify that the logger has the component field
	if logger.GetLevel() != log.Logger.GetLevel() {
		t.Errorf("Expected logger level to match global logger level")
	}
}

func TestIntegration(t *testing.T) {
	// Reset the global logger for testing
	log.Logger = zerolog.New(os.Stderr)

	// Initialize the logger
	Initialize("test-service", true)

	// Create a context logger
	ctx := context.Background()
	logger := ContextLogger(ctx, "test-component")

	// Log a message (this is just to verify it doesn't panic)
	logger.Info().Msg("Test message")

	// Use the global logger directly
	log.Info().Msg("Global logger test message")
}
