package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// initMu protects global logger assignment during initialization.
var initMu sync.Mutex

// OutputDestination defines where logs should be written.
// Multiple destinations can be combined using bitwise OR.
type OutputDestination int

const (
	OutputConsole OutputDestination = 1 << 0 // Output to console (stderr)
	OutputFile    OutputDestination = 1 << 1 // Output to file
)

// FileConfig configures file-based logging.
// File rotation should be managed by OS tools like logrotate.
type FileConfig struct {
	Path string // Log file path (required when OutputFile is used)
}

// InitializeWithFile sets up the zerolog global logger with flexible output options.
// Supports console output, file output, or both simultaneously.
//
// When file output is enabled, the returned io.Closer must be closed by the caller
// to release the file handle (typically via defer). When only console output is used,
// the returned closer is nil.
//
// Parameters:
//   - serviceName: Name of the service, added as a field to all log entries
//   - debug: If true, sets log level to Debug, otherwise Info
//   - output: Output destination flags (OutputConsole, OutputFile, or both combined with |)
//   - fileConfig: File configuration (required if OutputFile is specified, can be nil otherwise)
//
// Returns an io.Closer (non-nil when file output is enabled) and an error if configuration is invalid.
//
// Example:
//
//	// Console only
//	_, err := InitializeWithFile("my-service", true, OutputConsole, nil)
//
//	// File only
//	closer, err := InitializeWithFile("my-service", false, OutputFile, &FileConfig{Path: "app.log"})
//	if err != nil { log.Fatal(err) }
//	defer closer.Close()
//
//	// Both console and file
//	closer, err := InitializeWithFile("my-service", true, OutputConsole|OutputFile, &FileConfig{Path: "app.log"})
//	if err != nil { log.Fatal(err) }
//	defer closer.Close()
func InitializeWithFile(serviceName string, debug bool, output OutputDestination, fileConfig *FileConfig) (io.Closer, error) {
	initMu.Lock()
	defer initMu.Unlock()

	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)

	var writers []io.Writer
	var file *os.File

	// Console output (human-readable, colored)
	if output&OutputConsole != 0 {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	// File output (JSON, structured)
	if output&OutputFile != 0 {
		if fileConfig == nil || fileConfig.Path == "" {
			return nil, fmt.Errorf("fileConfig with Path is required when OutputFile is specified")
		}

		var err error
		file, err = os.OpenFile(fileConfig.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", fileConfig.Path, err)
		}

		writers = append(writers, file)
	}

	// Ensure at least one output is configured
	if len(writers) == 0 {
		return nil, fmt.Errorf("at least one output destination must be specified")
	}

	// Create multi-writer if multiple outputs
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = zerolog.MultiLevelWriter(writers...)
	}

	zlog.Logger = zerolog.New(writer).
		With().
		Timestamp().
		Str("service", serviceName).
		Int("pid", os.Getpid()).
		Caller().
		Logger().
		Level(level)

	return file, nil
}

// Initialize sets up the zerolog global logger with standard fields for console-only output.
// This function should be called once at the start of your application.
// After calling Initialize, you can use zerolog's log package functions directly
// (log.Debug(), log.Info(), etc.) or create component-specific loggers with ContextLogger.
//
// This is a convenience wrapper around InitializeWithFile for console-only output.
// For file output or multiple outputs, use InitializeWithFile directly.
//
// Parameters:
//   - serviceName: Name of the service, added as a field to all log entries
//   - debug: If true, sets log level to Debug, otherwise Info
//
// Returns an error if the logger cannot be initialized.
func Initialize(serviceName string, debug bool) error {
	_, err := InitializeWithFile(serviceName, debug, OutputConsole, nil)
	return err
}

// ContextLogger creates a component-scoped logger from the global logger.
// The context is associated with the logger for use by zerolog hooks that
// read from context (e.g., trace correlation), but context.WithValue entries
// are not automatically extracted into log fields.
//
// Parameters:
//   - ctx: Context associated with the logger (for hooks and cancellation, not value extraction)
//   - component: Name of the component, added as a field to all log entries
//
// Returns:
//   - A zerolog.Logger instance with the component field and associated context
func ContextLogger(ctx context.Context, component string) zerolog.Logger {
	return zlog.With().
		Ctx(ctx).
		Str("component", component).
		Logger()
}

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelNone  LogLevel = "none"
)
