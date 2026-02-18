// Package otel provides OpenTelemetry instrumentation utilities for github.com/jasoet/pkg/v2.
//
// This package offers:
//   - Centralized configuration for traces, metrics, and logs
//   - Library-specific semantic conventions
//   - No-op implementations when telemetry is disabled
//   - Integrated span and logging with automatic correlation
//   - Layer-aware instrumentation (Handler, Operations, Service, Repository)
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
// # Unified Layer Instrumentation
//
// Use LayerContext for simplified span + logging with automatic correlation:
//
//	// Service layer example
//	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
//	    otel.F("user_id", userID))
//	defer lc.End()
//
//	lc.Logger.Info("Creating user", otel.F("email", email))
//	if err := repo.Save(lc.Context(), data); err != nil {
//	    return lc.Error(err, "save failed")
//	}
//	return lc.Success("User created")
//
// Available layers: StartHandler, StartOperations, StartService, StartRepository
//
// # Standard Logging Helper
//
// This package provides otel.LogHelper for OTel-aware logging that automatically
// correlates logs with traces. It uses OTel LoggerProvider when available,
// otherwise falls back to zerolog. Logs automatically include trace_id and span_id
// when a span is active. See helper.go for details.
package otel
