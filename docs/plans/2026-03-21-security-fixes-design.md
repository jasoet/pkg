# Security Fixes Design — Critical & High Findings

**Date:** 2026-03-21
**Scope:** 8 critical + 24 high findings across 10 packages
**Source:** `docs/CODE_REVIEW_SUMMARY.md` and `docs/reviews/`

## Overview

Fix all critical and high-severity findings from the code review audit. Changes are grouped by package into one branch per package, executed in dependency order.

## Batching Strategy

| Branch | Package | C | H | Complexity |
|--------|---------|---|---|------------|
| `fix/otel-safety` | otel | 0 | 2 | Small |
| `fix/config-safety` | config | 1 | 2 | Small |
| `fix/compress-security` | compress | 2 | 2 | Medium |
| `fix/db-security` | db | 0 | 3 | Small |
| `fix/docker-safety` | docker | 0 | 3 | Small |
| `fix/server-security` | server | 0 | 2 | Small |
| `fix/rest-security` | rest | 0 | 2 | Small |
| `fix/grpc-security` | grpc | 2 | 3 | Medium |
| `fix/ssh-security` | ssh | 2 | 3 | Medium |
| `fix/argo-safety` | argo | 1 | 1 | Small |

Packages with no critical/high findings (logging, concurrent, temporal, retry, base32) are deferred to a future effort.

## Execution Order

Bottom-up by dependency (packages imported by others first):

1. otel (foundational)
2. config (foundational)
3. compress (leaf)
4. db (depends on otel)
5. docker (depends on otel)
6. server (depends on otel)
7. rest (depends on otel)
8. grpc (depends on otel, server)
9. ssh (depends on otel)
10. argo (depends on otel)

Each branch merges to main before the next starts.

## Fix Details

### 1. otel (0C + 2H)

**H11 — Slice aliasing in `LayeredSpanHelper.Start*` methods**
- File: `instrumentation.go` (5 methods)
- Fix: Replace `append([]Field{...}, fields...)` with pre-allocated slice:
  ```go
  allFields := make([]Field, 0, 1+len(fields))
  allFields = append(allFields, F("layer", "handler"))
  allFields = append(allFields, fields...)
  ```
- Apply to all five `Start*` methods

**H12 — No-op provider allocation per call**
- File: `config.go`
- Fix: Add package-level singletons:
  ```go
  var (
      noopTracerProvider = noopt.NewTracerProvider()
      noopMeterProvider  = noopm.NewMeterProvider()
      noopLoggerProvider = noopl.NewLoggerProvider()
  )
  ```
- Update `GetTracer`, `GetMeter`, `GetLogger` to use singletons

**Tests:** Verify existing tests pass. Add benchmark to confirm allocation reduction.

### 2. config (1C + 2H)

**C7 — Panic on negative `keyDepth`**
- File: `config.go`
- Fix: Add guard at top of `NestedEnvVars`:
  ```go
  if keyDepth < 0 {
      return
  }
  ```
- Test: Add `TestNestedEnvVars_NegativeKeyDepth`

**H20 — Race condition documentation**
- File: `config.go`
- Fix: Add doc comment to `NestedEnvVars` stating it is not goroutine-safe for a shared `*viper.Viper`

**H21 — Multi-segment field name truncation**
- File: `config.go`
- Fix: Replace `keyParts[keyDepth+1]` with `strings.Join(keyParts[keyDepth+1:], "_")`
- Test: Add `TestNestedEnvVars_MultiSegmentFieldName`

### 3. compress (2C + 2H)

**C5 — Silent truncation on decompression bomb**
- Files: `gz.go`, `tar.go`
- Fix: After `io.Copy` with `LimitReader`, check if `written >= maxFileSize`. Probe underlying reader for remaining data. Return `ErrSizeLimitExceeded`.
- Define sentinel errors:
  ```go
  var (
      ErrSizeLimitExceeded = errors.New("file size exceeds maximum allowed size")
      ErrPathTraversal     = errors.New("path traversal detected")
  )
  ```
- Test: Update `TestUnTarZipBombProtection` to assert error is returned

**C6 — Zip slip via symlink TOCTOU**
- File: `tar.go`
- Fix: Before writing each file, call `filepath.EvalSymlinks` on parent directory and verify result is still under `destinationDir`
- Test: Add test with symlink in destination dir pointing outside

**H18 — `UnGz` path traversal check broken**
- File: `gz.go`
- Fix: Replace `strings.Contains(cleanDst, "..")` with absolute path requirement:
  ```go
  if !filepath.IsAbs(dst) {
      return 0, fmt.Errorf("%w: destination must be an absolute path", ErrPathTraversal)
  }
  ```
- Test: Add test with relative path containing `..`

**H19 — `Tar` path stripping bug**
- File: `tar.go`
- Fix: Replace `strings.Replace(file, sourceDirectory, "", -1)` with `filepath.Rel(sourceDirectory, file)`
- Test: Add test with source directory name appearing in nested path

### 4. db (0C + 3H)

**H1 — Public `Dsn()` exposes password**
- File: `pool.go`
- Fix: Rename `Dsn()` to `dsn()`. Add `RedactedDsn()` that masks password with `***`
- Test: Verify `RedactedDsn()` output contains `***` and not actual password

**H2 — Error leaks DSN with password**
- File: `pool.go`
- Fix: Wrap `Ping`, `gorm.Open`, `db.DB()` errors:
  ```go
  return nil, fmt.Errorf("failed to connect to database at %s:%d: %w", c.Host, c.Port, err)
  ```
- Test: Verify error message does not contain password string

**H3 — SSL defaults to `"disable"`**
- File: `pool.go`
- Fix: Change `effectiveSSLMode()` default from `"disable"` to `"require"`
- Test: Update existing tests that rely on `"disable"` default

### 5. docker (0C + 3H)

**H6 — `MustCompile` panics on invalid regex**
- File: `wait.go`
- Fix: Change `WaitForLog` to use `regexp.Compile`. Change constructor to return `(*waitForLog, error)` or validate in `WaitUntilReady`.
- Test: Add test with invalid regex pattern

**H7 — Double `Start()` leaks containers**
- File: `executor.go`
- Fix: Add guard at top of `Start()`:
  ```go
  if e.containerID != "" {
      return fmt.Errorf("container already started: %s", e.containerID)
  }
  ```
- Test: Add test calling `Start()` twice

**H8 — Format-string injection in `ConnectionString`**
- File: `network.go`
- Fix: Replace `fmt.Sprintf(template, endpoint)` with `strings.ReplaceAll(template, "{{endpoint}}", endpoint)`
- Test: Add test with `%v` in template string

### 6. server (0C + 2H)

**H4 — No HTTP timeouts**
- File: `server.go`
- Fix: After `echo.New()`, set defaults on `e.Server`:
  ```go
  e.Server.ReadHeaderTimeout = 5 * time.Second
  e.Server.ReadTimeout = 30 * time.Second
  e.Server.WriteTimeout = 30 * time.Second
  e.Server.IdleTimeout = 120 * time.Second
  ```
- Test: Verify `echo.Server` fields are set after `setupEcho`

**H5 — No body size limit**
- File: `server.go`
- Fix: Add `e.Use(middleware.BodyLimit("4M"))` as default in `setupEcho`
- Test: Verify middleware is registered

### 7. rest (0C + 2H)

**H16 — Full response body in errors**
- Files: `client.go`, `config.go`
- Fix: Add `MaxResponseBodyLog int` field to `Config` (default 1024). Truncate `response.String()` before storing in `RequestInfo.Response` and error types:
  ```go
  body := response.String()
  if len(body) > c.restConfig.MaxResponseBodyLog {
      body = body[:c.restConfig.MaxResponseBodyLog] + "...(truncated)"
  }
  ```
- Test: Verify long response bodies are truncated in errors

**H17 — Nil guard for `WithOTelConfig`**
- File: `client.go`
- Fix: Add nil guard:
  ```go
  if client.restConfig != nil {
      client.restConfig.OTelConfig = cfg
  }
  ```
- Test: Verify no panic when applied to raw `&Client{}`

### 8. grpc (2C + 3H)

**C3 — `WithTLS()` is a no-op stub**
- File: `config.go`, `server.go`
- Fix: Remove `WithTLS()` option, `enableTLS`, `certFile`, `keyFile` fields entirely. No false security.
- Test: Verify `WithTLS` no longer exists (compile-time)

**C4 — Reflection enabled by default**
- File: `config.go`
- Fix: Change `enableReflection: true` to `enableReflection: false`
- Test: Update tests that rely on reflection being on

**H13 — Gateway-to-backend insecure**
- File: `echo_gateway.go`
- Fix: Accept optional `[]grpc.DialOption` parameter in `SetupGatewayForSeparate`. Document that callers must pass TLS credentials if needed.

**H14 — `waitForGRPCServer` dead retry loop**
- File: `echo_gateway.go`
- Fix: Replace `grpc.NewClient` with `net.DialTimeout("tcp", endpoint, 1*time.Second)`
- Test: Add test verifying actual connection probe

**H15 — No `ReadHeaderTimeout` on H2C server**
- File: `server.go`
- Fix: Add `ReadHeaderTimeout: 5 * time.Second` to H2C `http.Server`
- Test: Verify field is set

### 9. ssh (2C + 3H)

**C1/C2 — Data race + no shutdown signal**
- File: `tunnel.go`
- Fix: Add `stopCh chan struct{}` and `wg sync.WaitGroup` to `Tunnel` struct. Protect all `t.client` access with `t.mu`. Accept loop selects on `stopCh`. `forward()` adds to `wg`. `Close()` closes `stopCh`, waits `wg.Wait()`, then closes client under lock.
- Test: Add test for concurrent `Start()`/`Close()`, run with `-race`

**H9 — Password serializes to YAML**
- File: `tunnel.go`
- Fix: Change `Password` tag from `yaml:"password"` to `yaml:"-" mapstructure:"-"`
- Test: Verify YAML marshal does not include password

**H10 — Double `Start()` leaks resources**
- File: `tunnel.go`
- Fix: Add `if t.client != nil { return error }` guard at top of `Start()`
- Test: Add test calling `Start()` twice

### 10. argo (1C + 1H)

**C8 — `mustParseQuantity` panics**
- File: `builder/template/script.go`
- Fix: Replace `mustParseQuantity` with `resource.ParseQuantity`, propagate error through `Templates()` return
- Test: Add test with invalid quantity string

**H24 — `BuildWithEntrypoint` mutates shared state**
- File: `builder/builder.go`
- Fix: Copy `b.templates` before appending:
  ```go
  templates := make([]v1alpha1.Template, len(b.templates))
  copy(templates, b.templates)
  templates = append(templates, exitHandler)
  ```
- Test: Add test calling `BuildWithEntrypoint` twice on same builder

## Testing & Verification

For each package:
1. Run `task test` — confirm no regressions
2. Add targeted unit tests for each fix
3. Run `task lint` — confirm no linter issues
4. Run tests with `-race` flag for concurrency fixes (ssh, otel)

## Out of Scope

- Medium and low findings (deferred to future effort)
- Packages with no critical/high findings: logging, concurrent, temporal, retry, base32
- New feature additions (e.g., implementing actual TLS for grpc — we only remove the false stub)
