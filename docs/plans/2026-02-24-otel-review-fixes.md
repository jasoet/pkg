# Plan: otel Package — Code Review Fixes

**Date:** 2026-02-24
**Branch:** `fix/otel-review`
**Scope:** 4 critical, 2 important issues, 1 suggestion

---

## Fixes

### 1. [C1, C2, C3] Slice Mutation Bugs in helper.go

**Problem:** `append(h.baseFields, fields...)` in `WithFields()` (line 120), `log()` (line 157), and `Error()` (line 199) can mutate the parent helper's backing array when it has spare capacity. This causes data corruption when multiple child helpers share a parent.

**Fix:** Replace all three with defensive copy:
```go
allFields := append(make([]Field, 0, len(h.baseFields)+len(fields)), h.baseFields...)
allFields = append(allFields, fields...)
```

**Test:** Add test creating two child helpers from the same parent, verify fields don't interfere.

### 2. [C4] SeverityFatal Calls os.Exit(1) in logging.go

**Problem:** `severityToZerologEvent()` (line 235) maps `SeverityFatal` to `logger.Fatal()`, which calls `os.Exit(1)`. An OTel log record should never terminate the process.

**Fix:** Replace `logger.Fatal()` with `logger.WithLevel(zerolog.FatalLevel)`. This emits the log at fatal level without the `os.Exit(1)` side effect.

### 3. [I1] Silent Error Discard in defaultLoggerProvider

**Problem:** `config.go:160` discards the error from `NewLoggerProviderWithOptions()`. If provider creation fails, a nil provider is returned.

**Fix:** Log the error via zerolog and return a no-op provider as fallback instead of nil.

### 4. [I2, S1] Incomplete Shutdown

**Problem:** `Shutdown()` (config.go:166-182) only shuts down `LoggerProvider`, ignoring `TracerProvider` and `MeterProvider`.

**Fix:** Attempt shutdown on all three providers. Use type assertion against a `Shutdown(context.Context) error` interface. Collect errors with `errors.Join()`.

### 5. [I3] Duplicate DisableLogging Method

**Problem:** `WithoutLogging()` and `DisableLogging()` are identical methods.

**Fix:** Remove `DisableLogging()`. Keep `WithoutLogging()` which follows the `With*` naming convention.

### 6. [I4] Exported Fields Thread Safety

**Problem:** All `Config` fields are exported, allowing direct mutation.

**Fix:** Add doc comment to `Config` struct explicitly stating the immutability contract — create once, don't mutate after passing to consumers. Unexported fields + getters would be too large an API change for this batch.

---

## Verification

- Run `task test` — all existing unit tests pass
- Run new slice mutation isolation test
- Confirm no compilation errors across dependent packages: `go build ./...`
