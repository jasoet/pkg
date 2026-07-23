# Concurrent Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/concurrent.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/concurrent)

Type-safe fan-out execution of named functions with generics, error propagation, and automatic cancellation.

## Features

- **Type-safe generics**: compile-time type safety via `Func[T]`
- **Fail-fast**: the first error or panic cancels the shared context, signaling the remaining functions to stop
- **Panic recovery**: a panicking function is recovered and reported as an error instead of crashing the process
- **Causal error priority**: the first causal (non-context) error is returned, not the resulting `context.Canceled`
- **Typed results**: `ExecuteConcurrentlyTyped` builds a single typed value from the result map
- **Standard library only** (testify for tests)

## Installation

```bash
go get github.com/jasoet/pkg/v3/concurrent
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/jasoet/pkg/v3/concurrent"
)

func main() {
    ctx := context.Background()

    funcs := map[string]concurrent.Func[string]{
        "user": func(ctx context.Context) (string, error) {
            return "John Doe", nil
        },
        "email": func(ctx context.Context) (string, error) {
            return "john@example.com", nil
        },
    }

    results, err := concurrent.ExecuteConcurrently(ctx, funcs)
    if err != nil {
        panic(err)
    }

    fmt.Println(results["user"])  // "John Doe"
    fmt.Println(results["email"]) // "john@example.com"
}
```

## API Reference

### Types

```go
type Func[T any] func(ctx context.Context) (T, error)
```

A named unit of work. All functions in one call share the same `T`.

### ExecuteConcurrently

```go
func ExecuteConcurrently[T any](
    ctx context.Context,
    funcs map[string]Func[T],
) (map[string]T, error)
```

Executes every function in its own goroutine and returns the results indexed by
the map keys.

Behavior:

- A nil function in the map is rejected up front with an error naming its key.
- On the first error or panic, the shared context is canceled so the remaining
  functions can stop early. Functions that ignore `ctx` still run to completion.
- If any function fails, the returned map is nil and the error is the first
  causal error (secondary errors are discarded; a causal error is preferred
  over `context.Canceled`/`context.DeadlineExceeded` from siblings).
- A panic is recovered and converted to an error of the form `panic in "key": ...`.

### ExecuteConcurrentlyTyped

```go
func ExecuteConcurrentlyTyped[R any, T any](
    ctx context.Context,
    resultBuilder func(map[string]T) (R, error),
    funcs map[string]Func[T],
) (R, error)
```

Runs `ExecuteConcurrently` and, on success, folds the result map into a single
typed value with `resultBuilder`.

Type parameters are **result-first** — instantiate as
`ExecuteConcurrentlyTyped[Output, Input]`:

```go
summary, err := concurrent.ExecuteConcurrentlyTyped[string, int](
    ctx,
    func(results map[string]int) (string, error) {
        return fmt.Sprintf("total=%d", results["a"]+results["b"]), nil
    },
    funcs, // map[string]concurrent.Func[int]
)
```

If execution fails, the builder is not called and the zero value of `R` is
returned with the execution error. A builder error is returned as-is.

## Context Handling

Give the call a bounded context; check it inside long-running functions:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

funcs := map[string]concurrent.Func[string]{
    "slow": func(ctx context.Context) (string, error) {
        select {
        case <-time.After(10 * time.Second):
            return "done", nil
        case <-ctx.Done():
            return "", ctx.Err()
        }
    },
}

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

Canceling `ctx` (or a sibling failing) propagates to every function through the
shared context.

## Limitations

1. **First causal error only** — secondary errors are discarded; partial results are not returned.
2. **All-or-nothing** — the result map is nil if any function errors or panics.
3. **Unordered map results** — access results by key, not iteration order.
4. **Single element type** — all functions in one call return the same `T` (use `any` plus a builder for heterogeneous results).

## Examples

Runnable examples live in [examples/concurrent/](../examples/concurrent/) and are
behind the `example` build tag. From the repository root:

```bash
go run -tags=example ./examples/concurrent/
```

Compile-checked godoc examples are in [example_test.go](example_test.go).

## Testing

```bash
go test ./concurrent/ -count=1
```

The suite covers success, failure, cancellation, panic recovery, causal-error
priority, and typed building (~95% statement coverage).

## License

MIT License - see [LICENSE](../LICENSE) for details.
