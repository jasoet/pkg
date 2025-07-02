# Logging Package Examples

This directory contains examples demonstrating how to use the `logging` package for structured logging in Go applications.

## Overview

The `logging` package provides utilities for:
- Centralized logging setup with zerolog
- Context-aware logging with component identification
- Structured logging with consistent fields
- Debug and production logging configurations
- Integration with other packages in the library

## Running the Examples

To run the examples, use the following command from the `logging/examples` directory:

```bash
go run example.go
```

## Example Descriptions

The example.go file demonstrates several use cases:

### 1. Basic Logging Setup

Initialize the global logger for your application:

```go
// Initialize logging with service name and debug mode
logging.Initialize("my-service", true) // debug mode enabled

// Use the global logger directly
log.Info().Msg("Application started")
log.Debug().Str("version", "1.0.0").Msg("Debug information")
```

### 2. Context-Aware Logging

Create component-specific loggers with context:

```go
ctx := context.Background()
logger := logging.ContextLogger(ctx, "user-service")

logger.Info().Msg("User service started")
logger.Debug().Int("user_id", 123).Msg("Processing user")
```

### 3. Structured Logging

Log structured data with various field types:

```go
logger.Info().
    Str("method", "POST").
    Str("path", "/api/users").
    Int("status", 201).
    Dur("duration", 45*time.Millisecond).
    Msg("Request completed")
```

### 4. Different Log Levels

Use appropriate log levels for different scenarios:

```go
logger.Debug().Msg("Detailed debugging information")
logger.Info().Msg("General information")
logger.Warn().Msg("Warning: something might be wrong")
logger.Error().Err(err).Msg("An error occurred")
logger.Fatal().Msg("Fatal error - application will exit")
```

### 5. Error Logging

Properly log errors with context:

```go
if err := someOperation(); err != nil {
    logger.Error().
        Err(err).
        Str("operation", "database_query").
        Int("retry_count", 3).
        Msg("Operation failed after retries")
}
```

### 6. Performance Monitoring

Log performance metrics and timing:

```go
start := time.Now()
result, err := performOperation()
duration := time.Since(start)

logger.Info().
    Dur("duration", duration).
    Bool("success", err == nil).
    Int("result_count", len(result)).
    Msg("Operation completed")
```

### 7. HTTP Request Logging

Log HTTP requests with relevant details:

```go
func logRequest(logger zerolog.Logger, r *http.Request, status int, duration time.Duration) {
    logger.Info().
        Str("method", r.Method).
        Str("path", r.URL.Path).
        Str("remote_addr", r.RemoteAddr).
        Str("user_agent", r.UserAgent()).
        Int("status", status).
        Dur("duration", duration).
        Msg("HTTP request")
}
```

### 8. Database Operation Logging

Log database operations with context:

```go
func logDatabaseOperation(ctx context.Context, operation string, table string, duration time.Duration, err error) {
    logger := logging.ContextLogger(ctx, "database")
    
    event := logger.Info()
    if err != nil {
        event = logger.Error().Err(err)
    }
    
    event.
        Str("operation", operation).
        Str("table", table).
        Dur("duration", duration).
        Msg("Database operation")
}
```

## Configuration Options

### Log Levels

The package supports standard zerolog levels:

- **Debug**: Detailed information for debugging
- **Info**: General informational messages
- **Warn**: Warning messages for potential issues
- **Error**: Error messages for failures
- **Fatal**: Fatal errors that cause application exit

### Debug vs Production

```go
// Development mode (debug enabled)
logging.Initialize("my-service", true)

// Production mode (info level and above)
logging.Initialize("my-service", false)
```

### Logger Configuration

The global logger is configured with:
- **Console output**: Human-readable format for development
- **Timestamp**: RFC3339 format timestamps
- **Service name**: Consistent service identification
- **Process ID**: For multi-instance deployments
- **Caller information**: File and line number for debugging

## Field Types and Usage

### String Fields
```go
logger.Info().
    Str("user_id", "12345").
    Str("action", "login").
    Msg("User action")
```

### Numeric Fields
```go
logger.Info().
    Int("count", 42).
    Int64("timestamp", time.Now().Unix()).
    Float64("percentage", 95.5).
    Msg("Metrics")
```

### Boolean Fields
```go
logger.Info().
    Bool("success", true).
    Bool("cache_hit", false).
    Msg("Operation result")
```

### Duration Fields
```go
logger.Info().
    Dur("duration", 150*time.Millisecond).
    Dur("timeout", 30*time.Second).
    Msg("Timing information")
```

### Error Fields
```go
logger.Error().
    Err(err).
    Str("context", "user authentication").
    Msg("Authentication failed")
```

### Time Fields
```go
logger.Info().
    Time("started_at", startTime).
    Time("completed_at", time.Now()).
    Msg("Process timeline")
```

## Integration with Other Packages

### Database Package Integration

```go
import (
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
)

func setupDatabase(ctx context.Context) (*gorm.DB, error) {
    logger := logging.ContextLogger(ctx, "database-setup")
    
    config := &db.ConnectionConfig{
        DbType: db.Postgresql,
        Host:   "localhost",
        // ... other config
    }
    
    logger.Info().Str("db_type", string(config.DbType)).Msg("Connecting to database")
    
    database, err := config.Pool()
    if err != nil {
        logger.Error().Err(err).Msg("Database connection failed")
        return nil, err
    }
    
    logger.Info().Msg("Database connection successful")
    return database, nil
}
```

### REST Package Integration

```go
import (
    "github.com/jasoet/pkg/rest"
    "github.com/jasoet/pkg/logging"
)

func makeAPICall(ctx context.Context) {
    logger := logging.ContextLogger(ctx, "api-client")
    
    client, err := rest.NewClient(&rest.Config{
        BaseURL: "https://api.example.com",
    })
    
    logger.Info().Str("base_url", "https://api.example.com").Msg("Making API call")
    
    // Make request with logging
    response, err := client.Get("/users")
    if err != nil {
        logger.Error().Err(err).Msg("API call failed")
        return
    }
    
    logger.Info().Int("status", response.StatusCode).Msg("API call successful")
}
```

### Server Package Integration

```go
import (
    "github.com/jasoet/pkg/server"
    "github.com/jasoet/pkg/logging"
)

func startServer(ctx context.Context) {
    logger := logging.ContextLogger(ctx, "http-server")
    
    config := &server.Config{
        Port: 8080,
        // ... other config
    }
    
    logger.Info().Int("port", config.Port).Msg("Starting HTTP server")
    
    srv := server.New(config)
    // Server automatically includes logging middleware
}
```

## Best Practices

### 1. Initialize Once

```go
func main() {
    // Initialize logging at application startup
    logging.Initialize("my-service", os.Getenv("DEBUG") == "true")
    
    // Rest of application...
}
```

### 2. Use Context Loggers

```go
// Create component-specific loggers
func UserService(ctx context.Context) {
    logger := logging.ContextLogger(ctx, "user-service")
    
    // Use logger throughout the component
    logger.Info().Msg("User service operation")
}
```

### 3. Consistent Field Names

```go
// Use consistent field names across your application
logger.Info().
    Str("user_id", userID).        // Always use "user_id"
    Str("request_id", requestID).  // Always use "request_id"
    Dur("duration", duration).     // Always use "duration"
    Msg("Operation completed")
```

### 4. Meaningful Messages

```go
// Good: Descriptive message with context
logger.Info().
    Str("operation", "user_creation").
    Str("user_id", "12345").
    Msg("User created successfully")

// Avoid: Generic messages without context
logger.Info().Msg("Success")
```

### 5. Error Context

```go
// Provide context with errors
logger.Error().
    Err(err).
    Str("function", "CreateUser").
    Str("input", userInput).
    Msg("Failed to create user")
```

### 6. Performance Considerations

```go
// Use conditional logging for expensive operations
if logger.Debug().Enabled() {
    expensiveDebugData := generateDebugData()
    logger.Debug().
        Interface("debug_data", expensiveDebugData).
        Msg("Debug information")
}
```

## Log Analysis and Monitoring

### JSON Output for Production

For production environments, you might want JSON output:

```go
// Custom logger setup for production
func setupProductionLogger(serviceName string) {
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    log.Logger = zerolog.New(os.Stdout). // JSON output to stdout
        With().
        Timestamp().
        Str("service", serviceName).
        Int("pid", os.Getpid()).
        Logger()
}
```

### Structured Query Examples

With structured logging, you can easily query logs:

```bash
# Find all errors from a specific component
jq 'select(.level == "error" and .component == "database")' logs.json

# Find slow operations
jq 'select(.duration_ms > 1000)' logs.json

# Count requests by status code
jq -r '.status' logs.json | sort | uniq -c
```

## Testing with Logging

### Test Logger Setup

```go
func TestWithLogging(t *testing.T) {
    // Setup test logger
    logging.Initialize("test-service", true)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "test")
    
    // Your test code with logging
    logger.Info().Str("test", t.Name()).Msg("Running test")
}
```

### Capturing Logs in Tests

```go
func TestLogOutput(t *testing.T) {
    var buf bytes.Buffer
    
    // Create logger that writes to buffer
    testLogger := zerolog.New(&buf).With().Timestamp().Logger()
    
    testLogger.Info().Str("test", "example").Msg("Test message")
    
    // Verify log output
    output := buf.String()
    assert.Contains(t, output, "Test message")
    assert.Contains(t, output, "test")
}
```

## Common Patterns

### Request ID Tracking

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := generateRequestID()
    ctx := context.WithValue(r.Context(), "request_id", requestID)
    
    logger := logging.ContextLogger(ctx, "api")
    logger.Info().
        Str("request_id", requestID).
        Str("method", r.Method).
        Str("path", r.URL.Path).
        Msg("Request started")
    
    // Handle request...
    
    logger.Info().
        Str("request_id", requestID).
        Msg("Request completed")
}
```

### Service Boundaries

```go
func CallExternalService(ctx context.Context, serviceURL string) error {
    logger := logging.ContextLogger(ctx, "external-service")
    
    logger.Info().
        Str("service_url", serviceURL).
        Msg("Calling external service")
    
    start := time.Now()
    
    // Make call...
    
    logger.Info().
        Str("service_url", serviceURL).
        Dur("duration", time.Since(start)).
        Msg("External service call completed")
    
    return nil
}
```

## Troubleshooting

### Common Issues

1. **No Log Output**: Ensure `Initialize()` is called before using loggers
2. **Wrong Log Level**: Check debug parameter in `Initialize()`
3. **Missing Context**: Use `ContextLogger()` for component-specific logging
4. **Performance Impact**: Use conditional logging for expensive debug operations

### Debug Tips

- Use debug mode during development: `logging.Initialize("service", true)`
- Add request IDs for tracing requests across services
- Include timing information for performance analysis
- Use structured fields for easier log analysis