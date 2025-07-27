# Server Package Examples

This directory contains runnable examples demonstrating the features of the `server` package.

## 📍 Example Code Location

**Full example implementation:** [/server/examples/example.go](https://github.com/jasoet/pkg/blob/main/server/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/server"

// Option 1: Quick start with defaults
server.Start() // Starts on port 8080

// Option 2: With custom port
server.StartWithPort(3000)

// Option 3: Full configuration
config := &server.Config{
    Port: 8080,
    EchoConfigurer: func(e *echo.Echo) {
        // Add custom routes
        e.GET("/api/users", getUsersHandler)
        e.POST("/api/users", createUserHandler)
    },
    HealthChecker: func() error {
        // Custom health check logic
        return checkDatabaseConnection()
    },
}
server.StartWithConfig(*config)
```

**Built-in endpoints:**
- `/health` - Main health check
- `/health/ready` - Readiness probe
- `/health/live` - Liveness probe  
- `/metrics` - Prometheus metrics

## Overview

The examples in this directory complement the comprehensive documentation in the main server package README. These are practical, runnable demonstrations of:

- Basic server setup
- Custom configuration options
- Custom routes and middleware
- Health check implementations
- Graceful shutdown patterns

## Running the Examples

To run the interactive examples:

```bash
go run example.go
```

This will run through 5 different server examples in sequence, each demonstrating different aspects of the server package.

## Examples Included

### 1. Basic Server Setup
- Default configuration
- Built-in health endpoints (`/health`, `/health/ready`, `/health/live`)
- Built-in metrics endpoint (`/metrics`)
- Standard middleware (logging, CORS, recovery)

### 2. Custom Configuration
- Custom timeouts and ports
- Custom endpoint paths
- Advanced server configuration options

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

## Integration with Other Packages

The examples demonstrate integration with:
- **logging package**: Structured logging throughout
- **Echo framework**: Custom routes and middleware
- **Prometheus**: Metrics collection

## Key Features Demonstrated

- **Zero-configuration startup**: Minimal code to get a production-ready server
- **Flexible configuration**: Extensive customization options
- **Built-in observability**: Health checks, metrics, and logging
- **Production patterns**: Graceful shutdown, error handling, security middleware
- **Extensibility**: Custom routes, middleware, and health checks

## Related Documentation

For comprehensive documentation, configuration options, and additional examples, see the main server package README at `../README.md`.

The server package README includes:
- Complete configuration reference
- Advanced middleware examples
- Health check patterns
- Metrics customization
- Production deployment guides
- Integration examples with databases and external services