# gRPC Server Package

A production-ready, reusable gRPC server and HTTP gateway component for Go applications. This package provides a clean, configuration-driven API for setting up gRPC servers with HTTP/REST gateway support, built-in observability, health checks, and graceful shutdown capabilities.

## Features

- **Dual Protocol Support**: Run gRPC and HTTP services on the same port (H2C) or separate ports
- **Zero Configuration**: Works out-of-the-box with sensible defaults
- **Production Ready**: Built-in metrics, health checks, and graceful shutdown
- **Configurable**: Extensive configuration options for production deployments
- **Observability**: Prometheus metrics and structured health checks
- **Easy Integration**: Clean API that works with any gRPC service implementation

## Installation

```bash
go get github.com/jasoet/grpc-learn/pkg/grpc
```

## Quick Start

### Basic Usage (H2C Mode)

```go
package main

import (
    "log"

    taskv1 "github.com/jasoet/grpc-learn/gen/task/v1"
    "github.com/jasoet/grpc-learn/internal/repository"
    "github.com/jasoet/grpc-learn/internal/service"
    grpcserver "github.com/jasoet/grpc-learn/pkg/grpc"
    "google.golang.org/grpc"
)

func main() {
    // Create service dependencies
    taskRepo := repository.NewTaskRepository()
    taskSvc := service.NewTaskService(taskRepo)

    // Define service registrar
    serviceRegistrar := func(s *grpc.Server) {
        taskv1.RegisterTaskServiceServer(s, taskSvc)
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
- HTTP gateway: `http://localhost:8080/`
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

```go
package main

import (
    "log"
    "time"
    "net/http"

    grpcserver "github.com/jasoet/grpc-learn/pkg/grpc"
    "google.golang.org/grpc"
)

func main() {
    // Create advanced configuration
    config := grpcserver.Config{
        // Server Configuration
        GRPCPort: "9090",
        HTTPPort: "9091",
        Mode:     grpcserver.SeparateMode,

        // Timeouts
        ShutdownTimeout:       45 * time.Second,
        ReadTimeout:           10 * time.Second,
        WriteTimeout:          15 * time.Second,
        IdleTimeout:           120 * time.Second,
        MaxConnectionIdle:     30 * time.Minute,
        MaxConnectionAge:      60 * time.Minute,
        MaxConnectionAgeGrace: 10 * time.Second,

        // Production Features
        EnableMetrics:     true,
        MetricsPath:       "/metrics",
        EnableHealthCheck: true,
        HealthPath:        "/health",
        EnableLogging:     true,
        EnableReflection:  true,

        // Service Registration
        ServiceRegistrar: func(s *grpc.Server) {
            // Register your gRPC services here
        },

        // Customization Hooks
        GRPCConfigurer: func(s *grpc.Server) {
            log.Println("Applying custom gRPC configuration...")
            // Add interceptors, custom options, etc.
        },

        HTTPConfigurer: func(mux *http.ServeMux) {
            log.Println("Adding custom HTTP endpoints...")
            mux.HandleFunc("/version", versionHandler)
        },

        // Custom Shutdown
        Shutdown: func() error {
            log.Println("Running custom cleanup...")
            // Close connections, cleanup resources
            return nil
        },
    }

    if err := grpcserver.StartWithConfig(config); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

## Configuration Options

### Core Settings
- `GRPCPort`: Port for gRPC server (required)
- `HTTPPort`: Port for HTTP gateway (required for separate mode)
- `Mode`: Server mode (`H2CMode` or `SeparateMode`)

### Timeouts
- `ShutdownTimeout`: Graceful shutdown timeout (default: 30s)
- `ReadTimeout`: HTTP read timeout (default: 10s)
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
- `EnableReflection`: Enable gRPC reflection (default: false)

### Customization Hooks
- `ServiceRegistrar`: Function to register gRPC services
- `GRPCConfigurer`: Function to customize gRPC server
- `HTTPConfigurer`: Function to add custom HTTP handlers
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
```

## Metrics

Built-in Prometheus metrics include:

### gRPC Metrics
- `grpc_server_grpc_requests_total` - Total gRPC requests
- `grpc_server_grpc_request_duration_seconds` - Request duration
- `grpc_server_grpc_request_size_bytes` - Request payload size
- `grpc_server_grpc_response_size_bytes` - Response payload size
- `grpc_server_grpc_active_connections` - Active connections

### HTTP Metrics
- `grpc_server_http_requests_total` - Total HTTP requests
- `grpc_server_http_request_duration_seconds` - Request duration
- `grpc_server_http_request_size_bytes` - Request payload size
- `grpc_server_http_response_size_bytes` - Response payload size
- `grpc_server_http_active_requests` - Active requests

### Server Metrics
- `grpc_server_uptime_seconds` - Server uptime
- `grpc_server_start_time_seconds` - Server start timestamp

Access metrics at `http://localhost:{port}/metrics`

## API Reference

### Functions

#### `Start(port string, serviceRegistrar func(*grpc.Server)) error`
Starts a server in H2C mode with default configuration.

#### `StartH2C(port string, serviceRegistrar func(*grpc.Server)) error`
Explicitly starts a server in H2C mode.

#### `StartSeparate(grpcPort, httpPort string, serviceRegistrar func(*grpc.Server)) error`
Starts a server in separate mode with different ports for gRPC and HTTP.

#### `StartWithConfig(config Config) error`
Starts a server with custom configuration.

#### `New(config Config) (*Server, error)`
Creates a new server instance without starting it.

### Types

#### `Config`
Main configuration struct with all server options.

#### `Server`
Server instance with methods:
- `Start() error` - Start the server
- `Stop() error` - Gracefully stop the server
- `GetHealthManager() *HealthManager` - Get health check manager
- `GetMetricsManager() *MetricsManager` - Get metrics manager
- `IsRunning() bool` - Check if server is running

#### `ServerMode`
Server mode enumeration:
- `H2CMode` - Single port HTTP/2 cleartext mode
- `SeparateMode` - Separate ports for gRPC and HTTP

## Examples

See the `examples/` directory for complete usage examples:

- [`basic.go`](examples/basic.go) - Simple H2C setup
- [`advanced.go`](examples/advanced.go) - Production configuration with all features

## License

This package is part of the grpc-learn project.