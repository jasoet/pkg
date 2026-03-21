# Code Review: `logging` Package

**Date:** 2026-03-21

## Package Summary

Thin initialization wrapper around `github.com/rs/zerolog`. Configures the zerolog global logger with standard fields (service name, PID, caller, timestamp), supports console (human-readable) or file (JSON) output, or both via bitmask API. Three public functions: `Initialize`, `InitializeWithFile`, `ContextLogger`.

---

## Issues Found

### High

**H1 — Resource leak on partial failure in `InitializeWithFile`** (`logging.go:86-98`)

File handle opened but no explicit cleanup pattern if future error paths are added after `os.OpenFile`. Currently safe but structurally fragile.

**H2 — Global logger mutation not safe for concurrent initialization** (`logging.go:16,63-64,113`)

`initMu` protects the assignment, but derived logger copies held by other goroutines silently continue using old configuration. No enforcement mechanism for "initialize once at startup."

### Medium

- M1: `LogLevel` type defined but not used by the package's own API — inconsistency
- M2: File permissions world-readable `0o644` — should be `0o600`
- M3: PID logging as structural field has limited value in containers
- M4: No validation on `serviceName` or `component` parameters
- M5: README documents incorrect `InitializeWithFile` signature (missing `io.Closer` return)
- M6: `ContextLogger` creates a new logger on every call — allocation concern in hot paths

### Low

- L1: `OutputDestination` has no validation for unknown bits
- L2: Tests directly mutate global logger without cleanup
- L3: No Trace/Fatal log level in `LogLevel` enum
- L4: `ConsoleWriter.NoColor` not configurable (no TTY detection)
- L5: `Caller()` enabled unconditionally — performance cost and path leakage

### Security

- **SEC-1** (Low): Caller file paths expose internal source tree structure
- **SEC-2** (Medium): Log file mode `0644` may expose sensitive application data
- **SEC-3** (Low): No log injection sanitization for console output

### Recommendations

1. Fix README documentation — correct `InitializeWithFile` signature
2. Change default file permissions to `0600`
3. Guard file close on any future error paths
4. Add TTY detection for `ConsoleWriter.NoColor`
5. Make `Caller()` conditional on debug mode
