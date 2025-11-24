//go:build example

package main

import (
	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize with console output only (default behavior)
	logging.Initialize("console-example", true)

	// Basic logging
	log.Info().Msg("Service started")
	log.Debug().Str("mode", "development").Msg("Running in debug mode")

	// Structured logging
	log.Info().
		Str("user_id", "12345").
		Int("age", 30).
		Bool("premium", true).
		Msg("User logged in")

	// Warning and error
	log.Warn().Msg("Cache miss, fetching from database")
	log.Error().Str("error", "connection timeout").Msg("Failed to connect")

	log.Info().Msg("Example completed")
}
