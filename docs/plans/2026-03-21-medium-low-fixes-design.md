# Medium & Low Fixes Design

**Date:** 2026-03-21
**Scope:** ~172 medium and low findings across 15 packages
**Excluded:** 3 breaking API changes (see `2026-03-21-breaking-api-changes-design.md`)

## Strategy

One commit per package, same dependency order as the critical/high round. Most fixes are 1-5 line changes. TDD where testable.

## Execution Order

1. otel (6M + 8L)
2. config (3M + 4L)
3. logging (6M + 5L)
4. base32 (3M + 4L)
5. compress (3M + 5L)
6. concurrent (5M + 5L)
7. db (6M + 6L)
8. docker (5M + 7L)
9. server (4M + 6L) — skip M4 (breaking)
10. rest (7M + 7L)
11. retry (6M + 6L)
12. grpc (8M + 7L) — skip M6 (breaking)
13. ssh (6M + 6L)
14. temporal (8M + 7L)
15. argo (6M + 7L) — skip M1 (breaking)

## Per-Package Fix Summary

### 1. otel

| ID | Fix |
|----|-----|
| M1 | Add comment clarifying mutation-after-sharing is caller's responsibility (or make With* copy-on-write) |
| M2 | Log warning to stderr when logger provider creation fails |
| M3 | Validate OTLP endpoint is non-empty when insecure flag is set |
| M4 | Document that consoleExporter uses SimpleProcessor (synchronous) |
| M5 | Use no-op tracer (not global) when no config in context |
| M6 | Remove dead `fields` field from `LayerContext` |
| L1-L8 | Remove unreachable branch in `toString`, set `Timestamp` in `emitOTel`, use `fmt.Sprint` in default case, add PII note to examples, validate endpoint format, document no-op Shutdown/ForceFlush |

### 2. config

| ID | Fix |
|----|-----|
| M1 | Document `keyDepth` semantics clearly in godoc |
| M2 | Document that only first `envPrefix` is used (or change to single string with empty default) |
| M3 | Wrap errors with `fmt.Errorf("config: ...: %w", err)` |
| L1-L4 | Fix stale doc comment, use `t.Setenv`, add error path tests, add `mapstructure` tags to test struct |

### 3. logging

| ID | Fix |
|----|-----|
| M1 | Document `LogLevel` is for cross-package use by `otel` package |
| M2 | Change file permissions from `0o644` to `0o600` |
| M3 | Document PID field rationale or make conditional |
| M4 | Validate `serviceName` is non-empty |
| M5 | Fix README `InitializeWithFile` signature and examples |
| M6 | Document that `ContextLogger` allocates — callers should cache |
| L1-L5 | Add unknown-bits validation, use `t.Cleanup` for logger restore, document omitted log levels, add `NoColor` TTY detection comment, document `Caller()` performance note |

### 4. base32

| ID | Fix |
|----|-----|
| M1 | Return error from `CalculateChecksum` on empty string |
| M2 | Move `NormalizeBase32` doc comment above the function |
| M3 | Fix README function signatures and examples |
| L1-L4 | Remove unreachable overflow check, fix O(N²) prepend (reverse-build), fix sub-test names, add `\t\n\r` to normalizer |

### 5. compress

| ID | Fix |
|----|-----|
| M2 | Validate `sourceDirectory` is a directory in `Tar` |
| M3 | Stream `UnTarGzBase64` with `base64.NewDecoder` instead of buffering |
| M4 | Simplify file mode sanitization — use `header.FileInfo().Mode().Perm()` |
| L1-L5 | Remove unnecessary `os.Stat` before `MkdirAll`, document backslash check, fix named returns, fix skipped test assertion, add `UnGz` size options (use `DefaultMaxFileSize` constant) |

### 6. concurrent

| ID | Fix |
|----|-----|
| M1 | Document that only first error is returned (or use `errors.Join`) |
| M3 | Use `%w` when recovered value is an error |
| M4 | Document closer goroutine behavior |
| M5 | Add nil function guard with clear error |
| L1-L5 | Document wg.Add pattern, document map iteration order, move test types to `_test` package, fix README `Result[T]` type, loosen timing assertions |

### 7. db

| ID | Fix |
|----|-----|
| M2 | Validate `SSLMode` values per database type |
| M3 | Check gauge creation errors in `collectPoolMetrics` |
| M4 | Document `SQLDB()` resource management or fix to reuse pool |
| M5 | Use `PingContext` with timeout |
| M6 | Add `validate` tag to `Port` field |
| M7 | Add `MaxIdleConns <= MaxOpenConns` cross-validation |
| L1-L6 | Document PostgreSQL-only migration, simplify `setupMigration` return, fix `DatabaseType` naming, close pool on OTel failure, replace `time.Sleep` in tests, remove passwords from README examples |

### 8. docker

| ID | Fix |
|----|-----|
| M1 | Add doc warning about `BindMounts` security implications |
| M2 | Add doc warnings for `WithPrivileged`/`WithCapAdd` |
| M3 | Fix `WaitForLog` chunk boundary — use `bufio.Scanner` or accumulate buffer |
| M4 | Document `Stop`/`Restart` TOCTOU behavior |
| M5 | Check `errdefs.IsNotFound` before falling through to pull |
| L1-L7 | Extract port-parsing helper, use `defaultProtocol` constant, fix `Status` field mapping, set `client=nil` after close, fix benchmark error handling, document multiWait timeout, populate `LogEntry.Timestamp` |

### 9. server (skip M4 — breaking)

| ID | Fix |
|----|-----|
| M1 | Document that `Operation` panics propagate to caller |
| M2 | Apply `ShutdownTimeout` to entire stop sequence including callback |
| M3 | Validate port range (1-65535) or document port-0 behavior |
| M5 | Document health endpoint auth/rate-limit behavior |
| L1-L6 | Remove duplicate `syscall.SIGINT`, rename to `newHTTPServer`, use call context for logger, add struct tags, fix README error handler example, replace `if err == nil` with `require.NoError` in test |

### 10. rest

| ID | Fix |
|----|-----|
| M1 | Add `WithURLValidator` option or document SSRF responsibility |
| M2 | Add `WithTransport` option or document TLS via `GetRestClient()` |
| M3 | Document `GetRestClient()` thread-safety limitations |
| M4 | Remove unnecessary lock in option functions |
| M5 | Copy headers map before OTel propagator injection |
| M6 | Add retry condition: only retry on network errors and 5xx |
| M7 | Use `method` as span name, move full URL to attribute |
| L1-L7 | Include status in `UnauthorizedError.Error()`, remove unused `IsForbidden`, fix README defaults, add lock for `restConfig` read, check metric errors, copy `RequestInfo.Headers`, document response buffering |

### 11. retry

| ID | Fix |
|----|-----|
| M1 | Extract shared `doRetry` internal function to eliminate duplication |
| M2 | Create logger when logging is enabled even if tracing is disabled |
| M3 | Validate `RandomizationFactor` in [0,1) and `Multiplier` > 1 |
| M4 | Clarify "attempts" vs "retries" in error message |
| M5 | Remove pointless `notifyFunc` wrapper |
| M6 | Document attempt numbering convention |
| L1-L6 | Extract instrumentation name constant, fix README generics claim, add `RandomizationFactor` test, add `WithOTel` test, document span lifecycle, loosen timing assertions |

### 12. grpc (skip M6 — breaking)

| ID | Fix |
|----|-----|
| M1 | Fix health check cache thundering herd with double-checked locking |
| M2 | Copy checker map before invoking — release lock first |
| M3 | Implement W3C Trace Context propagation from gRPC metadata |
| M4 | Add `StreamServerInterceptor` for OTel |
| M5 | Add `WithCORSConfig` option |
| M7 | Make liveness endpoint return 200 unconditionally |
| M8 | Validate port strings as numeric |
| M9 | Remove `X-Gateway-Version` header |
| L1-L7 | Use structured logger, close listener on error, fix README API references, implement real memory health checker or remove, update deprecated test APIs, handle `HealthStatusUnknown`, document route ordering |

### 13. ssh

| ID | Fix |
|----|-----|
| M1 | Accept `context.Context` in `Start()` |
| M2 | Validate config fields in `New()` |
| M3 | Log `io.Copy` errors at debug level |
| M4 | Log warning when `InsecureIgnoreHostKey=true` |
| M5 | Improve `getHostKeyCallback` error message |
| M6 | Fix README limitations section — key-based auth is implemented |
| L1-L6 | Add `CloseWrite` half-close, fix error capitalization, replace committed test key, replace `time.Sleep` in tests, expose `LocalAddr()`, validate conflicting `InsecureIgnoreHostKey` + `KnownHostsFile` |

### 14. temporal

| ID | Fix |
|----|-----|
| M1 | Remove unnecessary logger from `DefaultConfig` |
| M2 | Document that `Start(ctx)` ctx is for logging only |
| M3 | Replace `time.Sleep` with log/port readiness strategy in testcontainer |
| M4 | Accept namespace as parameter in `NewWorkflowManager(client.Client)` |
| M5 | Store cancel function from `context.WithTimeout` |
| M6 | Enable metrics consistently for `ScheduleManager` |
| M7 | Document `GetClient()` lifecycle implications |
| M8 | Use `status.String()` instead of hardcoded strings in `GetDashboardStats` |
| L1-L7 | Document CallerSkip `+2`, fix taskQueue log label, use `context.Background()` in cleanup, remove unused types, change metrics bind to `127.0.0.1`, wrap errors consistently, fix test isolation |

### 15. argo (skip M1 — breaking)

| ID | Fix |
|----|-----|
| M2 | Fix `Script.Source()` to populate `ScriptTemplate.Source` |
| M3 | Add `yaml:"-"` to `AuthToken` (already has it — verify) |
| M4 | Document `WithConfig` shallow copy behavior |
| M5 | Document exit handler name-based prioritization or make opt-in |
| M6 | Add shell escaping in pattern functions (`parallel.go`) |
| M7 | Reentrant test for `BuildWithEntrypoint` (may already exist from critical fix) |
| L1-L7 | Replace string-switch with counter map, pass context to `buildClientConfig`, remove wasteful factory loggers, fix `SubmitWorkflow` log field name, remove hardcoded `sleep 2`, replace `time.Sleep` in tests, add `WithRetry` to Script |

## Testing

- `task test` after each package commit
- `task lint` at the end
- Run with `-race` for otel, concurrent, ssh, docker packages
