# Logging Package

Structured logging with zerolog, supporting flexible output destinations (console, file, or both).

## Features

- **Flexible Output**: Console, file, or both simultaneously
- **Structured Logging**: JSON format for files, human-readable for console
- **Multiple Log Levels**: Debug, Info, Warn, Error
- **Component Loggers**: Create loggers for specific components
- **Context Support**: Pass context values to loggers
- **Zero Dependencies**: Only stdlib + zerolog
- **OS-Managed Rotation**: Use logrotate or similar tools for file rotation

## Quick Start

### Console Only (Default)

```go
import (
    "github.com/jasoet/pkg/v2/logging"
    "github.com/rs/zerolog/log"
)

func main() {
    // Initialize with console output
    logging.Initialize("my-service", true) // debug=true
    
    // Use global logger
    log.Info().Msg("Service started")
    log.Debug().Str("config", "loaded").Msg("Configuration loaded")
}
```

### File Only

```go
import "github.com/jasoet/pkg/v2/logging"

func main() {
    // All logs go to file (no console output)
    logging.InitializeWithFile("my-service", false,
        logging.OutputFile,
        &logging.FileConfig{
            Path: "/var/log/myapp/app.log",
        })
    
    log.Info().Msg("This goes to file only")
}
```

### Both Console and File

```go
import "github.com/jasoet/pkg/v2/logging"

func main() {
    // Logs appear in both console and file
    logging.InitializeWithFile("my-service", true,
        logging.OutputConsole | logging.OutputFile, // Bitwise OR
        &logging.FileConfig{
            Path: "/var/log/myapp/app.log",
        })
    
    log.Info().Msg("Visible in console AND file")
}
```

## API Reference

### Initialize

```go
func Initialize(serviceName string, debug bool)
```

Sets up console-only logging (backward compatible).

**Parameters:**
- `serviceName`: Service name added to all logs
- `debug`: If true, sets level to Debug; otherwise Info

**Example:**
```go
logging.Initialize("my-service", true)
```

### InitializeWithFile

```go
func InitializeWithFile(serviceName string, debug bool, output OutputDestination, fileConfig *FileConfig)
```

Sets up logging with flexible output destinations.

**Parameters:**
- `serviceName`: Service name added to all logs
- `debug`: If true, sets level to Debug; otherwise Info
- `output`: Output destination flags (OutputConsole, OutputFile, or both)
- `fileConfig`: File configuration (required if OutputFile specified)

**Output Formats:**
- **Console**: Human-readable, colored (via `zerolog.ConsoleWriter`)
- **File**: JSON format for parsing and log aggregation

**Examples:**
```go
// Console only
logging.InitializeWithFile("service", true, logging.OutputConsole, nil)

// File only
logging.InitializeWithFile("service", false, 
    logging.OutputFile,
    &logging.FileConfig{Path: "app.log"})

// Both
logging.InitializeWithFile("service", true,
    logging.OutputConsole | logging.OutputFile,
    &logging.FileConfig{Path: "app.log"})
```

### ContextLogger

```go
func ContextLogger(ctx context.Context, component string) zerolog.Logger
```

Creates a component-specific logger with context values.

**Parameters:**
- `ctx`: Context (values will be added to logger)
- `component`: Component name

**Returns:** `zerolog.Logger` with component field

**Example:**
```go
logger := logging.ContextLogger(ctx, "user-service")
logger.Info().Str("user_id", "123").Msg("User created")
```

### OutputDestination

```go
type OutputDestination int

const (
    OutputConsole OutputDestination = 1 << 0  // Console (stderr)
    OutputFile    OutputDestination = 1 << 1  // File
)
```

Bitwise flags for output destinations. Combine with `|` operator:
```go
logging.OutputConsole | logging.OutputFile  // Both outputs
```

### FileConfig

```go
type FileConfig struct {
    Path string  // Log file path (required)
}
```

Configuration for file-based logging. File rotation should be managed by OS tools (logrotate, etc.).

## Output Formats

### Console Output

Human-readable with colors and timestamps:
```
2025-11-24T12:30:45+07:00 INF Service started service=my-service pid=12345
2025-11-24T12:30:46+07:00 DBG Configuration loaded config=loaded service=my-service pid=12345
```

### File Output

Structured JSON for parsing:
```json
{"level":"info","service":"my-service","pid":12345,"time":"2025-11-24T12:30:45+07:00","message":"Service started"}
{"level":"debug","service":"my-service","pid":12345,"config":"loaded","time":"2025-11-24T12:30:46+07:00","message":"Configuration loaded"}
```

## Usage Patterns

### Environment-Based Configuration

```go
import (
    "os"
    "github.com/jasoet/pkg/v2/logging"
)

func main() {
    env := os.Getenv("ENV")
    
    if env == "production" {
        // Production: file only, info level
        logging.InitializeWithFile("my-service", false,
            logging.OutputFile,
            &logging.FileConfig{Path: "/var/log/myapp/app.log"})
    } else if env == "staging" {
        // Staging: both console and file, debug level
        logging.InitializeWithFile("my-service", true,
            logging.OutputConsole | logging.OutputFile,
            &logging.FileConfig{Path: "/var/log/myapp/app.log"})
    } else {
        // Development: console only, debug level
        logging.Initialize("my-service", true)
    }
}
```

### Component-Specific Logging

```go
func ProcessOrder(ctx context.Context, orderID string) {
    logger := logging.ContextLogger(ctx, "order-processor")
    
    logger.Info().Str("order_id", orderID).Msg("Processing order")
    
    // ... process order ...
    
    logger.Info().
        Str("order_id", orderID).
        Str("status", "completed").
        Msg("Order processed")
}
```

### Structured Logging

```go
log.Info().
    Str("user_id", "123").
    Int("age", 30).
    Bool("premium", true).
    Dur("response_time", 150*time.Millisecond).
    Msg("User action completed")

// File output:
// {"level":"info","user_id":"123","age":30,"premium":true,"response_time":150,...}
```

### Error Logging

```go
if err != nil {
    log.Error().
        Err(err).
        Str("operation", "database_query").
        Msg("Database operation failed")
    return err
}
```

## File Rotation with logrotate

Since the package doesn't handle file rotation internally, use OS tools like `logrotate`:

### logrotate Configuration

Create `/etc/logrotate.d/myapp`:

```
/var/log/myapp/*.log {
    daily                    # Rotate daily
    rotate 7                 # Keep 7 days of logs
    compress                 # Compress old logs
    delaycompress            # Compress after 2nd rotation
    missingok                # Don't error if log missing
    notifempty               # Don't rotate empty logs
    create 0644 myapp myapp  # Create new file with permissions
    postrotate
        # Send SIGHUP to app to reopen log files (if needed)
        killall -SIGHUP myapp || true
    endscript
}
```

### Testing logrotate

```bash
# Test configuration
logrotate -d /etc/logrotate.d/myapp

# Force rotation
logrotate -f /etc/logrotate.d/myapp
```

## Log Levels

Use appropriate log levels:

```go
// Debug: Detailed information for debugging
log.Debug().Msg("Entering function ProcessUser")

// Info: General informational messages
log.Info().Msg("Service started successfully")

// Warn: Warning messages (not critical)
log.Warn().Msg("Cache miss, fetching from database")

// Error: Error conditions
log.Error().Err(err).Msg("Failed to connect to database")

// Fatal: Critical errors (exits with os.Exit(1))
log.Fatal().Msg("Unable to start server")

// Panic: Panic-level errors
log.Panic().Msg("Unrecoverable error")
```

## Best Practices

### 1. Initialize Once at Startup

```go
func main() {
    // Initialize logging first
    logging.InitializeWithFile("my-service", true,
        logging.OutputConsole | logging.OutputFile,
        &logging.FileConfig{Path: "app.log"})
    
    // Then start your application
    startServer()
}
```

### 2. Use Component Loggers

```go
// Create component-specific loggers
func NewUserService(ctx context.Context) *UserService {
    return &UserService{
        logger: logging.ContextLogger(ctx, "user-service"),
    }
}

func (s *UserService) CreateUser(user User) {
    s.logger.Info().Str("user_id", user.ID).Msg("Creating user")
}
```

### 3. Add Context to Logs

```go
log.Info().
    Str("request_id", requestID).
    Str("user_id", userID).
    Dur("latency", latency).
    Msg("Request processed")
```

### 4. Don't Log Sensitive Data

```go
// Bad
log.Info().Str("password", user.Password).Msg("User login")

// Good
log.Info().Str("user_id", user.ID).Msg("User login")
```

### 5. Use Structured Fields

```go
// Good: Structured and parseable
log.Info().
    Str("user_id", "123").
    Int("order_count", 5).
    Msg("User activity")

// Bad: Unstructured
log.Info().Msg("User 123 has 5 orders")
```

## Migration from v1

No changes needed! The `Initialize()` function remains backward compatible:

**v1 code:**
```go
logging.Initialize("my-service", true)
log.Info().Msg("Hello")
```

**Still works in v2!** To add file logging:

```go
logging.InitializeWithFile("my-service", true,
    logging.OutputConsole | logging.OutputFile,
    &logging.FileConfig{Path: "app.log"})
log.Info().Msg("Hello")
```

## Testing

When writing tests, you can redirect logs to a test file:

```go
func TestMyFunction(t *testing.T) {
    tempDir := t.TempDir()
    logFile := filepath.Join(tempDir, "test.log")
    
    logging.InitializeWithFile("test-service", true,
        logging.OutputFile,
        &logging.FileConfig{Path: logFile})
    
    // Run your test
    MyFunction()
    
    // Verify logs
    content, _ := os.ReadFile(logFile)
    assert.Contains(t, string(content), "expected log message")
}
```

## OpenTelemetry Integration

For OpenTelemetry-compatible logging with trace correlation, see the `otel` package:

```go
import "github.com/jasoet/pkg/v2/otel"

// Create OTel LoggerProvider
loggerProvider, _ := otel.NewLoggerProviderWithOptions("my-service",
    otel.WithLogLevel(logging.LogLevelInfo),
    otel.WithConsoleOutput(true))

cfg := &otel.Config{
    LoggerProvider: loggerProvider,
    // ... other OTel config
}
```

See [`otel/README.md`](../otel/README.md) for details.

## Troubleshooting

### Logs not appearing in file

1. Check file path exists and is writable
2. Verify OutputFile flag is set
3. Check FileConfig.Path is not empty
4. Verify file permissions (should be 0644)

### File grows indefinitely

1. Set up logrotate (see above)
2. Verify logrotate cron job is running
3. Check logrotate configuration syntax

### Cannot read log files

JSON logs can be pretty-printed:

```bash
# Pretty-print JSON logs
cat app.log | jq

# Filter by level
cat app.log | jq 'select(.level=="error")'

# Search for specific message
cat app.log | jq 'select(.message | contains("database"))'
```

### Performance concerns

- Console output is slower (formatting overhead)
- File output is fast (direct JSON write)
- For production: use file only (`OutputFile`)
- For development: use console or both

## License

Part of github.com/jasoet/pkg/v2 - follows repository license.
