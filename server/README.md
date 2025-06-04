# Server Package

This package provides a simple HTTP server implementation using the Echo framework with support for metrics, health checks, and graceful shutdown.

## Key Features

- HTTP server with configurable port
- Prometheus metrics integration
- Health check endpoints
- Graceful shutdown with configurable timeout
- Custom middleware support
- Operation and shutdown hooks

## Architecture Changes

The server implementation has been refactored to improve testability while maintaining production functionality:

1. **Separation of Concerns**: Signal handling is now separated from the server lifecycle
2. **Server Struct**: Introduced a `Server` struct that encapsulates the Echo instance and configuration
3. **Programmatic Control**: Added methods to programmatically start and stop the server
4. **Backward Compatibility**: Maintained the original API for production use

## Testing

The server package includes comprehensive unit tests that cover:

1. **Server Initialization**: Tests that the server is properly initialized with the given configuration
2. **Server Lifecycle**: Tests that the server can be started and stopped programmatically
3. **Health Check Endpoints**: Tests that the health check endpoints return the expected responses
4. **Metrics Functionality**: Tests that Prometheus metrics are properly configured and accessible
5. **Operation Execution**: Tests that the operation function is executed when the server starts
6. **Shutdown Execution**: Tests that the shutdown function is executed when the server stops
7. **Custom Middleware**: Tests that custom middleware is properly applied
8. **Integration**: Tests the entire server lifecycle in an integrated manner

## Usage

### Basic Usage

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
    "your-module/server"
)

func main() {
    // Define operation to run when server starts
    operation := func(e *echo.Echo) {
        log.Info().Msg("Server started, performing initialization...")
        // Add your initialization code here
    }

    // Define shutdown function to run when server stops
    shutdown := func(e *echo.Echo) {
        log.Info().Msg("Server stopping, performing cleanup...")
        // Add your cleanup code here
    }

    // Start the server on port 8080
    server.Start(8080, operation, shutdown)
}
```

### Advanced Usage with Custom Configuration

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
    "time"
    "your-module/server"
)

func main() {
    // Define operation and shutdown functions
    operation := func(e *echo.Echo) {
        log.Info().Msg("Server started, performing initialization...")
    }

    shutdown := func(e *echo.Echo) {
        log.Info().Msg("Server stopping, performing cleanup...")
    }

    // Create custom configuration
    config := server.DefaultConfig(8080, operation, shutdown)
    config.EnableMetrics = true
    config.MetricsPath = "/custom-metrics"
    config.ShutdownTimeout = 30 * time.Second

    // Add custom middleware
    config.Middleware = []echo.MiddlewareFunc{
        // Your custom middleware here
    }

    // Start the server with custom configuration
    server.StartWithConfig(config)
}
```

### Programmatic Control (for Testing)

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
    "time"
    "your-module/server"
)

func main() {
    // Define operation and shutdown functions
    operation := func(e *echo.Echo) {
        log.Info().Msg("Server started, performing initialization...")
    }

    shutdown := func(e *echo.Echo) {
        log.Info().Msg("Server stopping, performing cleanup...")
    }

    // Create server instance
    config := server.DefaultConfig(8080, operation, shutdown)
    srv := server.NewServer(config)

    // Start the server
    srv.Start()

    // Do something while the server is running
    time.Sleep(10 * time.Second)

    // Stop the server
    err := srv.Stop()
    if err != nil {
        log.Error().Err(err).Msg("Failed to stop server")
    }
}
```