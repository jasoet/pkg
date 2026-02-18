//go:build example

package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	// Create temp directory for logs
	tempDir, err := os.MkdirTemp("", "logging-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "app.log")

	// Initialize with BOTH console and file output
	if err := logging.InitializeWithFile("both-example", true,
		logging.OutputConsole|logging.OutputFile, // Bitwise OR
		&logging.FileConfig{
			Path: logFile,
		}); err != nil {
		panic(err)
	}

	println("=== Logging to both console and file ===\n")

	// Global logger
	log.Info().Msg("Service started")
	log.Debug().Str("environment", "development").Msg("Environment configured")

	// Component logger
	ctx := context.Background()
	userLogger := logging.ContextLogger(ctx, "user-service")

	userLogger.Info().
		Str("user_id", "user-123").
		Str("action", "registration").
		Msg("User registered")

	orderLogger := logging.ContextLogger(ctx, "order-service")

	orderLogger.Info().
		Str("order_id", "order-456").
		Int("items", 3).
		Float64("total", 99.99).
		Msg("Order placed")

	// Simulate processing
	time.Sleep(100 * time.Millisecond)

	orderLogger.Info().
		Str("order_id", "order-456").
		Str("status", "completed").
		Dur("processing_time", 100*time.Millisecond).
		Msg("Order processed")

	log.Warn().Str("cache_key", "user-123").Msg("Cache miss")
	log.Info().Msg("Service running normally")

	// Display file contents
	println("\n=== File Content (JSON format) ===")
	content, err := os.ReadFile(logFile)
	if err != nil {
		panic(err)
	}
	println(string(content))
	println("=== End of File Content ===")
	println("\nLog file location:", logFile)
	println("(File will be deleted after example exits)")
}
