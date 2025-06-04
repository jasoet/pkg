# HTTP Server Package

A clean, production-ready HTTP server implementation using the Echo framework with built-in support for metrics, health checks, and graceful shutdown.

## Quick Start

Get your server up and running with minimal configuration:

```go
package main

import (
    "github.com/labstack/echo/v4"
    "your-module/server"
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

## Configuration Options

The server can be customized using the `Config` struct:

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| Port | int | The port number to listen on | - |
| Operation | func(e *echo.Echo) | Function to run when server starts | - |
| Shutdown | func(e *echo.Echo) | Function to run when server stops | - |
| Middleware | []echo.MiddlewareFunc | Custom middleware to apply | [] |
| EnableMetrics | bool | Enable Prometheus metrics | true |
| MetricsPath | string | Path for metrics endpoint | "/metrics" |
| MetricsSubsystem | string | Prometheus metrics subsystem name | "echo" |
| ShutdownTimeout | time.Duration | Timeout for graceful shutdown | 10s |
| EchoConfigurer | func(e *echo.Echo) | Function to configure Echo instance | nil |

Example with custom configuration:

```go
config := server.DefaultConfig(8080, operation, shutdown)
config.EnableMetrics = true
config.MetricsPath = "/custom-metrics"
config.ShutdownTimeout = 30 * time.Second
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
    "your-module/server"
)

func main() {
    // Define operation and shutdown functions
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
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
    "time"
    "your-module/server"
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
            log.Info().
                Str("path", c.Path()).
                Dur("duration", duration).
                Msg("Request processed")

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

## Metrics Integration

The server includes built-in Prometheus metrics at the `/metrics` endpoint.

### Default Metrics

- HTTP request count
- HTTP request duration
- HTTP request size
- HTTP response size

### Customizing Metrics

```go
package main

import (
    "github.com/labstack/echo-contrib/echoprometheus"
    "github.com/labstack/echo/v4"
    "github.com/prometheus/client_golang/prometheus"
    "your-module/server"
)

func main() {
    // Create custom metrics
    customCounter := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "custom_counter",
        Help: "A custom counter metric",
    })

    // Register the metric
    prometheus.MustRegister(customCounter)

    operation := func(e *echo.Echo) {
        e.GET("/increment", func(c echo.Context) error {
            // Increment the custom counter
            customCounter.Inc()
            return c.String(200, "Counter incremented")
        })
    }

    // Configure server with custom metrics path
    config := server.DefaultConfig(8080, operation, func(e *echo.Echo) {})
    config.MetricsPath = "/custom-metrics"

    server.StartWithConfig(config)
}
```

## Graceful Shutdown

The server supports graceful shutdown, allowing in-flight requests to complete before shutting down.

### Basic Shutdown Handler

```go
shutdown := func(e *echo.Echo) {
    // Close database connections
    db.Close()

    // Close message queue connections
    mq.Close()

    // Log shutdown
    log.Info().Msg("All resources have been properly released")
}
```

### Advanced Shutdown with Context

```go
package main

import (
    "context"
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
    "time"
    "your-module/server"
)

func main() {
    // Create resources with context
    ctx, cancel := context.WithCancel(context.Background())

    // Start background workers
    worker := startBackgroundWorker(ctx)

    shutdown := func(e *echo.Echo) {
        log.Info().Msg("Shutting down background workers...")

        // Signal workers to stop
        cancel()

        // Wait for worker to finish with timeout
        select {
        case <-worker.Done():
            log.Info().Msg("Worker shutdown completed")
        case <-time.After(5 * time.Second):
            log.Warn().Msg("Worker shutdown timed out")
        }

        log.Info().Msg("Shutdown complete")
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
    "your-module/server"
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

### Using with TLS

```go
package main

import (
    "github.com/labstack/echo/v4"
    "your-module/server"
)

func main() {
    operation := func(e *echo.Echo) {
        // Configure TLS
        e.TLSServer.TLSConfig = getTLSConfig()

        // Your routes here
        e.GET("/secure", func(c echo.Context) error {
            return c.String(200, "Secure endpoint")
        })
    }

    shutdown := func(e *echo.Echo) {
        // Cleanup resources
    }

    // Start the server
    server.Start(443, operation, shutdown)
}
```

### Custom Error Handling

#### Using EchoConfigurer (Recommended)

```go
package main

import (
    "github.com/labstack/echo/v4"
    "net/http"
    "time"
    "your-module/server"
)

func main() {
    // Define your custom error handler
    customErrorHandler := func(err error, c echo.Context) {
        code := http.StatusInternalServerError
        message := "Internal Server Error"

        if he, ok := err.(*echo.HTTPError); ok {
            code = he.Code
            message = he.Message.(string)
        }

        // Log the error
        c.Logger().Error(err)

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
