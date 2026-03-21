# Code Review: `grpc` Package

**Date:** 2026-03-21

## Package Summary

Utility library wrapping `google.golang.org/grpc` and `echo/v4` for production gRPC servers. Two modes: H2C (gRPC + HTTP on one port) and Separate (distinct ports). Ships with gRPC gateway integration, Prometheus metrics, health checks, and OTel interceptors.

---

## Issues Found

### Critical

**C1 — TLS is a stub: `WithTLS` sets flags but is never used** (`config.go:354-360`)

`WithTLS(certFile, keyFile)` sets `enableTLS=true`, `certFile`, `keyFile` — but neither `server.go` nor any other file reads those fields at runtime. Server silently starts in plaintext. A caller who passes `WithTLS(...)` and believes the server is secured is exposed to false sense of security.

**Fix:** Either implement TLS loading or return `errNotImplemented`.

**C2 — gRPC reflection enabled by default** (`config.go:98`)

Exposes the complete service schema, all method names, and message types to any unauthenticated client.

**Fix:** Change default to `false`.

### High

**H1 — `SetupGatewayForSeparate` always uses insecure credentials** (`echo_gateway.go:58`)

Gateway-to-backend connection hardcoded as plaintext. No parameter to pass TLS credentials.

**H2 — Signal handlers never stopped** (`server.go:420-429,449-458,476-488`)

`signal.Notify` called but `signal.Stop` never called. Stale registrations accumulate on repeated start/stop cycles.

**H3 — `Stop()` has a race on `running` state** (`server.go:299-365`)

Lock released before `shutdownOnce.Do` — window where concurrent `Stop()` calls both pass the `!s.running` guard.

**H4 — `waitForGRPCServer` never actually probes the port** (`echo_gateway.go:72-76`)

`grpc.NewClient` returns immediately without connecting. The retry loop is dead code.

**Fix:** Use `net.DialTimeout` to actually verify the port is listening.

**H5 — No `ReadHeaderTimeout` on H2C `http.Server`** (`server.go:272-278`)

Vulnerable to slow-header attacks.

### Medium

- M1: Health check middleware has TOCTOU cache thundering herd
- M2: `CheckHealth` calls checker functions while holding `RLock` — deadlock risk
- M3: OTel tracing interceptor discards W3C Trace Context from metadata — breaks distributed tracing
- M4: No `StreamServerInterceptor` — streaming RPCs completely uninstrumented
- M5: Default CORS is wildcard `*` with no configuration option
- M6: Echo server created in `Start()`, not `New()` — can't inspect before running
- M7: Liveness endpoint runs all health checks — should be trivial alive check
- M8: Port values not validated as valid numbers
- M9: `X-Gateway-Version` header leaks implementation detail

### Low

- L1: `log.Printf` used instead of structured logger
- L2: gRPC listener not closed on error in separate mode
- L3: README references non-existent `DefaultConfig()`, `StartWithConfig()`
- L4: Default "memory" health checker is a hardcoded placeholder
- L5: Tests use deprecated `grpc.DialContext` + `grpc.WithInsecure`
- L6: `overallStatusFromResults` ignores `HealthStatusUnknown`
- L7: `echoConfigurer` called after gateway routes — ordering surprise

### Security

| Finding | Severity |
|---------|----------|
| TLS option is a no-op; server always runs plaintext | Critical |
| gRPC reflection on by default | Critical |
| Gateway-to-backend hardcoded insecure | High |
| W3C Trace Context not propagated | Medium |
| Default CORS is wildcard | Medium |
| No authentication layer provided or documented | Medium |

### Recommendations

1. Implement or remove TLS
2. Change reflection default to `false`
3. Fix `waitForGRPCServer` with `net.DialTimeout`
4. Add `signal.Stop`
5. Implement W3C Trace Context propagation
6. Add `StreamServerInterceptor`
7. Fix liveness probe semantics
8. Add `ReadHeaderTimeout` and `MaxHeaderBytes` to H2C server
