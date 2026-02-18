# Repository Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all compilation errors, logic bugs, code smells, and documentation inconsistencies found during the codebase review.

**Architecture:** Fixes are organized by severity (critical compilation errors first, then logic bugs, then code quality improvements). Each task is independent unless noted. The repository's existing patterns (functional options, OTel integration, structured logging via zerolog/otel.LogHelper) guide all changes.

**Tech Stack:** Go 1.24+, zerolog, OpenTelemetry, Echo v4, testify

---

## Phase 1: Critical Fixes (Compilation Errors)

### Task 1: Fix examples/docker - Multiple main() Declarations

**Files:**
- Create: `examples/docker/basic/main.go`
- Create: `examples/docker/database/main.go`
- Create: `examples/docker/logs/main.go`
- Create: `examples/docker/multi_container/main.go`
- Delete: `examples/docker/basic.go`
- Delete: `examples/docker/database.go`
- Delete: `examples/docker/logs.go`
- Delete: `examples/docker/multi_container.go`

**Context:** Four files in `examples/docker/` all declare `func main()` in `package main`. Go only allows one `main()` per package. Move each into its own subdirectory.

**Step 1:** Create subdirectories and move each file's content into `examples/docker/<name>/main.go`, keeping the `//go:build example` tag and all existing code.

**Step 2:** Delete the original files from `examples/docker/`.

**Step 3:** Verify compilation:
```bash
go build -o /dev/null -tags=example ./examples/docker/...
```
Expected: SUCCESS (no output)

**Step 4:** Commit
```bash
git add examples/docker/
git commit -m "fix(examples): split docker examples into separate subdirectories"
```

---

### Task 2: Fix examples/argo - Multiple main() Declarations

**Files:**
- Create: `examples/argo/basic/main.go` (from `main.go`)
- Create: `examples/argo/templates/main.go` (from `templates_example.go`)
- Create: `examples/argo/patterns/main.go` (from `patterns_example.go`)
- Create: `examples/argo/operations/main.go` (from `operations_example.go`)
- Create: `examples/argo/builder/main.go` (from `builder_example.go`)
- Create: `examples/argo/advanced/main.go` (from `advanced_features_example.go`)
- Delete: `examples/argo/main.go`, `examples/argo/templates_example.go`, `examples/argo/patterns_example.go`, `examples/argo/operations_example.go`, `examples/argo/builder_example.go`, `examples/argo/advanced_features_example.go`

**Context:** Six files in `examples/argo/` all declare `func main()`. Same fix as Task 1.

**Step 1:** Create subdirectories and move each file into its own `main.go`.

**Step 2:** Delete original files (keep `README.md`).

**Step 3:** Verify:
```bash
go build -o /dev/null -tags=example ./examples/argo/...
```
Expected: SUCCESS

**Step 4:** Commit
```bash
git add examples/argo/
git commit -m "fix(examples): split argo examples into separate subdirectories"
```

---

### Task 3: Fix examples/logging - Multiple main() + Undefined Functions

**Files:**
- Create: `examples/logging/otel/main.go` (from `otel_example.go`, rewritten)
- Delete: `examples/logging/otel_example.go`
- Verify: `examples/logging/example.go` (should remain as-is)

**Context:** `examples/logging/otel_example.go` has two problems:
1. Multiple `main()` in same package as `example.go`
2. Calls `logging.NewLoggerProvider()` which does not exist (5 occurrences)
3. Line 207 passes `false` (bool) where `otel.LoggerProviderOption` is expected

**Step 1:** Move `otel_example.go` to `examples/logging/otel/main.go`.

**Step 2:** Replace all `logging.NewLoggerProvider(serviceName, debug)` calls with the actual API: `otel.NewLoggerProviderWithOptions(serviceName, ...options)`. Use `otel.WithConsoleExporter()` for console output. Remove the invalid `false` argument on line 207.

**Step 3:** Verify:
```bash
go build -o /dev/null -tags=example ./examples/logging/...
```
Expected: SUCCESS

**Step 4:** Commit
```bash
git add examples/logging/
git commit -m "fix(examples): fix logging otel example - use correct API and split into subdirectory"
```

---

### Task 4: Fix examples/server - Missing OTelConfig Field

**Files:**
- Modify: `examples/server/example.go:128` (remove or comment out OTelConfig reference)

**Context:** `examples/server/example.go:128` references `config.OTelConfig` but `server.Config` (defined in `server/server.go:23-35`) has no such field. The server package was stripped of OTel in commit `4df260a`.

**Step 1:** In `examples/server/example.go`, remove or comment out the line `config.OTelConfig = otelConfig` (line 128). Add a comment explaining OTel is configured at middleware level, not server config level.

**Step 2:** Verify:
```bash
go build -o /dev/null -tags=example ./examples/server/...
```
Expected: SUCCESS

**Step 3:** Commit
```bash
git add examples/server/
git commit -m "fix(examples): remove reference to non-existent server.Config.OTelConfig"
```

---

## Phase 2: Logic Bug

### Task 5: Fix Inverted Metrics Condition in temporal/client.go

**Files:**
- Modify: `temporal/client.go:39`
- Test: `temporal/client_test.go` (if exists, verify behavior)

**Context:** In `temporal/client.go:39`, the condition `if !metricsEnabled` sets up Prometheus metrics when the parameter is `false`, and skips them when `true`. `NewClient(config)` on line 63-64 calls `NewClientWithMetrics(config, true)`, meaning the default path never sets up Prometheus. The boolean is inverted.

**Step 1:** Change line 39 from:
```go
if !metricsEnabled {
```
to:
```go
if metricsEnabled {
```

**Step 2:** Run existing tests:
```bash
go test -race -count=1 ./temporal/...
```
Expected: PASS

**Step 3:** Commit
```bash
git add temporal/client.go
git commit -m "fix(temporal): correct inverted metricsEnabled condition in NewClientWithMetrics"
```

---

## Phase 3: Code Quality - Replace panic/os.Exit in Library Code

### Task 6: Replace panic() with Error Returns in logging Package

**Files:**
- Modify: `logging/logging.go:52-106` (change `InitializeWithFile` signature)
- Modify: `logging/logging.go:119-121` (change `Initialize` signature)
- Modify: `logging/logging_test.go` (update tests for new error returns)
- Modify: All callers in `examples/` that call `Initialize` or `InitializeWithFile`

**Context:** `logging/logging.go` has three `panic()` calls at lines 74, 79, 87. Library functions should return errors. This is a breaking API change - `InitializeWithFile` returns `void` currently and must return `error`.

**Step 1:** Write a failing test that expects `InitializeWithFile` to return an error when given invalid config (e.g., `OutputFile` with nil `fileConfig`).

**Step 2:** Change function signatures:
```go
func InitializeWithFile(serviceName string, debug bool, output OutputDestination, fileConfig *FileConfig) error
func Initialize(serviceName string, debug bool) error
```

**Step 3:** Replace all three `panic()` calls with `return fmt.Errorf(...)`:
- Line 74: `return fmt.Errorf("fileConfig with Path is required when OutputFile is specified")`
- Line 79: `return fmt.Errorf("failed to open log file %s: %w", fileConfig.Path, err)`
- Line 87: `return fmt.Errorf("at least one output destination must be specified")`

Add `return nil` at the end of `InitializeWithFile`.

**Step 4:** Update `Initialize` to propagate the error:
```go
func Initialize(serviceName string, debug bool) error {
    return InitializeWithFile(serviceName, debug, OutputConsole, nil)
}
```

**Step 5:** Update all callers in `examples/` to handle the returned error (replace bare calls with `if err := logging.Initialize(...); err != nil { ... }`).

**Step 6:** Update tests in `logging/logging_test.go`.

**Step 7:** Verify:
```bash
go test -race -count=1 ./logging/...
go build -o /dev/null -tags=example ./examples/logging/...
```
Expected: PASS

**Step 8:** Commit
```bash
git add logging/ examples/
git commit -m "fix(logging): return errors instead of panicking in Initialize functions

BREAKING: Initialize and InitializeWithFile now return error"
```

---

### Task 7: Replace os.Exit and fmt.Print in server Package

**Files:**
- Modify: `server/server.go:95-105` (start method)
- Modify: `server/server.go:107-116` (stop method)
- Modify: `server/server.go:118-133` (StartWithConfig)
- Modify: `server/server_test.go` (update tests)
- Modify: `examples/server/example.go` (update for new error-returning API)

**Context:** `server/server.go` uses `fmt.Printf` for lifecycle logging (lines 99, 101, 108) and `os.Exit(1)` for fatal errors (lines 102, 131). A library should never call `os.Exit` - it prevents callers from cleanup. Replace with error returns and structured logging.

**Step 1:** Add `"github.com/rs/zerolog/log"` import. Replace `fmt.Printf/Println` calls with `log.Info().Msg(...)` and `log.Error().Err(err).Msg(...)`.

**Step 2:** Change `start()` to return `error` instead of calling `os.Exit`:
```go
func (s *httpServer) start() error {
    s.config.Operation(s.echo)
    errCh := make(chan error, 1)
    go func() {
        log.Info().Int("port", s.config.Port).Msg("Starting server")
        if err := s.echo.Start(fmt.Sprintf(":%v", s.config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
        }
        close(errCh)
    }()
    // Give the server a moment to fail on bind errors
    select {
    case err := <-errCh:
        return fmt.Errorf("failed to start server: %w", err)
    case <-time.After(100 * time.Millisecond):
        return nil
    }
}
```

**Step 3:** Change `StartWithConfig` to return `error`:
```go
func StartWithConfig(config Config) error {
    server := newHttpServer(config)
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    if err := server.start(); err != nil {
        return err
    }
    <-ctx.Done()
    return server.stop()
}
```

**Step 4:** Change `Start` to return `error`:
```go
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) error {
    config := DefaultConfig(port, operation, shutdown)
    config.Middleware = middleware
    return StartWithConfig(config)
}
```

**Step 5:** Update `stop()` to use structured logging:
```go
func (s *httpServer) stop() error {
    log.Info().Msg("Gracefully shutting down server")
    s.config.Shutdown(s.echo)
    ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
    defer cancel()
    return s.echo.Shutdown(ctx)
}
```

**Step 6:** Update `examples/server/example.go`:
- The example currently shows `server.StartWithConfig(config)` as a demonstration (printed, not called). Since the return type changes to `error`, update all display strings and any actual calls to reflect the new signature. For example:
```go
fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
```
- Also remove the `otelCfg` variable and `config.OTelConfig = otelCfg` assignment in `otelConfigExample()` (line 121-128) since `server.Config` has no `OTelConfig` field. Rewrite this example to show OTel configuration via Echo middleware instead.

**Step 7:** Update tests in `server/server_test.go`.

**Step 8:** Verify:
```bash
go test -race -count=1 ./server/...
go build -o /dev/null -tags=example ./examples/server/...
```
Expected: PASS

**Step 9:** Commit
```bash
git add server/ examples/server/
git commit -m "fix(server): return errors instead of calling os.Exit, use structured logging

BREAKING: Start, StartWithConfig now return error"
```

---

## Phase 4: Code Quality - Replace fmt.Print in Library Code

### Task 8: Replace fmt.Print with Structured Logging in ssh Package

**Files:**
- Modify: `ssh/tunnel.go:66,114,116,123,129`

**Context:** `ssh/tunnel.go` uses `fmt.Printf` and `fmt.Println` for error logging in production code. Replace with `zerolog/log` for consistency with the rest of the codebase.

**Step 1:** Add import `"github.com/rs/zerolog/log"`.

**Step 2:** Replace all `fmt.Printf`/`fmt.Println` calls:
- Line 66: `fmt.Printf("WARNING: ...")` -> `log.Warn().Str("hostname", hostname).Str("keyType", key.Type()).Msg("Unable to verify host key")`
- Line 114: `fmt.Println("SSH tunnel dial error:", err)` -> `log.Error().Err(err).Str("remoteAddr", remoteAddr).Msg("SSH tunnel dial error")`
- Line 116: `fmt.Printf("Error closing local connection: %v\n", closeErr)` -> `log.Error().Err(closeErr).Msg("Error closing local connection")`
- Line 123: `fmt.Println("SSH tunnel copy error:", err)` -> `log.Error().Err(err).Msg("SSH tunnel copy error")`
- Line 129: `fmt.Println("SSH tunnel copy error:", err)` -> `log.Error().Err(err).Msg("SSH tunnel copy error")`

**Step 3:** Remove `"fmt"` from imports (if no longer used).

**Step 4:** Verify:
```bash
go test -race -count=1 ./ssh/...
go vet ./ssh/...
```
Expected: PASS

**Step 5:** Commit
```bash
git add ssh/tunnel.go
git commit -m "fix(ssh): replace fmt.Print with structured zerolog logging"
```

---

### Task 9: Replace fmt.Printf in db/pool.go

**Files:**
- Modify: `db/pool.go:194`

**Context:** `db/pool.go:194` uses `fmt.Printf` for a metrics registration error in `collectPoolMetrics`. The `OTelConfig` is available on the receiver.

**Step 1:** Replace line 194:
```go
// Before:
fmt.Printf("Failed to register pool metrics callback: %v\n", err)

// After:
logger := pkgotel.NewLogHelper(context.Background(), c.OTelConfig,
    "github.com/jasoet/pkg/v2/db", "db.collectPoolMetrics")
logger.Error(err, "Failed to register pool metrics callback")
```

**Step 2:** Verify:
```bash
go test -race -count=1 ./db/...
go vet ./db/...
```
Expected: PASS

**Step 3:** Commit
```bash
git add db/pool.go
git commit -m "fix(db): replace fmt.Printf with otel.LogHelper in collectPoolMetrics"
```

---

## Phase 5: Fix Validate Tag and Remove Stub

### Task 10: Fix Invalid validate Tag on db.ConnectionConfig.Timeout

**Files:**
- Modify: `db/pool.go:36`

**Context:** `validate:"min=3s"` on a `time.Duration` field doesn't work as expected with `go-playground/validator`. The `min` tag compares numeric values. For `time.Duration` (which is `int64` nanoseconds), use the nanosecond value or switch to a custom validator. The simplest correct fix: remove the invalid tag since validation is never called by the library anyway.

**Step 1:** Change line 36 from:
```go
Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout" validate:"min=3s"`
```
to:
```go
Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
```

Also update `db/README.md:150` if it references this tag.

**Step 2:** Verify:
```bash
go test -race -count=1 ./db/...
```
Expected: PASS

**Step 3:** Commit
```bash
git add db/pool.go db/README.md
git commit -m "fix(db): remove invalid validate tag on Timeout duration field"
```

---

### Task 11: Remove DatabaseLoggingMiddleware Stub

**Files:**
- Modify: `rest/middleware.go:86-120` (remove `DatabaseLoggingMiddleware`)
- Modify: `rest/middleware_test.go` (remove related tests if any)

**Context:** `DatabaseLoggingMiddleware` is an exported stub with `// TODO: Implement actual database logging`. It does the same thing as `LoggingMiddleware`. Exporting unfinished placeholders as public API is misleading. Remove it.

**Step 1:** Remove `DatabaseLoggingMiddleware` struct, `NewDatabaseLoggingMiddleware`, `BeforeRequest`, and `AfterRequest` methods (lines 86-120).

**Step 2:** Check for any references to `DatabaseLoggingMiddleware` in tests or examples and remove them.

**Step 3:** Verify:
```bash
go test -race -count=1 ./rest/...
go build ./...
```
Expected: PASS

**Step 4:** Commit
```bash
git add rest/
git commit -m "fix(rest): remove unimplemented DatabaseLoggingMiddleware stub"
```

---

## Phase 6: Documentation Fixes

### Task 12: Fix CLAUDE.md Documentation Inaccuracies

**Files:**
- Modify: `CLAUDE.md`

**Context:** CLAUDE.md references functions that don't exist:
1. `config.Load[AppConfig]("config.yaml", "APP")` - only `LoadString[T]` exists
2. Shows `server.Config` with `OTelConfig` field (removed in recent commit)

**Step 1:** Replace `config.Load[AppConfig]("config.yaml", "APP")` example with:
```go
cfg, err := config.LoadString[AppConfig](yamlContent, "APP")
```

**Step 2:** Remove or update the server OTelConfig reference to match current API.

**Step 3:** Commit
```bash
git add CLAUDE.md
git commit -m "docs: fix CLAUDE.md references to non-existent functions"
```

---

### Task 13: Fix otel/doc.go Outdated API Example

**Files:**
- Modify: `otel/doc.go` (around line 39)

**Context:** `doc.go` shows `otel.Layers.StartService(ctx, cfg, "user", "CreateUser", ...)` but the actual signature is `StartService(ctx, component, operation, ...fields)` - no `cfg` parameter.

**Step 1:** Fix the example in doc.go to match the actual API signature.

**Step 2:** Verify:
```bash
go vet ./otel/...
```
Expected: PASS

**Step 3:** Commit
```bash
git add otel/doc.go
git commit -m "docs(otel): fix outdated API example in doc.go"
```

---

## Phase 7: Docker/Podman Test Resilience

### Task 14: Guard Docker Unit Tests Against Missing Container Runtime

**Files:**
- Create: `docker/testutil_test.go` (shared test helpers)
- Modify: `docker/wait_test.go`
- Modify: `docker/logs_test.go`
- Modify: `docker/executor_test.go` (if it has runtime-dependent tests)

**Context:** Unit tests in `docker/` fail when no container runtime is available (11 failures). These tests pull images and create containers, making them effectively dependent on a running daemon. The docker package already supports Podman transparently via `client.FromEnv` + `DOCKER_HOST` env var (`docker/executor.go:60`), so no code changes are needed for Podman support in the library itself.

**Podman compatibility note:** The docker package uses `client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())` which respects the `DOCKER_HOST` environment variable. To use Podman, set:
```bash
# Linux:
export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock
# macOS with podman machine:
export DOCKER_HOST=$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')
```
All Docker API calls used by this package (ContainerCreate, ContainerStart, ContainerStop, ContainerInspect, ImagePull, etc.) are fully supported by Podman's Docker-compatible REST API.

**Step 1:** Create `docker/testutil_test.go` with a shared helper that works with both Docker and Podman:
```go
package docker_test

import (
    "context"
    "testing"
    "time"

    "github.com/docker/docker/client"
)

// skipIfNoContainerRuntime skips the test if no Docker-compatible container runtime
// (Docker or Podman) is available. It respects DOCKER_HOST for Podman support.
func skipIfNoContainerRuntime(t *testing.T) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        t.Skip("Container runtime client not available:", err)
    }
    defer cli.Close()
    if _, err := cli.Ping(ctx); err != nil {
        t.Skip("Container runtime not running (set DOCKER_HOST for Podman):", err)
    }
}
```

**Step 2:** Add `skipIfNoContainerRuntime(t)` as the first line of each test that requires a container daemon. This covers all 11 failing tests in `wait_test.go` and `logs_test.go`, plus any in `executor_test.go`.

**Step 3:** Verify with no runtime available:
```bash
go test -race -count=1 -v ./docker/... 2>&1 | head -40
```
Expected: Tests SKIP gracefully with message "Container runtime not running"

**Step 4:** Verify with Podman (if available):
```bash
export DOCKER_HOST=$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')
go test -race -count=1 -v ./docker/... 2>&1 | head -40
```
Expected: Tests PASS using Podman as backend

**Step 5:** Commit
```bash
git add docker/
git commit -m "fix(docker): skip tests gracefully when no container runtime (Docker/Podman) is available"
```

---

## Summary

| Phase | Tasks | Impact |
|-------|-------|--------|
| Phase 1: Critical | Tasks 1-4 | Fix all example compilation errors |
| Phase 2: Logic Bug | Task 5 | Fix temporal metrics condition |
| Phase 3: panic/Exit | Tasks 6-7 | BREAKING: Return errors from logging.Initialize and server.Start |
| Phase 4: Logging | Tasks 8-9 | Replace fmt.Print with structured logging |
| Phase 5: Cleanup | Tasks 10-11 | Fix invalid tag, remove stub |
| Phase 6: Docs | Tasks 12-13 | Fix inaccurate documentation |
| Phase 7: Test Resilience | Task 14 | Docker/Podman test skip guards |

**Breaking changes:** Tasks 6 and 7 change public API signatures. Consider whether to batch these into a minor version bump.

**Total tasks:** 14
**Estimated commits:** 14
