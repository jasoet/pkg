package logging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	t.Run("sets debug level when debug is true", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr)

		// Call Initialize with debug=true
		Initialize("test-service", true)

		// Verify that the global level is set to Debug
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
		assert.Equal(t, zerolog.DebugLevel, zlog.Logger.GetLevel())
	})

	t.Run("sets info level when debug is false", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr)

		// Call Initialize with debug=false
		Initialize("prod-service", false)

		// Verify that the global level is set to Info
		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
		assert.Equal(t, zerolog.InfoLevel, zlog.Logger.GetLevel())
	})

	t.Run("uses console output by default", func(t *testing.T) {
		// Reset the global logger for testing
		zlog.Logger = zerolog.New(os.Stderr)

		// Initialize should work without panicking
		Initialize("test-service", false)

		// Verify logger is functional
		zlog.Logger.Info().Msg("test message")
	})
}

func TestInitializeWithFile(t *testing.T) {
	// Create temp directory for test logs
	tempDir := t.TempDir()

	t.Run("console only output", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		InitializeWithFile("console-service", true, OutputConsole, nil)

		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
		zlog.Logger.Info().Msg("console only message")
	})

	t.Run("file only output", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "file-only.log")
		InitializeWithFile("file-service", false, OutputFile, &FileConfig{Path: logFile})

		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

		// Write a log message
		zlog.Logger.Info().Str("test", "value").Msg("file only message")

		// Verify file exists and contains the message
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, "file only message")
		assert.Contains(t, logStr, "file-service")
		assert.Contains(t, logStr, `"test":"value"`)
		assert.Contains(t, logStr, `"level":"info"`)
	})

	t.Run("both console and file output", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "both.log")
		InitializeWithFile("dual-service", true, OutputConsole|OutputFile, &FileConfig{Path: logFile})

		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

		// Write a log message
		zlog.Logger.Debug().Str("key", "value").Msg("dual output message")

		// Verify file contains the message
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, "dual output message")
		assert.Contains(t, logStr, "dual-service")
		assert.Contains(t, logStr, `"key":"value"`)
		assert.Contains(t, logStr, `"level":"debug"`)
	})

	t.Run("file output with append mode", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "append.log")

		// First initialization
		InitializeWithFile("append-service", false, OutputFile, &FileConfig{Path: logFile})
		zlog.Logger.Info().Msg("first message")

		// Re-initialize (simulating app restart)
		zlog.Logger = zerolog.New(os.Stderr)
		InitializeWithFile("append-service", false, OutputFile, &FileConfig{Path: logFile})
		zlog.Logger.Info().Msg("second message")

		// Verify both messages are in the file
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, "first message")
		assert.Contains(t, logStr, "second message")

		// Verify we have two separate log entries
		lines := strings.Split(strings.TrimSpace(logStr), "\n")
		assert.Equal(t, 2, len(lines))
	})

	t.Run("panics when OutputFile specified without fileConfig", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		assert.Panics(t, func() {
			InitializeWithFile("panic-service", false, OutputFile, nil)
		})
	})

	t.Run("panics when OutputFile specified with empty path", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		assert.Panics(t, func() {
			InitializeWithFile("panic-service", false, OutputFile, &FileConfig{Path: ""})
		})
	})

	t.Run("panics when no output destination specified", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		assert.Panics(t, func() {
			InitializeWithFile("panic-service", false, 0, nil)
		})
	})

	t.Run("panics when file cannot be opened", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		invalidPath := "/invalid/nonexistent/directory/file.log"

		assert.Panics(t, func() {
			InitializeWithFile("panic-service", false, OutputFile, &FileConfig{Path: invalidPath})
		})
	})

	t.Run("creates file with correct permissions", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "permissions.log")
		InitializeWithFile("perm-service", false, OutputFile, &FileConfig{Path: logFile})

		// Write a message to ensure file is created
		zlog.Logger.Info().Msg("test")

		// Check file permissions
		info, err := os.Stat(logFile)
		require.NoError(t, err)

		// Verify permissions are 0644
		mode := info.Mode().Perm()
		assert.Equal(t, os.FileMode(0644), mode)
	})

	t.Run("multiple log levels to file", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "levels.log")
		InitializeWithFile("levels-service", true, OutputFile, &FileConfig{Path: logFile})

		// Write multiple levels
		zlog.Logger.Debug().Msg("debug message")
		zlog.Logger.Info().Msg("info message")
		zlog.Logger.Warn().Msg("warn message")
		zlog.Logger.Error().Msg("error message")

		// Verify all levels are in the file
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, `"level":"debug"`)
		assert.Contains(t, logStr, `"level":"info"`)
		assert.Contains(t, logStr, `"level":"warn"`)
		assert.Contains(t, logStr, `"level":"error"`)
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
		assert.Equal(t, globalLogger.GetLevel(), logger.GetLevel())

		// Verify logger is not nil and can log without panic
		logger.Info().Msg("test message")
	})

	t.Run("inherits level from global logger", func(t *testing.T) {
		// Set global logger with specific level
		zlog.Logger = zerolog.New(os.Stderr).Level(zerolog.WarnLevel)

		ctx := context.Background()
		logger := ContextLogger(ctx, "warn-component")

		// Verify logger inherits warn level
		assert.Equal(t, zerolog.WarnLevel, logger.GetLevel())
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

	t.Run("works with file output", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "context.log")

		zlog.Logger = zerolog.New(os.Stderr)
		InitializeWithFile("context-service", false, OutputFile, &FileConfig{Path: logFile})

		ctx := context.Background()
		logger := ContextLogger(ctx, "my-component")

		logger.Info().Str("user_id", "123").Msg("user action")

		// Verify file contains the message with component
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, "user action")
		assert.Contains(t, logStr, `"component":"my-component"`)
		assert.Contains(t, logStr, `"user_id":"123"`)
	})
}

func TestIntegration(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("complete workflow with file logging", func(t *testing.T) {
		zlog.Logger = zerolog.New(os.Stderr)

		logFile := filepath.Join(tempDir, "integration.log")

		// Initialize the logger with both outputs
		InitializeWithFile("integration-service", true, OutputConsole|OutputFile, &FileConfig{Path: logFile})

		// Create a context logger
		ctx := context.Background()
		logger := ContextLogger(ctx, "integration-component")

		// Log various messages
		logger.Debug().Msg("Debug message")
		logger.Info().Str("key", "value").Msg("Info message")
		logger.Warn().Int("count", 42).Msg("Warning message")

		// Also use global logger
		globalLogger := zlog.Logger
		globalLogger.Info().Msg("Global logger message")

		// Verify file contains all messages
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		logStr := string(content)
		assert.Contains(t, logStr, "Debug message")
		assert.Contains(t, logStr, "Info message")
		assert.Contains(t, logStr, "Warning message")
		assert.Contains(t, logStr, "Global logger message")
		assert.Contains(t, logStr, `"component":"integration-component"`)
		assert.Contains(t, logStr, "integration-service")
	})
}

func TestOutputDestination(t *testing.T) {
	t.Run("bitwise operations work correctly", func(t *testing.T) {
		// Test individual flags
		assert.Equal(t, OutputDestination(1), OutputConsole)
		assert.Equal(t, OutputDestination(2), OutputFile)

		// Test combination
		combined := OutputConsole | OutputFile
		assert.Equal(t, OutputDestination(3), combined)

		// Test checking flags
		assert.NotEqual(t, 0, combined&OutputConsole)
		assert.NotEqual(t, 0, combined&OutputFile)

		// Test single flag
		consoleOnly := OutputConsole
		assert.NotEqual(t, 0, consoleOnly&OutputConsole)
		assert.Equal(t, OutputDestination(0), consoleOnly&OutputFile)
	})
}
