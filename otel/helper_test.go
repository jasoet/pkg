package otel

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/noop"
)

func TestNewLogHelper(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel config", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		require.NotNil(t, helper)
		assert.Nil(t, helper.otelLogger, "otelLogger must be nil when config is nil")
		assert.Equal(t, "test.Function", helper.function)
	})

	t.Run("with OTel config but logging disabled", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service"} // LoggerProvider nil → disabled
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		require.NotNil(t, helper)
		assert.Nil(t, helper.otelLogger, "otelLogger must be nil when logging is disabled")
	})

	t.Run("with OTel config and logging enabled", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			LoggerProvider: noop.NewLoggerProvider(),
		}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		require.NotNil(t, helper)
		assert.NotNil(t, helper.otelLogger, "otelLogger must be set when logging is enabled")
		assert.Equal(t, "test.Function", helper.function)
	})
}

// TestNewLogHelper_FallbackLevelAndFields verifies the zerolog fallback used
// when OTel is not configured: it defaults to Info level (Debug is filtered)
// and does not mislabel the instrumentation scope as the service name.
func TestNewLogHelper_FallbackLevelAndFields(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	// nil config → zerolog fallback. scopeName is a module path, not a service.
	h := NewLogHelper(context.Background(), nil, "github.com/jasoet/pkg/v3/argo", "argo.Run")
	h.Debug("debug-should-be-filtered")
	h.Info("info-should-appear")

	require.NoError(t, w.Close())
	os.Stderr = origStderr
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	got := string(out)

	// Default level is Info: Debug must be filtered out.
	assert.NotContains(t, got, "debug-should-be-filtered", "fallback logger must default to Info level")
	assert.Contains(t, got, "info-should-appear")

	// The scope (a module path) must not be mislabeled as the service field.
	assert.NotContains(t, got, "service=", "nil-config fallback must not emit a service field")
	assert.Contains(t, got, "scope=", "fallback must record the scope under a distinct field")
	assert.Contains(t, got, "github.com/jasoet/pkg/v3/argo")
	assert.Contains(t, got, "argo.Run")
}

// TestNewLogHelper_FallbackUsesServiceNameWhenAvailable verifies that when a
// config carries a ServiceName, the fallback labels it as service (not scope).
func TestNewLogHelper_FallbackUsesServiceNameWhenAvailable(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	// Config with a ServiceName but logging disabled → zerolog fallback.
	cfg := &Config{ServiceName: "billing"}
	h := NewLogHelper(context.Background(), cfg, "service.billing", "")
	h.Info("service-name-present")

	require.NoError(t, w.Close())
	os.Stderr = origStderr
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	got := string(out)

	assert.Contains(t, got, "service=")
	assert.Contains(t, got, "billing")
}

func TestLogHelper_Debug(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		assert.NotPanics(t, func() {
			helper.Debug("debug message")
			helper.Debug("debug message with fields", F("key", "value"), F("count", 42))
		})
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service", LoggerProvider: noop.NewLoggerProvider()}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Debug("debug message")
			helper.Debug("debug message with fields", F("key", "value"), F("count", 42))
		})
	})
}

func TestLogHelper_Info(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		assert.NotPanics(t, func() {
			helper.Info("info message")
			helper.Info("info message with fields", F("key", "value"), F("enabled", true))
		})
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service", LoggerProvider: noop.NewLoggerProvider()}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Info("info message")
			helper.Info("info message with fields", F("key", "value"), F("enabled", true))
		})
	})
}

func TestLogHelper_Warn(t *testing.T) {
	ctx := context.Background()

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		assert.NotPanics(t, func() {
			helper.Warn("warning message")
			helper.Warn("warning message with fields", F("key", "value"), F("ratio", 0.75))
		})
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service", LoggerProvider: noop.NewLoggerProvider()}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Warn("warning message")
			helper.Warn("warning message with fields", F("key", "value"), F("ratio", 0.75))
		})
	})
}

func TestLogHelper_Error(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	t.Run("without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		assert.NotPanics(t, func() {
			helper.Error(testErr, "error message")
			helper.Error(testErr, "error message with fields", F("key", "value"), F("code", 500))
		})
	})

	t.Run("with OTel", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service", LoggerProvider: noop.NewLoggerProvider()}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Error(testErr, "error message")
			helper.Error(testErr, "error message with fields", F("key", "value"), F("code", 500))
		})
	})
}

func TestLogHelper_MixedTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("various data types without OTel", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function")
		assert.NotPanics(t, func() {
			helper.Info("mixed types",
				F("string", "value"),
				F("int", 123),
				F("int64", int64(456)),
				F("bool", true),
				F("float64", 3.14),
			)
		})
	})

	t.Run("various data types with OTel", func(t *testing.T) {
		cfg := &Config{ServiceName: "test-service", LoggerProvider: noop.NewLoggerProvider()}
		helper := NewLogHelper(ctx, cfg, "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Info("mixed types",
				F("string", "value"),
				F("int", 123),
				F("int64", int64(456)),
				F("bool", true),
				F("float64", 3.14),
			)
		})
	})
}

// TestLogHelper_LogLevelFiltering tests that logs are filtered based on the
// configured level without panicking.
func TestLogHelper_LogLevelFiltering(t *testing.T) {
	ctx := context.Background()

	newCfg := func(t *testing.T, level LogLevel) *Config {
		t.Helper()
		lp, err := NewLoggerProviderWithOptions("test-service", WithLogLevel(level))
		require.NoError(t, err)
		shutdownProvider(t, lp)
		return &Config{ServiceName: "test-service", LoggerProvider: lp}
	}

	t.Run("warn level filters info and debug", func(t *testing.T) {
		helper := NewLogHelper(ctx, newCfg(t, LogLevelWarn), "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Debug("filtered")
			helper.Info("filtered")
			helper.Warn("appears")
			helper.Error(errors.New("test error"), "appears")
		})
	})

	t.Run("info level filters debug only", func(t *testing.T) {
		helper := NewLogHelper(ctx, newCfg(t, LogLevelInfo), "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Debug("filtered")
			helper.Info("appears")
			helper.Warn("appears")
			helper.Error(errors.New("test error"), "appears")
		})
	})

	t.Run("error level filters all except errors", func(t *testing.T) {
		helper := NewLogHelper(ctx, newCfg(t, LogLevelError), "test-scope", "test.Function")
		assert.NotPanics(t, func() {
			helper.Debug("filtered")
			helper.Info("filtered")
			helper.Warn("filtered")
			helper.Error(errors.New("test error"), "appears")
		})
	})
}

func TestLogHelper_WithFields_SliceIsolation(t *testing.T) {
	ctx := context.Background()

	t.Run("sibling helpers do not share fields", func(t *testing.T) {
		parent := NewLogHelper(ctx, nil, "", "test.Function").
			WithFields(F("base", "value"))

		child1 := parent.WithFields(F("child", "one"))
		child2 := parent.WithFields(F("child", "two"))

		require.Len(t, parent.baseFields, 1)
		require.Len(t, child1.baseFields, 2)
		require.Len(t, child2.baseFields, 2)

		assert.Equal(t, "one", child1.baseFields[1].Value)
		assert.Equal(t, "two", child2.baseFields[1].Value)
		assert.Equal(t, "value", parent.baseFields[0].Value)
	})

	t.Run("log calls do not mutate baseFields", func(t *testing.T) {
		helper := NewLogHelper(ctx, nil, "", "test.Function").
			WithFields(F("base", "value"))

		originalLen := len(helper.baseFields)

		helper.Info("msg1", F("extra", "a"))
		helper.Info("msg2", F("extra", "b"))
		helper.Error(errors.New("err"), "msg3", F("extra", "c"))

		assert.Len(t, helper.baseFields, originalLen)
	})
}
