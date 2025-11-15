# Retry Examples

This directory contains comprehensive examples demonstrating the retry package functionality.

## Running Examples

```bash
# Run all examples
go run -tags=example ./examples/retry

# Or from the repository root
go run -tags=example github.com/jasoet/pkg/v2/examples/retry
```

## Examples Included

### 1. Basic Retry
Demonstrates basic retry with default configuration (5 retries, 500ms initial interval, exponential backoff).

### 2. Custom Backoff
Shows how to configure custom backoff parameters:
- Initial interval
- Maximum interval
- Multiplier
- Max retries

### 3. Permanent Errors
Demonstrates how to use `retry.Permanent()` to stop retrying for non-transient errors like validation failures.

### 4. Context Cancellation
Shows how retry respects context cancellation and stops immediately.

### 5. Unlimited Retries with Timeout
Demonstrates unlimited retries (MaxRetries=0) combined with context timeout for polling scenarios.

### 6. Custom Notifications
Shows how to use `retry.DoWithNotify()` to get notified on each retry attempt for custom logging or metrics.

## Expected Output

```
=== Retry Package Examples ===

Example 1: Basic Retry
----------------------
  Attempt 1...
  Attempt 2...
  Attempt 3...
  ✅ Success after 3 attempts

Example 2: Custom Backoff
-------------------------
  Attempt 1 at 0ms
  Attempt 2 at 100ms
  Attempt 3 at 250ms
  Attempt 4 at 475ms
  ✅ Success after 4 attempts

Example 3: Permanent Errors
---------------------------
  Attempt 1
  ❌ Failed immediately (no retry): validation error: empty input
  Total attempts: 1 (expected: 1)

Example 4: Context Cancellation
-------------------------------
  Attempt 1
  Attempt 2
  🛑 Cancelling context...
  Attempt 3
  ❌ Cancelled: example.cancel cancelled after 3 attempts: context canceled
  Stopped after 3 attempts

Example 5: Unlimited Retries with Timeout
-----------------------------------------
  Attempt 1
  Attempt 2
  Attempt 3
  Attempt 4
  ✅ Success after 4 attempts

Example 6: Custom Notifications
--------------------------------
  🔄 Retry scheduled in 50ms due to: failure #1
  🔄 Retry scheduled in 100ms due to: failure #2
  ✅ Success after 3 attempts
```

## Learn More

- [Retry Package Documentation](../../retry/README.md)
- [API Reference](https://pkg.go.dev/github.com/jasoet/pkg/v2/retry)
