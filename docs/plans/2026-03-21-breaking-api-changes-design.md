# Breaking API Changes — Feature Plan

**Date:** 2026-03-21
**Status:** Deferred — implement as separate feature PRs
**Source:** Code review medium findings that require breaking API changes

## Deferred Items

### 1. argo — Add context propagation through builder (M1)

**Problem:** All `Add()`, `AddParallel()`, `AddExitHandler()`, `Build()`, `BuildWithEntrypoint()`, and all `Steps()`/`Templates()` methods use `context.Background()`. Trace spans are disconnected from the caller.

**Root cause:** The `WorkflowSource` interface (`source.go`) does not accept `context.Context`, making it impossible to propagate context without a breaking change.

**Proposed fix:**
- Add `context.Context` parameter to `WorkflowSource` interface methods
- Update all implementations (Container, Script, HTTP, Noop)
- Update builder methods to accept and propagate context
- Branch: `feat/argo-context-propagation`

### 2. server — Export composable Server type (M4)

**Problem:** `httpServer` is unexported. No exported way to stop a running server other than sending a signal. Not composable for embedding in larger lifecycle managers.

**Proposed fix:**
- Export a `Server` type with `Start(ctx context.Context) error` / `Stop() error`
- Remove hardcoded `os/signal` handling from the library core
- Provide a `Run()` convenience function that wraps `Start` with signal handling for simple cases
- Branch: `feat/server-composable-api`

### 3. grpc — Move Echo creation from Start() to New() (M6)

**Problem:** Echo server is not created until `Start()`. Users cannot interact with the Echo instance (add custom routes) between construction and start — must use `WithEchoConfigurer` callback.

**Proposed fix:**
- Create Echo instance in `New()` instead of `Start()`
- Expose `Echo() *echo.Echo` accessor for pre-start configuration
- Branch: `feat/grpc-echo-early-init`

## Implementation Notes

- Each item should be a separate feature branch and PR
- Include migration guide in PR description
- Update README and examples
- These are v2 minor version bumps (non-breaking for Go modules since method additions to interfaces are breaking)
- Consider whether argo M1 should wait for a v3 release
