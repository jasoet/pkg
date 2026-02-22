# HTTP Server Package (v2)

A clean, production-ready HTTP server implementation using the Echo framework with built-in health checks and graceful shutdown.

> **Note:** `Start` and `StartWithConfig` return `error` instead of calling `os.Exit(1)`.
> Callers must handle the returned error. See examples below.

## Quick Start

Get your server up and running with minimal configuration:

```go
package main

import (
    "github.com/jasoet/pkg/v2/server"
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
    if err := server.Start(8080, operation, shutdown); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
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
| ShutdownTimeout | time.Duration | Timeout for graceful shutdown | 10s |
| EchoConfigurer | func(e *echo.Echo) | Function to configure Echo instance | nil |

Example with custom configuration:

```go
config := server.DefaultConfig(8080, operation, shutdown)
config.ShutdownTimeout = 30 * time.Second
if err := server.StartWithConfig(config); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
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

if err := server.StartWithConfig(config); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

## Middleware Examples

### Adding Custom Middleware

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/jasoet/pkg/v2/server"
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
    if err := server.Start(8080, operation, shutdown, corsMiddleware, rateLimiter); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

### Creating Your Own Middleware

```go
package main

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "github.com/jasoet/pkg/v2/server"
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
    if err := server.Start(8080,
        func(e *echo.Echo) {},
        func(e *echo.Echo) {},
        timingMiddleware); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
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
    "github.com/jasoet/pkg/v2/server"
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

    if err := server.StartWithConfig(config); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

## Advanced Usage

### Integrating with Existing Applications

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/jasoet/pkg/v2/server"
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
    if err := server.Start(8080, operation, shutdown); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
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
    "github.com/jasoet/pkg/v2/server"
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
    if err := server.StartWithConfig(config); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

## Examples

For complete, runnable examples, see the [examples directory](../examples/server/).

The examples demonstrate:
- Basic server setup
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

if err := server.Start(8080, operation, shutdown); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

### 2. Use Middleware for Cross-Cutting Concerns

```go
// Apply middleware for auth, logging, rate limiting, etc.
authMiddleware := createAuthMiddleware()
rateLimiter := middleware.RateLimiterWithConfig(...)

if err := server.Start(8080, operation, shutdown, authMiddleware, rateLimiter); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

### 3. Implement Proper Health Checks

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

#### `Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) error`
Starts the HTTP server with simplified configuration. Returns an error if the server fails to start or shut down.

#### `StartWithConfig(config Config) error`
Starts the HTTP server with the given configuration. Returns an error if the server fails to start or shut down.

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

### Graceful shutdown timeout
- Increase `ShutdownTimeout` in config
- Check for long-running operations in handlers
- Ensure Shutdown function completes quickly

## License

This package is part of github.com/jasoet/pkg and follows the repository's license.
