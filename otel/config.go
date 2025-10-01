package otel

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopm "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration for instrumentation.
// All providers are optional - nil values result in no-op implementations.
type Config struct {
	// TracerProvider for distributed tracing
	// If nil, tracing will be disabled (no-op tracer)
	TracerProvider trace.TracerProvider

	// MeterProvider for metrics collection
	// If nil, metrics will be disabled (no-op meter)
	MeterProvider metric.MeterProvider

	// LoggerProvider for structured logging via OTel
	// If nil, OTel logging will be disabled (no-op logger)
	LoggerProvider log.LoggerProvider

	// ServiceName identifies the service in telemetry data
	ServiceName string

	// ServiceVersion identifies the service version
	ServiceVersion string
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
