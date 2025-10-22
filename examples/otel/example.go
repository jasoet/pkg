// Package main demonstrates comprehensive usage of the otel package.
//
// This example shows:
//   - Basic OTel configuration setup
//   - Using LogHelper for OTel-aware logging
//   - No-op vs active provider configurations
//   - Integration patterns
//
// Run with: go run ./examples/otel
package main

import (
	"context"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	fmt.Println("=== OTel Package Examples ===")

	// Example 1: Basic Configuration
	basicConfiguration()

	// Example 2: No-op Configuration
	noopConfiguration()

	// Example 3: LogHelper Usage
	logHelperUsage()

	// Example 4: Telemetry Pillars
	telemetryPillars()

	// Example 5: Configuration Validation
	configurationValidation()
}

// Example 1: Basic OTel configuration
func basicConfiguration() {
	fmt.Println("--- Example 1: Basic Configuration ---")

	// Create OTel providers (in real app, these would be properly initialized)
	tracerProvider := trace.NewTracerProvider()
	meterProvider := metric.NewMeterProvider()
	loggerProvider := log.NewLoggerProvider()

	// Create OTel configuration
	cfg := &otel.Config{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		LoggerProvider: loggerProvider,
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
	}

	fmt.Printf("Service Name:    %s\n", cfg.ServiceName)
	fmt.Printf("Service Version: %s\n", cfg.ServiceVersion)
	fmt.Printf("Tracing:         %v\n", cfg.IsTracingEnabled())
	fmt.Printf("Metrics:         %v\n", cfg.IsMetricsEnabled())
	fmt.Printf("Logging:         %v\n\n", cfg.IsLoggingEnabled())

	// Cleanup
	_ = tracerProvider.Shutdown(context.Background())
	_ = meterProvider.Shutdown(context.Background())
	_ = loggerProvider.Shutdown(context.Background())
}

// Example 2: No-op configuration (telemetry disabled)
func noopConfiguration() {
	fmt.Println("--- Example 2: No-op Configuration (Telemetry Disabled) ---")

	// Configuration with nil providers - no overhead
	cfg := &otel.Config{
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
	}

	fmt.Printf("Service Name:    %s\n", cfg.ServiceName)
	fmt.Printf("Service Version: %s\n", cfg.ServiceVersion)
	fmt.Printf("Tracing:         %v (no-op)\n", cfg.IsTracingEnabled())
	fmt.Printf("Metrics:         %v (no-op)\n", cfg.IsMetricsEnabled())
	fmt.Printf("Logging:         %v (no-op)\n", cfg.IsLoggingEnabled())
	fmt.Println()
	fmt.Println("When providers are nil, the package uses no-op implementations")
	fmt.Println("with zero overhead. This allows easy toggling of telemetry.")
	fmt.Println()
}

// Example 3: LogHelper usage patterns
func logHelperUsage() {
	fmt.Println("--- Example 3: LogHelper Usage ---")

	ctx := context.Background()

	// Scenario 1: Without OTel (falls back to zerolog)
	fmt.Println("\n1. Without OTel (zerolog fallback):")
	logger1 := otel.NewLogHelper(ctx, nil, "", "example.doWork")
	logger1.Info("Starting work", otel.F("worker_id", 123))
	logger1.Debug("Processing item", otel.F("item_id", 456))

	// Scenario 2: With OTel configured
	fmt.Println("\n2. With OTel configured:")
	loggerProvider := log.NewLoggerProvider()
	cfg := &otel.Config{
		LoggerProvider: loggerProvider,
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
	}

	logger2 := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/example", "example.processData")
	logger2.Info("Data processed successfully",
		otel.F("records_processed", 1000),
		otel.F("duration_ms", 250))
	logger2.Debug("Cache hit",
		otel.F("cache_key", "user:123"),
		otel.F("hit_rate", 0.95))

	// Cleanup
	_ = loggerProvider.Shutdown(context.Background())

	fmt.Println("\nNote: With OTel, logs automatically include trace_id and span_id")
	fmt.Println("for correlation with distributed traces.")
	fmt.Println()
}

// Example 4: Telemetry pillars (traces, metrics, logs)
func telemetryPillars() {
	fmt.Println("--- Example 4: Telemetry Pillars ---")

	// You can enable any combination of telemetry pillars
	fmt.Println("Telemetry can be selectively enabled:")
	fmt.Println()

	// Scenario 1: Traces only
	fmt.Println("1. Traces Only:")
	tracerProvider := trace.NewTracerProvider()
	cfg1 := &otel.Config{
		TracerProvider: tracerProvider,
		ServiceName:    "tracing-service",
	}
	fmt.Printf("   Tracing: %v, Metrics: %v, Logging: %v\n",
		cfg1.IsTracingEnabled(),
		cfg1.IsMetricsEnabled(),
		cfg1.IsLoggingEnabled())
	_ = tracerProvider.Shutdown(context.Background())

	// Scenario 2: Metrics only
	fmt.Println("\n2. Metrics Only:")
	meterProvider := metric.NewMeterProvider()
	cfg2 := &otel.Config{
		MeterProvider: meterProvider,
		ServiceName:   "metrics-service",
	}
	fmt.Printf("   Tracing: %v, Metrics: %v, Logging: %v\n",
		cfg2.IsTracingEnabled(),
		cfg2.IsMetricsEnabled(),
		cfg2.IsLoggingEnabled())
	_ = meterProvider.Shutdown(context.Background())

	// Scenario 3: All pillars enabled
	fmt.Println("\n3. All Pillars Enabled:")
	tracerProvider3 := trace.NewTracerProvider()
	meterProvider3 := metric.NewMeterProvider()
	loggerProvider3 := log.NewLoggerProvider()
	cfg3 := &otel.Config{
		TracerProvider: tracerProvider3,
		MeterProvider:  meterProvider3,
		LoggerProvider: loggerProvider3,
		ServiceName:    "full-telemetry-service",
	}
	fmt.Printf("   Tracing: %v, Metrics: %v, Logging: %v\n",
		cfg3.IsTracingEnabled(),
		cfg3.IsMetricsEnabled(),
		cfg3.IsLoggingEnabled())
	_ = tracerProvider3.Shutdown(context.Background())
	_ = meterProvider3.Shutdown(context.Background())
	_ = loggerProvider3.Shutdown(context.Background())

	fmt.Println("\nEach pillar is independent - enable what you need!")
	fmt.Println()
}

// Example 5: Configuration validation
func configurationValidation() {
	fmt.Println("--- Example 5: Configuration Validation ---")

	// Check what's enabled
	tracerProvider := trace.NewTracerProvider()
	cfg := &otel.Config{
		TracerProvider: tracerProvider,
		ServiceName:    "validation-example",
		ServiceVersion: "2.0.0",
	}

	fmt.Println("Checking configuration state:")
	fmt.Printf("  Service Name:    %s\n", cfg.ServiceName)
	fmt.Printf("  Service Version: %s\n", cfg.ServiceVersion)
	fmt.Printf("  Tracing:         %v\n", cfg.IsTracingEnabled())
	fmt.Printf("  Metrics:         %v\n", cfg.IsMetricsEnabled())
	fmt.Printf("  Logging:         %v\n", cfg.IsLoggingEnabled())

	// Get tracer
	if cfg.IsTracingEnabled() {
		tracer := cfg.GetTracer("example-scope")
		fmt.Printf("\n  Tracer obtained: %T\n", tracer)
	}

	// Attempt to get meter (will return no-op since not configured)
	meter := cfg.GetMeter("example-scope")
	fmt.Printf("  Meter obtained:  %T (no-op since not configured)\n", meter)

	// Cleanup
	_ = tracerProvider.Shutdown(context.Background())

	fmt.Println("\nConfiguration methods allow safe checking of enabled features")
	fmt.Println("before attempting to use them.")
	fmt.Println()
}
