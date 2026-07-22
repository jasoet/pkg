# v3 Phase 7: docker De-Leak + Surface Cleanup

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the docker client from `WaitStrategy`'s public contract, clean up the package surface (tags, name collision, dead exports), and fix the broken docs/example endpoint pattern.

**Architecture:** A library-owned `ContainerTarget` (wraps the docker client internally, exposes only `ID()`, `Logs(ctx)`, `State(ctx)`) replaces `*client.Client` in `WaitStrategy.WaitUntilReady` and `WaitForFunc`. Own `ContainerState` struct replaces `container.InspectResponse` for strategy use.

**Tech Stack:** Go 1.26, docker/docker client (internal), testify, Docker daemon for integration tests.

## Global Constraints

- Work on `next`, module `github.com/jasoet/pkg/v3`. Conventional Commits; NEVER AI attribution. Breaking commits carry `!` + `BREAKING CHANGE:` footer.
- Verification per task: `nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...` plus focused tests; `task check` green at phase end.
- Backlog: `docs/plans/2026-07-22-v3-audit-backlog.md` (docker section).

## Current-State Facts (verified — trust these)

- Leak: `WaitStrategy.WaitUntilReady(ctx, cli *client.Client, containerID string)` and `WaitForFunc(fn func(ctx, cli *client.Client, containerID string) error)` — wait.go:20, 305.
- Strategies only need: `ContainerLogs` (waitForLog) and `ContainerInspect` (port/http/healthy) — small surface.
- `ContainerRequest.OTelConfig *otel.Config` at docker/config.go:92 has NO struct tags (needs `yaml:"-" mapstructure:"-"`).
- Name collision: `WaitForHealthy() *waitForHealthy` (wait.go:254, strategy constructor) vs `(e *Executor) WaitForHealthy(ctx, timeout) error` (status.go:226).
- Exported nat-typed helpers with zero external callers: `NatPort` (network.go:196), `PortBindings` (network.go:213), `ExposedPorts` (network.go:236) — used only inside docker if at all.
- Dead field: `LogEntry.Timestamp` (logs.go:24) — explicitly documented as never populated.
- `NewFromRequest(req, opts...)` = prepend `WithRequest` + `New` (executor.go:101-105) — KEEP as documented sugar (decided).
- OTel: executor uses its own span helpers (e.otel.startSpan), not `otel.Layers`.

---

### Task 1: ContainerTarget de-leak of WaitStrategy

**Files:**
- Create: `docker/target.go`
- Modify: `docker/wait.go` (interface + all strategies), `docker/executor.go` (passes target), `docker/wait_test.go` + any strategy tests, examples using `WaitForFunc`

**Interfaces:**
- Produces:
  ```go
  // ContainerTarget is the runtime surface a WaitStrategy can inspect.
  type ContainerTarget struct { /* wraps *client.Client + containerID, unexported */ }
  func (t ContainerTarget) ID() string
  func (t ContainerTarget) Logs(ctx context.Context) (io.ReadCloser, error)     // stdout+stderr, follow
  func (t ContainerTarget) State(ctx context.Context) (ContainerState, error)

  type ContainerState struct {
      Running      bool
      HealthStatus string            // "" when no healthcheck
      Ports        map[string][]string // containerPort ("80/tcp") → hostPorts
  }

  type WaitStrategy interface {
      WaitUntilReady(ctx context.Context, target ContainerTarget) error
  }
  // WaitForFunc(fn func(ctx context.Context, target ContainerTarget) error) *waitFunc
  ```
- REMOVED from public contract: `*client.Client` in WaitStrategy and WaitForFunc signatures.

- [ ] **Step 1: Write the failing test**

Create `docker/target_test.go`: construct strategies per the NEW interface and assert the interface compiles with ContainerTarget (compile-level), plus a fake-target test if feasible (e.g., an unexported constructor or interface seam for tests — implementer's choice: export a test helper `newContainerTarget(cli, id)` in package docker and white-box test that waitForLog matches via a stubbed ContainerAPI internally). At minimum: a test asserting `WaitForFunc` accepts `func(ctx, target ContainerTarget) error` and that its timeout wrapper works (fn returns sentinel error → WaitUntilReady wraps it).

Run: FAIL — ContainerTarget undefined.

- [ ] **Step 2: Implement**

- `docker/target.go`: ContainerTarget wrapping the client + id; `Logs` = ContainerLogs(ShowStdout+Stderr, Follow); `State` maps ContainerInspect → ContainerState (Running, Health status string or "", Ports map[string][]string from NetworkSettings.Ports).
- `wait.go`: interface + 5 strategies (log/port/http/healthy/func) + multiWait rewritten to the target API (port/http strategies use `State().Ports` + dial/GET against localhost:hostPort as today; healthy uses `State().HealthStatus == "healthy"`).
- `executor.go`: wherever strategies are invoked, construct `newContainerTarget(e.client, containerID)`.
- Update all tests/examples using `WaitForFunc` with the old signature.

- [ ] **Step 3: Verify**

```bash
nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./docker/ -count=1
nix develop -c go test ./docker/ -count=1 -run 'TestExecutor_.*Nginx|TestWait' -v  # live-daemon strategies still work
```

- [ ] **Step 4: Commit**

```bash
git add docker/ examples/
git commit -m "feat(docker)!: ContainerTarget replaces docker client in WaitStrategy

BREAKING CHANGE: WaitStrategy.WaitUntilReady and WaitForFunc now take docker.ContainerTarget instead of *client.Client + containerID."
```

---

### Task 2: Surface cleanup (tags, collision, dead exports)

**Files:**
- Modify: `docker/config.go` (OTelConfig tags), `docker/network.go` (nat helpers), `docker/status.go` (rename), `docker/logs.go` (dead field), `internal/archtest/archtest_test.go`

**Interfaces:**
- Produces: `ContainerRequest.OTelConfig` tagged `yaml:"-" mapstructure:"-"`; `NatPort/PortBindings/ExposedPorts` unexported (natPort/portBindings/exposedPorts) or deleted if fully unused; `Executor.WaitForHealthy` renamed `WaitHealthy` (strategy constructor `WaitForHealthy()` keeps its name); `LogEntry.Timestamp` removed; docker.ContainerRequest registered in archtest.

- [ ] **Step 1: Write the failing test**

In `internal/archtest/archtest_test.go` add `"docker": reflect.TypeOf(docker.ContainerRequest{}),` to `compliantConfigs` (import docker).
Run: `nix develop -c go test ./internal/archtest/ -run TestConfigStructsCarryOTelConfig/docker -v`
Expected: FAIL — missing tags.

- [ ] **Step 2: Implement**

- config.go: add the tags.
- network.go: check internal usage of the three helpers (`grep -n 'NatPort(\|PortBindings(\|ExposedPorts(' docker/*.go`); unexport if internally used, delete if unused. Ensure no exported signature retains `nat.*` types afterward: `grep -n 'nat\.' docker/*.go | grep -v _test` must show only unexported usages.
- status.go: rename the Executor method to `WaitHealthy`; update callers (`grep -rn '\.WaitForHealthy(' --include='*.go' . | grep -v vendor`).
- logs.go: remove `LogEntry.Timestamp` field + its doc comment; check constructors/literals don't set it.
- options_test.go (archtest): add `_ func(*otel.Config) docker.Option = docker.WithOTelConfig` — ALREADY EXISTS from Phase 1; verify, don't duplicate.

- [ ] **Step 3: Verify**

```bash
nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./docker/ ./internal/archtest/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git add docker/ internal/archtest/
git commit -m "feat(docker)!: surface cleanup — OTelConfig tags, WaitHealthy rename, drop nat helpers and dead field

BREAKING CHANGE: Executor.WaitForHealthy renamed WaitHealthy; NatPort/PortBindings/ExposedPorts unexported; LogEntry.Timestamp removed."
```

---

### Task 3: docker README + endpoint-pattern bug + Example tests

**Files:**
- Modify: `docker/README.md`, `examples/docker/database/main.go` (and README if present)
- Test: `docker/example_test.go` (new)

**Interfaces:**
- Produces: fixed wait-pattern docs (`{{endpoint}}` vs `%s` drift — audit: README/database example's pattern never matches container logs, so the wait never succeeds); compile-checked examples.

- [ ] **Step 1: Reproduce and fix the endpoint bug**

Read the database example and README section: identify the mismatched wait pattern (the audit says `%s` vs `{{endpoint}}` drift — find where the pattern string must match actual container log output). Fix so `go run -tags=example ./examples/docker/database` actually becomes ready. Verify by running it (Docker available).

- [ ] **Step 2: Example tests**

Create `docker/example_test.go`: compile-checked examples (`ExampleNew`, `ExampleWaitForLog`) — daemon-dependent examples get the `// Output is non-deterministic; compile-checked only.` comment; pure-construction parts (options assembly) can be shown without starting containers.

- [ ] **Step 3: Rewrite docker/README.md**

/v3 paths; document ContainerTarget + new WaitStrategy contract; WaitHealthy rename; removed nat helpers/Timestamp; fix any other stale signatures (`NewFromRequest` kept — document as sugar over `New(WithRequest(req), ...)`).

- [ ] **Step 4: Verify** — `nix develop -c go test ./docker/ -count=1` green; database example runs to readiness.

- [ ] **Step 5: Commit**

```bash
git add docker/ examples/docker/
git commit -m "docs(docker): fix endpoint wait-pattern drift; rewrite README for ContainerTarget API"
```

---

### Task 4: Phase verification and push

- [ ] **Step 1: Full gate**

```bash
task check
nix develop -c go build -tags=example,integration ./...
```

- [ ] **Step 2: Push** — `git push origin next`
