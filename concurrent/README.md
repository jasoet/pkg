# Concurrent Execution

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/concurrent.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/concurrent)

Type-safe concurrent execution utilities with generics, error aggregation, and automatic cancellation.

## Overview

The `concurrent` package provides production-ready utilities for executing multiple operations concurrently with full type safety using Go generics. It handles error propagation, context cancellation, and result aggregation automatically.

## Features

- **Type-Safe Generics**: Full compile-time type safety
- **Auto Cancellation**: Cancels remaining operations on first error
- **Error Handling**: Returns first error encountered
- **Context Support**: Respects context cancellation and timeouts
- **Flexible Results**: Map-based or typed struct results
- **Zero Dependencies**: Only uses Go standard library

## Installation

```bash
go get github.com/jasoet/pkg/v2/concurrent
```

## Quick Start

### Basic Concurrent Execution

```go
package main

import (
    "context"
    "fmt"
    "github.com/jasoet/pkg/v2/concurrent"
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

### Type-Safe Results

```go
import "github.com/jasoet/pkg/v2/concurrent"

type UserData struct {
    Name  string
    Email string
}

func main() {
    ctx := context.Background()

    funcs := map[string]concurrent.Func[string]{
        "name":  fetchName,
        "email": fetchEmail,
    }

    // Build typed result
    userData, err := concurrent.ExecuteConcurrentlyTyped(
        ctx,
        func(results map[string]string) (UserData, error) {
            return UserData{
                Name:  results["name"],
                Email: results["email"],
            }, nil
        },
        funcs,
    )

    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", userData)
}
```

## API Reference

### Types

#### Func[T any]

Generic function type for concurrent execution:

```go
type Func[T any] func(ctx context.Context) (T, error)
```

#### Result[T any]

Holds the result of a concurrent operation:

```go
type Result[T any] struct {
    Value T
    Err   error
}
```

### Functions

#### ExecuteConcurrently

Execute multiple functions concurrently:

```go
func ExecuteConcurrently[T any](
    ctx context.Context,
    funcs map[string]Func[T],
) (map[string]T, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `funcs`: Map of named functions to execute

**Returns:**
- `map[string]T`: Results indexed by function names
- `error`: First error encountered (if any)

**Behavior:**
- Executes all functions concurrently
- Returns first error and cancels remaining operations
- Results are nil if any function errors

#### ExecuteConcurrentlyTyped

Type-safe concurrent execution with result builder:

```go
func ExecuteConcurrentlyTyped[T any, R any](
    ctx context.Context,
    resultBuilder func(map[string]T) (R, error),
    funcs map[string]Func[T],
) (R, error)
```

**Parameters:**
- `ctx`: Context for cancellation
- `resultBuilder`: Function to build typed result from map
- `funcs`: Map of functions to execute

**Returns:**
- `R`: Built result of type R
- `error`: Error from execution or builder

## Usage Examples

### Database Queries

```go
type Product struct {
    ID    int
    Name  string
    Price float64
}

funcs := map[string]concurrent.Func[*Product]{
    "product1": func(ctx context.Context) (*Product, error) {
        return db.GetProduct(ctx, 1)
    },
    "product2": func(ctx context.Context) (*Product, error) {
        return db.GetProduct(ctx, 2)
    },
    "product3": func(ctx context.Context) (*Product, error) {
        return db.GetProduct(ctx, 3)
    },
}

products, err := concurrent.ExecuteConcurrently(ctx, funcs)
if err != nil {
    log.Fatal(err)
}

for key, product := range products {
    fmt.Printf("%s: %+v\n", key, product)
}
```

### API Calls

```go
type APIResponse struct {
    Data   string
    Status int
}

funcs := map[string]concurrent.Func[*APIResponse]{
    "api1": func(ctx context.Context) (*APIResponse, error) {
        return callAPI(ctx, "https://api1.example.com")
    },
    "api2": func(ctx context.Context) (*APIResponse, error) {
        return callAPI(ctx, "https://api2.example.com")
    },
}

responses, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### File Processing

```go
funcs := map[string]concurrent.Func[[]byte]{
    "file1.txt": func(ctx context.Context) ([]byte, error) {
        return os.ReadFile("file1.txt")
    },
    "file2.txt": func(ctx context.Context) ([]byte, error) {
        return os.ReadFile("file2.txt")
    },
}

contents, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### Aggregated Results

```go
type DashboardData struct {
    UserCount    int
    OrderCount   int
    RevenueTotal float64
}

funcs := map[string]concurrent.Func[float64]{
    "users":   countUsers,
    "orders":  countOrders,
    "revenue": calculateRevenue,
}

dashboard, err := concurrent.ExecuteConcurrentlyTyped(
    ctx,
    func(results map[string]float64) (DashboardData, error) {
        return DashboardData{
            UserCount:    int(results["users"]),
            OrderCount:   int(results["orders"]),
            RevenueTotal: results["revenue"],
        }, nil
    },
    funcs,
)
```

## Context Handling

### Timeout

```go
// Set timeout for all operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Operations timed out")
    }
}
```

### Cancellation

```go
// Manual cancellation
ctx, cancel := context.WithCancel(context.Background())

// Cancel after some condition
go func() {
    time.Sleep(2 * time.Second)
    cancel() // Cancels all running operations
}()

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### Early Termination

```go
// Automatically cancels remaining operations on first error
funcs := map[string]concurrent.Func[string]{
    "fast": func(ctx context.Context) (string, error) {
        return "done", nil
    },
    "slow": func(ctx context.Context) (string, error) {
        time.Sleep(10 * time.Second)
        return "done", nil // Won't complete if "error" fails first
    },
    "error": func(ctx context.Context) (string, error) {
        return "", errors.New("failed") // Cancels "slow"
    },
}

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
// err != nil, "slow" was cancelled
```

## Error Handling

### First Error Returns

```go
funcs := map[string]concurrent.Func[int]{
    "success": func(ctx context.Context) (int, error) {
        return 42, nil
    },
    "failure": func(ctx context.Context) (int, error) {
        return 0, errors.New("operation failed")
    },
}

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
if err != nil {
    // err contains first error encountered
    // results is nil
    log.Printf("Concurrent execution failed: %v", err)
}
```

### Builder Errors

```go
results, err := concurrent.ExecuteConcurrentlyTyped(
    ctx,
    func(results map[string]int) (MyStruct, error) {
        // Validate results
        if results["required"] == 0 {
            return MyStruct{}, errors.New("required field missing")
        }
        return MyStruct{Value: results["required"]}, nil
    },
    funcs,
)
```

## Best Practices

### 1. Use Context Timeouts

```go
// ✅ Good: Always use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, _ := concurrent.ExecuteConcurrently(ctx, funcs)

// ❌ Bad: No timeout
ctx := context.Background()
results, _ := concurrent.ExecuteConcurrently(ctx, funcs)
```

### 2. Handle Context in Functions

```go
// ✅ Good: Check context cancellation
func fetchData(ctx context.Context) (string, error) {
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    default:
        // Do work
        return "data", nil
    }
}

// ❌ Bad: Ignore context
func fetchData(ctx context.Context) (string, error) {
    time.Sleep(10 * time.Second) // Doesn't respect cancellation
    return "data", nil
}
```

### 3. Keep Functions Independent

```go
// ✅ Good: Independent functions
funcs := map[string]concurrent.Func[int]{
    "task1": independentTask1,
    "task2": independentTask2,
}

// ❌ Bad: Dependent functions (use sequential execution)
funcs := map[string]concurrent.Func[int]{
    "task1": func(ctx context.Context) (int, error) {
        return 1, nil
    },
    "task2": func(ctx context.Context) (int, error) {
        // Depends on task1 result - this won't work!
        return task1Result + 1, nil
    },
}
```

### 4. Use Typed Builders

```go
// ✅ Good: Type-safe result building
type Result struct {
    Users  int
    Orders int
}

concurrent.ExecuteConcurrentlyTyped(
    ctx,
    func(results map[string]int) (Result, error) {
        return Result{
            Users:  results["users"],
            Orders: results["orders"],
        }, nil
    },
    funcs,
)

// ❌ Bad: Manual type assertions
results, _ := concurrent.ExecuteConcurrently(ctx, funcs)
users := results["users"]   // Requires type knowledge
orders := results["orders"]
```

### 5. Check All Results

```go
// ✅ Good: Validate builder results
concurrent.ExecuteConcurrentlyTyped(
    ctx,
    func(results map[string]Data) (Aggregate, error) {
        if len(results) != expectedCount {
            return Aggregate{}, errors.New("incomplete results")
        }
        // Build aggregate
    },
    funcs,
)
```

## Testing

The package includes comprehensive tests with 100% coverage:

```bash
# Run tests
go test ./concurrent -v

# With coverage
go test ./concurrent -cover
```

### Test Examples

```go
func TestConcurrentExecution(t *testing.T) {
    ctx := context.Background()

    funcs := map[string]concurrent.Func[int]{
        "double": func(ctx context.Context) (int, error) {
            return 10, nil
        },
        "triple": func(ctx context.Context) (int, error) {
            return 15, nil
        },
    }

    results, err := concurrent.ExecuteConcurrently(ctx, funcs)

    assert.NoError(t, err)
    assert.Equal(t, 10, results["double"])
    assert.Equal(t, 15, results["triple"])
}
```

## Performance

- **Goroutine Overhead**: ~2KB per goroutine
- **Channel Overhead**: Minimal buffered channel
- **Type Safety**: Zero runtime overhead (generics compile-time only)

**Benchmark:**
```
BenchmarkExecuteConcurrently-8    10000    ~100µs/op (5 functions)
BenchmarkTypedExecution-8         10000    ~105µs/op (includes builder)
```

## Limitations

1. **First Error Only**: Returns first error, others are lost
2. **All-or-Nothing**: All results are nil if any function errors
3. **Map Results**: Results are unordered (use keys to access)
4. **Same Type**: All functions must return same type T

## Examples

See [examples/](.../examples/concurrent/concurrent/) directory for:
- Basic concurrent execution
- Typed result building
- Context handling
- Error handling
- Real-world use cases

## Related Packages

- **[db](../db/)** - Database operations
- **[rest](../rest/)** - HTTP client

## License

MIT License - see [LICENSE](../LICENSE) for details.
