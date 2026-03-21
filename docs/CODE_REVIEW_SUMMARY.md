# Code Review & Security Audit — Summary

**Date:** 2026-03-21
**Scope:** All 15 packages in `github.com/jasoet/pkg/v2`
**Reviewed:** ~150+ Go source files, tests, READMEs, and examples

## Executive Summary

The codebase is generally well-structured, follows Go conventions, and uses consistent patterns (functional options, OTel integration, testify). However, the review uncovered several critical and high-severity issues that should be addressed before any new release.

**Totals: 8 Critical, 24 High, 48 Medium, 55+ Low findings**

---

## Critical Findings (Fix Immediately)

| # | Package | Finding | Location |
|---|---------|---------|----------|
| C1 | **ssh** | `t.client` written/read without mutex — data race between `Start()`, `forward()`, and `Close()` | `tunnel.go:145,187,222` |
| C2 | **ssh** | Accept loop continues after `Close()` — no shutdown signal channel | `tunnel.go:168-177` |
| C3 | **grpc** | `WithTLS()` is a no-op stub — server always runs plaintext but caller believes TLS is active | `config.go:354, server.go` |
| C4 | **grpc** | gRPC reflection enabled by default — exposes full schema unauthenticated | `config.go:98` |
| C5 | **compress** | Decompression bomb: `io.LimitReader` silently truncates with no error returned | `gz.go:54, tar.go:168` |
| C6 | **compress** | Zip slip via symlink TOCTOU — no `filepath.EvalSymlinks` before write | `tar.go:243` |
| C7 | **config** | `NestedEnvVars` panics on negative `keyDepth` (index out of range) | `config.go:77` |
| C8 | **argo** | `mustParseQuantity` panics on invalid input in `Script.Templates()` | `builder/template/script.go:344` |

---

## High Findings (Fix Before Next Release)

| # | Package | Finding |
|---|---------|---------|
| H1 | **db** | `Dsn()` is public and returns plaintext password in connection string |
| H2 | **db** | Ping/connection errors may leak DSN with password |
| H3 | **db** | SSL defaults to `"disable"` for all database types |
| H4 | **server** | No HTTP timeouts (`ReadTimeout`, `WriteTimeout`, `ReadHeaderTimeout`) — DoS via slow-loris |
| H5 | **server** | No default request body size limit — memory exhaustion |
| H6 | **docker** | `regexp.MustCompile` panics on invalid user-supplied regex in `WaitForLog` |
| H7 | **docker** | `Start()` can be called twice, silently leaking containers |
| H8 | **docker** | `ConnectionString` is a format-string injection surface |
| H9 | **ssh** | `Password` field serializes to YAML (missing `yaml:"-"` tag) |
| H10 | **ssh** | `Start()` has no guard against double-call — leaks SSH client and listener |
| H11 | **otel** | Slice aliasing race in `LayeredSpanHelper.Start*` methods |
| H12 | **otel** | `GetTracer/GetMeter/GetLogger` allocate new no-op providers per call |
| H13 | **grpc** | Gateway-to-backend connection hardcoded as insecure |
| H14 | **grpc** | `waitForGRPCServer` never actually probes the port — dead retry loop |
| H15 | **grpc** | No `ReadHeaderTimeout` on H2C `http.Server` |
| H16 | **rest** | Full response body (may contain tokens/PII) embedded in errors and logs |
| H17 | **rest** | Full request body stored in `RequestInfo` accessible to all middleware |
| H18 | **compress** | `UnGz` path traversal check is effectively dead code after `filepath.Clean` |
| H19 | **compress** | `Tar` uses `strings.Replace` instead of `filepath.Rel` — path stripping bug |
| H20 | **config** | Race condition on shared `*viper.Viper` in `NestedEnvVars` |
| H21 | **config** | Silent data loss for multi-segment env var keys |
| H22 | **temporal** | `StartAll` partial failure leaves already-started workers running |
| H23 | **temporal** | Lock upgrade TOCTOU in `Close` |
| H24 | **argo** | `BuildWithEntrypoint` mutates shared builder state |

---

## Security Findings by Category

### Credential & Secret Handling

| Package | Risk | Finding |
|---------|------|---------|
| **db** | High | Public `Dsn()` exposes plaintext passwords; errors may leak connection strings |
| **ssh** | High | `Password` field serializable to YAML; test file embeds real Ed25519 key |
| **rest** | High | Response/request bodies (may contain tokens) stored in errors/logs with no truncation |
| **argo** | Medium | Auth token stored as plain string; no protection against reflection exposure |
| **temporal** | Medium | No TLS or auth credential support in `Config` |

### Transport Security

| Package | Risk | Finding |
|---------|------|---------|
| **grpc** | Critical | `WithTLS()` is a stub — server always runs plaintext |
| **db** | Medium | SSL defaults to `"disable"` for all databases |
| **rest** | Medium | No TLS configuration surface; consumers cannot enforce minimum version |
| **ssh** | Medium | No warning when `InsecureIgnoreHostKey=true` |
| **temporal** | Medium | All connections to Temporal server are plaintext |

### Injection Risks

| Package | Risk | Finding |
|---------|------|---------|
| **argo** | High | Shell injection via unsanitized `fmt.Sprintf` in pattern functions |
| **docker** | High | Format-string injection in `ConnectionString` |
| **argo** | Medium | Workflow expression injection via unvalidated `When()` strings |
| **rest** | Medium | No URL validation — SSRF surface |

### DoS / Resource Exhaustion

| Package | Risk | Finding |
|---------|------|---------|
| **server** | High | No HTTP timeouts; no body size limit |
| **grpc** | High | No `ReadHeaderTimeout` on H2C server |
| **compress** | Critical | Silent truncation on decompression bomb (no error returned) |
| **compress** | Medium | `UnTarGzBase64` buffers entire archive in memory |

### Information Disclosure

| Package | Risk | Finding |
|---------|------|---------|
| **grpc** | Critical | Reflection enabled by default — exposes full service schema |
| **grpc** | Low | `X-Gateway-Version` header leaks implementation details |
| **logging** | Low | `.Caller()` exposes source file paths unconditionally |
| **logging** | Medium | Log files created world-readable (`0644`) |

---

## Per-Package Summary

| Package | Critical | High | Medium | Low | Top Priority Fix |
|---------|----------|------|--------|-----|-----------------|
| **otel** | 0 | 2 | 6 | 8 | Slice aliasing in `Start*` methods |
| **config** | 1 | 2 | 3 | 4 | Panic on negative `keyDepth` |
| **logging** | 0 | 2 | 6 | 5 | Fix README signatures; file perms `0600` |
| **db** | 0 | 2 | 7 | 6 | Make `Dsn()` unexported; SSL default `require` |
| **docker** | 0 | 3 | 6 | 7 | `regexp.Compile` instead of `MustCompile` |
| **server** | 0 | 2 | 5 | 6 | Add HTTP timeouts |
| **grpc** | 2 | 3 | 9 | 7 | Implement or remove TLS; reflection default off |
| **rest** | 0 | 3 | 7 | 7 | Truncate response body in errors |
| **concurrent** | 0 | 2 | 5 | 5 | Consider using `errgroup` |
| **temporal** | 2 | 2 | 8 | 7 | Fix `StartAll` rollback; TLS support |
| **ssh** | 2 | 3 | 6 | 6 | Mutex on `t.client`; shutdown channel |
| **compress** | 2 | 2 | 4 | 5 | Error on size limit hit; fix path traversal |
| **argo** | 2 | 4 | 7 | 7 | Fix `mustParseQuantity`; shell escaping |
| **retry** | 0 | 3 | 6 | 6 | Guard nil `lastErr`; deduplicate `Do`/`DoWithNotify` |
| **base32** | 0 | 0 | 3 | 4 | Reject empty string in `CalculateChecksum` |

---

## Top 10 Recommended Actions

1. **grpc**: Implement TLS or return error from `WithTLS()` — false security is worse than no security
2. **ssh**: Add mutex protection on `t.client` and shutdown channel for accept loop
3. **compress**: Return error when `LimitReader` truncates; add `EvalSymlinks` defense
4. **db**: Make `Dsn()` unexported; change SSL default to `"require"`
5. **server**: Set `ReadHeaderTimeout` (5s), `ReadTimeout` (30s), `WriteTimeout` (30s), body limit
6. **docker**: Replace `MustCompile` with `Compile`; guard `Start()` idempotency
7. **config**: Add negative `keyDepth` guard; fix multi-segment field name truncation
8. **argo**: Replace `mustParseQuantity` with error-returning variant; add shell escaping in patterns
9. **rest**: Add configurable response body truncation in errors; copy headers before OTel injection
10. **grpc**: Change reflection default to `false`; fix `waitForGRPCServer` to actually probe the port

---

See individual package reports in `docs/reviews/` for full details.
