# HTTP Server Package (v2)

A clean, production-ready HTTP server implementation using the Echo framework with built-in support for OpenTelemetry, health checks, and graceful shutdown.

## 🚨 v2 Breaking Changes

**Version 2.x** introduces OpenTelemetry as the standard for observability, replacing Prometheus metrics.

### Removed from v1:
- `EnableMetrics` field
- `MetricsPath` field
- `MetricsSubsystem` field
- Prometheus integration
- `github.com/rs/zerolog` logging

### Added in v2:
- `OTelConfig *otel.Config` - OpenTelemetry configuration
- Support for traces, metrics, and logs via OpenTelemetry
- Default LoggerProvider (stdout) when using `otel.NewConfig()`
- Independent control of telemetry pillars
- Simple `fmt` logging for server lifecycle events

### Migration from v1 to v2

**Before (v1):**
```go
config := server.Config{
    Port: 8080,
    EnableMetrics: true,
    MetricsPath: "/metrics",
    MetricsSubsystem: "my_service",
}
server.StartWithConfig(config)
```

**After (v2):**
```go
// Option 1: Without telemetry
operation := func(e *echo.Echo) {
    // Your routes
}
shutdown := func(e *echo.Echo) {
    // Cleanup
}
config := server.DefaultConfig(8080, operation, shutdown)
server.StartWithConfig(config)

// Option 2: With OpenTelemetry logging (default)
otelCfg := otel.NewConfig("my-service")  // Logs to stdout by default
config.OTelConfig = otelCfg

// Option 3: With full OpenTelemetry (traces + metrics + logs)
otelCfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithServiceVersion("1.0.0")
config.OTelConfig = otelCfg
```

## Quick Start

Get your server up and running with minimal configuration:

```go
package main

import (
    "github.com/jasoet/pkg/server"
    "github.com/labstack/echo/v4"
)

func main() {
    // Define what happens when server starts
    operation := func(e *echo.Echo) {
        // Register your routes here
        e.GET("/hello", func(c echo.Context) error {
            return c.String(200, "Hello, World!")
        })
    }

    // Define cleanup actions when server stops
    shutdown := func(e *echo.Echo) {
        // Cleanup resources here
    }

    // Start server on port 8080
    server.Start(8080, operation, shutdown)
}
```

## OpenTelemetry Integration

Version 2 uses OpenTelemetry for all observability needs.

### Basic Usage (Logging Only)

```go
import (
    "github.com/jasoet/pkg/otel"
    "github.com/jasoet/pkg/server"
    "github.com/labstack/echo/v4"
)

func main() {
    // Create OTel config with default stdout logging
    otelCfg := otel.NewConfig("my-service").
        WithServiceVersion("1.0.0")

    operation := func(e *echo.Echo) {
        e.GET("/api/users", getUsersHandler)
    }

    shutdown := func(e *echo.Echo) {
        // Cleanup
    }

    config := server.DefaultConfig(8080, operation, shutdown)
    config.OTelConfig = otelCfg

    server.StartWithConfig(config)
}
```

### Full Telemetry (Traces + Metrics + Logs)

```go
import (
    "context"
    "github.com/jasoet/pkg/otel"
    "github.com/jasoet/pkg/server"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
    ctx := context.Background()

    // Setup TracerProvider
    traceExporter, _ := otlptracehttp.New(ctx)
    tracerProvider := trace.NewTracerProvider(
        trace.WithBatcher(traceExporter),
    )

    // Setup MeterProvider
    meterProvider := metric.NewMeterProvider()

    // Create OTel config with all pillars
    otelCfg := otel.NewConfig("my-service").
        WithTracerProvider(tracerProvider).
        WithMeterProvider(meterProvider).
        WithServiceVersion("1.0.0")

    operation := func(e *echo.Echo) {
        e.GET("/api/users", getUsersHandler)
    }

    shutdown := func(e *echo.Echo) {
        // Shutdown OTel providers
        otelCfg.Shutdown(ctx)
    }

    config := server.DefaultConfig(8080, operation, shutdown)
    config.OTelConfig = otelCfg

    server.StartWithConfig(config)
}
```

### Disabling Telemetry

```go
// Option 1: No OTelConfig (nil)
config := server.DefaultConfig(8080, operation, shutdown)
// config.OTelConfig is nil - no telemetry

// Option 2: Disable logging specifically
otelCfg := otel.NewConfig("my-service").
    WithoutLogging()  // Disables default stdout logging
config.OTelConfig = otelCfg
```

## Configuration Options

The server can be customized using the `Config` struct:

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| Port | int | The port number to listen on | - |
| Operation | func(e *echo.Echo) | Function to run when server starts | - |
| Shutdown | func(e *echo.Echo) | Function to run when server stops | - |
| Middleware | []echo.MiddlewareFunc | Custom middleware to apply | [] |
| OTelConfig | *otel.Config | OpenTelemetry configuration | nil |
| ShutdownTimeout | time.Duration | Timeout for graceful shutdown | 10s |
| EchoConfigurer | func(e *echo.Echo) | Function to configure Echo instance | nil |

Example with custom configuration:

```go
config := server.DefaultConfig(8080, operation, shutdown)
config.ShutdownTimeout = 30 * time.Second
config.OTelConfig = otel.NewConfig("my-service")
server.StartWithConfig(config)
```

### Using EchoConfigurer

The `EchoConfigurer` allows you to configure the Echo instance directly after it's created but before the server starts. This is useful for Echo-specific configurations like custom error handlers, validators, or other Echo settings.

```go
config := server.DefaultConfig(8080, operation, shutdown)

// Configure Echo instance
config.EchoConfigurer = func(e *echo.Echo) {
    // Custom error handler
    e.HTTPErrorHandler = myCustomErrorHandler

    // Custom validator
    e.Validator = myValidator

    // Other Echo-specific configurations
    e.Debug = true
}

server.StartWithConfig(config)
```

## Middleware Examples

### Adding Custom Middleware

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/jasoet/pkg/server"
)

func main() {
    operation := func(e *echo.Echo) {
        // Your routes here
    }

    shutdown := func(e *echo.Echo) {
        // Your cleanup here
    }

    // Add custom middleware
    corsMiddleware := middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins: []string{"https://example.com"},
        AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
    })

    rateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
        Skipper: middleware.DefaultSkipper,
        Store:   middleware.NewRateLimiterMemoryStore(20),
    })

    // Start server with middleware
    server.Start(8080, operation, shutdown, corsMiddleware, rateLimiter)
}
```

### Creating Your Own Middleware

```go
package main

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "github.com/jasoet/pkg/server"
    "time"
)

func main() {
    // Create custom timing middleware
    timingMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()

            // Execute the next handler
            err := next(c)

            // Log the time taken
            duration := time.Since(start)
            fmt.Printf("[%s] %s - %v\n",
                c.Request().Method,
                c.Path(),
                duration)

            return err
        }
    }

    // Start server with custom middleware
    server.Start(8080,
        func(e *echo.Echo) {},
        func(e *echo.Echo) {},
        timingMiddleware)
}
```

## Health Checks

The server includes built-in health check endpoints:

| Endpoint | Description | Response |
|----------|-------------|----------|
| `/health` | General health status | `{"status":"UP"}` |
| `/health/ready` | Readiness check | `{"status":"READY"}` |
| `/health/live` | Liveness check | `{"status":"ALIVE"}` |

### Customizing Health Checks

You can customize the health check endpoints in your operation function:

```go
operation := func(e *echo.Echo) {
    // Override the default health endpoint
    e.GET("/health", func(c echo.Context) error {
        // Check your application's health
        dbHealthy := checkDatabaseConnection()
        cacheHealthy := checkCacheConnection()

        if !dbHealthy || !cacheHealthy {
            return c.JSON(500, map[string]interface{}{
                "status": "DOWN",
                "components": map[string]string{
                    "database": dbHealthy ? "UP" : "DOWN",
                    "cache": cacheHealthy ? "UP" : "DOWN",
                },
            })
        }

        return c.JSON(200, map[string]interface{}{
            "status": "UP",
            "components": map[string]string{
                "database": "UP",
                "cache": "UP",
            },
        })
    })
}
```

## OpenTelemetry Metrics

When `OTelConfig` is provided with a `MeterProvider`, the server automatically collects HTTP metrics:

### Default Metrics

- `http.server.request.count` - Total number of HTTP requests
- `http.server.request.duration` - HTTP request duration (histogram)
- `http.server.active_requests` - Number of active HTTP requests (up/down counter)

All metrics include semantic convention attributes:
- `http.request.method` - HTTP method
- `http.route` - Route pattern
- `http.response.status_code` - HTTP status code

## Graceful Shutdown

The server supports graceful shutdown, allowing in-flight requests to complete before shutting down.

### Basic Shutdown Handler

```go
shutdown := func(e *echo.Echo) {
    // Close database connections
    db.Close()

    // Close message queue connections
    mq.Close()

    fmt.Println("All resources have been properly released")
}
```

### Advanced Shutdown with Context

```go
package main

import (
    "context"
    "fmt"
    "github.com/labstack/echo/v4"
    "github.com/jasoet/pkg/server"
    "time"
)

func main() {
    // Create resources with context
    ctx, cancel := context.WithCancel(context.Background())

    // Start background workers
    worker := startBackgroundWorker(ctx)

    shutdown := func(e *echo.Echo) {
        fmt.Println("Shutting down background workers...")

        // Signal workers to stop
        cancel()

        // Wait for worker to finish with timeout
        select {
        case <-worker.Done():
            fmt.Println("Worker shutdown completed")
        case <-time.After(5 * time.Second):
            fmt.Println("Worker shutdown timed out")
        }

        fmt.Println("Shutdown complete")
    }

    // Configure server with longer shutdown timeout
    config := server.DefaultConfig(8080, func(e *echo.Echo) {}, shutdown)
    config.ShutdownTimeout = 30 * time.Second

    server.StartWithConfig(config)
}
```

## Advanced Usage

### Integrating with Existing Applications

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/jasoet/pkg/server"
    "your-module/auth"
    "your-module/database"
)

func main() {
    // Initialize your application components
    db := database.New()
    authService := auth.New(db)

    // Create your API handlers
    userHandler := NewUserHandler(db, authService)
    productHandler := NewProductHandler(db)

    // Define server operation
    operation := func(e *echo.Echo) {
        // Group routes by API version
        v1 := e.Group("/api/v1")

        // User routes
        v1.POST("/users", userHandler.Create)
        v1.GET("/users/:id", userHandler.Get)
        v1.PUT("/users/:id", userHandler.Update)
        v1.DELETE("/users/:id", userHandler.Delete)

        // Product routes
        v1.GET("/products", productHandler.List)
        v1.GET("/products/:id", productHandler.Get)

        // Add authentication middleware to protected routes
        admin := v1.Group("/admin")
        admin.Use(authService.AdminMiddleware)
        admin.GET("/stats", productHandler.Stats)
    }

    // Define shutdown
    shutdown := func(e *echo.Echo) {
        db.Close()
    }

    // Start the server
    server.Start(8080, operation, shutdown)
}
```

### Custom Error Handling

#### Using EchoConfigurer (Recommended)

```go
package main

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "net/http"
    "time"
    "github.com/jasoet/pkg/server"
)

func main() {
    // Define your custom error handler
    customErrorHandler := func(err error, c echo.Context) {
        code := http.StatusInternalServerError
        message := "Internal Server Error"

        if he, ok := err.(*echo.HTTPError); ok {
            code = he.Code
            message = fmt.Sprintf("%v", he.Message)
        }

        // Log the error
        fmt.Printf("Error: %v\n", err)

        // Return a custom error response
        c.JSON(code, map[string]interface{}{
            "error": message,
            "status": code,
            "timestamp": time.Now().Format(time.RFC3339),
        })
    }

    // Define operation for business logic
    operation := func(e *echo.Echo) {
        // Register your routes here
        e.GET("/api/users", listUsers)
    }

    // Define shutdown function
    shutdown := func(e *echo.Echo) {
        // Cleanup resources
    }

    // Create config with EchoConfigurer
    config := server.DefaultConfig(8080, operation, shutdown)

    // Set Echo-specific configurations
    config.EchoConfigurer = func(e *echo.Echo) {
        // Set custom error handler
        e.HTTPErrorHandler = customErrorHandler

        // Other Echo configurations
        e.Debug = true
        e.Validator = myCustomValidator
    }

    // Start the server
    server.StartWithConfig(config)
}
```

## Examples

For complete, runnable examples, see the [examples directory](./examples/).

The examples demonstrate:
- Basic server setup
- OpenTelemetry configuration (traces, metrics, logs)
- Custom routes and middleware
- Health check implementations
- Graceful shutdown patterns

Run the examples:
```bash
cd examples
go run -tags example example.go
```

## Best Practices

### 1. Always Use Operation and Shutdown Functions

```go
// Good
operation := func(e *echo.Echo) {
    // Register routes
    e.GET("/users", getUsersHandler)
}

shutdown := func(e *echo.Echo) {
    // Cleanup resources
    db.Close()
}

server.Start(8080, operation, shutdown)
```

### 2. Enable OpenTelemetry for Production

```go
// Production configuration
otelCfg := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithServiceVersion(version)

config := server.DefaultConfig(8080, operation, shutdown)
config.OTelConfig = otelCfg
config.ShutdownTimeout = 30 * time.Second
server.StartWithConfig(config)
```

### 3. Use Middleware for Cross-Cutting Concerns

```go
// Apply middleware for auth, logging, rate limiting, etc.
authMiddleware := createAuthMiddleware()
rateLimiter := middleware.RateLimiterWithConfig(...)

server.Start(8080, operation, shutdown, authMiddleware, rateLimiter)
```

### 4. Implement Proper Health Checks

```go
operation := func(e *echo.Echo) {
    e.GET("/health", func(c echo.Context) error {
        // Check dependencies
        if !db.Ping() {
            return c.JSON(503, map[string]string{
                "status": "DOWN",
                "reason": "database unavailable",
            })
        }
        return c.JSON(200, map[string]string{"status": "UP"})
    })
}
```

## API Reference

### Functions

#### `Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc)`
Starts the HTTP server with simplified configuration.

#### `StartWithConfig(config Config)`
Starts the HTTP server with the given configuration.

#### `DefaultConfig(port int, operation Operation, shutdown Shutdown) Config`
Returns a default server configuration.

### Types

#### `Operation func(e *echo.Echo)`
Function to execute when server starts. Use this to register routes and handlers.

#### `Shutdown func(e *echo.Echo)`
Function to execute when server stops. Use this for cleanup.

#### `EchoConfigurer func(e *echo.Echo)`
Function to configure the Echo instance directly.

## Troubleshooting

### Server won't start
- Check if the port is already in use
- Verify Operation function doesn't have errors
- Check for panics in route handlers

### Telemetry not working
- Verify `OTelConfig` is not nil
- Check that providers are properly initialized
- Use `otel.NewConfig()` for default stdout logging

### Graceful shutdown timeout
- Increase `ShutdownTimeout` in config
- Check for long-running operations in handlers
- Ensure Shutdown function completes quickly

## License

This package is part of github.com/jasoet/pkg and follows the repository's license.
