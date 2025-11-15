# Retry Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/retry.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/retry)

Production-ready retry mechanism with exponential backoff using `cenkalti/backoff/v4`. This package provides a clean, reusable API for retrying operations without manual retry logic implementation.

## Features

- **Exponential Backoff**: Configurable backoff strategy with jitter
- **Context Support**: Respects context cancellation and timeouts
- **OpenTelemetry Integration**: Automatic tracing and logging
- **Permanent Errors**: Stop retrying for non-transient errors
- **Flexible Configuration**: Fluent API with sensible defaults
- **Type-Safe**: Uses generics for type-safe operation definitions

## Installation

```bash
go get github.com/jasoet/pkg/v2/retry
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/jasoet/pkg/v2/retry"
)

func main() {
    ctx := context.Background()

    // Default configuration (5 retries, 500ms initial interval, 2x multiplier)
    cfg := retry.DefaultConfig().
        WithName("database.connect").
        WithMaxRetries(3)

    err := retry.Do(ctx, cfg, func(ctx context.Context) error {
        // Your operation that might fail
        return connectToDatabase()
    })

    if err != nil {
        fmt.Printf("Failed after retries: %v\n", err)
    }
}
```

### With OpenTelemetry

```go
import (
    "github.com/jasoet/pkg/v2/retry"
    "github.com/jasoet/pkg/v2/otel"
)

func main() {
    ctx := context.Background()

    // Setup OTel
    otelConfig := otel.NewConfig("my-service").
        WithTracerProvider(tracerProvider).
        WithMeterProvider(meterProvider)

    // Retry with OTel instrumentation
    cfg := retry.DefaultConfig().
        WithName("api.fetch").
        WithMaxRetries(5).
        WithInitialInterval(1 * time.Second).
        WithOTel(otelConfig)

    err := retry.Do(ctx, cfg, func(ctx context.Context) error {
        return fetchFromAPI(ctx)
    })
}
```

### Custom Backoff Strategy

```go
cfg := retry.DefaultConfig().
    WithName("s3.upload").
    WithMaxRetries(10).
    WithInitialInterval(100 * time.Millisecond).
    WithMaxInterval(30 * time.Second).
    WithMultiplier(1.5)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return uploadToS3(data)
})
```

### Permanent Errors (No Retry)

```go
import "github.com/jasoet/pkg/v2/retry"

func validateAndProcess(data string) error {
    if len(data) == 0 {
        // This error should not be retried
        return retry.Permanent(fmt.Errorf("invalid data: empty string"))
    }

    // This error will be retried
    return processData(data)
}

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return validateAndProcess(data)
})
```

### With Custom Notifications

```go
err := retry.DoWithNotify(ctx, cfg,
    func(ctx context.Context) error {
        return riskyOperation()
    },
    func(err error, backoff time.Duration) {
        log.Printf("Retry after %v: %v", backoff, err)
        // Send metrics, alerts, etc.
    },
)
```

### Unlimited Retries (Use with Timeout)

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

cfg := retry.DefaultConfig().
    WithName("poll.status").
    WithMaxRetries(0). // Unlimited retries
    WithInitialInterval(1 * time.Second).
    WithMaxInterval(10 * time.Second)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    status, err := checkJobStatus()
    if err != nil {
        return err
    }
    if status != "completed" {
        return fmt.Errorf("job not ready: %s", status)
    }
    return nil
})
```

## Configuration

### Config Fields

```go
type Config struct {
    // MaxRetries is the maximum number of retry attempts (0 means unlimited)
    // Default: 5
    MaxRetries uint64

    // InitialInterval is the initial retry interval
    // Default: 500ms
    InitialInterval time.Duration

    // MaxInterval caps the maximum retry interval
    // Default: 60s
    MaxInterval time.Duration

    // Multiplier is the exponential backoff multiplier
    // Default: 2.0 (each retry waits 2x longer)
    Multiplier float64

    // OperationName is used for logging and tracing
    // Default: "retry.operation"
    OperationName string

    // OTelConfig enables OpenTelemetry tracing and logging
    // Optional: if nil, no OTel instrumentation
    OTelConfig *otel.Config
}
```

### Fluent API Methods

- `WithName(name string)` - Set operation name for logging/tracing
- `WithMaxRetries(n uint64)` - Set maximum retry attempts (0 = unlimited)
- `WithInitialInterval(d time.Duration)` - Set initial retry interval
- `WithMaxInterval(d time.Duration)` - Set maximum retry interval cap
- `WithMultiplier(m float64)` - Set exponential backoff multiplier
- `WithOTel(cfg *otel.Config)` - Enable OpenTelemetry instrumentation

## How It Works

1. **Exponential Backoff**: Each retry waits longer than the previous one
   - Interval = InitialInterval × Multiplier^(attempt-1)
   - Capped at MaxInterval

2. **Jitter**: Built-in randomization to prevent thundering herd

3. **Context Awareness**:
   - Respects context cancellation
   - Respects context deadlines/timeouts
   - Returns context error when cancelled

4. **Permanent Errors**:
   - Wrap errors with `retry.Permanent()` to stop retrying
   - Useful for validation errors, 4xx HTTP errors, etc.

## Examples

See [examples/retry](../examples/retry/) for complete working examples:

- Basic retry with defaults
- Custom backoff configuration
- OpenTelemetry integration
- Permanent error handling
- Context cancellation
- Unlimited retries with timeout

## Best Practices

1. **Set Appropriate MaxRetries**: Don't retry forever, use context timeout for long-running operations

2. **Use Permanent Errors**: Mark non-transient errors as permanent to avoid unnecessary retries

3. **Configure Backoff Based on Service**:
   - Fast operations: 100ms initial, 1.5x multiplier
   - Network calls: 500ms-1s initial, 2x multiplier
   - Heavy operations: 1s+ initial, 2-3x multiplier

4. **Add Context Timeout**: Always use context with timeout to prevent infinite waiting

5. **Monitor with OTel**: Enable OpenTelemetry for production visibility

## Common Use Cases

### Database Connection

```go
cfg := retry.DefaultConfig().
    WithName("database.connect").
    WithMaxRetries(5).
    WithInitialInterval(500 * time.Millisecond)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return db.Ping()
})
```

### HTTP API Call

```go
cfg := retry.DefaultConfig().
    WithName("api.call").
    WithMaxRetries(3).
    WithInitialInterval(1 * time.Second)

var response *http.Response
err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    if resp.StatusCode >= 500 {
        resp.Body.Close()
        return fmt.Errorf("server error: %d", resp.StatusCode)
    }
    if resp.StatusCode >= 400 {
        resp.Body.Close()
        return retry.Permanent(fmt.Errorf("client error: %d", resp.StatusCode))
    }
    response = resp
    return nil
})
```

### File Upload with S3

```go
cfg := retry.DefaultConfig().
    WithName("s3.upload").
    WithMaxRetries(10).
    WithInitialInterval(200 * time.Millisecond).
    WithMaxInterval(30 * time.Second)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return s3Client.Upload(ctx, bucket, key, data)
})
```

## Testing

The package includes comprehensive tests covering:
- Success scenarios (first attempt, after retries)
- Failure scenarios (all retries exhausted)
- Context cancellation and timeout
- Exponential backoff behavior
- Permanent errors
- Unlimited retries
- Custom notifications

Run tests:

```bash
go test -v -race ./retry
```

## Performance Considerations

- **Low Overhead**: Minimal allocations, uses `cenkalti/backoff` efficiently
- **Context-Aware**: Respects cancellation immediately
- **No Goroutine Leaks**: Properly cleans up on context cancellation

## Related Packages

- [cenkalti/backoff](https://github.com/cenkalti/backoff) - Underlying backoff implementation
- [otel](../otel/) - OpenTelemetry integration for observability

## License

MIT License - see [LICENSE](../LICENSE) for details
