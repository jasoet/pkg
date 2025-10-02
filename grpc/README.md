# gRPC Server Package

A production-ready, reusable gRPC server with Echo HTTP framework integration for Go applications. This package provides a clean, configuration-driven API for setting up gRPC servers with HTTP/REST gateway support, built-in observability, health checks, and graceful shutdown capabilities.

## Features

- **Echo Framework Integration**: Full-featured HTTP server using Echo v4
- **Dual Protocol Support**: Run gRPC and HTTP services on the same port (H2C) or separate ports
- **gRPC Gateway**: Automatic HTTP/REST endpoints for gRPC services
- **Zero Configuration**: Works out-of-the-box with sensible defaults
- **Production Ready**: Built-in Prometheus metrics, health checks, and graceful shutdown
- **Highly Configurable**: Extensive configuration options including CORS, rate limiting, and custom middleware
- **Observability**: Prometheus metrics with detailed request tracking for both gRPC and HTTP
- **Easy Integration**: Clean API that works with any gRPC service implementation

## Installation

```bash
go get github.com/jasoet/pkg/grpc
```

## Quick Start

### Basic Usage (H2C Mode)

```go
package main

import (
    "log"
    "google.golang.org/grpc"

    grpcserver "github.com/jasoet/pkg/grpc"
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
- HTTP gateway: `http://localhost:8080/api/v1/`
- Health checks: `http://localhost:8080/health`
- Metrics: `http://localhost:8080/metrics`

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

## Advanced Configuration

### Echo Integration with Custom Routes

```go
package main

import (
    "log"
    "time"

    "github.com/labstack/echo/v4"
    "google.golang.org/grpc"

    grpcserver "github.com/jasoet/pkg/grpc"
)

func main() {
    // Create advanced configuration
    config := grpcserver.DefaultConfig()

    // Server Configuration
    config.GRPCPort = "50051"
    config.Mode = grpcserver.H2CMode

    // Timeouts
    config.ShutdownTimeout = 45 * time.Second
    config.ReadTimeout = 10 * time.Second
    config.WriteTimeout = 15 * time.Second
    config.IdleTimeout = 120 * time.Second
    config.MaxConnectionIdle = 30 * time.Minute
    config.MaxConnectionAge = 60 * time.Minute
    config.MaxConnectionAgeGrace = 10 * time.Second

    // Production Features
    config.EnableMetrics = true
    config.EnableHealthCheck = true
    config.EnableLogging = true
    config.EnableReflection = true

    // Echo-specific Features
    config.EnableCORS = true
    config.EnableRateLimit = true
    config.RateLimit = 100.0 // requests per second

    // Gateway Configuration
    config.GatewayBasePath = "/api/v1"

    // Register gRPC services
    config.ServiceRegistrar = func(srv *grpc.Server) {
        // Register your gRPC services here
        log.Println("Registering gRPC services...")
    }

    // Configure Echo with custom routes
    config.EchoConfigurer = func(e *echo.Echo) {
        // Add custom REST endpoints
        e.GET("/status", func(c echo.Context) error {
            return c.JSON(200, map[string]interface{}{
                "service": "my-service",
                "status":  "running",
            })
        })

        // Add custom middleware
        e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
            return func(c echo.Context) error {
                log.Printf("Custom middleware: %s %s", c.Request().Method, c.Path())
                return next(c)
            }
        })

        log.Println("Custom Echo routes configured")
    }

    // Custom gRPC configuration
    config.GRPCConfigurer = func(s *grpc.Server) {
        log.Println("Applying custom gRPC configuration...")
        // Add interceptors, custom options, etc.
    }

    // Custom shutdown handler
    config.Shutdown = func() error {
        log.Println("Running custom cleanup...")
        // Close connections, cleanup resources
        return nil
    }

    // Start server
    if err := grpcserver.StartWithConfig(config); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

### Using Echo Middleware

```go
import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

config := grpcserver.DefaultConfig()

// Add Echo middleware via configuration
config.Middleware = []echo.MiddlewareFunc{
    middleware.RequestID(),
    middleware.Secure(),
    middleware.Gzip(),
}

// Or configure via EchoConfigurer
config.EchoConfigurer = func(e *echo.Echo) {
    e.Use(middleware.RequestID())
    e.Use(middleware.Secure())
    e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
        Level: 5,
    }))
}
```

## OpenTelemetry Integration

The gRPC server package supports OpenTelemetry for comprehensive observability with distributed tracing, metrics, and structured logging. When `OTelConfig` is provided, it replaces traditional Prometheus metrics with OpenTelemetry instrumentation.

### Basic OpenTelemetry Setup

```go
package main

import (
    "context"
    "log"

    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/otel"
    grpcserver "github.com/jasoet/pkg/grpc"
    "google.golang.org/grpc"
)

func main() {
    // Create OTel config with logging (traces and metrics optional)
    otelCfg := otel.NewConfig("my-grpc-service").
        WithServiceVersion("1.0.0")

    // Or use logging package for better log-span correlation
    loggerProvider := logging.NewLoggerProvider("my-grpc-service", false)
    otelCfg.WithLoggerProvider(loggerProvider)

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

    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/otel"
    grpcserver "github.com/jasoet/pkg/grpc"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
    "google.golang.org/grpc"
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
    loggerProvider := logging.NewLoggerProvider("my-grpc-service", false)

    // Create OTel config
    otelCfg := &otel.Config{
        ServiceName:    "my-grpc-service",
        ServiceVersion: "1.0.0",
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
        LoggerProvider: loggerProvider,
    }

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

When `OTelConfig` is provided, the server automatically instruments:

#### gRPC Server
- **Traces**: Distributed tracing for all gRPC methods with semantic conventions
- **Metrics**:
  - `rpc.server.request.count` - Total gRPC requests by method and status
  - `rpc.server.duration` - Request duration histogram
  - `rpc.server.active_requests` - Active concurrent requests
- **Logs**: Structured logs with automatic trace_id/span_id correlation

#### HTTP Gateway
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

### Backwards Compatibility

The gRPC server maintains backwards compatibility:
- **Without OTelConfig**: Uses traditional Prometheus metrics and standard logging
- **With OTelConfig**: Uses OpenTelemetry instrumentation
- Health checks work in both modes
- No breaking changes to existing code

### Migration from Prometheus to OTel

**Before (Prometheus):**
```go
server, err := grpcserver.New(
    grpcserver.WithGRPCPort("50051"),
    grpcserver.WithMetrics(),  // Prometheus metrics
    grpcserver.WithLogging(),  // Standard logging
    grpcserver.WithServiceRegistrar(serviceRegistrar),
)
```

**After (OpenTelemetry):**
```go
otelCfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

server, err := grpcserver.New(
    grpcserver.WithGRPCPort("50051"),
    grpcserver.WithOTelConfig(otelCfg),  // Replaces metrics and logging
    grpcserver.WithServiceRegistrar(serviceRegistrar),
)
```

## Configuration Options

### Core Settings
- `GRPCPort`: Port for gRPC server (required)
- `HTTPPort`: Port for HTTP gateway (required for separate mode)
- `Mode`: Server mode (`H2CMode` or `SeparateMode`)

### Timeouts
- `ShutdownTimeout`: Graceful shutdown timeout (default: 30s)
- `ReadTimeout`: HTTP read timeout (default: 5s)
- `WriteTimeout`: HTTP write timeout (default: 10s)
- `IdleTimeout`: HTTP idle timeout (default: 60s)
- `MaxConnectionIdle`: Max connection idle time (default: 15m)
- `MaxConnectionAge`: Max connection age (default: 30m)
- `MaxConnectionAgeGrace`: Connection age grace period (default: 5s)

### Features
- `EnableMetrics`: Enable Prometheus metrics (default: true)
- `MetricsPath`: Metrics endpoint path (default: "/metrics")
- `EnableHealthCheck`: Enable health check endpoints (default: true)
- `HealthPath`: Health check path (default: "/health")
- `EnableLogging`: Enable request logging (default: true)
- `EnableReflection`: Enable gRPC reflection (default: true)

### Echo-Specific Features
- `EnableCORS`: Enable CORS middleware (default: false)
- `EnableRateLimit`: Enable rate limiting middleware (default: false)
- `RateLimit`: Requests per second for rate limiting (default: 100.0)
- `Middleware`: Custom Echo middleware functions

### Gateway Configuration
- `GatewayBasePath`: Base path for gRPC gateway routes (default: "/api/v1")

### Customization Hooks
- `ServiceRegistrar`: Function to register gRPC services
- `GRPCConfigurer`: Function to customize gRPC server
- `EchoConfigurer`: Function to configure Echo server and add custom routes
- `Shutdown`: Custom shutdown handler

## Health Checks

The server provides comprehensive health check endpoints:

- `GET /health` - Overall health status
- `GET /health/ready` - Readiness probe
- `GET /health/live` - Liveness probe

### Custom Health Checks

```go
server, err := grpcserver.New(config)
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

## Metrics

Built-in Prometheus metrics include:

### gRPC Metrics
- `grpc_server_grpc_requests_total` - Total gRPC requests by method and status
- `grpc_server_grpc_request_duration_seconds` - Request duration histogram
- `grpc_server_grpc_request_size_bytes` - Request payload size histogram
- `grpc_server_grpc_response_size_bytes` - Response payload size histogram
- `grpc_server_grpc_active_connections` - Active gRPC connections

### HTTP Metrics (Echo)
- `grpc_server_http_requests_total` - Total HTTP requests by method, path, and status
- `grpc_server_http_request_duration_seconds` - Request duration histogram
- `grpc_server_http_request_size_bytes` - Request payload size histogram
- `grpc_server_http_response_size_bytes` - Response payload size histogram
- `grpc_server_http_active_requests` - Active HTTP requests

### Server Metrics
- `grpc_server_uptime_seconds` - Server uptime in seconds
- `grpc_server_start_time_seconds` - Server start timestamp

Access metrics at `http://localhost:{port}/metrics`

## API Reference

### Quick Start Functions

#### `Start(port string, serviceRegistrar func(*grpc.Server)) error`
Starts a server in H2C mode with default configuration.

```go
grpcserver.Start("8080", func(s *grpc.Server) {
    // Register services
})
```

#### `StartH2C(port string, serviceRegistrar func(*grpc.Server)) error`
Explicitly starts a server in H2C mode.

```go
grpcserver.StartH2C("8080", serviceRegistrar)
```

#### `StartSeparate(grpcPort, httpPort string, serviceRegistrar func(*grpc.Server)) error`
Starts a server in separate mode with different ports for gRPC and HTTP.

```go
grpcserver.StartSeparate("9090", "9091", serviceRegistrar)
```

#### `StartWithConfig(config Config) error`
Starts a server with custom configuration.

```go
config := grpcserver.DefaultConfig()
// Configure...
grpcserver.StartWithConfig(config)
```

### Advanced Usage Functions

#### `New(config Config) (*Server, error)`
Creates a new server instance without starting it. Useful for advanced control and testing.

```go
server, err := grpcserver.New(config)
if err != nil {
    log.Fatal(err)
}

// Access managers before starting
healthManager := server.GetHealthManager()
metricsManager := server.GetMetricsManager()

// Start when ready
if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

#### `DefaultConfig() Config`
Returns a configuration with sensible defaults.

```go
config := grpcserver.DefaultConfig()
config.GRPCPort = "50051"
```

### Types

#### `Config`
Main configuration struct with all server options. See Configuration Options section above.

#### `Server`
Server instance with methods:
- `Start() error` - Start the server
- `Stop() error` - Gracefully stop the server
- `GetHealthManager() *HealthManager` - Get health check manager
- `GetMetricsManager() *MetricsManager` - Get metrics manager
- `GetGRPCServer() *grpc.Server` - Get underlying gRPC server
- `IsRunning() bool` - Check if server is running

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
                        - Metrics
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
                                       - Metrics
                                       - Custom Routes
```

## Examples

The `examples/` directory contains a complete calculator service demonstrating:

- **Unary RPC**: Basic request-response operations (Add, Subtract, Multiply, Divide)
- **Server Streaming**: Server sends multiple responses (Factorial)
- **Client Streaming**: Client sends multiple requests (Sum)
- **Bidirectional Streaming**: Both send multiple messages (RunningAverage)
- **Echo Integration**: Custom HTTP routes alongside gRPC
- **Full Observability**: Health checks and metrics

### Running the Example

```bash
# Navigate to examples directory
cd examples

# Run the server
go run -tags examples cmd/server/main.go

# In another terminal, run the client
go run -tags examples cmd/client/main.go

# Test HTTP endpoints
curl http://localhost:50051/status
curl http://localhost:50051/health
curl http://localhost:50051/metrics
curl http://localhost:50051/calculator
```

## Testing

The package includes comprehensive tests covering:
- Server lifecycle (start, stop, restart)
- Configuration validation
- Health check functionality
- Metrics collection
- H2C and Separate modes
- Graceful shutdown

Run tests:
```bash
go test ./...
```

## Best Practices

1. **Use H2C Mode for Development**: Simplifies local testing with a single port
2. **Use Separate Mode for Production**: Better isolation and flexibility
3. **Enable All Observability Features**: Metrics and health checks are essential for production
4. **Configure Timeouts**: Set appropriate timeouts based on your service requirements
5. **Use gRPC Reflection in Development**: Makes testing with tools like grpcurl easier
6. **Disable Reflection in Production**: Security best practice
7. **Add Custom Health Checks**: Monitor critical dependencies (database, cache, etc.)
8. **Use Echo Middleware**: Leverage Echo's rich middleware ecosystem

## Dependencies

- `google.golang.org/grpc` - gRPC framework
- `github.com/grpc-ecosystem/grpc-gateway/v2` - gRPC gateway
- `github.com/labstack/echo/v4` - Echo HTTP framework
- `github.com/prometheus/client_golang` - Prometheus metrics
- `golang.org/x/net` - HTTP/2 support

## License

This package is part of the jasoet/pkg project.