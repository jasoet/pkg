# Code Review: `docker` Package

**Date:** 2026-03-21

## Package Summary

Go utility library for managing Docker/Podman container lifecycles. Wraps the official Docker SDK and provides two API styles: functional options and declarative struct. Core capabilities: container lifecycle, log streaming, port/network inspection, wait strategies, and optional OTel instrumentation.

---

## Issues Found

### High

**H1 — `regexp.MustCompile` panics on invalid user-supplied patterns** (`wait.go:33`)

`WaitForLog(pattern)` calls `MustCompile` which panics if the pattern is invalid regex. The caller's string is untrusted input.

**Fix:** Use `regexp.Compile` with error return.

**H2 — `Start()` can be called multiple times, silently leaking containers** (`executor.go:108-162`)

No guard against calling `Start()` when `containerID` is already set. A second call creates a new container and permanently loses the reference to the first.

**Fix:** Check `e.containerID != ""` at the top.

**H3 — `ConnectionString` is a format-string injection surface** (`network.go:183-190`)

`template` parameter passed directly to `fmt.Sprintf`. If constructed from untrusted input, additional `%v` verbs can expose internal state.

**Fix:** Use `strings.ReplaceAll` with a fixed placeholder.

### Medium

- M1: `BindMounts` allows arbitrary bind-mount specs without validation
- M2: `WithPrivileged` and `WithCapAdd` have no security documentation
- M3: `WaitForLog` pattern matching is chunk-boundary unsafe (reads 8192 bytes at a time)
- M4: `Stop()` and `Restart()` have TOCTOU window with `containerID`
- M5: Image inspection error is swallowed — silently falls through to pull
- M6: `errors.Is` not used for `io.EOF` comparison (direct equality)

### Low

- L1: Duplicate port-parsing logic in four places
- L2: `WithPortBindings` hardcodes `"tcp"` instead of `defaultProtocol` constant
- L3: `Status.Status` and `Status.State` set to the same value
- L4: `Close()` does not prevent further use of the executor
- L5: Benchmark tests ignore start errors
- L6: `multiWait` timeout may double-count nested timeouts
- L7: `LogEntry.Timestamp` field is never populated

### Security

- No command injection: all Docker interaction via SDK over Unix socket (no `exec.Command`)
- No path traversal beyond Docker daemon enforcement
- OTel instrumentation correctly avoids recording sensitive config values

### Recommendations

1. Fix `WaitForLog` to use `regexp.Compile` and return error
2. Add idempotency guard to `Start()`
3. Replace `ConnectionString`'s `fmt.Sprintf` pattern with safe substitution
4. Fix `pullImage` to check `errdefs.IsNotFound` before falling through
5. Fix `WaitForLog` chunk-boundary bug using `bufio.Scanner`
6. Replace `err == io.EOF` with `errors.Is(err, io.EOF)`
