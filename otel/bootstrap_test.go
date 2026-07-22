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

func TestLogLevel_Constants(t *testing.T) {
	assert.Equal(t, otel.LogLevel("debug"), otel.LogLevelDebug)
	assert.Equal(t, otel.LogLevel("info"), otel.LogLevelInfo)
	assert.Equal(t, otel.LogLevel("warn"), otel.LogLevelWarn)
	assert.Equal(t, otel.LogLevel("error"), otel.LogLevelError)
	assert.Equal(t, otel.LogLevel("none"), otel.LogLevelNone)
}
