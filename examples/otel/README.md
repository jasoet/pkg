# OTel Package Examples

This directory contains examples demonstrating how to use the `otel` package for OpenTelemetry instrumentation configuration in Go applications.

## 📍 Example Code Location

**Full example implementation:** [/otel/examples/example.go](https://github.com/jasoet/pkg/blob/main/otel/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/v2/otel"

// Create configuration with all pillars
cfg := &otel.Config{
    TracerProvider: tracerProvider,  // For distributed tracing
    MeterProvider:  meterProvider,   // For metrics
    LoggerProvider: loggerProvider,  // For structured logging
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
}

// Check what's enabled
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("scope-name")
}

// Use LogHelper for OTel-aware logging
logger := otel.NewLogHelper(ctx, cfg, "github.com/myorg/myapp", "myFunc")
logger.Info("Processing request", otel.F("request_id", reqID))
logger.Error(err, "Failed to process", otel.F("user_id", userID))

// No-op mode (zero overhead when nil)
cfg := &otel.Config{ServiceName: "my-service"}  // All providers nil = no-op
```

**Key Features:**
- Centralized configuration for traces, metrics, and logs
- No-op implementations when providers are nil (zero overhead)
- LogHelper for automatic log-trace correlation
- Selective enabling of telemetry pillars

## Overview

The `otel` package provides:
- **Centralized Configuration**: Single config struct for all OTel providers
- **Telemetry Pillars**: Independent control of traces, metrics, and logs
- **LogHelper**: OTel-aware logging with automatic trace correlation
- **No-op Support**: Zero overhead when telemetry is disabled

## Running the Examples

To run the examples, use the following command from the repository root:

```bash
go run ./examples/otel
```

Or from the `otel/examples` directory:

```bash
go run example.go
```

This will demonstrate:
1. Basic OTel configuration setup
2. No-op configuration (telemetry disabled)
3. LogHelper usage patterns
4. Selective telemetry pillar enabling
5. Configuration validation

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/otel/examples/example.go) file demonstrates several configuration patterns:

### 1. Basic Configuration

Shows how to create a complete OTel configuration:

```go
cfg := &otel.Config{
    TracerProvider: tracerProvider,
    MeterProvider:  meterProvider,
    LoggerProvider: loggerProvider,
    ServiceName:    "example-service",
    ServiceVersion: "1.0.0",
}

// Check what's enabled
fmt.Println(cfg.IsTracingEnabled())  // true
fmt.Println(cfg.IsMetricsEnabled())  // true
fmt.Println(cfg.IsLoggingEnabled())  // true
```

### 2. No-op Configuration

Demonstrates zero-overhead telemetry when disabled:

```go
// All providers nil = no-op mode
cfg := &otel.Config{
    ServiceName:    "example-service",
    ServiceVersion: "1.0.0",
}

fmt.Println(cfg.IsTracingEnabled())  // false
fmt.Println(cfg.IsMetricsEnabled())  // false
fmt.Println(cfg.IsLoggingEnabled())  // false

// No overhead - safe to use in production
```

### 3. LogHelper Usage

Shows OTel-aware logging patterns:

```go
// Without OTel (falls back to zerolog)
logger1 := otel.NewLogHelper(ctx, nil, "", "example.doWork")
logger1.Info("Starting work", otel.F("worker_id", 123))

// With OTel (automatic trace_id/span_id injection)
logger2 := otel.NewLogHelper(ctx, cfg, "github.com/myorg/myapp", "example.processData")
logger2.Info("Data processed",
    otel.F("records_processed", 1000),
    otel.F("duration_ms", 250))

// Error logging
logger2.Error(err, "Processing failed",
    otel.F("error_code", "E001"),
    otel.F("retry_count", 3))
```

### 4. Telemetry Pillars

Demonstrates selective enabling of telemetry:

```go
// Scenario 1: Traces only
cfg1 := &otel.Config{
    TracerProvider: tracerProvider,
    ServiceName:    "tracing-only-service",
}

// Scenario 2: Metrics only
cfg2 := &otel.Config{
    MeterProvider: meterProvider,
    ServiceName:   "metrics-only-service",
}

// Scenario 3: All pillars
cfg3 := &otel.Config{
    TracerProvider: tracerProvider,
    MeterProvider:  meterProvider,
    LoggerProvider: loggerProvider,
    ServiceName:    "full-telemetry-service",
}
```

### 5. Configuration Validation

Shows safe configuration checking:

```go
// Check before use
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("my-scope")
    // Use tracer...
}

// Safe to call even if not enabled (returns no-op)
meter := cfg.GetMeter("my-scope")
```

## Telemetry Pillars

The otel package supports three independent telemetry pillars:

### 1. Traces (Distributed Tracing)

```go
cfg := &otel.Config{
    TracerProvider: tracerProvider,
    ServiceName:    "my-service",
}

tracer := cfg.GetTracer("my-scope")
// Use for distributed tracing...
```

### 2. Metrics (Measurements)

```go
cfg := &otel.Config{
    MeterProvider: meterProvider,
    ServiceName:   "my-service",
}

meter := cfg.GetMeter("my-scope")
// Use for metrics collection...
```

### 3. Logs (Structured Logging)

```go
cfg := &otel.Config{
    LoggerProvider: loggerProvider,
    ServiceName:    "my-service",
}

logger := cfg.GetLogger("my-scope")
// Use for structured logging with trace correlation...
```

## LogHelper Features

The LogHelper provides:
- **Automatic Trace Correlation**: Logs include trace_id and span_id
- **Fallback to Zerolog**: Works without OTel configuration
- **Structured Logging**: Type-safe field addition with `F()` function
- **Multiple Log Levels**: Debug, Info, Warn, Error

### Basic Usage

```go
logger := otel.NewLogHelper(ctx, cfg, "github.com/myorg/myapp", "functionName")

// Info logging
logger.Info("User logged in",
    otel.F("user_id", 123),
    otel.F("email", "user@example.com"))

// Debug logging
logger.Debug("Cache lookup",
    otel.F("cache_key", "user:123"),
    otel.F("hit", true))

// Warning logging
logger.Warn("Rate limit approaching",
    otel.F("current_rate", 95),
    otel.F("max_rate", 100))

// Error logging
logger.Error(err, "Failed to process payment",
    otel.F("payment_id", "PAY-123"),
    otel.F("amount", 99.99))
```

## Integration Patterns

### With Server Package

```go
serverConfig := &server.Config{
    OtelConfig: &otel.Config{
        TracerProvider: tracerProvider,
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
    },
}
```

### With GRPC Package

```go
grpcOptions := []grpc.ServerOption{
    grpc.OtelConfig(&otel.Config{
        TracerProvider: tracerProvider,
        ServiceName:    "my-grpc-service",
    }),
}
```

## Best Practices

### 1. Centralize OTel Configuration

```go
// Create once, reuse everywhere
otelCfg := &otel.Config{
    TracerProvider: tracerProvider,
    MeterProvider:  meterProvider,
    LoggerProvider: loggerProvider,
    ServiceName:    "my-service",
    ServiceVersion: version,
}

// Pass to all components
serverCfg.OtelConfig = otelCfg
grpcCfg.OtelConfig = otelCfg
```

### 2. Always Check Before Using

```go
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("my-scope")
    // Safe to use tracer
}
```

### 3. Use LogHelper Consistently

```go
// Create logger for each function/scope
logger := otel.NewLogHelper(ctx, cfg, scopeName, functionName)

// Use throughout function
logger.Debug("Starting...")
logger.Info("Processing...")
logger.Error(err, "Failed...")
```

### 4. Structured Logging with F()

```go
// Type-safe field addition
logger.Info("Event occurred",
    otel.F("event_id", 123),
    otel.F("user_id", userID),
    otel.F("timestamp", time.Now()))
```

### 5. Enable Selectively

```go
// Development: All pillars
devCfg := &otel.Config{
    TracerProvider: tp,
    MeterProvider:  mp,
    LoggerProvider: lp,
}

// Production: Traces and metrics only
prodCfg := &otel.Config{
    TracerProvider: tp,
    MeterProvider:  mp,
}
```

## Configuration Options

### Required Fields

- `ServiceName` - Name of your service

### Optional Fields

- `ServiceVersion` - Version of your service
- `TracerProvider` - For distributed tracing (nil = no-op)
- `MeterProvider` - For metrics (nil = no-op)
- `LoggerProvider` - For logging (nil = no-op)

## No-op Behavior

When providers are nil:
- `IsTracingEnabled()` returns `false`
- `IsMetricsEnabled()` returns `false`
- `IsLoggingEnabled()` returns `false`
- `GetTracer()` returns no-op tracer
- `GetMeter()` returns no-op meter
- `GetLogger()` returns no-op logger
- **Zero overhead** - no performance impact

This allows easy toggling of telemetry without code changes.

## Differences from fullstack-otel

The `examples/fullstack-otel` directory contains a complete application with:
- Full OpenTelemetry setup
- Jaeger/Prometheus integration
- Multi-service architecture
- Production-ready configuration

The `examples/otel` directory focuses on:
- Basic configuration patterns
- Simple usage examples
- LogHelper demonstration
- Configuration validation

For production setup guidance, see `examples/fullstack-otel`.

## Further Reading

- [OTel Package Documentation](https://github.com/jasoet/pkg/tree/main/otel)
- [Fullstack OTel Example](https://github.com/jasoet/pkg/tree/main/examples/fullstack-otel)
- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Main pkg Repository](https://github.com/jasoet/pkg)
