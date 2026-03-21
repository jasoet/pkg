# Code Review: `rest` Package

**Date:** 2026-03-21

## Package Summary

Thin wrapper around go-resty that adds a middleware pipeline (Before/After hooks), typed error classification by HTTP status, optional OTel tracing/metrics/logging, and configurable retry/timeout behavior.

---

## Issues Found

### High

**H1 — `WithOTelConfig` panics if `restConfig` is nil** (`client.go:57-59`)

Dereferences `client.restConfig` unconditionally. Safe via `NewClient()` but panics on raw `&Client{}`.

**H2 — Full response body stored in error messages unconditionally** (`client.go:266-279`, `error.go`)

Every non-success response has full body captured as `response.String()` and embedded into returned errors. If downstream API echoes tokens, PII, or stack traces, these propagate to logs.

**H3 — Full request body stored in `RequestInfo` accessible to all middleware** (`middleware.go:14`, `client.go:228-233`)

No scrubbing at the middleware interface level. If a custom middleware logs `info.Body`, credentials are exposed.

### Medium

- M1: No URL validation — accepts arbitrary caller-supplied URLs (SSRF surface)
- M2: No TLS configuration surface; consumers cannot enforce minimum version or pin CAs
- M3: `GetRestClient()` returns unprotected underlying resty client — thread-safety issues
- M4: Lock in option functions is unnecessary during construction
- M5: OTel tracing injects W3C `traceparent` into caller-supplied headers map in-place — corrupts reused maps
- M6: No retry condition configured — retries on all errors including 4xx by default
- M7: Full URL used as OTel span name — causes metric cardinality explosion

### Low

- L1: `UnauthorizedError.Error()` returns only `Msg`, discarding status code and body
- L2: `IsForbidden` is defined but never used
- L3: README default configuration values are wrong (stale)
- L4: `doRequest` reads `restConfig` without lock
- L5: Metric errors silently swallowed with `nolint:errcheck`
- L6: `RequestInfo.Headers` stores original map by reference
- L7: `doRequest` buffers full response as string unconditionally — second copy

### Security

| Finding | Severity |
|---------|----------|
| Full response body in errors/logs | High |
| Full request body in `RequestInfo` | High |
| No URL validation (SSRF) | Medium |
| No TLS configuration surface | Medium |
| Trace headers injected in-place | Medium |
| Retries on non-idempotent 4xx | Medium |

### Recommendations

1. Add configurable `MaxResponseBodyLog` field (default 1 KB); truncate in errors
2. Add `WithURLValidator` option for SSRF defense
3. Expose `WithTLSConfig` option
4. Add retry conditions to restrict to network errors and 5xx only
5. Copy headers before OTel injection
6. Fix span naming cardinality — use route template, not full URL
7. Fix README default config values
