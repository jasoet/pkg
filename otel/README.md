# OpenTelemetry Integration

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/otel.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/otel)

Unified OpenTelemetry v2 configuration and instrumentation utilities for the `pkg` library ecosystem.

## Overview

The `otel` package provides centralized OpenTelemetry configuration that enables observability across all library packages. It supports the three pillars of observability:

- **Traces** - Distributed tracing for request flows
- **Metrics** - Performance and health measurements
- **Logs** - Structured logging via OpenTelemetry standard

## Features

- **Unified Configuration**: Single config object for all telemetry pillars
- **Selective Enablement**: Enable only the telemetry you need
- **No-op by Default**: Zero overhead when providers are not configured
- **Method Chaining**: Fluent API for configuration
- **Standard Logging Helper**: OTel-aware logging with automatic trace correlation
- **Graceful Shutdown**: Proper resource cleanup

## Installation

```bash
go get github.com/jasoet/pkg/v2/otel
```

## Quick Start

### Basic Configuration

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/v2/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Create tracer and meter providers (your setup)
    tracerProvider := trace.NewTracerProvider(/* ... */)
    meterProvider := metric.NewMeterProvider(/* ... */)

    // Create unified OTel config
    otelConfig := otel.NewConfig("my-service").
        WithTracerProvider(tracerProvider).
        WithMeterProvider(meterProvider).
        WithServiceVersion("1.0.0")

    // Use with library packages
    // server.Start(server.Config{OTelConfig: otelConfig, ...})
    // db.Pool(db.Config{OTelConfig: otelConfig, ...})

    // Cleanup on shutdown
    defer otelConfig.Shutdown(context.Background())
}
```

### Selective Telemetry

Enable only what you need:

```go
// Tracing only
cfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithoutLogging()  // Disable default logging

// Metrics only
cfg := otel.NewConfig("my-service").
    WithMeterProvider(meterProvider).
    WithoutLogging()

// All three pillars
cfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithLoggerProvider(loggerProvider)
```

### Custom Logger Provider

Use the `logging` package for better formatting and automatic trace correlation:

```go
import (
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/otel"
)

// Production-ready logger with trace correlation
loggerProvider := logging.NewLoggerProvider("my-service", false)

cfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithLoggerProvider(loggerProvider)
```

## Configuration API

### Config Struct

```go
type Config struct {
    TracerProvider trace.TracerProvider  // nil = no tracing
    MeterProvider  metric.MeterProvider  // nil = no metrics
    LoggerProvider log.LoggerProvider    // nil = no OTel logs
    ServiceName    string
    ServiceVersion string
}
```

### Builder Methods

| Method | Description |
|--------|-------------|
| `NewConfig(name)` | Create config with service name and default logger |
| `WithTracerProvider(tp)` | Enable distributed tracing |
| `WithMeterProvider(mp)` | Enable metrics collection |
| `WithLoggerProvider(lp)` | Set custom logger provider |
| `WithServiceVersion(v)` | Set service version |
| `WithoutLogging()` | Disable default stdout logging |

### Helper Methods

```go
// Check what's enabled
cfg.IsTracingEnabled()  // bool
cfg.IsMetricsEnabled()  // bool
cfg.IsLoggingEnabled()  // bool

// Get instrumentation components
tracer := cfg.GetTracer("scope-name")   // Returns no-op if disabled
meter := cfg.GetMeter("scope-name")     // Returns no-op if disabled
logger := cfg.GetLogger("scope-name")   // Returns no-op if disabled

// Cleanup
cfg.Shutdown(context.Background())
```

## Standard Logging Helper

The `otel` package provides `LogHelper` for OTel-aware logging with automatic log-span correlation:

```go
import "github.com/jasoet/pkg/v2/otel"

// Create a logger (uses OTel when configured, falls back to zerolog otherwise)
logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v2/mypackage", "mypackage.DoWork")

// Log with automatic trace_id/span_id injection (when OTel is enabled)
logger.Debug("Starting work", "workerId", 123)
logger.Info("Work completed", "duration", elapsed)
logger.Error(err, "Work failed", "workerId", 123)
```

**Benefits:**
- Automatic trace_id/span_id injection when OTel is configured
- Graceful fallback to zerolog when OTel is not configured
- Consistent API across all packages
- Errors automatically recorded in active spans

See [helper.go](./helper.go) for full documentation.

## Integration Examples

### HTTP Server

```go
import (
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/server"
)

otelConfig := otel.NewConfig("my-api").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

server.Start(server.Config{
    Port:       8080,
    OTelConfig: otelConfig,
})
```

### gRPC Server

```go
import (
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/grpc"
)

otelConfig := otel.NewConfig("my-grpc-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

grpcServer := grpc.NewServer(
    grpc.NewConfig("my-service", 9090).
        WithOTelConfig(otelConfig),
)
```

### Database

```go
import (
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/db"
)

otelConfig := otel.NewConfig("my-db-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

pool, _ := db.ConnectionConfig{
    DbType:     db.Postgresql,
    Host:       "localhost",
    OTelConfig: otelConfig,
}.Pool()

// All queries are automatically traced
pool.Find(&users)
```

### REST Client

```go
import (
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/rest"
)

otelConfig := otel.NewConfig("my-client").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

client := rest.NewClient(rest.ClientConfig{
    BaseURL:    "https://api.example.com",
    OTelConfig: otelConfig,
})

// Requests are automatically traced
client.Get("/users", &result)
```

## Complete Example

See the [fullstack OTel example](../examples/fullstack-otel) for a complete application demonstrating all three telemetry pillars across multiple packages.

## Testing

The package includes comprehensive tests with 97.1% coverage:

```bash
# Run tests
go test ./otel -v

# With coverage
go test ./otel -cover
```

### Test Utilities

Use no-op providers for testing:

```go
import (
    "github.com/jasoet/pkg/v2/otel"
    noopm "go.opentelemetry.io/otel/metric/noop"
    noopt "go.opentelemetry.io/otel/trace/noop"
)

func TestMyCode(t *testing.T) {
    cfg := otel.NewConfig("test-service").
        WithTracerProvider(noopt.NewTracerProvider()).
        WithMeterProvider(noopm.NewMeterProvider()).
        WithoutLogging()

    // Test your code with cfg
}
```

## Best Practices

### 1. Create Once, Share Everywhere

```go
// ✅ Good: Single config shared across packages
otelConfig := otel.NewConfig("my-service").
    WithTracerProvider(tp).
    WithMeterProvider(mp)

serverCfg := server.Config{OTelConfig: otelConfig}
dbCfg := db.Config{OTelConfig: otelConfig}
```

### 2. Always Shutdown

```go
// ✅ Good: Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := otelConfig.Shutdown(ctx); err != nil {
    log.Printf("OTel shutdown error: %v", err)
}
```

### 3. Check Before Using

```go
// ✅ Good: Check enablement
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("my-scope")
    // Use tracer
}
```

### 4. Use LogHelper for Consistent Logging

```go
// ✅ Good: Use otel.LogHelper for automatic log-span correlation
logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v2/mypackage", "mypackage.DoWork")
logger.Info("Work completed", "duration", elapsed)
```

## Architecture

### Design Principles

1. **Zero Dependencies**: Only depends on OTel SDK (no custom exporters)
2. **No-op Safety**: Nil providers result in no-op implementations
3. **Lazy Initialization**: Providers created only when needed
4. **Immutable Config**: Thread-safe after creation

### Package Structure

```
otel/
├── config.go        # Config struct and builder methods
├── helper.go        # Standard logging helper with OTel integration
├── helper_test.go   # LogHelper tests
├── doc.go          # Package documentation
└── config_test.go  # Config tests
```

## Troubleshooting

### No Telemetry Data

**Problem**: Not seeing traces/metrics/logs

**Solutions**:
```go
// 1. Check if enabled
fmt.Println("Tracing:", cfg.IsTracingEnabled())
fmt.Println("Metrics:", cfg.IsMetricsEnabled())
fmt.Println("Logging:", cfg.IsLoggingEnabled())

// 2. Verify providers are set
if cfg.TracerProvider == nil {
    // Tracing will be no-op
}

// 3. Ensure shutdown is called
defer cfg.Shutdown(context.Background())
```

### Default Logger Too Verbose

**Problem**: Stdout logger creating too much output

**Solution**:
```go
// Disable default logger
cfg := otel.NewConfig("my-service").WithoutLogging()

// Or use custom logger
cfg := otel.NewConfig("my-service").
    WithLoggerProvider(myLoggerProvider)
```

### Provider Already Registered

**Problem**: Global provider conflicts

**Solution**: This package doesn't use global providers - it returns scoped instruments from `GetTracer()`, `GetMeter()`, and `GetLogger()`.

## Version Compatibility

- **OpenTelemetry**: v1.38.0+
- **Go**: 1.25+
- **pkg library**: v2.0.0+

## Migration from v1

v2 uses OpenTelemetry v2 API:

```go
// v1 (OTel v1)
import "go.opentelemetry.io/otel"
tracer := otel.Tracer("my-scope")

// v2 (OTel v2)
import "github.com/jasoet/pkg/v2/otel"
cfg := otel.NewConfig("my-service").WithTracerProvider(tp)
tracer := cfg.GetTracer("my-scope")
```

See [VERSIONING_GUIDE.md](../VERSIONING_GUIDE.md) for complete migration guide.

## Related Packages

- **[logging](../logging/)** - Structured logging with OTel integration
- **[server](../server/)** - HTTP server with automatic tracing
- **[grpc](../grpc/)** - gRPC server with automatic instrumentation
- **[db](../db/)** - Database with query tracing
- **[rest](../rest/)** - REST client with distributed tracing

## License

MIT License - see [LICENSE](../LICENSE) for details.
