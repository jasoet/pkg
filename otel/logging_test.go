package otel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// memLogExporter is an in-memory sdklog.Exporter that records every exported
// log record for assertions. It is safe for concurrent use.
type memLogExporter struct {
	mu   sync.Mutex
	recs []sdklog.Record
}

func (e *memLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range records {
		e.recs = append(e.recs, records[i].Clone())
	}
	return nil
}

func (e *memLogExporter) Shutdown(context.Context) error   { return nil }
func (e *memLogExporter) ForceFlush(context.Context) error { return nil }

func (e *memLogExporter) records() []sdklog.Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]sdklog.Record(nil), e.recs...)
}

// recordAttrs collects a record's attributes into a map for easy assertions.
func recordAttrs(r *sdklog.Record) map[string]any {
	out := make(map[string]any)
	r.WalkAttributes(func(kv log.KeyValue) bool {
		out[kv.Key] = kv.Value.AsString()
		switch kv.Value.Kind() {
		case log.KindBool:
			out[kv.Key] = kv.Value.AsBool()
		case log.KindInt64:
			out[kv.Key] = kv.Value.AsInt64()
		case log.KindFloat64:
			out[kv.Key] = kv.Value.AsFloat64()
		case log.KindString:
			out[kv.Key] = kv.Value.AsString()
		}
		return true
	})
	return out
}

// shutdownProvider registers a t.Cleanup that shuts the provider down, which
// stops any background BatchProcessor goroutine from leaking across tests.
func shutdownProvider(t *testing.T, provider log.LoggerProvider) {
	t.Helper()
	sp, ok := provider.(interface {
		Shutdown(context.Context) error
	})
	require.True(t, ok, "provider must support Shutdown")
	t.Cleanup(func() {
		assert.NoError(t, sp.Shutdown(context.Background()))
	})
}

// TestWithConsoleOutput tests the WithConsoleOutput option
func TestWithConsoleOutput(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "console output enabled", enabled: true, expected: true},
		{name: "console output disabled", enabled: false, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			WithConsoleOutput(tt.enabled)(cfg)
			assert.Equal(t, tt.expected, cfg.consoleOutput)
		})
	}
}

// TestWithOTLPEndpoint tests the WithOTLPEndpoint option stores the raw values.
func TestWithOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		endpoint         string
		insecure         bool
		expectedEndpoint string
		expectedInsecure bool
	}{
		{"host:port secure", "localhost:4318", false, "localhost:4318", false},
		{"host:port insecure", "localhost:4318", true, "localhost:4318", true},
		{"https url", "https://otel-collector:4318", false, "https://otel-collector:4318", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			WithOTLPEndpoint(tt.endpoint, tt.insecure)(cfg)
			assert.Equal(t, tt.expectedEndpoint, cfg.otlpEndpoint)
			assert.Equal(t, tt.expectedInsecure, cfg.otlpInsecure)
			assert.True(t, cfg.otlpEndpointSet, "otlpEndpointSet must record that the option was applied")
		})
	}
}

// TestWithLogLevel tests the WithLogLevel option
func TestWithLogLevel(t *testing.T) {
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelNone}
	for _, lvl := range levels {
		t.Run(string(lvl), func(t *testing.T) {
			cfg := &loggerProviderConfig{}
			WithLogLevel(lvl)(cfg)
			assert.Equal(t, lvl, cfg.logLevel)
		})
	}
}

// TestNewLoggerProviderWithOptions_NoOTLP tests console-only providers.
func TestNewLoggerProviderWithOptions_NoOTLP(t *testing.T) {
	tests := []struct {
		name string
		opts []LoggerProviderOption
	}{
		{"debug mode without OTLP", []LoggerProviderOption{WithLogLevel(LogLevelDebug)}},
		{"info mode without OTLP", nil},
		{"explicit warn level without OTLP", []LoggerProviderOption{WithLogLevel(LogLevelWarn)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewLoggerProviderWithOptions("test-service", tt.opts...)
			require.NoError(t, err)
			require.NotNil(t, provider)
			shutdownProvider(t, provider)
		})
	}
}

// TestNewLoggerProviderWithOptions_EmptyOTLPEndpointErrors verifies that
// explicitly enabling OTLP with an empty endpoint is surfaced as an error
// rather than silently disabling OTLP export.
func TestNewLoggerProviderWithOptions_EmptyOTLPEndpointErrors(t *testing.T) {
	provider, err := NewLoggerProviderWithOptions(
		"test-service",
		WithOTLPEndpoint("", true),
	)
	require.Error(t, err, "empty OTLP endpoint must error")
	assert.Nil(t, provider)
}

// TestNewLoggerProviderWithOptions_OTLPEndpointURL verifies that a URL-shaped
// endpoint (with scheme) is routed to the collector's /v1/logs path. With the
// previous WithEndpoint-only wiring the scheme was treated as part of the host
// and the export silently failed.
func TestNewLoggerProviderWithOptions_OTLPEndpointURL(t *testing.T) {
	var gotPath atomic.Value // string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	provider, err := NewLoggerProviderWithOptions(
		"test-service",
		WithOTLPEndpoint(srv.URL, true), // srv.URL carries an http:// scheme
		WithConsoleOutput(false),
	)
	require.NoError(t, err)
	require.NotNil(t, provider)
	shutdownProvider(t, provider)

	logger := provider.Logger("test-scope")
	var rec log.Record
	rec.SetBody(log.StringValue("hello"))
	rec.SetSeverity(log.SeverityInfo)
	logger.Emit(context.Background(), rec)

	ff, ok := provider.(interface {
		ForceFlush(context.Context) error
	})
	require.True(t, ok)
	require.NoError(t, ff.ForceFlush(context.Background()))

	require.Eventually(t, func() bool {
		v := gotPath.Load()
		return v != nil && v.(string) == "/v1/logs"
	}, 3*time.Second, 10*time.Millisecond, "collector should receive an export at /v1/logs")
}

// TestNewLoggerProviderWithOptions_OTLPProviderLifecycle verifies a provider
// with an OTLP endpoint is created without eagerly dialing, and is shut down
// via cleanup so its BatchProcessor goroutine does not leak.
func TestNewLoggerProviderWithOptions_OTLPProviderLifecycle(t *testing.T) {
	provider, err := NewLoggerProviderWithOptions(
		"test-service",
		WithOTLPEndpoint("nonexistent-host:9999", true),
		WithConsoleOutput(true),
		WithLogLevel(LogLevelInfo),
	)
	require.NoError(t, err, "otlploghttp.New must not dial eagerly")
	require.NotNil(t, provider)
	shutdownProvider(t, provider)
}

// TestNewLoggerProviderWithOptions_LogLevelPriority tests log level handling.
func TestNewLoggerProviderWithOptions_LogLevelPriority(t *testing.T) {
	levels := []LogLevel{LogLevelError, LogLevelDebug, LogLevelWarn, ""}
	for _, lvl := range levels {
		name := string(lvl)
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			var opts []LoggerProviderOption
			if lvl != "" {
				opts = append(opts, WithLogLevel(lvl))
			}
			provider, err := NewLoggerProviderWithOptions("test-service", opts...)
			require.NoError(t, err)
			require.NotNil(t, provider)
			shutdownProvider(t, provider)
		})
	}
}

// TestNewLoggerProviderWithOptions_MultipleOptions tests combining options.
func TestNewLoggerProviderWithOptions_MultipleOptions(t *testing.T) {
	t.Run("all options combined without OTLP", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(true),
			WithLogLevel(LogLevelWarn),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})

	t.Run("disable console output yields a valid silent provider", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(false),
			WithLogLevel(LogLevelInfo),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})
}

// TestNewLoggerProviderWithOptions_ConsoleDisabledIsSilent verifies that
// disabling console output with no OTLP endpoint produces a provider that
// does not re-add a console exporter behind the caller's back.
func TestNewLoggerProviderWithOptions_ConsoleDisabledIsSilent(t *testing.T) {
	provider, err := NewLoggerProviderWithOptions(
		"test-service",
		WithConsoleOutput(false),
	)
	require.NoError(t, err)
	require.NotNil(t, provider)
	shutdownProvider(t, provider)

	sdkProvider, ok := provider.(*sdklog.LoggerProvider)
	require.True(t, ok)

	// A provider with no processors reports Enabled=false, proving no console
	// exporter was silently re-added.
	logger := sdkProvider.Logger("scope")
	enabled := logger.(interface {
		Enabled(context.Context, log.EnabledParameters) bool
	}).Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityError})
	assert.False(t, enabled, "console-disabled provider with no OTLP must have no processors")
}

// TestLoggerProviderConfig_Defaults tests default configuration values.
func TestLoggerProviderConfig_Defaults(t *testing.T) {
	cfg := &loggerProviderConfig{serviceName: "test-service", consoleOutput: true}
	assert.True(t, cfg.consoleOutput)
	assert.Empty(t, cfg.otlpEndpoint)
	assert.False(t, cfg.otlpInsecure)
	assert.Empty(t, string(cfg.logLevel))
}

// TestSetupZerologConsole exercises provider creation with different levels.
func TestSetupZerologConsole(t *testing.T) {
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelNone, "unknown"}
	for _, lvl := range levels {
		t.Run(string(lvl), func(t *testing.T) {
			provider, err := NewLoggerProviderWithOptions(
				"test-service",
				WithLogLevel(lvl),
				WithConsoleOutput(true),
			)
			require.NoError(t, err)
			require.NotNil(t, provider)
			shutdownProvider(t, provider)
		})
	}
}

// TestNewLoggerProviderWithOptions_Integration tests realistic usage patterns.
func TestNewLoggerProviderWithOptions_Integration(t *testing.T) {
	t.Run("local development setup", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("my-service", WithConsoleOutput(true))
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
		assert.NotNil(t, provider.Logger("test-scope"))
	})

	t.Run("production-like setup without collector", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"my-service",
			WithConsoleOutput(true),
			WithLogLevel(LogLevelInfo),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})

	t.Run("silent mode", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"my-service",
			WithConsoleOutput(false),
			WithLogLevel(LogLevelNone),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})
}

// TestLoggerProviderCompatibility verifies interface conformance.
func TestLoggerProviderCompatibility(t *testing.T) {
	t.Run("provider implements log.LoggerProvider", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("test-service")
		require.NoError(t, err)
		shutdownProvider(t, provider)
		var _ log.LoggerProvider = provider
	})

	t.Run("logger can be obtained from provider", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions("test-service")
		require.NoError(t, err)
		shutdownProvider(t, provider)

		logger := provider.Logger("test-scope")
		require.NotNil(t, logger)
		var _ log.Logger = logger
	})
}

// TestNewLoggerProviderWithOptions_EmptyServiceName tests empty service name.
func TestNewLoggerProviderWithOptions_EmptyServiceName(t *testing.T) {
	provider, err := NewLoggerProviderWithOptions("")
	require.NoError(t, err)
	require.NotNil(t, provider)
	shutdownProvider(t, provider)
}

// TestLoggerProviderOptions_Chaining tests option chaining and last-wins.
func TestLoggerProviderOptions_Chaining(t *testing.T) {
	t.Run("chain multiple options", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithConsoleOutput(true),
			WithLogLevel(LogLevelDebug),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})

	t.Run("last option wins for same config", func(t *testing.T) {
		provider, err := NewLoggerProviderWithOptions(
			"test-service",
			WithLogLevel(LogLevelDebug),
			WithLogLevel(LogLevelError),
		)
		require.NoError(t, err)
		require.NotNil(t, provider)
		shutdownProvider(t, provider)
	})
}

// TestLoggerProvider_NoopComparison compares behavior with the noop provider.
func TestLoggerProvider_NoopComparison(t *testing.T) {
	ourProvider, err := NewLoggerProviderWithOptions("test-service")
	require.NoError(t, err)
	shutdownProvider(t, ourProvider)

	noopProvider := noop.NewLoggerProvider()

	ourLogger := ourProvider.Logger("test-scope")
	noopLogger := noopProvider.Logger("test-scope")
	require.NotNil(t, ourLogger)
	require.NotNil(t, noopLogger)

	ctx := context.Background()
	var record log.Record
	record.SetBody(log.StringValue("test message"))

	assert.NotPanics(t, func() {
		ourLogger.Emit(ctx, record)
		noopLogger.Emit(ctx, record)
	})
}

// TestLoggingPipeline_TraceCorrelationAndSeverityMapping verifies the full
// logging pipeline: trace_id/span_id correlation on emitted records, severity
// and severity-text mapping, and typed-attribute mapping, using an in-memory
// sdklog exporter rather than only asserting "does not panic".
func TestLoggingPipeline_TraceCorrelationAndSeverityMapping(t *testing.T) {
	exporter := &memLogExporter{}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)))
	t.Cleanup(func() { require.NoError(t, lp.Shutdown(context.Background())) })

	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

	cfg := &Config{
		ServiceName:    "pipeline-service",
		TracerProvider: tp,
		LoggerProvider: lp,
	}
	ctx := ContextWithConfig(context.Background(), cfg)

	span := StartSpan(ctx, "service.pipeline", "DoWork")
	logger := span.FunctionLogger("service.pipeline", "DoWork")

	logger.Debug("debug msg", F("dbg", 1))
	logger.Info("info msg", F("user.id", "u-1"), F("count", int64(7)), F("ratio", 0.5), F("ok", true))
	logger.Warn("warn msg")
	logger.Error(errors.New("boom"), "error msg", F("code", 500))
	span.End()

	recs := exporter.records()
	require.Len(t, recs, 4, "all four levels must be emitted to the exporter")

	// Trace correlation: every record carries the active span's IDs.
	sc := span.Span().SpanContext()
	require.True(t, sc.TraceID().IsValid(), "span must have a valid trace id")
	for i := range recs {
		assert.Equal(t, sc.TraceID(), recs[i].TraceID(), "record %d trace_id must match span", i)
		assert.Equal(t, sc.SpanID(), recs[i].SpanID(), "record %d span_id must match span", i)
	}

	// Severity + severity text mapping.
	assert.Equal(t, log.SeverityDebug, recs[0].Severity())
	assert.Equal(t, "DEBUG", recs[0].SeverityText())
	assert.Equal(t, log.SeverityInfo, recs[1].Severity())
	assert.Equal(t, "INFO", recs[1].SeverityText())
	assert.Equal(t, log.SeverityWarn, recs[2].Severity())
	assert.Equal(t, "WARN", recs[2].SeverityText())
	assert.Equal(t, log.SeverityError, recs[3].Severity())
	assert.Equal(t, "ERROR", recs[3].SeverityText())

	// Body mapping.
	assert.Equal(t, "info msg", recs[1].Body().AsString())

	// Typed-attribute mapping on the info record.
	infoAttrs := recordAttrs(&recs[1])
	assert.Equal(t, "DoWork", infoAttrs["function"])
	assert.Equal(t, "u-1", infoAttrs["user.id"])
	assert.Equal(t, int64(7), infoAttrs["count"])
	assert.InEpsilon(t, 0.5, infoAttrs["ratio"], 1e-9)
	assert.Equal(t, true, infoAttrs["ok"])

	// The error record includes the error message and extra fields.
	errAttrs := recordAttrs(&recs[3])
	assert.Equal(t, "boom", errAttrs["error"])
	assert.Equal(t, int64(500), errAttrs["code"])
}
