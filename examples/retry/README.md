# Retry Examples

This directory contains examples demonstrating the retry package functionality.

## Running Examples

```bash
# From the repository root
go run -tags=example ./examples/retry

# Or by module path
go run -tags=example github.com/jasoet/pkg/v3/examples/retry
```

## Examples Included

### 1. Basic Retry
Basic retry with `retry.New(...)`: fails twice, succeeds on the third attempt.

### 2. Custom Backoff
Custom backoff parameters — initial interval, maximum interval, multiplier, max retries — with jitter disabled so the intervals are exact.

### 3. Permanent Errors
`retry.Permanent()` stops retrying for non-transient errors like validation failures: one attempt, error returned immediately.

### 4. Context Cancellation
Retry stops as soon as the context is cancelled.

### 5. Unlimited Retries with Timeout
`WithMaxRetries(0)` retries without an attempt limit; a context timeout acts as the safety net while polling.

### 6. Custom Notifications
`retry.DoWithNotify()` invokes a callback before each retry wait, for custom logging or metrics.

## Expected Output

The output below is reproducible: the examples disable jitter (`WithRandomizationFactor(0)`) and avoid timing- or randomness-dependent printout.

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
  Attempt 1
  Attempt 2
  Attempt 3
  Attempt 4
  ✅ Success after 4 attempts

Example 3: Permanent Errors
---------------------------
  Attempt 1
  ❌ Failed immediately (no retry): example.permanent failed after 1 attempts (1 initial + 0 retries): validation error: empty input
  Total attempts: 1 (expected: 1)

Example 4: Context Cancellation
-------------------------------
  Attempt 1
  Attempt 2
  🛑 Cancelling context...
  ❌ Cancelled: example.cancel canceled after 2 attempts: context canceled
  Stopped after 2 attempts

Example 5: Unlimited Retries with Timeout
-----------------------------------------
  Attempt 1
  Attempt 2
  Attempt 3
  Attempt 4
  Attempt 5
  ✅ Success after 5 attempts

Example 6: Custom Notifications
--------------------------------
  🔄 Retry scheduled in 50ms due to: failure #1
  🔄 Retry scheduled in 100ms due to: failure #2
  ✅ Success after 3 attempts
```

## Learn More

- [Retry Package Documentation](../../retry/README.md)
- [API Reference](https://pkg.go.dev/github.com/jasoet/pkg/v3/retry)
