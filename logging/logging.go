package logging

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
	"time"
)

var initOnce sync.Once

// Initialize sets up the zerolog global logger with standard fields.
// This function should be called once at the start of your application.
// After calling Initialize, you can use zerolog's log package functions directly
// (log.Debug(), log.Info(), etc.) or create component-specific loggers with ContextLogger.
//
// Parameters:
//   - serviceName: Name of the service, added as a field to all log entries
//   - debug: If true, sets log level to Debug, otherwise Info
func Initialize(serviceName string, debug bool) {
	initOnce.Do(func() {
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
	})
}

// ContextLogger creates a logger with context values.
// This function uses the global logger configured by Initialize.
// It adds context values and a component name to the logger.
//
// Parameters:
//   - ctx: Context that may contain values to be added to the logger
//   - component: Name of the component, added as a field to all log entries
//
// Returns:
//   - A zerolog.Logger instance with context and component information
func ContextLogger(ctx context.Context, component string) zerolog.Logger {
	return log.With().
		Ctx(ctx).
		Str("component", component).
		Logger()
}
