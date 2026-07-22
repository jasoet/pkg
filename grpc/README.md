# gRPC Server Package

A production-ready, reusable gRPC server with Echo HTTP framework integration for Go applications. This package provides a functional-options API for setting up gRPC servers with HTTP/REST gateway support, built-in observability, health checks, and graceful shutdown capabilities.

## Features

- **Echo Framework Integration**: Full-featured HTTP server using Echo v4
- **Dual Protocol Support**: Run gRPC and HTTP services on the same port (H2C) or separate ports
- **gRPC Gateway**: Mount a grpc-gateway mux on Echo and register generated handlers via `WithGatewayRegistrar`
- **Zero Configuration**: Works out-of-the-box with sensible defaults
- **Production Ready**: Built-in OpenTelemetry instrumentation, health checks, and graceful shutdown
- **Highly Configurable**: Functional options for CORS, rate limiting, timeouts, and custom middleware
- **Observability**: OpenTelemetry metrics, tracing, and structured logging for both gRPC and HTTP
- **Easy Integration**: Clean API that works with any gRPC service implementation

## Installation

```bash
go get github.com/jasoet/pkg/v3/grpc
```

## Quick Start

### Basic Usage (H2C Mode)

```go
package main

import (
    "log"

    "google.golang.org/grpc"

    grpcserver "github.com/jasoet/pkg/v3/grpc"
    calculatorv1 "your-module/gen/calculator/v1"
    "your-module/internal/service"
)

func main() {
    // Define service registrar
    serviceRegistrar := func(s *grpc.Server) {
        calculatorService := service.NewCalculatorService()
        calculatorv1.RegisterCalculatorServiceServer(s, calculatorService)
    }

    // Start server with minimal configuration
    log.Println("Starting gRPC server on :8080...")
    if err := grpcserver.Start("8080", serviceRegistrar); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

This starts a server in H2C mode where both gRPC and HTTP endpoints are available on port 8080:
- gRPC endpoints: `localhost:8080`
- HTTP gateway: `http://localhost:8080/api/v1/` (serves routes registered via `WithGatewayRegistrar`)
- Health checks: `http://localhost:8080/health`

## Server Modes

### H2C Mode (Default)
Single port serving both gRPC and HTTP traffic using HTTP/2 cleartext protocol:

```go
grpcserver.Start("8080", serviceRegistrar)
// or explicitly
grpcserver.StartH2C("8080", serviceRegistrar)
```

### Separate Mode
Different ports for gRPC and HTTP services:

```go
grpcserver.StartSeparate("9090", "9091", serviceRegistrar)
```

## Configuration with Options

The server is configured with functional options passed to `New` (or to the `Start*` convenience functions, which accept trailing options). All options are optional; sensible defaults apply.

```go
server, err := grpcserver.New(
    // Ports & mode
    grpcserver.WithH2CMode(),                       // or WithSeparateMode("9090", "9091")
    grpcserver.WithGRPCPort("50051"),

    // Timeouts
    grpcserver.WithShutdownTimeout(45*time.Second),
    grpcserver.WithReadTimeout(10*time.Second),
    grpcserver.WithWriteTimeout(15*time.Second),
    grpcserver.WithIdleTimeout(120*time.Second),
    grpcserver.WithConnectionTimeouts(30*time.Minute, 60*time.Minute, 10*time.Second),

    // Features
    grpcserver.WithHealthCheck(),
    grpcserver.WithHealthPath("/health"),
    grpcserver.WithReflection(),
    grpcserver.WithCORS(),
    grpcserver.WithRateLimit(100.0), // requests per second

    // Gateway
    grpcserver.WithGatewayBasePath("/api/v1"),

    // Hooks
    grpcserver.WithServiceRegistrar(serviceRegistrar),
    grpcserver.WithEchoConfigurer(func(e *echo.Echo) {
        e.GET("/status", func(c echo.Context) error {
            return c.JSON(200, map[string]string{"status": "running"})
        })
    }),
    grpcserver.WithShutdownHandler(func() error {
        // Close connections, clean up resources
        return nil
    }),
)
if err != nil {
    log.Fatal(err)
}

if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

### Option Reference

**Ports & Mode**
- `WithH2CMode()` — gRPC and HTTP on one port (default)
- `WithSeparateMode(grpcPort, httpPort string)` — separate ports for gRPC and HTTP
- `WithGRPCPort(port string)` — gRPC port (default `"8080"`)
- `WithHTTPPort(port string)` — HTTP gateway port, SeparateMode only (default `"8081"`)

**Timeouts**
- `WithShutdownTimeout(d)` — graceful shutdown timeout (default 30s)
- `WithReadTimeout(d)` / `WithWriteTimeout(d)` / `WithIdleTimeout(d)` — HTTP server timeouts (defaults 5s / 10s / 60s)
- `WithConnectionTimeouts(idle, age, grace)` — gRPC keepalive limits (defaults 15m / 30m / 5s)
- `WithMaxConnectionIdle(d)` / `WithMaxConnectionAge(d)` / `WithMaxConnectionAgeGrace(d)` — individual keepalive limits

**Health**
- `WithHealthCheck()` / `WithoutHealthCheck()` — toggle health endpoints (default enabled)
- `WithHealthPath(path)` — health base path (default `"/health"`)

**Reflection**
- `WithReflection()` / `WithoutReflection()` — toggle gRPC server reflection (default disabled)

**CORS**
- `WithCORS()` — enable CORS with the default (wildcard) policy
- `WithCORSConfig(middleware.CORSConfig)` — enable CORS with a custom configuration

**Rate limit**
- `WithRateLimit(rps float64)` — enable Echo rate limiting (default 100 rps when enabled)

**Gateway**
- `WithGatewayBasePath(path)` — base path for gateway routes (default `"/api/v1"`)
- `WithGatewayRegistrar(fn func(*runtime.ServeMux))` — register handlers on the gateway mux (see below)

**Lifecycle & hooks**
- `WithServiceRegistrar(fn func(*grpc.Server))` — register gRPC services
- `WithGRPCConfigurer(fn func(*grpc.Server))` — customize the gRPC server
- `WithEchoConfigurer(fn func(*echo.Echo))` — add custom routes/middleware; runs after the gateway mount, so its routes take precedence
- `WithShutdownHandler(fn func() error)` — custom shutdown hook
- `WithMiddleware(mw ...echo.MiddlewareFunc)` — additional Echo middleware

**OTel**
- `WithOTelConfig(cfg *otel.Config)` — enable OpenTelemetry (nil/absent disables it)

## gRPC Gateway

When a service or gateway registrar is configured, the server creates a grpc-gateway `runtime.ServeMux` and mounts it on Echo under the gateway base path (default `/api/v1`). The mount is a catch-all — **the gateway only serves what you register on the mux**, and the only way to register on it is `WithGatewayRegistrar`. The function runs during `Start`, after the mux is created and before it is mounted, which is where you hook in generated gateway code:

```go
server, err := grpcserver.New(
    grpcserver.WithH2CMode(),
    grpcserver.WithGRPCPort("8080"),
    grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
        calculatorv1.RegisterCalculatorServiceServer(s, calculatorService)
    }),
    grpcserver.WithGatewayRegistrar(func(mux *runtime.ServeMux) {
        // Register generated gateway handlers, e.g.:
        //   conn, _ := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
        //   calculatorv1.RegisterCalculatorServiceHandler(context.Background(), mux, conn)
        // or register plain HTTP routes directly:
        _ = mux.HandlePath(http.MethodGet, "/ping",
            func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
                _, _ = w.Write([]byte("pong"))
            })
    }),
)
```

Note the mount strips the base path before requests reach the mux, so the mux sees proto http-rule paths verbatim — a pattern of `/ping` (as generated code registers it) is served at `/api/v1/ping`. Do not include the base path prefix in `HandlePath` patterns.

`CreateGatewayMux()` and `MountGatewayOnEcho(e, mux, basePath)` are also exported if you need to assemble a gateway manually.

## OpenTelemetry Integration

The gRPC server package supports OpenTelemetry for comprehensive observability with distributed tracing, metrics, and structured logging. Provide an `*otel.Config` via `WithOTelConfig` to enable instrumentation. Instrumentation is implemented as gRPC **interceptors** (unary and stream) plus Echo middleware — it does not use `otel.Layers`.

### Basic OpenTelemetry Setup

```go
package main

import (
    "log"

    "google.golang.org/grpc"

    grpcserver "github.com/jasoet/pkg/v3/grpc"
    "github.com/jasoet/pkg/v3/otel"
)

func main() {
    // Optional: logger provider with better log-span correlation
    loggerProvider, err := otel.NewLoggerProviderWithOptions("my-grpc-service")
    if err != nil {
        log.Fatal(err)
    }

    // Create OTel config once (traces and metrics optional)
    otelCfg := otel.NewConfig("my-grpc-service",
        otel.WithServiceVersion("1.0.0"),
        otel.WithLoggerProvider(loggerProvider))

    // Start server with OTel
    server, err := grpcserver.New(
        grpcserver.WithGRPCPort("50051"),
        grpcserver.WithOTelConfig(otelCfg),
        grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
            // Register your services
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Full OpenTelemetry Stack (Traces + Metrics + Logs)

```go
package main

import (
    "context"
    "log"
    "time"

    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
    "google.golang.org/grpc"

    grpcserver "github.com/jasoet/pkg/v3/grpc"
    "github.com/jasoet/pkg/v3/otel"
)

func main() {
    ctx := context.Background()

    // Setup resource attributes
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String("my-grpc-service"),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Setup TracerProvider with OTLP exporter
    traceExporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }

    tracerProvider := trace.NewTracerProvider(
        trace.WithBatcher(traceExporter),
        trace.WithResource(res),
    )

    // Setup MeterProvider
    meterProvider := metric.NewMeterProvider(
        metric.WithResource(res),
    )

    // Setup LoggerProvider with trace correlation
    loggerProvider, err := otel.NewLoggerProviderWithOptions("my-grpc-service")
    if err != nil {
        log.Fatal(err)
    }

    // Create OTel config
    otelCfg := otel.NewConfig("my-grpc-service",
        otel.WithServiceVersion("1.0.0"),
        otel.WithTracerProvider(tracerProvider),
        otel.WithMeterProvider(meterProvider),
        otel.WithLoggerProvider(loggerProvider))

    // Start gRPC server with full OTel instrumentation
    server, err := grpcserver.New(
        grpcserver.WithGRPCPort("50051"),
        grpcserver.WithOTelConfig(otelCfg),
        grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
            // Register your services
        }),
        grpcserver.WithShutdownHandler(func() error {
            // Shutdown OTel providers
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()

            if err := tracerProvider.Shutdown(ctx); err != nil {
                log.Printf("Error shutting down tracer provider: %v", err)
            }
            if err := meterProvider.Shutdown(ctx); err != nil {
                log.Printf("Error shutting down meter provider: %v", err)
            }
            return nil
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### What Gets Instrumented

When `WithOTelConfig` is provided, the server automatically instruments:

#### gRPC Server (via interceptors)
- **Traces**: Distributed tracing for all gRPC methods with semantic conventions
- **Metrics**:
  - `rpc.server.request.count` - Total gRPC requests by method and status
  - `rpc.server.duration` - Request duration histogram
  - `rpc.server.active_requests` - Active concurrent requests
- **Logs**: Structured logs with automatic trace_id/span_id correlation

#### HTTP Gateway (via Echo middleware)
- **Traces**: HTTP request spans linked to gRPC spans
- **Metrics**:
  - `http.server.request.count` - Total HTTP requests
  - `http.server.request.duration` - Request duration histogram
  - `http.server.active_requests` - Active concurrent requests
- **Logs**: HTTP access logs with trace correlation

### Log-Span Correlation

When using the `logging` package LoggerProvider, all logs automatically include `trace_id` and `span_id` fields, enabling you to:
- Click a span in Grafana → See all related logs
- Click a log → Jump to the trace
- Filter logs by trace ID

```json
{
  "level": "info",
  "scope": "grpc.server",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "rpc.system": "grpc",
  "rpc.method": "/calculator.v1.CalculatorService/Add",
  "rpc.grpc.status_code": 0,
  "message": "gRPC /calculator.v1.CalculatorService/Add"
}
```

### Without OTelConfig

When no `*otel.Config` is provided, the server runs without metrics or structured logging instrumentation. Health checks still work. To enable observability, provide a config via `WithOTelConfig()`.

## Health Checks

The server provides comprehensive health check endpoints:

- `GET /health` - Overall health status
- `GET /health/ready` - Readiness probe
- `GET /health/live` - Liveness probe

### Custom Health Checks

```go
server, err := grpcserver.New(grpcserver.WithGRPCPort("8080"))
if err != nil {
    log.Fatal(err)
}

healthManager := server.GetHealthManager()

// Register custom health check
healthManager.RegisterCheck("database", func() grpcserver.HealthCheckResult {
    // Check database connectivity
    return grpcserver.HealthCheckResult{
        Status: grpcserver.HealthStatusUp,
        Details: map[string]interface{}{
            "connection": "active",
            "latency": "5ms",
        },
    }
})

// Start the server
if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

## Metrics (OpenTelemetry)

When `WithOTelConfig` is provided with a `MeterProvider`, the following OTel metrics are emitted:

### gRPC Metrics
- `rpc.server.request.count` - Total gRPC requests by method and status
- `rpc.server.duration` - Request duration histogram (ms)
- `rpc.server.active_requests` - Active concurrent gRPC requests
- `rpc.server.stream.count` - Total gRPC streams
- `rpc.server.stream.duration` - Stream duration histogram (ms)
- `rpc.server.active_streams` - Active concurrent streams

### HTTP Gateway Metrics
- `http.server.request.count` - Total HTTP gateway requests
- `http.server.request.duration` - Request duration histogram (ms)
- `http.server.active_requests` - Active concurrent HTTP requests

### Server Metrics
- `server.uptime` - Server uptime in seconds
- `server.start_time` - Server start time as Unix timestamp

Metrics flow through the OTel pipeline (OTLP exporter → collector → backend).

## API Reference

### Convenience Start Functions

All three block until shutdown, handle SIGINT/SIGTERM gracefully, and accept any number of trailing `Option`s.

#### `Start(port string, serviceRegistrar func(*grpc.Server), opts ...Option) error`
Starts a server in H2C mode (default mode) on the given port.

```go
grpcserver.Start("8080", func(s *grpc.Server) {
    // Register services
}, grpcserver.WithReflection())
```

#### `StartH2C(port string, serviceRegistrar func(*grpc.Server), opts ...Option) error`
Explicitly starts a server in H2C mode.

#### `StartSeparate(grpcPort, httpPort string, serviceRegistrar func(*grpc.Server), opts ...Option) error`
Starts a server in separate mode with different ports for gRPC and HTTP.

### Server Constructor

#### `New(opts ...Option) (*Server, error)`
Creates a server instance without starting it, for full lifecycle control:

```go
server, err := grpcserver.New(
    grpcserver.WithGRPCPort("50051"),
    grpcserver.WithServiceRegistrar(serviceRegistrar),
)
if err != nil {
    log.Fatal(err)
}

// Access managers before starting
healthManager := server.GetHealthManager()

// Start when ready (blocks)
if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

### Types

#### `Server`
Server instance with methods:
- `Start() error` - Start the server (blocks); supports Start/Stop/Start cycles
- `Stop() error` - Gracefully stop the server
- `IsRunning() bool` - Check if the server is running
- `GetHealthManager() *HealthManager` - Get the health check manager
- `GetGRPCServer() *grpc.Server` - Get the underlying gRPC server (nil after Stop until the next Start)

#### `Option`
Functional option: `type Option func(*config)`. See the Option Reference above.

#### `ServerMode`
Server mode enumeration:
- `H2CMode` - Single port HTTP/2 cleartext mode (gRPC + HTTP on same port)
- `SeparateMode` - Separate ports for gRPC and HTTP

## Architecture

### H2C Mode Architecture
```
Client Request → Port 8080
                    ↓
            HTTP/2 Cleartext Handler
                    ↓
         ┌──────────┴──────────┐
         ↓                     ↓
    gRPC Server           Echo HTTP Server
    (application/grpc)    (HTTP/1.1 & HTTP/2)
         ↓                     ↓
    Your Services        - gRPC Gateway
                         - Health Checks
                         - Custom Routes
```

### Separate Mode Architecture
```
Client Request
    ↓
    ├─→ Port 9090 → gRPC Server → Your Services
    │
    └─→ Port 9091 → Echo HTTP Server → - gRPC Gateway
                                       - Health Checks
                                       - Custom Routes
```

## Examples

The `examples/grpc/` directory contains a complete calculator service demonstrating:

- **Unary RPC**: Basic request-response operations (Add, Subtract, Multiply, Divide)
- **Server Streaming**: Server sends multiple responses (Factorial)
- **Client Streaming**: Client sends multiple requests (Sum)
- **Bidirectional Streaming**: Both send multiple messages (RunningAverage)
- **Echo Integration**: Custom HTTP routes alongside gRPC
- **Full Observability**: Health checks and metrics

### Running the Example

From the repository root:

```bash
# Run the server
go run -tags=example ./examples/grpc/cmd/server

# In another terminal, run the client
go run -tags=example ./examples/grpc/cmd/client

# Test HTTP endpoints
curl http://localhost:50051/status
curl http://localhost:50051/health
curl http://localhost:50051/calculator
```

## Testing

The package includes comprehensive tests covering:
- Server lifecycle (start, stop, restart)
- Configuration validation
- Health check functionality
- OTel instrumentation
- H2C and Separate modes
- Graceful shutdown

Run tests:
```bash
go test ./...
```

## Best Practices

1. **Use H2C Mode for Development**: Simplifies local testing with a single port
2. **Use Separate Mode for Production**: Better isolation and flexibility
3. **Enable OTel Observability**: Provide an `*otel.Config` with `MeterProvider` and `TracerProvider` for production
4. **Configure Timeouts**: Set appropriate timeouts based on your service requirements
5. **Use gRPC Reflection in Development**: Makes testing with tools like grpcurl easier
6. **Disable Reflection in Production**: Security best practice
7. **Add Custom Health Checks**: Monitor critical dependencies (database, cache, etc.)
8. **Use Echo Middleware**: Leverage Echo's rich middleware ecosystem via `WithMiddleware` or `WithEchoConfigurer`

## Dependencies

- `google.golang.org/grpc` - gRPC framework
- `github.com/grpc-ecosystem/grpc-gateway/v2` - gRPC gateway
- `github.com/labstack/echo/v4` - Echo HTTP framework
- `go.opentelemetry.io/otel` - OpenTelemetry instrumentation
- `golang.org/x/net` - HTTP/2 support

## License

This package is part of the jasoet/pkg project.
