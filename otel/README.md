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
- **OTLP Logging Support**: Export logs to OpenTelemetry collectors with flexible options
- **Granular Log Levels**: Fine-grained control over log verbosity (debug, info, warn, error, none)
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

### OTLP Logging with Flexible Options

Create a logger provider with OTLP export and granular control:

```go
import "github.com/jasoet/pkg/v2/otel"

// Console-only logging (default, no OTLP)
loggerProvider, err := otel.NewLoggerProviderWithOptions("my-service")

// OTLP logging with console output (local development)
loggerProvider, err := otel.NewLoggerProviderWithOptions(
    "my-service",
    otel.WithOTLPEndpoint("localhost:4318", true), // insecure for local
    otel.WithConsoleOutput(true),
    otel.WithLogLevel(logging.LogLevelInfo),
)

// OTLP-only logging (production)
loggerProvider, err := otel.NewLoggerProviderWithOptions(
    "my-service",
    otel.WithOTLPEndpoint("otel-collector.prod:4318", false), // secure
    otel.WithConsoleOutput(false), // disable console in prod
    otel.WithLogLevel(logging.LogLevelWarn),
)

// Use with OTel config
cfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
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

// Context management (recommended)
ctx = otel.ContextWithConfig(ctx, cfg)  // Store config in context
cfg = otel.ConfigFromContext(ctx)       // Retrieve config from context

// Cleanup
cfg.Shutdown(context.Background())
```

### Logger Provider Options

Create flexible logger providers with `NewLoggerProviderWithOptions`:

| Option | Description |
|--------|-------------|
| `WithOTLPEndpoint(endpoint, insecure)` | Enable OTLP log export to collector |
| `WithConsoleOutput(enabled)` | Enable/disable console logging (default: true) |
| `WithLogLevel(level)` | Set log level: `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarn`, `LogLevelError`, `LogLevelNone` |

**Log Level Priority:**
1. Explicit `WithLogLevel()` (highest priority)
2. Default to `info` level

**Examples:**

```go
import "github.com/jasoet/pkg/v2/logging"

// Default info level
provider, _ := otel.NewLoggerProviderWithOptions("service")

// Debug mode (all logs)
provider, _ := otel.NewLoggerProviderWithOptions("service",
    otel.WithLogLevel(logging.LogLevelDebug))

// Specific log level
provider, _ := otel.NewLoggerProviderWithOptions("service",
    otel.WithLogLevel(logging.LogLevelWarn))

// OTLP + console for development
provider, _ := otel.NewLoggerProviderWithOptions("service",
    otel.WithOTLPEndpoint("localhost:4318", true),
    otel.WithConsoleOutput(true),
    otel.WithLogLevel(logging.LogLevelDebug))

// OTLP-only for production
provider, _ := otel.NewLoggerProviderWithOptions("service",
    otel.WithOTLPEndpoint("collector:4318", false),
    otel.WithConsoleOutput(false),
    otel.WithLogLevel(logging.LogLevelInfo))
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

## Context-Based Config Propagation

The recommended pattern for passing OTel config through your application layers is to store it in the context once at the entry point:

```go
import "github.com/jasoet/pkg/v2/otel"

// At the HTTP handler entry point
func (h *Handler) HandleRequest(c echo.Context) error {
    // Store config in context once
    ctx := otel.ContextWithConfig(c.Request().Context(), h.otelConfig)

    // Config automatically available to all nested operations
    return h.service.ProcessRequest(ctx, req)
}

// In service layer - no need to pass config explicitly
func (s *Service) ProcessRequest(ctx context.Context, req Request) error {
    // Config retrieved from context automatically
    // Fields passed here are automatically included in all log calls
    lc := otel.Layers.StartService(ctx, "user", "ProcessRequest",
        otel.F("request.id", req.ID))
    defer lc.End()

    // Logger is always available (zerolog fallback when no config)
    // Fields "layer=service" and "request.id" are automatically included
    lc.Logger.Info("Processing request")

    return s.repo.Save(lc.Context(), data)
}

// In repository layer - config still available
func (r *Repository) Save(ctx context.Context, data Data) error {
    lc := otel.Layers.StartRepository(ctx, "user", "Save",
        otel.F("data.id", data.ID))
    defer lc.End()

    // Fields "layer=repository" and "data.id" automatically in logs
    lc.Logger.Debug("Saving to database")

    return lc.Success("Data saved")
}
```

**Benefits:**
- Set config once at entry point, available everywhere
- No need to pass config as parameter through all layers
- Natural propagation through context (like span data)
- Clean API - fewer parameters
- **Logger always available** (zerolog fallback when no config)
- **Fields automatically included** in all log calls

**API Pattern:**
```go
// Store config in context (once at entry point)
ctx = otel.ContextWithConfig(ctx, cfg)

// Create layer contexts - all return both Span and Logger
// Fields passed here are automatically included in all log calls
lc := otel.Layers.StartHandler(ctx, "user", "GetUser", otel.F("http.method", "GET"))
lc := otel.Layers.StartService(ctx, "user", "CreateUser", otel.F("user.email", email))
lc := otel.Layers.StartRepository(ctx, "user", "FindByID", otel.F("user.id", id))
lc := otel.Layers.StartOperations(ctx, "user", "ProcessQueue", otel.F("queue.name", queue))
lc := otel.Layers.StartMiddleware(ctx, "auth", "ValidateToken", otel.F("token.type", "JWT"))

// All log calls automatically include the fields
lc.Logger.Info("Processing")           // Includes all fields
lc.Logger.Debug("Details", F("extra", val)) // Adds extra field
lc.Error(err, "Failed")               // Includes all fields
lc.Success("Done")                    // Includes all fields

// Get logger from span (config retrieved automatically)
span := otel.StartSpan(ctx, "service.user", "DoWork")
logger := span.Logger("service.user") // No config parameter needed
```

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
├── config_test.go   # Config tests
├── logging.go       # OTLP logger provider with flexible options
├── logging_test.go  # Logger provider tests
├── helper.go        # Standard logging helper with OTel integration
├── helper_test.go   # LogHelper tests
├── instrumentation.go        # Instrumentation utilities
├── instrumentation_test.go   # Instrumentation tests
└── doc.go          # Package documentation
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
