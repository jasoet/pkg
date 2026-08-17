# Concurrent Package Examples

This directory contains examples demonstrating how to use the `concurrent` package for parallel execution of functions in Go applications.

## 📍 Example Code Location

**Full example implementation:** [example.go](example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/v3/concurrent"

// Define functions to run concurrently
funcs := map[string]concurrent.Func[string]{
    "api1": func(ctx context.Context) (string, error) {
        // Call API 1
        return callAPI1(ctx)
    },
    "api2": func(ctx context.Context) (string, error) {
        // Call API 2
        return callAPI2(ctx)
    },
}

// Execute all functions concurrently
results, err := concurrent.ExecuteConcurrently(ctx, funcs)
if err != nil {
    // Handle error (fail-fast: if one fails, all are cancelled)
}

// Access results by key
api1Result := results["api1"]
api2Result := results["api2"]
```

**Key features:**
- Type-safe with Go generics
- Fail-fast: cancels all on first error
- Context-aware for proper cancellation
- Returns map of results by key

## Overview

The `concurrent` package provides utilities for:
- Executing multiple functions concurrently with type safety
- Collecting results from parallel operations
- Handling errors with fail-fast behavior and context cancellation
- Building typed result structures from concurrent operations

## Running the Examples

The example is behind the `example` build tag. From the repository root:

```bash
go run -tags=example ./examples/concurrent/
```

## Example Descriptions

The [example.go](example.go) file demonstrates several use cases:

### 1. Basic Concurrent Execution

Execute multiple functions concurrently and collect their results:

```go
funcs := map[string]concurrent.Func[string]{
    "task1": func(ctx context.Context) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Result from task 1", nil
    },
    "task2": func(ctx context.Context) (string, error) {
        time.Sleep(200 * time.Millisecond)
        return "Result from task 2", nil
    },
}

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### 2. Database Operations Concurrency

Perform multiple database queries concurrently:

```go
dbFuncs := map[string]concurrent.Func[[]User]{
    "active_users": func(ctx context.Context) ([]User, error) {
        return db.GetActiveUsers(ctx)
    },
    "inactive_users": func(ctx context.Context) ([]User, error) {
        return db.GetInactiveUsers(ctx)
    },
    "admin_users": func(ctx context.Context) ([]User, error) {
        return db.GetAdminUsers(ctx)
    },
}

userGroups, err := concurrent.ExecuteConcurrently(ctx, dbFuncs)
```

### 3. Typed Result Building

Use typed result structures for better type safety:

```go
type DashboardData struct {
    Users    []User
    Products []Product
    Orders   []Order
}

resultBuilder := func(results map[string]interface{}) (DashboardData, error) {
    return DashboardData{
        Users:    results["users"].([]User),
        Products: results["products"].([]Product),
        Orders:   results["orders"].([]Order),
    }, nil
}

dashboard, err := concurrent.ExecuteConcurrentlyTyped(ctx, resultBuilder, dataFuncs)
```

### 4. API Calls Concurrency

Make multiple API calls concurrently:

```go
apiFuncs := map[string]concurrent.Func[ApiResponse]{
    "weather": func(ctx context.Context) (ApiResponse, error) {
        return weatherAPI.GetCurrentWeather(ctx, "New York")
    },
    "news": func(ctx context.Context) (ApiResponse, error) {
        return newsAPI.GetLatestNews(ctx, "technology")
    },
    "stock": func(ctx context.Context) (ApiResponse, error) {
        return stockAPI.GetStockPrice(ctx, "AAPL")
    },
}

apiResults, err := concurrent.ExecuteConcurrently(ctx, apiFuncs)
```

### 5. Error Handling and Context Cancellation

Handle errors with fail-fast behavior:

```go
funcs := map[string]concurrent.Func[string]{
    "success": func(ctx context.Context) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Success", nil
    },
    "failure": func(ctx context.Context) (string, error) {
        time.Sleep(50 * time.Millisecond)
        return "", errors.New("operation failed")
    },
    "slow": func(ctx context.Context) (string, error) {
        select {
        case <-time.After(1 * time.Second):
            return "Slow result", nil
        case <-ctx.Done():
            return "", ctx.Err()
        }
    },
}

// The slow operation will be cancelled when the failure occurs
results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### 6. Batch Processing

Process batches of data concurrently:

```go
func processBatch(items []Item) ([]ProcessedItem, error) {
    batchSize := 10
    batches := make(map[string]concurrent.Func[[]ProcessedItem])
    
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        batchKey := fmt.Sprintf("batch_%d", i/batchSize)
        
        batches[batchKey] = func(ctx context.Context) ([]ProcessedItem, error) {
            return processItems(ctx, batch)
        }
    }
    
    batchResults, err := concurrent.ExecuteConcurrently(ctx, batches)
    if err != nil {
        return nil, err
    }
    
    var allResults []ProcessedItem
    for _, result := range batchResults {
        allResults = append(allResults, result...)
    }
    
    return allResults, nil
}
```

## Key Features

### Type Safety
- Uses Go generics for compile-time type safety
- Supports any return type with `Func[T any]`
- Type-safe result building with `ExecuteConcurrentlyTyped`

### Error Handling
- **Fail-fast behavior**: the first error or panic cancels the shared context, signaling the other functions to stop
- **Context cancellation**: proper context propagation for cancellation
- **Error propagation**: `ExecuteConcurrently` waits for every goroutine to exit before returning, then returns the first causal error. Cancellation only *signals* siblings to stop — a function that ignores `ctx` still runs to completion, so the call blocks until the slowest function returns

### Performance
- **Concurrent execution**: All functions run in parallel
- **Efficient coordination**: Uses channels and WaitGroups for coordination
- **Memory efficient**: Results are collected as they complete

### Flexibility
- **Generic functions**: Support for any function signature that returns `(T, error)`
- **Keyed results**: Results are organized by string keys for easy access
- **Custom result builders**: Build complex result structures with type safety

## Best Practices

### 1. Context Management
```go
// Always use context with timeout for long-running operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

### 2. Error Handling
```go
// Handle errors appropriately.
// Note: execution is all-or-nothing — on error `results` is nil, so there are
// no partial results to inspect. The returned error is the first causal error.
results, err := concurrent.ExecuteConcurrently(ctx, funcs)
if err != nil {
    log.Printf("Concurrent execution failed: %v", err)
    return
}
// results is only non-nil when every function succeeded.
```

### 3. Function Design
```go
// Make functions context-aware for proper cancellation
func fetchUserData(ctx context.Context) (UserData, error) {
    select {
    case <-ctx.Done():
        return UserData{}, ctx.Err()
    default:
        // Perform actual work
        return getUserFromDB(ctx)
    }
}
```

### 4. Resource Management

`ExecuteConcurrently` starts *every* function in the map at once — it does not
bound concurrency itself. To limit how many run simultaneously, process the
items in fixed-size batches and call `ExecuteConcurrently` once per batch. This
caps concurrency at `maxConcurrency` while still processing **every** item (do
not simply `break` out of the loop past the limit — that silently drops the
remaining items).

```go
// Process all items, at most maxConcurrency at a time.
var allResults []string
for start := 0; start < len(items); start += maxConcurrency {
    end := start + maxConcurrency
    if end > len(items) {
        end = len(items)
    }

    funcs := make(map[string]concurrent.Func[string])
    for i, item := range items[start:end] {
        funcs[fmt.Sprintf("item_%d", start+i)] = createProcessingFunc(item)
    }

    batch, err := concurrent.ExecuteConcurrently(ctx, funcs)
    if err != nil {
        return nil, err
    }
    for _, r := range batch {
        allResults = append(allResults, r)
    }
}
```

For finer-grained control (a fixed pool of long-lived workers pulling from a
channel), use a classic worker pool instead of `ExecuteConcurrently`.

## Use Cases

### 1. Data Aggregation
- Fetch data from multiple sources (databases, APIs, files)
- Aggregate results for dashboard views
- Parallel data validation and enrichment

### 2. Batch Processing
- Process large datasets in parallel batches
- Concurrent file processing
- Parallel data transformations

### 3. API Orchestration
- Fan-out API calls to multiple services
- Parallel external service integration
- Concurrent data fetching for microservices

### 4. Performance Optimization
- Reduce overall execution time for independent operations
- Improve throughput for I/O-bound operations
- Optimize resource utilization

## Performance Considerations

### Concurrency vs Parallelism
- **I/O-bound operations**: High concurrency (hundreds of operations)
- **CPU-bound operations**: Limit to number of CPU cores
- **Memory-bound operations**: Consider memory usage per operation

### Context Timeouts
```go
// Set appropriate timeouts based on operation characteristics
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

### Error Recovery
```go
// Consider whether to retry failed operations
if err != nil {
    if isRetryable(err) {
        // Implement retry logic
        return retryOperation(ctx, funcs)
    }
    return nil, err
}
```

## Troubleshooting

### Common Issues

1. **Context Cancellation**: Operations cancelled due to one failure
   - **Solution**: Use separate contexts if operations should be independent

2. **Resource Exhaustion**: Too many concurrent operations
   - **Solution**: Limit concurrency or use worker pools

3. **Deadlocks**: Improper channel usage or blocking operations
   - **Solution**: Ensure proper context handling and non-blocking operations

4. **Memory Leaks**: Goroutines not properly cleaned up
   - **Solution**: Always use context cancellation and proper defer statements