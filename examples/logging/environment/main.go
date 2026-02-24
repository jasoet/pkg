//go:build example

package main

import (
	"os"
	"path/filepath"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	// Get environment from ENV variable (or default to development)
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	println("=== Environment-Based Logging Configuration ===")
	println("Environment:", env)
	println()

	// Configure logging based on environment
	switch env {
	case "production":
		// Production: file only, info level
		tempDir, _ := os.MkdirTemp("", "logging-prod-*")
		defer os.RemoveAll(tempDir)

		logFile := filepath.Join(tempDir, "production.log")
		closer, err := logging.InitializeWithFile("my-service", false,
			logging.OutputFile,
			&logging.FileConfig{Path: logFile})
		if err != nil {
			panic(err)
		}
		defer closer.Close()

		println("Production mode: logging to file only")
		println("Log file:", logFile)

	case "staging":
		// Staging: both console and file, debug level
		tempDir, _ := os.MkdirTemp("", "logging-staging-*")
		defer os.RemoveAll(tempDir)

		logFile := filepath.Join(tempDir, "staging.log")
		closer, err := logging.InitializeWithFile("my-service", true,
			logging.OutputConsole|logging.OutputFile,
			&logging.FileConfig{Path: logFile})
		if err != nil {
			panic(err)
		}
		defer closer.Close()

		println("Staging mode: logging to console and file")
		println("Log file:", logFile)

	default:
		// Development: console only, debug level
		if err := logging.Initialize("my-service", true); err != nil {
			panic(err)
		}
		println("Development mode: logging to console only")
	}

	println()

	// Log some messages
	log.Info().Str("environment", env).Msg("Application started")
	log.Debug().Msg("Debug information")
	log.Info().
		Str("user_id", "test-123").
		Str("action", "login").
		Msg("User logged in")

	log.Warn().Msg("Warning message")
	log.Info().Msg("Application running")

	println("\n=== Try running with different environments ===")
	println("ENV=development go run -tags=example ./logging/examples/environment")
	println("ENV=staging go run -tags=example ./logging/examples/environment")
	println("ENV=production go run -tags=example ./logging/examples/environment")
}
