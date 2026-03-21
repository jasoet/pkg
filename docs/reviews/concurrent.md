# Code Review: `concurrent` Package

**Date:** 2026-03-21

## Package Summary

Provides two generic functions for fan-out concurrent execution of named functions: `ExecuteConcurrently[T]` and `ExecuteConcurrentlyTyped[T, R]`. Small package (103 lines of implementation).

---

## Issues Found

### High

**H1 — Panic recovery double-send structural fragility** (`execution.go:41-46`)

The panic-recovery defer and normal send path are mutually exclusive in current code. However, the structure is fragile: if code is added between the normal send and function end, a double-send would overflow the buffered channel and deadlock.

**Fix:** Restructure so exactly one send per goroutine is guaranteed unconditionally.

**H2 — `errgroup` available but unused**

`golang.org/x/sync/errgroup` is already an indirect dependency. The hand-rolled implementation re-invents it with subtle differences.

### Medium

- M1: Only first error returned — secondary errors silently discarded. Consider `errors.Join`.
- M2: Successful partial results discarded on any error — no way to get partial map
- M3: `fmt.Errorf` uses `%v` instead of `%w` for panic recovery — breaks `errors.Is`/`errors.As`
- M4: Closer goroutine is not cancellable
- M5: No nil function guard in input map

### Low

- L1: `wg.Add(len(funcs))` before loop — correct but pattern requires care on refactor
- L2: Map iteration order non-deterministic — affects error identity
- L3: Test types in `package concurrent` instead of `package concurrent_test`
- L4: README documents non-existent `Result[T]` type
- L5: Timing assertions in tests are flaky on loaded CI machines

### Security

- No unsafe operations. Imports only `context`, `errors`, `fmt`, `sync`.
- Panic recovery prevents goroutine-level panics from crashing the process.
- Context propagation is correct.

### Recommendations

1. Restructure goroutine body for guaranteed single send
2. Consider replacing with `errgroup`
3. Use `errors.Join` or document silent error discard
4. Use `%w` when recovered value is an `error`
5. Fix README: remove non-existent `Result[T]` type
