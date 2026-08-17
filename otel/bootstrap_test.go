package otel_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/otel"
)

func TestInitialize_ConsoleOnly(t *testing.T) {
	err := otel.Initialize("test-svc", false)
	assert.NoError(t, err)
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInitializeWithFile_WritesToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	closer, err := otel.InitializeWithFile("test-svc", false, otel.OutputFile, &otel.FileConfig{Path: path})
	require.NoError(t, err)
	require.NotNil(t, closer)
	defer closer.Close()

	logger := otel.ContextLogger(context.Background(), "test")
	logger.Info().Msg("hello-file")
	require.NoError(t, closer.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "hello-file")
	assert.Contains(t, string(content), "test-svc")
}

func TestInitializeWithFile_ConsoleOnlyReturnsNilCloser(t *testing.T) {
	closer, err := otel.InitializeWithFile("test-svc", false, otel.OutputConsole, nil)
	require.NoError(t, err)
	// The returned closer must be a true nil interface, not a typed-nil
	// *os.File, so that the documented `if closer != nil { closer.Close() }`
	// guard does not misfire and call Close on a nil file handle.
	require.True(t, closer == nil, "console-only output must return a true nil io.Closer, got %T", closer)
}

func TestInitializeWithFile_FailedValidationDoesNotMutateGlobalLevel(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(zerolog.InfoLevel) })

	// OutputFile without a fileConfig is invalid. With debug=true, a global
	// level mutation that runs before validation would leave the global level
	// at Debug even though initialization failed.
	closer, err := otel.InitializeWithFile("test-svc", true, otel.OutputFile, nil)
	require.Error(t, err)
	require.True(t, closer == nil, "failed init must return a nil closer, got %T", closer)
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel(),
		"failed initialization must not mutate the zerolog global level")
}

func TestLogLevel_Constants(t *testing.T) {
	assert.Equal(t, otel.LogLevel("debug"), otel.LogLevelDebug)
	assert.Equal(t, otel.LogLevel("info"), otel.LogLevelInfo)
	assert.Equal(t, otel.LogLevel("warn"), otel.LogLevelWarn)
	assert.Equal(t, otel.LogLevel("error"), otel.LogLevelError)
	assert.Equal(t, otel.LogLevel("none"), otel.LogLevelNone)
}
