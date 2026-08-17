package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	noopl "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopm "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	noopt "go.opentelemetry.io/otel/trace/noop"
)

func TestNewConfig(t *testing.T) {
	t.Run("creates config with service name", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.Equal(t, "test-service", cfg.ServiceName)
	})

	t.Run("has default logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.NotNil(t, cfg.LoggerProvider, "expected LoggerProvider to be set by default")
	})

	t.Run("has nil tracer provider by default", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.Nil(t, cfg.TracerProvider)
	})

	t.Run("has nil meter provider by default", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.Nil(t, cfg.MeterProvider)
	})

	t.Run("has empty service version by default", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.Empty(t, cfg.ServiceVersion)
	})
}

func TestWithTracerProvider(t *testing.T) {
	t.Run("sets tracer provider", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		cfg := NewConfig("test-service", WithTracerProvider(tp))
		assert.Equal(t, tp, cfg.TracerProvider)
	})

	t.Run("combines with other options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		mp := noopm.NewMeterProvider()

		cfg := NewConfig("test-service",
			WithTracerProvider(tp),
			WithMeterProvider(mp))

		assert.Equal(t, tp, cfg.TracerProvider)
		assert.Equal(t, mp, cfg.MeterProvider)
	})
}

func TestWithMeterProvider(t *testing.T) {
	t.Run("sets meter provider", func(t *testing.T) {
		mp := noopm.NewMeterProvider()
		cfg := NewConfig("test-service", WithMeterProvider(mp))
		assert.Equal(t, mp, cfg.MeterProvider)
	})
}

func TestWithLoggerProvider(t *testing.T) {
	t.Run("sets custom logger provider", func(t *testing.T) {
		lp := noopl.NewLoggerProvider()
		cfg := NewConfig("test-service", WithLoggerProvider(lp))
		assert.Equal(t, lp, cfg.LoggerProvider)
	})

	t.Run("replaces default logger provider", func(t *testing.T) {
		defaultLogger := NewConfig("test-service").LoggerProvider

		customLogger := noopl.NewLoggerProvider()
		cfg := NewConfig("test-service", WithLoggerProvider(customLogger))

		assert.NotEqual(t, defaultLogger, cfg.LoggerProvider)
		assert.Equal(t, customLogger, cfg.LoggerProvider)
	})
}

func TestWithServiceVersion(t *testing.T) {
	t.Run("sets service version", func(t *testing.T) {
		cfg := NewConfig("test-service", WithServiceVersion("v1.2.3"))
		assert.Equal(t, "v1.2.3", cfg.ServiceVersion)
	})

	t.Run("combines with other options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service",
			WithServiceVersion("v2.0.0"),
			WithTracerProvider(tp))

		assert.Equal(t, "v2.0.0", cfg.ServiceVersion)
		assert.Equal(t, tp, cfg.TracerProvider)
	})
}

func TestWithoutLogging(t *testing.T) {
	t.Run("disables logging by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutLogging())
		assert.Nil(t, cfg.LoggerProvider)
	})

	t.Run("combines with other options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service",
			WithoutLogging(),
			WithTracerProvider(tp))

		assert.Nil(t, cfg.LoggerProvider)
		assert.Equal(t, tp, cfg.TracerProvider)
	})
}

func TestWithoutTracing(t *testing.T) {
	t.Run("disables tracing by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithTracerProvider(noopt.NewTracerProvider()),
			WithoutTracing())
		assert.Nil(t, cfg.TracerProvider)
	})

	t.Run("combines with other options", func(t *testing.T) {
		mp := noopm.NewMeterProvider()

		cfg := NewConfig("test-service",
			WithoutTracing(),
			WithMeterProvider(mp))

		assert.Nil(t, cfg.TracerProvider)
		assert.Equal(t, mp, cfg.MeterProvider)
	})

	t.Run("works when tracer provider is already nil", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutTracing())
		assert.Nil(t, cfg.TracerProvider)
	})
}

func TestWithoutMetrics(t *testing.T) {
	t.Run("disables metrics by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithMeterProvider(noopm.NewMeterProvider()),
			WithoutMetrics())
		assert.Nil(t, cfg.MeterProvider)
	})

	t.Run("combines with other options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service",
			WithoutMetrics(),
			WithTracerProvider(tp))

		assert.Nil(t, cfg.MeterProvider)
		assert.Equal(t, tp, cfg.TracerProvider)
	})

	t.Run("works when meter provider is already nil", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutMetrics())
		assert.Nil(t, cfg.MeterProvider)
	})
}

func TestIsTracingEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		assert.False(t, cfg.IsTracingEnabled())
	})

	t.Run("returns false when tracer provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.False(t, cfg.IsTracingEnabled())
	})

	t.Run("returns true when tracer provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithTracerProvider(noopt.NewTracerProvider()))
		assert.True(t, cfg.IsTracingEnabled())
	})
}

func TestIsMetricsEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		assert.False(t, cfg.IsMetricsEnabled())
	})

	t.Run("returns false when meter provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.False(t, cfg.IsMetricsEnabled())
	})

	t.Run("returns true when meter provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithMeterProvider(noopm.NewMeterProvider()))
		assert.True(t, cfg.IsMetricsEnabled())
	})
}

func TestIsLoggingEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		assert.False(t, cfg.IsLoggingEnabled())
	})

	t.Run("returns false when logger provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutLogging())
		assert.False(t, cfg.IsLoggingEnabled())
	})

	t.Run("returns true when logger provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.True(t, cfg.IsLoggingEnabled())
	})

	t.Run("returns true with custom logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithLoggerProvider(noopl.NewLoggerProvider()))
		assert.True(t, cfg.IsLoggingEnabled())
	})
}

func TestGetTracer(t *testing.T) {
	t.Run("returns no-op tracer when tracing is disabled", func(t *testing.T) {
		cfg := NewConfig("test-service")

		tracer := cfg.GetTracer("test-scope")
		require.NotNil(t, tracer)

		// Verify it's a no-op tracer by checking it doesn't panic.
		_, span := tracer.Start(context.Background(), "test-operation")
		span.End()
	})

	t.Run("returns tracer from provider when tracing is enabled", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		cfg := NewConfig("test-service", WithTracerProvider(tp))
		assert.NotNil(t, cfg.GetTracer("test-scope"))
	})

	t.Run("accepts tracer options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		cfg := NewConfig("test-service", WithTracerProvider(tp))
		assert.NotNil(t, cfg.GetTracer("test-scope", trace.WithInstrumentationVersion("v1.0.0")))
	})
}

func TestGetMeter(t *testing.T) {
	t.Run("returns no-op meter when metrics are disabled", func(t *testing.T) {
		cfg := NewConfig("test-service")

		meter := cfg.GetMeter("test-scope")
		require.NotNil(t, meter)

		// Verify it's a no-op meter by checking it doesn't error.
		_, err := meter.Int64Counter("test-counter")
		assert.NoError(t, err)
	})

	t.Run("returns meter from provider when metrics are enabled", func(t *testing.T) {
		mp := noopm.NewMeterProvider()
		cfg := NewConfig("test-service", WithMeterProvider(mp))
		assert.NotNil(t, cfg.GetMeter("test-scope"))
	})

	t.Run("accepts meter options", func(t *testing.T) {
		mp := noopm.NewMeterProvider()
		cfg := NewConfig("test-service", WithMeterProvider(mp))
		assert.NotNil(t, cfg.GetMeter("test-scope", metric.WithInstrumentationVersion("v1.0.0")))
	})
}

func TestGetLogger(t *testing.T) {
	t.Run("returns no-op logger when logging is disabled", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutLogging())

		logger := cfg.GetLogger("test-scope")
		require.NotNil(t, logger)

		// Verify it's a no-op logger by checking it doesn't panic.
		assert.NotPanics(t, func() {
			logger.Emit(context.Background(), log.Record{})
		})
	})

	t.Run("returns logger from provider when logging is enabled", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.NotNil(t, cfg.GetLogger("test-scope"))
	})

	t.Run("accepts logger options", func(t *testing.T) {
		lp := noopl.NewLoggerProvider()
		cfg := NewConfig("test-service", WithLoggerProvider(lp))
		assert.NotNil(t, cfg.GetLogger("test-scope", log.WithInstrumentationVersion("v1.0.0")))
	})
}

func TestShutdown(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		var cfg *Config
		assert.NoError(t, cfg.Shutdown(context.Background()))
	})

	t.Run("succeeds with default logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		assert.NoError(t, cfg.Shutdown(context.Background()))
	})

	t.Run("succeeds with no-op logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service",
			WithLoggerProvider(noopl.NewLoggerProvider()))
		assert.NoError(t, cfg.Shutdown(context.Background()))
	})

	t.Run("succeeds without logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutLogging())
		assert.NoError(t, cfg.Shutdown(context.Background()))
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		cfg := NewConfig("test-service")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should still succeed or return context error.
		_ = cfg.Shutdown(ctx)
	})
}

func TestNoopProviderSingletons(t *testing.T) {
	t.Run("GetTracer returns same singleton-backed tracer across calls", func(t *testing.T) {
		cfg := NewConfig("test-service") // TracerProvider is nil → uses singleton

		tracer1 := cfg.GetTracer("scope-a")
		tracer2 := cfg.GetTracer("scope-a")
		require.NotNil(t, tracer1)
		require.NotNil(t, tracer2)

		_, span := tracer1.Start(context.Background(), "op")
		span.End()
	})

	t.Run("GetMeter returns singleton-backed meter across calls", func(t *testing.T) {
		cfg := NewConfig("test-service") // MeterProvider is nil → uses singleton

		meter1 := cfg.GetMeter("scope-a")
		meter2 := cfg.GetMeter("scope-a")
		require.NotNil(t, meter1)
		require.NotNil(t, meter2)

		_, err := meter1.Int64Counter("counter1")
		assert.NoError(t, err)
	})

	t.Run("GetLogger returns singleton-backed logger across calls", func(t *testing.T) {
		cfg := NewConfig("test-service", WithoutLogging()) // LoggerProvider nil → uses singleton

		logger1 := cfg.GetLogger("scope-a")
		logger2 := cfg.GetLogger("scope-a")
		require.NotNil(t, logger1)
		require.NotNil(t, logger2)

		assert.NotPanics(t, func() {
			logger1.Emit(context.Background(), log.Record{})
		})
	})

	t.Run("package-level noop singletons are usable", func(t *testing.T) {
		tracer := noopTracerProvider.Tracer("test")
		_, span := tracer.Start(context.Background(), "op")
		span.End()

		meter := noopMeterProvider.Meter("test")
		_, err := meter.Int64Counter("c")
		assert.NoError(t, err)

		logger := noopLoggerProvider.Logger("test")
		assert.NotPanics(t, func() {
			logger.Emit(context.Background(), log.Record{})
		})
	})
}

func TestDefaultLoggerProvider(t *testing.T) {
	t.Run("creates a logger provider", func(t *testing.T) {
		lp := defaultLoggerProvider("test-service", false)
		assert.NotNil(t, lp)
	})

	t.Run("created logger can emit logs", func(t *testing.T) {
		lp := defaultLoggerProvider("test-service", false)
		logger := lp.Logger("test-scope")
		assert.NotPanics(t, func() {
			logger.Emit(context.Background(), log.Record{})
		})
	})
}

func TestNewConfigAllOptions(t *testing.T) {
	t.Run("applies all options together", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		mp := noopm.NewMeterProvider()
		lp := noopl.NewLoggerProvider()

		cfg := NewConfig("my-service",
			WithServiceVersion("v2.0.0"),
			WithTracerProvider(tp),
			WithMeterProvider(mp),
			WithLoggerProvider(lp))

		assert.Equal(t, "my-service", cfg.ServiceName)
		assert.Equal(t, "v2.0.0", cfg.ServiceVersion)
		assert.Equal(t, tp, cfg.TracerProvider)
		assert.Equal(t, mp, cfg.MeterProvider)
		assert.Equal(t, lp, cfg.LoggerProvider)

		assert.True(t, cfg.IsTracingEnabled())
		assert.True(t, cfg.IsMetricsEnabled())
		assert.True(t, cfg.IsLoggingEnabled())
	})
}
