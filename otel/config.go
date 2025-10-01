package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopm "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration for instrumentation.
// TracerProvider and MeterProvider are optional - nil values result in no-op implementations.
// LoggerProvider defaults to stdout exporter when using NewConfig().
type Config struct {
	// TracerProvider for distributed tracing
	// If nil, tracing will be disabled (no-op tracer)
	TracerProvider trace.TracerProvider

	// MeterProvider for metrics collection
	// If nil, metrics will be disabled (no-op meter)
	MeterProvider metric.MeterProvider

	// LoggerProvider for structured logging via OTel
	// Defaults to stdout exporter when using NewConfig()
	// Set to nil explicitly to disable logging
	LoggerProvider log.LoggerProvider

	// ServiceName identifies the service in telemetry data
	ServiceName string

	// ServiceVersion identifies the service version
	ServiceVersion string
}

// NewConfig creates a new OpenTelemetry configuration with default LoggerProvider.
// The default LoggerProvider uses stdout exporter for easy debugging.
// Use With* methods to add TracerProvider and MeterProvider.
//
// For production use with better formatting and automatic log-span correlation,
// use logging.NewLoggerProvider() instead:
//
//	import "github.com/jasoet/pkg/logging"
//	cfg := &otel.Config{
//	    ServiceName:    "my-service",
//	    LoggerProvider: logging.NewLoggerProvider("my-service", false),
//	}
//	cfg.WithTracerProvider(tp).WithMeterProvider(mp)
//
// Example with default stdout logger:
//
//	cfg := otel.NewConfig("my-service").
//	    WithTracerProvider(tp).
//	    WithMeterProvider(mp)
func NewConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		LoggerProvider: defaultLoggerProvider(),
	}
}

// WithTracerProvider sets the TracerProvider for distributed tracing
func (c *Config) WithTracerProvider(tp trace.TracerProvider) *Config {
	c.TracerProvider = tp
	return c
}

// WithMeterProvider sets the MeterProvider for metrics collection
func (c *Config) WithMeterProvider(mp metric.MeterProvider) *Config {
	c.MeterProvider = mp
	return c
}

// WithLoggerProvider sets a custom LoggerProvider, replacing the default stdout logger
func (c *Config) WithLoggerProvider(lp log.LoggerProvider) *Config {
	c.LoggerProvider = lp
	return c
}

// WithServiceVersion sets the service version for telemetry data
func (c *Config) WithServiceVersion(version string) *Config {
	c.ServiceVersion = version
	return c
}

// WithoutLogging disables the default logging by setting LoggerProvider to nil
func (c *Config) WithoutLogging() *Config {
	c.LoggerProvider = nil
	return c
}

// defaultLoggerProvider creates a simple stdout-based LoggerProvider
// This is used as the default to ensure logging works out of the box
func defaultLoggerProvider() log.LoggerProvider {
	exporter, err := stdoutlog.New()
	if err != nil {
		// If stdout exporter fails, return no-op
		return noop.NewLoggerProvider()
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	return provider
}

// Shutdown gracefully shuts down all configured providers
// Call this when your application exits to flush any pending telemetry
func (c *Config) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}

	// Shutdown logger provider if it supports it
	if lp, ok := c.LoggerProvider.(*sdklog.LoggerProvider); ok {
		if err := lp.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Note: TracerProvider and MeterProvider shutdown
	// should be handled by the user who created them

	return nil
}

// IsTracingEnabled returns true if tracing is configured
func (c *Config) IsTracingEnabled() bool {
	return c != nil && c.TracerProvider != nil
}

// IsMetricsEnabled returns true if metrics collection is configured
func (c *Config) IsMetricsEnabled() bool {
	return c != nil && c.MeterProvider != nil
}

// IsLoggingEnabled returns true if OTel logging is configured
func (c *Config) IsLoggingEnabled() bool {
	return c != nil && c.LoggerProvider != nil
}

// GetTracer returns a tracer for the given instrumentation scope.
// Returns a no-op tracer if tracing is not configured.
func (c *Config) GetTracer(scopeName string, opts ...trace.TracerOption) trace.Tracer {
	if !c.IsTracingEnabled() {
		return trace.NewNoopTracerProvider().Tracer(scopeName, opts...)
	}
	return c.TracerProvider.Tracer(scopeName, opts...)
}

// GetMeter returns a meter for the given instrumentation scope.
// Returns a no-op meter if metrics are not configured.
func (c *Config) GetMeter(scopeName string, opts ...metric.MeterOption) metric.Meter {
	if !c.IsMetricsEnabled() {
		return noopm.NewMeterProvider().Meter(scopeName, opts...)
	}
	return c.MeterProvider.Meter(scopeName, opts...)
}

// GetLogger returns a logger for the given instrumentation scope.
// Returns a no-op logger if logging is not configured.
func (c *Config) GetLogger(scopeName string, opts ...log.LoggerOption) log.Logger {
	if !c.IsLoggingEnabled() {
		return noop.NewLoggerProvider().Logger(scopeName, opts...)
	}
	return c.LoggerProvider.Logger(scopeName, opts...)
}
