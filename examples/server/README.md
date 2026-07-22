# Server Package Examples (v3)

This directory contains runnable examples demonstrating the features of the `server` package with OpenTelemetry support.

## 📍 Example Code Location

**Full example implementation:** [example.go](./example.go) (in this directory)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern with v3 (OpenTelemetry)
import (
    "github.com/jasoet/pkg/v3/server"
    "github.com/jasoet/pkg/v3/otel"
    "github.com/labstack/echo/v4"
)

// Option 1: Basic server without telemetry
operation := func(e *echo.Echo) {
    e.GET("/api/users", getUsersHandler)
}
shutdown := func(e *echo.Echo) {
    // Cleanup
}
srv, _ := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
)
srv.Start() // blocks until srv.Shutdown(ctx) is called

// Option 2: With OpenTelemetry (logging only)
otelCfg := otel.NewConfig("my-service")  // Default logging to stdout
srv, _ = server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithOTelConfig(otelCfg),
)
srv.Start()

// Option 3: Full OpenTelemetry (traces + metrics + logs)
otelCfg = otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithServiceVersion("1.0.0")
srv, _ = server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithOTelConfig(otelCfg),
)
srv.Start()
```

**Built-in endpoints:**
- `/health` - Main health check
- `/health/ready` - Readiness probe
- `/health/live` - Liveness probe

## Overview

The examples in this directory complement the comprehensive documentation in the main server package README. These are practical, runnable demonstrations of:

- Basic server setup
- OpenTelemetry configuration (traces, metrics, logs)
- Custom routes and middleware
- Health check implementations
- Graceful shutdown patterns

## Running the Examples

To run the interactive examples from the repository root:

```bash
go run -tags=example ./examples/server
```

This will run through 5 different server examples in sequence, each demonstrating different aspects of the server package.

## Examples Included

### 1. Basic Server Setup
- Minimal configuration with Operation and Shutdown functions
- Built-in health endpoints (`/health`, `/health/ready`, `/health/live`)
- No telemetry (OTelConfig is nil)

### 2. Server with OpenTelemetry Configuration
- Default LoggerProvider (stdout) enabled via `otel.NewConfig()`
- Demonstrates logging without tracing/metrics
- Shows fluent API for configuration
- Independent control of telemetry pillars

### 3. Custom Routes and Middleware
- RESTful API endpoints
- Authentication middleware
- Request ID middleware
- Route grouping and protection

### 4. Health Checks
- Custom health check implementations
- Service dependency monitoring
- Detailed health status reporting

### 5. Graceful Shutdown
- Signal handling (SIGTERM, SIGINT)
- Cleanup handlers
- In-flight request completion
- Configurable shutdown timeouts

## v3 Breaking Changes

**⚠️ Important:** v3 introduces breaking changes from v2:

### Removed (v2):
- `server.Start(port, operation, shutdown, middleware...)` (blocked on OS signals)
- `server.StartWithConfig(config)` (blocked on OS signals)
- `server.DefaultConfig(port, operation, shutdown)`

### Added (v3):
- `server.New(opts ...Option) (*Server, error)` — validates config, prepares Echo without binding
- `srv.Start()` — blocks until `Shutdown` is called; returns `nil` on a clean shutdown
- `srv.Shutdown(ctx)` — programmatic, idempotent graceful shutdown
- `srv.Addr()` — bound listener address (discovers the OS-assigned port with `WithPort(0)`)
- `srv.Echo()` — access to the underlying Echo instance before `Start`
- Auto-installed OTel tracing and metrics middleware when `OTelConfig` is set

Note: a stopped `Server` cannot be restarted — `Start` returns an error; create a new one with `New`.

### Migration Guide

**Before (v2):**
```go
config := server.DefaultConfig(8080, operation, shutdown)
config.ShutdownTimeout = 30 * time.Second
err := server.StartWithConfig(config) // blocked until SIGINT/SIGTERM
```

**After (v3):**
```go
// Without telemetry
srv, _ := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
)

// With OpenTelemetry logging
otelCfg := otel.NewConfig("my-service")
srv, _ = server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithOTelConfig(otelCfg),
)

// With full telemetry (traces + metrics + logs)
otelCfg = otel.NewConfig("my-service").
    WithTracerProvider(tp).
    WithMeterProvider(mp)
srv, _ = server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithOTelConfig(otelCfg),
)
```

## Integration with Other Packages

The examples demonstrate integration with:
- **otel package**: OpenTelemetry traces, metrics, and logs
- **Echo framework**: Custom routes and middleware
- **Standard library**: Simple console logging for lifecycle events

## Key Features Demonstrated

- **Flexible telemetry**: Enable logging, tracing, and metrics independently
- **Zero-configuration startup**: Minimal code for production-ready server
- **Flexible configuration**: Extensive customization options
- **Built-in observability**: Health checks and OpenTelemetry support
- **Production patterns**: Graceful shutdown, error handling, security middleware
- **Extensibility**: Custom routes, middleware, and health checks

## Related Documentation

For comprehensive documentation, configuration options, and additional examples, see the main server package README at [`../../server/README.md`](../../server/README.md).

The server package README includes:
- Complete configuration reference
- OpenTelemetry integration guide
- Advanced middleware examples
- Health check patterns
- Telemetry customization
- Production deployment guides
- Integration examples with databases and external services
