# Server Package Examples (v2)

This directory contains runnable examples demonstrating the features of the `server` package with OpenTelemetry support.

## 📍 Example Code Location

**Full example implementation:** [/server/examples/example.go](https://github.com/jasoet/pkg/blob/main/server/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern with v2 (OpenTelemetry)
import (
    "github.com/jasoet/pkg/server"
    "github.com/jasoet/pkg/otel"
    "github.com/labstack/echo/v4"
)

// Option 1: Basic server without telemetry
operation := func(e *echo.Echo) {
    e.GET("/api/users", getUsersHandler)
}
shutdown := func(e *echo.Echo) {
    // Cleanup
}
server.Start(8080, operation, shutdown)

// Option 2: With OpenTelemetry (logging only)
otelCfg := otel.NewConfig("my-service")  // Default logging to stdout
config := server.DefaultConfig(8080, operation, shutdown)
config.OTelConfig = otelCfg
server.StartWithConfig(config)

// Option 3: Full OpenTelemetry (traces + metrics + logs)
otelCfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithServiceVersion("1.0.0")
config.OTelConfig = otelCfg
server.StartWithConfig(config)
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

To run the interactive examples:

```bash
cd /path/to/pkg/server/examples
go run -tags example example.go
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

## v2 Breaking Changes

**⚠️ Important:** v2 introduces breaking changes from v1:

### Removed (v1):
- `EnableMetrics` field
- `MetricsPath` field
- `MetricsSubsystem` field
- Prometheus metrics integration
- `github.com/rs/zerolog` logging

### Added (v2):
- `OTelConfig *otel.Config` field
- OpenTelemetry traces, metrics, and logs
- Default LoggerProvider via `otel.NewConfig()`
- Independent control of telemetry pillars
- Simple `fmt` logging for server lifecycle

### Migration Guide

**Before (v1):**
```go
config := server.Config{
    Port: 8080,
    EnableMetrics: true,
    MetricsPath: "/metrics",
}
```

**After (v2):**
```go
// Without telemetry
config := server.DefaultConfig(8080, operation, shutdown)

// With OpenTelemetry logging
otelCfg := otel.NewConfig("my-service")
config.OTelConfig = otelCfg

// With full telemetry (traces + metrics + logs)
otelCfg := otel.NewConfig("my-service").
    WithTracerProvider(tp).
    WithMeterProvider(mp)
config.OTelConfig = otelCfg
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

For comprehensive documentation, configuration options, and additional examples, see the main server package README at `../README.md`.

The server package README includes:
- Complete configuration reference
- OpenTelemetry integration guide
- Advanced middleware examples
- Health check patterns
- Telemetry customization
- Production deployment guides
- Integration examples with databases and external services
