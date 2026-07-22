# HTTP Server Package (v3)

A clean, production-ready HTTP server implementation using the Echo framework with built-in health checks, graceful shutdown, and optional OpenTelemetry instrumentation.

> **Note:** `srv.Start()` blocks until `srv.Shutdown(ctx)` is called (or serving fails),
> returning `nil` on a clean shutdown. Signal handling is up to the caller. See examples below.

## Quick Start

Get your server up and running with minimal configuration:

```go
package main

import (
    "github.com/jasoet/pkg/v3/server"
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

    // Create the server, then start it (blocks until Shutdown is called)
    srv, err := server.New(
        server.WithPort(8080),
        server.WithOperation(operation),
        server.WithShutdown(shutdown),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }
    if err := srv.Start(); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

## Configuration Options

The server is configured with functional options, which populate a `Config`:

| Field | Option | Type | Description | Default |
|-------|--------|------|-------------|---------|
| Port | `WithPort` | int | The port number to listen on (`0` = OS-assigned ephemeral port) | 0 |
| Operation | `WithOperation` | func(e *echo.Echo) | Runs after Echo is configured, before listening | nil |
| Shutdown | `WithShutdown` | func(e *echo.Echo) | Runs during graceful shutdown, before Echo drains | nil |
| Middleware | `WithMiddleware` | ...echo.MiddlewareFunc | Custom middleware to apply | none |
| ShutdownTimeout | `WithShutdownTimeout` | time.Duration | Deadline for graceful shutdown | 10s |
| EchoConfigurer | `WithEchoConfigurer` | func(e *echo.Echo) | Customizes the Echo instance during setup | nil |
| OTelConfig | `WithOTelConfig` | *otel.Config | OpenTelemetry configuration (see below) | nil |

Example with custom configuration:

```go
srv, err := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithShutdownTimeout(30*time.Second),
)
if err != nil {
    log.Fatal().Err(err).Msg("invalid server config")
}
if err := srv.Start(); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

### Using EchoConfigurer

The `EchoConfigurer` allows you to configure the Echo instance directly after it's created but before the server starts. This is useful for Echo-specific configurations like custom error handlers, validators, or other Echo settings.

```go
srv, err := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),

    // Configure Echo instance
    server.WithEchoConfigurer(func(e *echo.Echo) {
        // Custom error handler
        e.HTTPErrorHandler = myCustomErrorHandler

        // Custom validator
        e.Validator = myValidator

        // Other Echo-specific configurations
        e.Debug = true
    }),
)
if err != nil {
    log.Fatal().Err(err).Msg("invalid server config")
}
if err := srv.Start(); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

## OpenTelemetry Instrumentation

Pass an `*otel.Config` via `WithOTelConfig` and the server auto-installs request instrumentation middleware (before your own middleware). All instrumentation uses the scope name `http.server`.

### Tracing (when tracing is enabled on the config)

One server span per request, named `{method} {route}` (e.g. `GET /users/:id`), with attributes:

- `http.request.method`
- `url.full`
- `http.response.status_code`
- `http.route`

### Metrics (when metrics is enabled on the config)

- `http.server.request.count` — counter of total HTTP requests, unit `{request}`
- `http.server.request.duration` — histogram of request duration, unit `ms`

Both are attributed by `http.request.method` and `http.response.status_code`.

```go
import (
    "github.com/jasoet/pkg/v3/otel"
    "github.com/jasoet/pkg/v3/server"
)

otelCfg := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider),
)

srv, err := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithOTelConfig(otelCfg),
)
```

With no `OTelConfig` (the default), no spans or metrics are emitted.

## Middleware Examples

### Adding Custom Middleware

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/jasoet/pkg/v3/server"
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
    srv, err := server.New(
        server.WithPort(8080),
        server.WithOperation(operation),
        server.WithShutdown(shutdown),
        server.WithMiddleware(corsMiddleware, rateLimiter),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }
    if err := srv.Start(); err != nil {
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
    "github.com/jasoet/pkg/v3/server"
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
    srv, err := server.New(
        server.WithPort(8080),
        server.WithMiddleware(timingMiddleware),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }
    if err := srv.Start(); err != nil {
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

> **Note:** Health routes are registered **after** user middleware, so any middleware you add via
> `WithMiddleware` (including auth) also applies to them. If you need unauthenticated Kubernetes
> probes, don't register global auth middleware, or exempt the health paths in your middleware
> (e.g. with a skipper).

### Customizing Health Checks

You can replace the health check endpoints in your operation function:

```go
operation := func(e *echo.Echo) {
    // Override the default health endpoint
    e.GET("/health", func(c echo.Context) error {
        // Check your application's health
        dbStatus := "UP"
        if !checkDatabaseConnection() {
            dbStatus = "DOWN"
        }
        cacheStatus := "UP"
        if !checkCacheConnection() {
            cacheStatus = "DOWN"
        }

        if dbStatus != "UP" || cacheStatus != "UP" {
            return c.JSON(500, map[string]interface{}{
                "status": "DOWN",
                "components": map[string]string{
                    "database": dbStatus,
                    "cache":    cacheStatus,
                },
            })
        }

        return c.JSON(200, map[string]interface{}{
            "status": "UP",
            "components": map[string]string{
                "database": "UP",
                "cache":    "UP",
            },
        })
    })
}
```

## Graceful Shutdown

The server supports graceful shutdown, allowing in-flight requests to complete before shutting down. Call `Shutdown(ctx)` from another goroutine — for example from your own signal handler — and `Start` returns `nil` once draining completes.

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
    "github.com/jasoet/pkg/v3/server"
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
    srv, err := server.New(
        server.WithPort(8080),
        server.WithShutdown(shutdown),
        server.WithShutdownTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }

    // Trigger shutdown however you like; Shutdown(ctx) drains in-flight
    // requests within ShutdownTimeout.
    go func() {
        <-someShutdownSignal
        _ = srv.Shutdown(context.Background())
    }()

    if err := srv.Start(); err != nil {
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
    "github.com/jasoet/pkg/v3/server"
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

    // Start the server (blocks until Shutdown is called)
    srv, err := server.New(
        server.WithPort(8080),
        server.WithOperation(operation),
        server.WithShutdown(shutdown),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }
    if err := srv.Start(); err != nil {
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
    "github.com/jasoet/pkg/v3/server"
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

        // Log the error (do not leak internal details to the client)
        fmt.Printf("Error: %v\n", err)

        // Sanitize 5xx responses to avoid leaking internal error details
        if code >= http.StatusInternalServerError {
            _ = c.JSON(code, map[string]string{"error": "internal server error"})
            return
        }

        // Return a custom error response
        _ = c.JSON(code, map[string]interface{}{
            "error":     message,
            "status":    code,
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

    // Create the server with EchoConfigurer
    srv, err := server.New(
        server.WithPort(8080),
        server.WithOperation(operation),
        server.WithShutdown(shutdown),

        // Set Echo-specific configurations
        server.WithEchoConfigurer(func(e *echo.Echo) {
            // Set custom error handler
            e.HTTPErrorHandler = customErrorHandler

            // Other Echo configurations
            e.Debug = true
            e.Validator = myCustomValidator
        }),
    )
    if err != nil {
        log.Fatal().Err(err).Msg("invalid server config")
    }

    // Start the server
    if err := srv.Start(); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

## Examples

For complete, runnable examples, see the [examples/server directory](../examples/server/).

The examples demonstrate:
- Basic server setup
- Custom routes and middleware
- Health check implementations
- Graceful shutdown patterns

Run the examples from the repository root:
```bash
go run -tags=example ./examples/server
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

srv, err := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
)
if err != nil {
    log.Fatal().Err(err).Msg("invalid server config")
}
if err := srv.Start(); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

### 2. Use Middleware for Cross-Cutting Concerns

```go
// Apply middleware for auth, logging, rate limiting, etc.
authMiddleware := createAuthMiddleware()
rateLimiter := middleware.RateLimiterWithConfig(...)

srv, err := server.New(
    server.WithPort(8080),
    server.WithOperation(operation),
    server.WithShutdown(shutdown),
    server.WithMiddleware(authMiddleware, rateLimiter),
)
if err != nil {
    log.Fatal().Err(err).Msg("invalid server config")
}
if err := srv.Start(); err != nil {
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

#### `New(opts ...Option) (*Server, error)`
Creates a server from functional options (`WithPort`, `WithOperation`, `WithShutdown`, `WithMiddleware`, `WithShutdownTimeout`, `WithEchoConfigurer`, `WithOTelConfig`). Validates the configuration (port must be 0-65535) and prepares the Echo instance without binding or serving.

#### `NewConfig(opts ...Option) Config`
Builds a `Config` from functional options with sensible defaults (10s shutdown timeout).

### Methods

#### `(s *Server) Start() error`
Runs the `Operation` callback first, then binds the listener and serves, blocking until `Shutdown` is called or serving fails. Because `Operation` runs before binding, `Addr()` returns `""` inside `Operation` — with `Port: 0` the OS-assigned port is only known after binding. Returns `nil` on a clean shutdown (`http.ErrServerClosed` is filtered). Calling `Start` while already running returns an error. A stopped `Server` cannot be restarted — `Start` returns an error; create a new one with `New`.

#### `(s *Server) Shutdown(ctx context.Context) error`
Invokes the `Shutdown` callback and drains the Echo server, honoring `ShutdownTimeout` on top of the caller's context (whichever deadline is earlier). Idempotent: the callback and drain run exactly once.

#### `(s *Server) Addr() string`
Returns the bound listener address (e.g. `[::]:8080`), or `""` before the server is listening (and after shutdown). This is how callers discover the OS-assigned port when using `Port: 0`.

#### `(s *Server) Echo() *echo.Echo`
Returns the underlying Echo instance for route registration or customization before `Start`.

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
- Increase `ShutdownTimeout` via `WithShutdownTimeout`
- Check for long-running operations in handlers
- Ensure the Shutdown function completes quickly

## License

This package is part of github.com/jasoet/pkg/v3 and follows the repository's license.
