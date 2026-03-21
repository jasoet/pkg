# Code Review: `server` Package

**Date:** 2026-03-21

## Package Summary

Thin lifecycle-management wrapper around the Echo HTTP framework. Provides functional options configuration, three built-in health-check endpoints (`/health`, `/health/ready`, `/health/live`), graceful OS-signal-based shutdown, and optional OTel-integrated logging.

---

## Issues Found

### High

**H1 — No HTTP request timeouts on the underlying `net/http.Server`** (`server.go:113`)

`echo.New()` never sets `ReadTimeout`, `WriteTimeout`, `ReadHeaderTimeout`, or `IdleTimeout`. These default to zero (no timeout). DoS risk via slow-loris or large-body attacks.

**Fix:** Expose `WithReadTimeout`, `WithWriteTimeout`, `WithReadHeaderTimeout` options with secure defaults (e.g., 30s write, 5s read header).

**H2 — No request body size limit**

No body size limit is applied by default. A client can POST an arbitrarily large body, exhausting server memory.

**Fix:** Apply a default body limit middleware or add a `WithBodyLimit` option.

### Medium

- M1: `Operation` callback runs synchronously; panics are unrecovered
- M2: `Shutdown` callback has no timeout guard — can block indefinitely, preventing HTTP drain
- M3: `Port: 0` is valid but undocumented; negative/invalid ports not validated
- M4: `httpServer` is unexported — server cannot be stopped programmatically without signals
- M5: Health endpoints are not authenticated or rate-limited by default

### Low

- L1: Redundant signal registration (`os.Interrupt` and `syscall.SIGINT` are the same)
- L2: `newHttpServer` should be `newHTTPServer` per Go naming conventions
- L3: Logger created with `context.Background()` instead of call context
- L4: `Config.Port` has no `yaml`/`mapstructure` tags
- L5: Test custom error handler leaks internal error message; README mirrors this pattern
- L6: `TestIntegration` has silent error swallow with `if err == nil` guard

### Security

- No critical vulnerabilities. TLS, CORS, and headers are delegated to user-supplied middleware (correct design).
- Bind error detection is correct and immediate (uses `net.Listen` before goroutine).
- Signal handling is idiomatic.

### Recommendations

1. Set sane timeout defaults on `echo.Server` — `ReadHeaderTimeout` at minimum
2. Apply a default body-size limit middleware
3. Wrap `Shutdown` callback inside the `ShutdownTimeout` context
4. Export a composable `Server` type with `Start(ctx)`/`Stop()` methods
