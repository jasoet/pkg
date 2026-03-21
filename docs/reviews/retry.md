# Code Review: `retry` Package

**Date:** 2026-03-21

## Package Summary

Production-oriented retry utility built on `cenkalti/backoff/v4`. Provides exponential backoff with jitter via `Config` (value-type fluent API), two entry points (`Do` and `DoWithNotify`), OTel tracing and structured logging integration, `Permanent` error wrapping, and context cancellation support.

---

## Issues Found

### High

**H1 — `attempt` counter synchronicity assumption** (`retry.go:157,277`)

`attempt` is mutated inside the closure called by `backoff.Retry`. Currently safe because the library calls synchronously, but a latent risk if the library changes.

**H2 — `lastErr` may be nil when used in error wrapping** (`retry.go:219,338`)

If `backoff.Retry` returns non-nil error that isn't the operation error, `lastErr` could be nil. `fmt.Errorf("... %w", lastErr)` with nil `lastErr` produces `"... <nil>"`.

**Fix:** Guard: `if lastErr != nil { ... } else { return generic error }`.

**H3 — Context cancellation detection not exhaustive** (`retry.go:203,322`)

Uses `ctx.Err() != nil` instead of `errors.Is(err, context.Canceled)` — less robust, order-dependent.

### Medium

- M1: Massive code duplication between `Do` and `DoWithNotify` (~120 lines identical)
- M2: OTel check uses `IsTracingEnabled()` but ignores logging-only config — logger stays nil
- M3: `WithRandomizationFactor` has no input validation — values >= 1.0 can cause panic in backoff
- M4: `attempt` count semantics misleading — `MaxRetries(3)` produces error "failed after 4 attempts"
- M5: `notifyFunc` wrapper in `DoWithNotify` is pointless indirection
- M6: First `Debug` log says "attempt: 1" vs backoff library counting from 0

### Low

- L1: Instrumentation name string duplicated verbatim in both functions
- L2: README claims generics are used — false
- L3: Test doesn't assert default `RandomizationFactor`
- L4: No test for `WithOTel` builder method
- L5: `span.SetAttributes` after `span.End()` risk (currently safe but latent)
- L6: Timing assertion in test may be flaky

### Security

- No timing attack surface (not crypto-related)
- Error messages may leak internal details via `%w` chain
- No input sanitization on `OperationName` before OTel span names (unbounded length)

### Recommendations

1. Guard `lastErr` nil check before wrapping
2. Use `errors.Is(err, context.Canceled/DeadlineExceeded)`
3. Refactor `Do` and `DoWithNotify` to share common implementation
4. Fix OTel gate to enable logging independently of tracing
5. Add validation for `RandomizationFactor` and `Multiplier`
6. Fix README: remove false generics claim
