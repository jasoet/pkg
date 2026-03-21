# Code Review: `temporal` Package

**Date:** 2026-03-21

## Package Summary

Wraps the Temporal workflow engine SDK providing: `Config`, client factories with optional Prometheus/OTel, `WorkerManager`, `ScheduleManager`, `WorkflowManager`, `ZerologAdapter`, and a testcontainer subpackage.

---

## Issues Found

### Critical

**C1 — `StartAll` partial failure leaves already-started workers running** (`worker.go:143-154`)

On any `w.Start()` error, returns immediately without stopping previously started workers. Caller believes nothing is running, but N-1 workers may be active.

**Fix:** Stop all previously started workers before returning error.

**C2 — Lock upgrade TOCTOU in `Close`** (`worker.go:44-77`)

Reads count under `RLock`, releases, re-acquires to iterate. Between acquisitions, another goroutine could call `Register`, changing the slice.

### High

- H1: `DeleteSchedules` holds `RLock` during long network I/O, then non-atomic lock upgrade
- H2: Raw unsanitized query string in `ListWorkflows`/`CountWorkflows`
- H3: OTel interceptor creation failure silently swallowed
- H4: `Close()` not idempotent — double-close on gRPC connection undefined behavior

### Medium

- M1: `DefaultConfig` creates logger just for one debug line
- M2: `Start(ctx, w)` ignores passed `ctx` entirely
- M3: Testcontainer uses hard `time.Sleep` instead of readiness probe
- M4: `NewWorkflowManager(client.Client)` hardcodes namespace to `"default"`
- M5: Context leak — `context.WithTimeout` without storing cancel
- M6: `ScheduleManager` metrics disabled inconsistently vs other managers
- M7: `GetClient()` returns unprotected internal reference
- M8: `GetDashboardStats` uses hardcoded strings instead of enum-to-string

### Low

- L1: `WithCallerSkip` skip count is magic number `+2`
- L2: `Start(ctx, w)` logs index as "taskQueue" — misleading
- L3: Cleanup closure captures `ctx` by reference — if cancelled, `Terminate` fails
- L4: `WorkflowFilter`/`TimeRange` types defined but never used
- L5: `MetricsListenAddress` defaults to `0.0.0.0:9090` — exposes metrics externally
- L6: Error wrapping inconsistent
- L7: Multiple subtests share single `WorkerManager`

### Security

- **SEC-1** (High): No TLS configuration support — all connections plaintext
- **SEC-2** (High): No authentication/namespace credential support
- **SEC-3** (Medium): Prometheus metrics endpoint has no auth or binding restriction
- **SEC-4** (Medium): `validateQueryParam` regex overly restrictive and not applied uniformly
- **SEC-5** (Low): `temporalio/temporal:latest` unpinned image tag
- **SEC-6** (Low): Default metrics bind address `0.0.0.0`

### Recommendations

1. Add TLS and credential support to `Config`
2. Fix `StartAll` rollback on partial failure
3. Fix double-lock TOCTOU in `Close` and `DeleteSchedules`
4. Fix namespace inference — accept as explicit argument
5. Change default `MetricsListenAddress` to `127.0.0.1:9090`
6. Pin testcontainer default image
