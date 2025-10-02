# Logging Package

OpenTelemetry LoggerProvider implementation with zerolog as the backend. This package provides automatic log-span correlation for observability platforms like Grafana.

## Features

- **OTel LoggerProvider**: Implements `log.LoggerProvider` interface
- **Zerolog Backend**: Beautiful console logging with structured data
- **Automatic Trace Correlation**: Logs automatically include `trace_id` and `span_id`
- **Grafana Integration**: Click spans to see related logs
- **Backward Compatible**: Supports legacy `Initialize()` and `ContextLogger()` functions

## How Log-Span Correlation Works

When you click a span in Grafana and see related logs, it's because both share the same `trace_id` and `span_id`. Here's how our implementation achieves this:

### The Mechanism

```
1. Tracer creates span → Context contains trace_id + span_id
2. Logger emits log   → Extracts trace_id + span_id from context
3. Backend stores     → Links logs to spans via matching IDs
4. Grafana displays   → Shows logs when you click a span
```

### Implementation

The `zerologLogger.Emit()` function automatically extracts trace context:

```go
func (l *zerologLogger) Emit(ctx context.Context, record log.Record) {
    // ... create zerolog event ...

    // Extract trace context from the context parameter
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.IsValid() {
        event = event.
            Str("trace_id", spanCtx.TraceID().String()).  // Links to trace
            Str("span_id", spanCtx.SpanID().String())     // Links to span
    }

    // ... emit log ...
}
```

**Key point**: The `ctx context.Context` parameter passed to `Emit()` contains the active span. The logger extracts the trace ID and span ID from it and adds them as log fields.

## Usage

### Basic Setup (Legacy)

```go
import "github.com/jasoet/pkg/v2/logging"

// Initialize global logger
logging.Initialize("my-service", true)

// Use global logger
log.Info().Msg("Service started")

// Create component logger
logger := logging.ContextLogger(ctx, "user-service")
logger.Info().Str("user_id", "123").Msg("User created")
```

### OTel LoggerProvider (Recommended for v2)

```go
import (
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/otel"
)

// Create LoggerProvider with zerolog backend
loggerProvider := logging.NewLoggerProvider("my-service", false)

// Use with otel.Config
cfg := &otel.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    LoggerProvider: loggerProvider,
    TracerProvider: yourTracerProvider,  // Your tracer setup
    MeterProvider:  yourMeterProvider,   // Your meter setup
}

// Get a logger
logger := cfg.GetLogger("business-logic")

// Emit logs with trace context
ctx, span := tracer.Start(ctx, "ProcessOrder")
defer span.End()

// This log will automatically include trace_id and span_id!
var record log.Record
// ... configure record ...
logger.Emit(ctx, record)
```

### Server Integration Example

```go
import (
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/server"
)

func main() {
    // Setup OTel with zerolog backend
    loggerProvider := logging.NewLoggerProvider("api-server", false)

    otelCfg := &otel.Config{
        ServiceName:    "api-server",
        LoggerProvider: loggerProvider,
        TracerProvider: setupTracer(),   // Your tracer
        MeterProvider:  setupMetrics(),  // Your metrics
    }

    // Configure server with OTel
    operation := func(e *echo.Echo) {
        e.GET("/users/:id", getUserHandler)
    }

    serverCfg := server.DefaultConfig(8080, operation, func(e *echo.Echo) {})
    serverCfg.OTelConfig = otelCfg

    // Start server
    // All HTTP requests will be traced AND logged
    // Logs will include trace_id and span_id automatically!
    server.StartWithConfig(serverCfg)
}
```

## Log Output Format

With trace context, your logs will look like:

```
{"level":"info","service":"my-service","scope":"business-logic","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","trace_flags":"01","message":"Processing user request"}
```

In Grafana:
1. Find trace `4bf92f3577b34da6a3ce929d0e0e4736`
2. Click span `00f067aa0ba902b7`
3. See this log (and all others with same span_id)

## API Reference

### NewLoggerProvider

```go
func NewLoggerProvider(serviceName string, debug bool) log.LoggerProvider
```

Creates an OpenTelemetry LoggerProvider backed by zerolog.

**Parameters:**
- `serviceName`: Service name added to all logs
- `debug`: If true, sets log level to Debug; otherwise Info

**Returns:** A `log.LoggerProvider` that implements OTel logging interface

**Features:**
- Automatic trace context extraction
- Pretty console output with colors
- Structured logging with timestamps
- PID and caller information

### Initialize (Legacy)

```go
func Initialize(serviceName string, debug bool)
```

Sets up the global zerolog logger. For backward compatibility with v1.

### ContextLogger (Legacy)

```go
func ContextLogger(ctx context.Context, component string) zerolog.Logger
```

Creates a component-specific logger from the global logger.

## Best Practices

### 1. Use OTel LoggerProvider for New Code

```go
// Good (v2)
loggerProvider := logging.NewLoggerProvider("service", false)
cfg := &otel.Config{LoggerProvider: loggerProvider}

// Old (v1 - still works but no trace correlation)
logging.Initialize("service", false)
```

### 2. Always Pass Context to Logging

```go
// Good - trace context will be extracted
func processRequest(ctx context.Context) {
    logger.Emit(ctx, record)
}

// Bad - no trace context, logs won't link to spans
func processRequest() {
    logger.Emit(context.Background(), record)
}
```

### 3. Use Spans for Important Operations

```go
func handleUserRequest(ctx context.Context) {
    // Create span
    ctx, span := tracer.Start(ctx, "HandleUserRequest")
    defer span.End()

    // All logs within this function will share this span's trace_id and span_id
    logger.Emit(ctx, logRecord("User request started"))

    // Child operations create child spans
    processUser(ctx)  // Logs will have same trace_id, different span_id
}
```

### 4. Integration with Server Package

When using the server package with OTelConfig, HTTP request logging automatically includes trace context:

```go
serverCfg.OTelConfig = &otel.Config{
    LoggerProvider: logging.NewLoggerProvider("api", false),
    TracerProvider: tracerProvider,
}

// Every HTTP request will:
// 1. Create a span (via TracerProvider)
// 2. Log the request (via LoggerProvider)
// 3. Automatically link them (same trace_id and span_id)
```

## Migration from v1

**v1 (zerolog only):**
```go
logging.Initialize("my-service", true)
logger := logging.ContextLogger(ctx, "component")
logger.Info().Msg("message")
```

**v2 (OTel with trace correlation):**
```go
loggerProvider := logging.NewLoggerProvider("my-service", true)
cfg := &otel.Config{LoggerProvider: loggerProvider}
logger := cfg.GetLogger("component")

var record log.Record
// configure record...
logger.Emit(ctx, record)  // Includes trace_id and span_id!
```

## Technical Details

### Trace Context Extraction

The implementation uses `trace.SpanContextFromContext(ctx)` to extract:
- **trace_id**: Unique ID for the entire request trace
- **span_id**: Unique ID for this specific operation
- **trace_flags**: Sampling decision (01 = sampled, 00 = not sampled)

### Severity Mapping

OTel severity levels map to zerolog levels:
- `SeverityFatal` → `zerolog.Fatal()`
- `SeverityError` → `zerolog.Error()`
- `SeverityWarn` → `zerolog.Warn()`
- `SeverityInfo` → `zerolog.Info()`
- `SeverityDebug` → `zerolog.Debug()`
- Default → `zerolog.Trace()`

### Attribute Conversion

All OTel log attributes are converted to zerolog fields:
- `KindBool` → `event.Bool(key, val)`
- `KindInt64` → `event.Int64(key, val)`
- `KindFloat64` → `event.Float64(key, val)`
- `KindString` → `event.Str(key, val)`
- `KindBytes` → `event.Bytes(key, val)`
- `KindSlice`, `KindMap` → `event.Interface(key, val)`

## Troubleshooting

### Logs not appearing in Grafana

1. Check if TracerProvider is configured
2. Verify spans are being created
3. Ensure context is passed to `Emit()`
4. Check if logs have `trace_id` and `span_id` fields

### Logs appear but not linked to spans

1. Verify trace_id and span_id match between logs and spans
2. Check Grafana datasource configuration
3. Ensure logs and traces are sent to the same backend
4. Verify field names (should be `trace_id` and `span_id`)

### Performance concerns

- The LoggerProvider uses zerolog's efficient field system
- Trace context extraction is a simple map lookup (very fast)
- No-op if span context is invalid (zero overhead)

## License

Part of github.com/jasoet/pkg - follows repository license.
