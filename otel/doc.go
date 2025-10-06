// Package otel provides OpenTelemetry instrumentation utilities for github.com/jasoet/pkg/v2.
//
// This package offers:
//   - Centralized configuration for traces, metrics, and logs
//   - Library-specific semantic conventions
//   - No-op implementations when telemetry is disabled
//
// # Configuration
//
// Create an otel.Config with the desired providers:
//
//	cfg := &otel.Config{
//	    TracerProvider: tracerProvider,  // optional
//	    MeterProvider:  meterProvider,   // optional
//	    LoggerProvider: loggerProvider,  // optional
//	    ServiceName:    "my-service",
//	    ServiceVersion: "1.0.0",
//	}
//
// Then pass this config to package configurations (server.Config, grpc options, etc.).
//
// # Telemetry Pillars
//
// Enable any combination of:
//   - Traces (distributed tracing)
//   - Metrics (measurements and aggregations)
//   - Logs (structured log export via OpenTelemetry standard)
//
// Each pillar is independently controlled by setting its provider.
// Nil providers result in no-op implementations with zero overhead.
//
// # Standard Logging Helper
//
// This package provides otel.LogHelper for OTel-aware logging that automatically
// correlates logs with traces. It uses OTel LoggerProvider when available,
// otherwise falls back to zerolog. See helper.go for details.
package otel
