# Retry Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/retry.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/retry)

Production-ready retry mechanism with exponential backoff using `cenkalti/backoff/v4`. This package provides a clean, reusable API for retrying operations without manual retry logic implementation.

## Features

- **Exponential Backoff**: Configurable backoff strategy with jitter
- **Context Support**: Respects context cancellation and timeouts
- **OpenTelemetry Integration**: Automatic tracing and logging
- **Permanent Errors**: Stop retrying for non-transient errors
- **Functional Options**: Sensible defaults via `DefaultConfig`, overridden with `retry.New(...)` options
- **No Panics**: Invalid configuration is reported as an error by `Do`/`DoWithNotify` before the first attempt

## Installation

```bash
go get github.com/jasoet/pkg/v3/retry
```

## Quick Start

### Configure with functional options

`retry.New` starts from `DefaultConfig()` and applies each option in order; fields you don't set keep their defaults.

```go
cfg := retry.New(
    retry.WithName("db.connect"),
    retry.WithMaxRetries(3),
    retry.WithInitialInterval(100*time.Millisecond),
)
```

Backed by [`ExampleNew`](./example_test.go).

### Retry an operation

`Do` calls the operation, and retries it with exponential backoff while it returns an error. With `MaxRetries = N` the operation runs at most N+1 times (1 initial attempt + up to N retries). Here the operation fails twice and succeeds on the third attempt:

```go
cfg := retry.New(
    retry.WithName("flaky.op"),
    retry.WithMaxRetries(5),
    retry.WithInitialInterval(time.Millisecond),
)

attempts := 0
err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    attempts++
    if attempts < 3 {
        return errors.New("temporary failure")
    }
    return nil
})
// attempts == 3, err == nil
```

Backed by [`ExampleDo`](./example_test.go).

### Permanent errors (no retry)

Wrap an error with `retry.Permanent` to stop retrying immediately — useful for validation errors, 4xx HTTP responses, and other non-transient failures. `Do` returns after the first attempt; the returned error wraps the original one, so `errors.Is`/`errors.As` still match it.

```go
err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return retry.Permanent(errors.New("invalid input"))
})
// 1 attempt; err == "validate.input failed after 1 attempts (1 initial + 0 retries): invalid input"
```

Backed by [`ExamplePermanent`](./example_test.go).

### Custom notifications

`DoWithNotify` calls a notify function before each retry wait, with the error and the upcoming backoff duration — handy for custom logging or metrics:

```go
err := retry.DoWithNotify(ctx, cfg,
    func(ctx context.Context) error {
        return riskyOperation()
    },
    func(err error, backoff time.Duration) {
        log.Printf("retrying in %v: %v", backoff, err)
    },
)
```

Backed by [`ExampleDoWithNotify`](./example_test.go).

### Unlimited retries (use with a timeout)

`retry.WithMaxRetries(0)` means unlimited retries — the loop ends only when the operation succeeds or the context is done. Always combine it with `context.WithTimeout` (or a deadline) so the loop terminates.

## Configuration

### Config fields

All fields are exported and carry `yaml`/`mapstructure` tags (camelCase), so a `Config` can also be loaded from a config file.

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `MaxRetries` | `uint64` | `5` | Retries after the initial attempt; `0` means unlimited |
| `InitialInterval` | `time.Duration` | `500ms` | Wait before the first retry; must be > 0 |
| `MaxInterval` | `time.Duration` | `60s` | Cap for the backoff interval; must be >= `InitialInterval` |
| `Multiplier` | `float64` | `2.0` | Backoff growth per retry; must be > 1 |
| `RandomizationFactor` | `float64` | `0.5` | Jitter factor in `[0, 1]`; `0` disables jitter, `0.5` means +/-50% |
| `Name` | `string` | `"retry.operation"` | Operation name used in log messages, error messages, and OTel spans |
| `OTelConfig` | `*otel.Config` | `nil` | OpenTelemetry instrumentation; `nil` disables it. Not serializable (`yaml:"-"`) |

### Options

- `WithName(name string)` — operation name for logging/tracing
- `WithMaxRetries(n uint64)` — max retries after the initial attempt (0 = unlimited)
- `WithInitialInterval(d time.Duration)` — initial retry interval
- `WithMaxInterval(d time.Duration)` — retry interval cap
- `WithMultiplier(m float64)` — exponential backoff multiplier
- `WithRandomizationFactor(f float64)` — jitter factor in `[0, 1]`
- `WithOTelConfig(cfg *otel.Config)` — OpenTelemetry instrumentation

### Validation (no panics)

Options never panic. `Do` and `DoWithNotify` validate the config before the first attempt and return a descriptive error if any rule is violated — the operation is never called with an invalid config:

- `Multiplier` must be > 1
- `InitialInterval` must be > 0
- `MaxInterval` must be >= `InitialInterval`
- `RandomizationFactor` must be in `[0, 1]`

## How It Works

1. **Exponential backoff**: each retry waits `InitialInterval × Multiplier^(retry-1)`, capped at `MaxInterval`. There is no overall time limit (`MaxElapsedTime` is disabled); termination is governed by `MaxRetries` and context.
2. **Jitter**: `RandomizationFactor` spreads intervals to prevent a thundering herd.
3. **Context awareness**: cancellation and deadlines stop the retry loop immediately; the returned error wraps `ctx.Err()`.
4. **Permanent errors**: `retry.Permanent(err)` short-circuits the retry loop on the current attempt.

## Examples

See [examples/retry](../examples/retry/) for a runnable program covering basic retry, custom backoff, permanent errors, context cancellation, unlimited retries with timeout, and custom notifications.

## Best Practices

1. **Set appropriate MaxRetries**: don't retry forever; use a context timeout for long-running polls.
2. **Use permanent errors**: mark non-transient errors with `retry.Permanent` to avoid pointless retries.
3. **Size the backoff for the dependency**: fast in-process operations ~100ms initial / 1.5x multiplier; network calls 500ms–1s / 2x; heavy operations 1s+ / 2–3x.
4. **Keep jitter on** in production (`RandomizationFactor > 0`) so synchronized clients don't stampede a recovering service.
5. **Name your operations**: `WithName` shows up in error messages and OTel spans — invaluable when several retried operations interleave.
6. **Monitor with OTel**: pass an `*otel.Config` via `WithOTelConfig` for tracing and structured logs.

## Testing

```bash
go test -v -race ./retry
```

The suite covers success/failure paths, backoff timing, context cancellation, permanent errors, unlimited retries, notifications, and Do-time config validation, plus compile-checked example tests in [`example_test.go`](./example_test.go).

## Related Packages

- [cenkalti/backoff](https://github.com/cenkalti/backoff) — underlying backoff implementation
- [otel](../otel/) — OpenTelemetry integration for observability

## License

MIT License - see [LICENSE](../LICENSE) for details
