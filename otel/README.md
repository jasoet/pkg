# OpenTelemetry Integration

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/otel.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/otel)

Unified OpenTelemetry configuration, instrumentation, and logging utilities for the `pkg` library ecosystem.

## Overview

The `otel` package provides centralized OpenTelemetry configuration that enables observability across all library packages. It supports the three pillars of observability:

- **Traces** - Distributed tracing for request flows
- **Metrics** - Performance and health measurements
- **Logs** - Structured logging via OpenTelemetry standard

Since v3, the former `logging` package is absorbed into `otel`: global zerolog bootstrap (`otel.Initialize`, `otel.InitializeWithFile`, `otel.ContextLogger`) and OTLP logger providers (`otel.NewLoggerProviderWithOptions`) live here.

## Features

- **Unified Configuration**: Single config object for all telemetry pillars, built with functional options
- **Selective Enablement**: Enable only the telemetry you need
- **No-op by Default**: Zero overhead when providers are not configured
- **Layer Instrumentation**: `otel.Layers.Start*()` spans with integrated, correlated logging
- **Standard Logging Helper**: OTel-aware logging with automatic trace correlation
- **Global Logger Bootstrap**: zerolog global logger setup with console and/or file output
- **OTLP Logging Support**: Export logs to OpenTelemetry collectors with flexible options
- **Granular Log Levels**: Fine-grained control over log verbosity (debug, info, warn, error, none)
- **Graceful Shutdown**: Proper resource cleanup

## Installation

```bash
go get github.com/jasoet/pkg/v3/otel
```

## Quick Start

### Basic Configuration

```go
import "github.com/jasoet/pkg/v3/otel"

// Create unified OTel config (compile-checked: ExampleNewConfig)
otelConfig := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider), // optional
    otel.WithMeterProvider(meterProvider),   // optional
    otel.WithServiceVersion("1.0.0"))

// Use with library packages via their OTelConfig field / WithOTelConfig option

// Cleanup on shutdown
defer otelConfig.Shutdown(context.Background())
```

### Selective Telemetry

Enable only what you need:

```go
// Tracing only (default logging disabled)
cfg := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithoutLogging())

// Metrics only (default logging disabled)
cfg := otel.NewConfig("my-service",
    otel.WithMeterProvider(meterProvider),
    otel.WithoutLogging())

// All three pillars
cfg := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider),
    otel.WithLoggerProvider(loggerProvider))
```

### Custom Logger Provider

Use `otel.NewLoggerProviderWithOptions` for better formatting and automatic trace correlation (compile-checked: `ExampleNewLoggerProviderWithOptions`):

```go
import "github.com/jasoet/pkg/v3/otel"

loggerProvider, err := otel.NewLoggerProviderWithOptions("my-service",
    otel.WithLogLevel(otel.LogLevelDebug))
if err != nil {
    panic(err)
}

cfg := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithLoggerProvider(loggerProvider))
```

### OTLP Logging with Flexible Options

Console output is enabled by default; add an OTLP endpoint to also export logs to a collector:

```go
import "github.com/jasoet/pkg/v3/otel"

// OTLP logging with console output (local development)
loggerProvider, err := otel.NewLoggerProviderWithOptions(
    "my-service",
    otel.WithOTLPEndpoint("https://localhost:4318", true), // insecure for local
    otel.WithConsoleOutput(true),
    otel.WithLogLevel(otel.LogLevelInfo),
)

// OTLP-only logging (production)
loggerProvider, err = otel.NewLoggerProviderWithOptions(
    "my-service",
    otel.WithOTLPEndpoint("https://otel-collector.prod:4318", false), // secure
    otel.WithConsoleOutput(false), // disable console in prod
    otel.WithLogLevel(otel.LogLevelWarn),
)
```

Note: this package uses `otlploghttp`, so OTLP endpoints are full URLs with scheme.

## Global Logger Bootstrap

For plain (non-OTel) logging, initialize the global zerolog logger once at startup (compile-checked: `ExampleInitialize`):

```go
import "github.com/jasoet/pkg/v3/otel"

// Console-only global logger at info level (debug=true for debug level + caller)
err := otel.Initialize("my-service", false)

// Console + file output
closer, err := otel.InitializeWithFile("my-service", true,
    otel.OutputConsole|otel.OutputFile,
    &otel.FileConfig{Path: "app.log"})
if err != nil {
    log.Fatal(err)
}
defer closer.Close()

// Component-scoped logger derived from the global logger
logger := otel.ContextLogger(ctx, "repository")
```

Global log records are written to stderr (console) and/or the configured file.

## Configuration API

### Functional Options

| Option | Description |
|--------|-------------|
| `NewConfig(name, opts...)` | Create config with service name, default logger, and options |
| `WithTracerProvider(tp)` | Enable distributed tracing |
| `WithMeterProvider(mp)` | Enable metrics collection |
| `WithLoggerProvider(lp)` | Set custom logger provider |
| `WithServiceVersion(v)` | Set service version |
| `WithoutTracing()` | Disable tracing |
| `WithoutMetrics()` | Disable metrics |
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
| `WithOTLPEndpoint(endpoint, insecure)` | Enable OTLP log export to collector (full URL with scheme) |
| `WithConsoleOutput(enabled)` | Enable/disable console logging (default: true) |
| `WithLogLevel(level)` | Set log level: `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarn`, `LogLevelError`, `LogLevelNone` |

**Log Level Priority:**
1. Explicit `WithLogLevel()` (highest priority)
2. Default to `info` level

**Examples:**

```go
import "github.com/jasoet/pkg/v3/otel"

// Default info level
provider, _ := otel.NewLoggerProviderWithOptions("service")

// Debug mode (all logs)
provider, _ = otel.NewLoggerProviderWithOptions("service",
    otel.WithLogLevel(otel.LogLevelDebug))

// OTLP + console for development
provider, _ = otel.NewLoggerProviderWithOptions("service",
    otel.WithOTLPEndpoint("https://localhost:4318", true),
    otel.WithConsoleOutput(true),
    otel.WithLogLevel(otel.LogLevelDebug))

// OTLP-only for production
provider, _ = otel.NewLoggerProviderWithOptions("service",
    otel.WithOTLPEndpoint("https://collector:4318", false),
    otel.WithConsoleOutput(false),
    otel.WithLogLevel(otel.LogLevelInfo))
```

## Standard Logging Helper

The `otel` package provides `LogHelper` for OTel-aware logging with automatic log-span correlation (compile-checked: `Example_optionalFunctionParameter`):

```go
import "github.com/jasoet/pkg/v3/otel"

// Create a logger (uses OTel when configured, falls back to zerolog otherwise)
logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v3/mypackage", "mypackage.DoWork")

// Log with automatic trace_id/span_id injection (when OTel is enabled)
logger.Debug("Starting work", otel.F("workerId", 123))
logger.Info("Work completed", otel.F("duration", elapsed))
logger.Error(err, "Work failed", otel.F("workerId", 123))
```

**Benefits:**
- Automatic trace_id/span_id injection when OTel is configured
- Graceful fallback to zerolog when OTel is not configured
- Consistent API across all packages
- Errors automatically recorded in active spans

See [helper.go](./helper.go) for full documentation.

## Layer Instrumentation

`otel.Layers` provides five starters — `StartHandler`, `StartMiddleware`, `StartOperations`, `StartService`, `StartRepository` — each returning a `LayerContext` with both a span and a correlated logger (compile-checked: `Example_layerContextIntegration`, `Example_middlewareLayer`):

```go
// Fields passed here are automatically included in all log calls
lc := otel.Layers.StartService(ctx, "user", "CreateUser",
    otel.F("user.id", "12345"))
defer lc.End()

lc.Logger.Info("Creating user", otel.F("email", "user@example.com"))

if err := repo.Save(lc.Context(), data); err != nil {
    return lc.Error(err, "save failed")
}
lc.Success("User created")
```

Note: `Success` sets the span status to `codes.Ok`; per the OTel specification the status description is dropped for `Ok`, so the message appears in the log but not on the span.

## Context-Based Config Propagation

The recommended pattern for passing OTel config through your application layers is to store it in the context once at the entry point (compile-checked: `Example_withOTelConfig`, `Example_layerPropagation`):

```go
import "github.com/jasoet/pkg/v3/otel"

// At the HTTP handler entry point
func (h *Handler) HandleRequest(c echo.Context) error {
    // Store config in context once
    ctx := otel.ContextWithConfig(c.Request().Context(), h.otelConfig)

    // Config automatically available to all nested operations
    return h.service.ProcessRequest(ctx, req)
}

// In service layer - no need to pass config explicitly
func (s *Service) ProcessRequest(ctx context.Context, req Request) error {
    lc := otel.Layers.StartService(ctx, "user", "ProcessRequest",
        otel.F("request.id", req.ID))
    defer lc.End()

    lc.Logger.Info("Processing request")

    return s.repo.Save(lc.Context(), data)
}
```

**Benefits:**
- Set config once at entry point, available everywhere
- No need to pass config as parameter through all layers
- Natural propagation through context (like span data)
- **Logger always available** (zerolog fallback when no config — see `Example_withoutOTelConfig`)
- **Fields automatically included** in all log calls

Config is optional but recommended for production; you can adopt it gradually (see `Example_gradualOTelAdoption`, `Example_configOptionalButRecommended`).

**API Pattern:**
```go
// Store config in context (once at entry point)
ctx = otel.ContextWithConfig(ctx, cfg)

// Create layer contexts - all return both Span and Logger
lc := otel.Layers.StartHandler(ctx, "user", "GetUser", otel.F("http.method", "GET"))
lc := otel.Layers.StartMiddleware(ctx, "auth", "ValidateToken", otel.F("token.type", "JWT"))
lc := otel.Layers.StartOperations(ctx, "user", "ProcessQueue", otel.F("queue.name", queue))
lc := otel.Layers.StartService(ctx, "user", "CreateUser", otel.F("user.email", email))
lc := otel.Layers.StartRepository(ctx, "user", "FindByID", otel.F("user.id", id))

// All log calls automatically include the fields
lc.Logger.Info("Processing")                    // Includes all fields
lc.Logger.Debug("Details", otel.F("extra", val)) // Adds extra field
lc.Error(err, "Failed")                         // Includes all fields
lc.Success("Done")                              // Includes all fields

// Get logger from span (config retrieved automatically)
span := otel.StartSpan(ctx, "service.user", "DoWork")
logger := span.Logger("service.user") // No config parameter needed
```

## Using with Library Packages

Create one `*otel.Config` and inject it into each package's configuration. Every instrumented package exposes either an `OTelConfig *otel.Config` config field or a `WithOTelConfig(cfg)` option:

```go
import "github.com/jasoet/pkg/v3/otel"

otelConfig := otel.NewConfig("my-app",
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider))

// server.Config{OTelConfig: otelConfig, ...}
// grpc.WithOTelConfig(otelConfig)
// db config with OTelConfig field
// rest.WithOTelConfig(otelConfig)
```

See each package's README for its exact wiring (`server`, `grpc`, `db`, `rest`, `temporal`, `docker`, `argo`).

## Complete Example

See the [fullstack OTel example](../examples/fullstack-otel) for a complete application demonstrating all three telemetry pillars across multiple packages.

## Testing

```bash
# Run tests (includes Output-verified examples)
go test ./otel -v

# With coverage
go test ./otel -cover
```

### Test Utilities

Use no-op providers for testing:

```go
import (
    "github.com/jasoet/pkg/v3/otel"
    noopm "go.opentelemetry.io/otel/metric/noop"
    noopt "go.opentelemetry.io/otel/trace/noop"
)

func TestMyCode(t *testing.T) {
    cfg := otel.NewConfig("test-service",
        otel.WithTracerProvider(noopt.NewTracerProvider()),
        otel.WithMeterProvider(noopm.NewMeterProvider()),
        otel.WithoutLogging())

    // Test your code with cfg
}
```

## Best Practices

### 1. Create Once, Share Everywhere

```go
// Good: Single config shared across packages
otelConfig := otel.NewConfig("my-service",
    otel.WithTracerProvider(tp),
    otel.WithMeterProvider(mp))
```

### 2. Always Shutdown

```go
// Good: Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := otelConfig.Shutdown(ctx); err != nil {
    log.Printf("OTel shutdown error: %v", err)
}
```

### 3. Check Before Using

```go
// Good: Check enablement
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("my-scope")
    // Use tracer
}
```

### 4. Use LogHelper for Consistent Logging

```go
// Good: Use otel.LogHelper for automatic log-span correlation
logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v3/mypackage", "mypackage.DoWork")
logger.Info("Work completed", otel.F("duration", elapsed))
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
├── config.go                       # Config struct and functional options
├── config_test.go                  # Config tests
├── options_test.go                 # Functional options tests
├── bootstrap.go                    # Global zerolog logger bootstrap (Initialize, ContextLogger)
├── bootstrap_test.go               # Bootstrap tests
├── logging.go                      # OTLP logger provider with flexible options
├── logging_test.go                 # Logger provider tests
├── helper.go                       # Standard logging helper with OTel integration
├── helper_test.go                  # LogHelper tests
├── instrumentation.go              # Span/layer instrumentation utilities
├── instrumentation_test.go         # Instrumentation tests
├── instrumentation_behavior_test.go # Behavioral tests with in-memory exporter
├── examples_test.go                # Compile-checked, Output-verified examples
├── instrumentation_example_test.go # Compile-checked, Output-verified examples
└── doc.go                          # Package documentation
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
cfg := otel.NewConfig("my-service", otel.WithoutLogging())

// Or use custom logger
cfg := otel.NewConfig("my-service",
    otel.WithLoggerProvider(myLoggerProvider))
```

### Provider Already Registered

**Problem**: Global provider conflicts

**Solution**: This package doesn't use global providers - it returns scoped instruments from `GetTracer()`, `GetMeter()`, and `GetLogger()`.

## Version Compatibility

- **OpenTelemetry**: v1.38.0+
- **Go**: 1.25+
- **pkg library**: v3.0.0+

## Migration from v2

v3 absorbs the `logging` package into `otel` and switches `Config` construction to functional options:

```go
// v2
import "github.com/jasoet/pkg/v2/logging"
err := logging.Initialize("my-service", false)
cfg := otel.NewConfig("my-service").WithServiceVersion("1.0.0")

// v3
import "github.com/jasoet/pkg/v3/otel"
err := otel.Initialize("my-service", false)
cfg := otel.NewConfig("my-service", otel.WithServiceVersion("1.0.0"))
```

See the [v3 audit backlog](../docs/plans/2026-07-22-v3-audit-backlog.md) for the full list of changes; a complete migration guide ships with v3.0.0.

## Related Packages

- **[server](../server/)** - HTTP server with automatic tracing
- **[grpc](../grpc/)** - gRPC server with automatic instrumentation
- **[db](../db/)** - Database with query tracing
- **[rest](../rest/)** - REST client with distributed tracing

## License

MIT License - see [LICENSE](../LICENSE) for details.
