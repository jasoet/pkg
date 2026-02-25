package otel

import (
	"context"
	"testing"

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

		if cfg.ServiceName != "test-service" {
			t.Errorf("expected ServiceName to be 'test-service', got '%s'", cfg.ServiceName)
		}
	})

	t.Run("has default logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")

		if cfg.LoggerProvider == nil {
			t.Error("expected LoggerProvider to be set by default")
		}
	})

	t.Run("has nil tracer provider by default", func(t *testing.T) {
		cfg := NewConfig("test-service")

		if cfg.TracerProvider != nil {
			t.Error("expected TracerProvider to be nil by default")
		}
	})

	t.Run("has nil meter provider by default", func(t *testing.T) {
		cfg := NewConfig("test-service")

		if cfg.MeterProvider != nil {
			t.Error("expected MeterProvider to be nil by default")
		}
	})

	t.Run("has empty service version by default", func(t *testing.T) {
		cfg := NewConfig("test-service")

		if cfg.ServiceVersion != "" {
			t.Errorf("expected ServiceVersion to be empty, got '%s'", cfg.ServiceVersion)
		}
	})
}

func TestWithTracerProvider(t *testing.T) {
	t.Run("sets tracer provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		tp := noopt.NewTracerProvider()

		cfg.WithTracerProvider(tp)

		if cfg.TracerProvider != tp {
			t.Error("expected TracerProvider to be set")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service")
		tp := noopt.NewTracerProvider()

		result := cfg.WithTracerProvider(tp)

		if result != cfg {
			t.Error("expected WithTracerProvider to return same config instance")
		}
	})

	t.Run("allows method chaining", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		mp := noopm.NewMeterProvider()

		cfg := NewConfig("test-service").
			WithTracerProvider(tp).
			WithMeterProvider(mp)

		if cfg.TracerProvider != tp {
			t.Error("expected TracerProvider to be set")
		}
		if cfg.MeterProvider != mp {
			t.Error("expected MeterProvider to be set")
		}
	})
}

func TestWithMeterProvider(t *testing.T) {
	t.Run("sets meter provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		mp := noopm.NewMeterProvider()

		cfg.WithMeterProvider(mp)

		if cfg.MeterProvider != mp {
			t.Error("expected MeterProvider to be set")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service")
		mp := noopm.NewMeterProvider()

		result := cfg.WithMeterProvider(mp)

		if result != cfg {
			t.Error("expected WithMeterProvider to return same config instance")
		}
	})
}

func TestWithLoggerProvider(t *testing.T) {
	t.Run("sets custom logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		lp := noopl.NewLoggerProvider()

		cfg.WithLoggerProvider(lp)

		if cfg.LoggerProvider != lp {
			t.Error("expected LoggerProvider to be set to custom provider")
		}
	})

	t.Run("replaces default logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")
		defaultLogger := cfg.LoggerProvider

		customLogger := noopl.NewLoggerProvider()
		cfg.WithLoggerProvider(customLogger)

		if cfg.LoggerProvider == defaultLogger {
			t.Error("expected LoggerProvider to be replaced")
		}
		if cfg.LoggerProvider != customLogger {
			t.Error("expected LoggerProvider to be custom provider")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service")
		lp := noopl.NewLoggerProvider()

		result := cfg.WithLoggerProvider(lp)

		if result != cfg {
			t.Error("expected WithLoggerProvider to return same config instance")
		}
	})
}

func TestWithServiceVersion(t *testing.T) {
	t.Run("sets service version", func(t *testing.T) {
		cfg := NewConfig("test-service")

		cfg.WithServiceVersion("v1.2.3")

		if cfg.ServiceVersion != "v1.2.3" {
			t.Errorf("expected ServiceVersion to be 'v1.2.3', got '%s'", cfg.ServiceVersion)
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service")

		result := cfg.WithServiceVersion("v1.0.0")

		if result != cfg {
			t.Error("expected WithServiceVersion to return same config instance")
		}
	})

	t.Run("allows method chaining with other methods", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service").
			WithServiceVersion("v2.0.0").
			WithTracerProvider(tp)

		if cfg.ServiceVersion != "v2.0.0" {
			t.Errorf("expected ServiceVersion to be 'v2.0.0', got '%s'", cfg.ServiceVersion)
		}
		if cfg.TracerProvider != tp {
			t.Error("expected TracerProvider to be set")
		}
	})
}

func TestWithoutLogging(t *testing.T) {
	t.Run("disables logging by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service")

		// Verify default logger is set
		if cfg.LoggerProvider == nil {
			t.Error("expected default LoggerProvider to be set")
		}

		cfg.WithoutLogging()

		if cfg.LoggerProvider != nil {
			t.Error("expected LoggerProvider to be nil after WithoutLogging")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service")

		result := cfg.WithoutLogging()

		if result != cfg {
			t.Error("expected WithoutLogging to return same config instance")
		}
	})

	t.Run("allows method chaining", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service").
			WithoutLogging().
			WithTracerProvider(tp)

		if cfg.LoggerProvider != nil {
			t.Error("expected LoggerProvider to be nil")
		}
		if cfg.TracerProvider != tp {
			t.Error("expected TracerProvider to be set")
		}
	})
}

func TestDisableTracing(t *testing.T) {
	t.Run("disables tracing by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())

		// Verify tracer is set
		if cfg.TracerProvider == nil {
			t.Error("expected TracerProvider to be set")
		}

		cfg.DisableTracing()

		if cfg.TracerProvider != nil {
			t.Error("expected TracerProvider to be nil after DisableTracing")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())

		result := cfg.DisableTracing()

		if result != cfg {
			t.Error("expected DisableTracing to return same config instance")
		}
	})

	t.Run("allows method chaining", func(t *testing.T) {
		mp := noopm.NewMeterProvider()

		cfg := NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider()).
			DisableTracing().
			WithMeterProvider(mp)

		if cfg.TracerProvider != nil {
			t.Error("expected TracerProvider to be nil")
		}
		if cfg.MeterProvider != mp {
			t.Error("expected MeterProvider to be set")
		}
	})

	t.Run("works when tracer provider is already nil", func(t *testing.T) {
		cfg := NewConfig("test-service")

		cfg.DisableTracing()

		if cfg.TracerProvider != nil {
			t.Error("expected TracerProvider to remain nil")
		}
	})
}

func TestDisableMetrics(t *testing.T) {
	t.Run("disables metrics by setting provider to nil", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		// Verify meter is set
		if cfg.MeterProvider == nil {
			t.Error("expected MeterProvider to be set")
		}

		cfg.DisableMetrics()

		if cfg.MeterProvider != nil {
			t.Error("expected MeterProvider to be nil after DisableMetrics")
		}
	})

	t.Run("returns config for method chaining", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		result := cfg.DisableMetrics()

		if result != cfg {
			t.Error("expected DisableMetrics to return same config instance")
		}
	})

	t.Run("allows method chaining", func(t *testing.T) {
		tp := noopt.NewTracerProvider()

		cfg := NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider()).
			DisableMetrics().
			WithTracerProvider(tp)

		if cfg.MeterProvider != nil {
			t.Error("expected MeterProvider to be nil")
		}
		if cfg.TracerProvider != tp {
			t.Error("expected TracerProvider to be set")
		}
	})

	t.Run("works when meter provider is already nil", func(t *testing.T) {
		cfg := NewConfig("test-service")

		cfg.DisableMetrics()

		if cfg.MeterProvider != nil {
			t.Error("expected MeterProvider to remain nil")
		}
	})
}


func TestIsTracingEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		if cfg.IsTracingEnabled() {
			t.Error("expected IsTracingEnabled to return false for nil config")
		}
	})

	t.Run("returns false when tracer provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service")
		if cfg.IsTracingEnabled() {
			t.Error("expected IsTracingEnabled to return false when TracerProvider is nil")
		}
	})

	t.Run("returns true when tracer provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())

		if !cfg.IsTracingEnabled() {
			t.Error("expected IsTracingEnabled to return true when TracerProvider is set")
		}
	})
}

func TestIsMetricsEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		if cfg.IsMetricsEnabled() {
			t.Error("expected IsMetricsEnabled to return false for nil config")
		}
	})

	t.Run("returns false when meter provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service")
		if cfg.IsMetricsEnabled() {
			t.Error("expected IsMetricsEnabled to return false when MeterProvider is nil")
		}
	})

	t.Run("returns true when meter provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithMeterProvider(noopm.NewMeterProvider())

		if !cfg.IsMetricsEnabled() {
			t.Error("expected IsMetricsEnabled to return true when MeterProvider is set")
		}
	})
}

func TestIsLoggingEnabled(t *testing.T) {
	t.Run("returns false when config is nil", func(t *testing.T) {
		var cfg *Config
		if cfg.IsLoggingEnabled() {
			t.Error("expected IsLoggingEnabled to return false for nil config")
		}
	})

	t.Run("returns false when logger provider is nil", func(t *testing.T) {
		cfg := NewConfig("test-service").WithoutLogging()
		if cfg.IsLoggingEnabled() {
			t.Error("expected IsLoggingEnabled to return false when LoggerProvider is nil")
		}
	})

	t.Run("returns true when logger provider is set", func(t *testing.T) {
		cfg := NewConfig("test-service")
		if !cfg.IsLoggingEnabled() {
			t.Error("expected IsLoggingEnabled to return true when LoggerProvider is set")
		}
	})

	t.Run("returns true with custom logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithLoggerProvider(noopl.NewLoggerProvider())

		if !cfg.IsLoggingEnabled() {
			t.Error("expected IsLoggingEnabled to return true with custom LoggerProvider")
		}
	})
}

func TestGetTracer(t *testing.T) {
	t.Run("returns no-op tracer when tracing is disabled", func(t *testing.T) {
		cfg := NewConfig("test-service")

		tracer := cfg.GetTracer("test-scope")

		if tracer == nil {
			t.Error("expected GetTracer to return a tracer")
		}

		// Verify it's a no-op tracer by checking it doesn't panic
		_, span := tracer.Start(context.Background(), "test-operation")
		span.End()
	})

	t.Run("returns tracer from provider when tracing is enabled", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		cfg := NewConfig("test-service").WithTracerProvider(tp)

		tracer := cfg.GetTracer("test-scope")

		if tracer == nil {
			t.Error("expected GetTracer to return a tracer")
		}
	})

	t.Run("accepts tracer options", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		cfg := NewConfig("test-service").WithTracerProvider(tp)

		tracer := cfg.GetTracer("test-scope", trace.WithInstrumentationVersion("v1.0.0"))

		if tracer == nil {
			t.Error("expected GetTracer to return a tracer with options")
		}
	})
}

func TestGetMeter(t *testing.T) {
	t.Run("returns no-op meter when metrics are disabled", func(t *testing.T) {
		cfg := NewConfig("test-service")

		meter := cfg.GetMeter("test-scope")

		if meter == nil {
			t.Error("expected GetMeter to return a meter")
		}

		// Verify it's a no-op meter by checking it doesn't panic
		_, err := meter.Int64Counter("test-counter")
		if err != nil {
			t.Errorf("expected no-op meter to not error, got: %v", err)
		}
	})

	t.Run("returns meter from provider when metrics are enabled", func(t *testing.T) {
		mp := noopm.NewMeterProvider()
		cfg := NewConfig("test-service").WithMeterProvider(mp)

		meter := cfg.GetMeter("test-scope")

		if meter == nil {
			t.Error("expected GetMeter to return a meter")
		}
	})

	t.Run("accepts meter options", func(t *testing.T) {
		mp := noopm.NewMeterProvider()
		cfg := NewConfig("test-service").WithMeterProvider(mp)

		meter := cfg.GetMeter("test-scope", metric.WithInstrumentationVersion("v1.0.0"))

		if meter == nil {
			t.Error("expected GetMeter to return a meter with options")
		}
	})
}

func TestGetLogger(t *testing.T) {
	t.Run("returns no-op logger when logging is disabled", func(t *testing.T) {
		cfg := NewConfig("test-service").WithoutLogging()

		logger := cfg.GetLogger("test-scope")

		if logger == nil {
			t.Error("expected GetLogger to return a logger")
		}

		// Verify it's a no-op logger by checking it doesn't panic
		logger.Emit(context.Background(), log.Record{})
	})

	t.Run("returns logger from provider when logging is enabled", func(t *testing.T) {
		cfg := NewConfig("test-service")

		logger := cfg.GetLogger("test-scope")

		if logger == nil {
			t.Error("expected GetLogger to return a logger")
		}
	})

	t.Run("accepts logger options", func(t *testing.T) {
		lp := noopl.NewLoggerProvider()
		cfg := NewConfig("test-service").WithLoggerProvider(lp)

		logger := cfg.GetLogger("test-scope", log.WithInstrumentationVersion("v1.0.0"))

		if logger == nil {
			t.Error("expected GetLogger to return a logger with options")
		}
	})
}

func TestShutdown(t *testing.T) {
	t.Run("returns nil when config is nil", func(t *testing.T) {
		var cfg *Config
		err := cfg.Shutdown(context.Background())
		if err != nil {
			t.Errorf("expected no error for nil config, got: %v", err)
		}
	})

	t.Run("succeeds with default logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service")

		err := cfg.Shutdown(context.Background())
		if err != nil {
			t.Errorf("expected Shutdown to succeed, got error: %v", err)
		}
	})

	t.Run("succeeds with no-op logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service").
			WithLoggerProvider(noopl.NewLoggerProvider())

		err := cfg.Shutdown(context.Background())
		if err != nil {
			t.Errorf("expected Shutdown to succeed with no-op logger, got error: %v", err)
		}
	})

	t.Run("succeeds without logger provider", func(t *testing.T) {
		cfg := NewConfig("test-service").WithoutLogging()

		err := cfg.Shutdown(context.Background())
		if err != nil {
			t.Errorf("expected Shutdown to succeed without logger, got error: %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		cfg := NewConfig("test-service")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should still succeed or return context error
		_ = cfg.Shutdown(ctx)
	})
}

func TestDefaultLoggerProvider(t *testing.T) {
	t.Run("creates a logger provider", func(t *testing.T) {
		lp := defaultLoggerProvider("test-service", false)

		if lp == nil {
			t.Error("expected defaultLoggerProvider to return a provider")
		}
	})

	t.Run("created logger can emit logs", func(t *testing.T) {
		lp := defaultLoggerProvider("test-service", false)
		logger := lp.Logger("test-scope")

		// Should not panic
		logger.Emit(context.Background(), log.Record{})
	})
}

func TestFullConfigChaining(t *testing.T) {
	t.Run("supports full method chaining", func(t *testing.T) {
		tp := noopt.NewTracerProvider()
		mp := noopm.NewMeterProvider()
		lp := noopl.NewLoggerProvider()

		cfg := NewConfig("my-service").
			WithServiceVersion("v2.0.0").
			WithTracerProvider(tp).
			WithMeterProvider(mp).
			WithLoggerProvider(lp)

		if cfg.ServiceName != "my-service" {
			t.Error("ServiceName not set correctly")
		}
		if cfg.ServiceVersion != "v2.0.0" {
			t.Error("ServiceVersion not set correctly")
		}
		if cfg.TracerProvider != tp {
			t.Error("TracerProvider not set correctly")
		}
		if cfg.MeterProvider != mp {
			t.Error("MeterProvider not set correctly")
		}
		if cfg.LoggerProvider != lp {
			t.Error("LoggerProvider not set correctly")
		}

		if !cfg.IsTracingEnabled() {
			t.Error("Tracing should be enabled")
		}
		if !cfg.IsMetricsEnabled() {
			t.Error("Metrics should be enabled")
		}
		if !cfg.IsLoggingEnabled() {
			t.Error("Logging should be enabled")
		}
	})
}
