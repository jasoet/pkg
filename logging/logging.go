package logging

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

// Initialize sets up the global logger with standard fields
func Initialize(serviceName string, debug bool) {
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", serviceName).
		Int("pid", os.Getpid()).
		Caller().
		Logger()
}

// ContextLogger creates a logger with context values
func ContextLogger(ctx context.Context, component string) zerolog.Logger {
	return log.With().
		Ctx(ctx).
		Str("component", component).
		Logger()
}
