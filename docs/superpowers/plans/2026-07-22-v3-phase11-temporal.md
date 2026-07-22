# v3 Phase 11: temporal Unification

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace temporal's `interface{}` constructors with typed ones, add the options/OTelConfig convention, ctx-accepting Close, backfill unit tests with the SDK mock client, and document the SDK-integration posture.

**Architecture:** Managers take a caller-owned `client.Client` (no more `clientOrConfig interface{}`, no internal ownership flags). `NewClient` gains functional options following the db.NewPool pattern. The Temporal SDK leak is BY DESIGN (SDK-integration package) — documented, not wrapped.

**Tech Stack:** Go 1.26, go.temporal.io/sdk (+ mocks), testify.

## Global Constraints

- Work on `next`, module `github.com/jasoet/pkg/v3`. Conventional Commits; NEVER AI attribution. Breaking commits carry `!` + `BREAKING CHANGE:` footer.
- Verification per task: `nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...` plus focused tests; `task check` green at phase end.
- Backlog: `docs/plans/2026-07-22-v3-audit-backlog.md` (temporal section). The `temporal/job` subpackage is OUT OF SCOPE (already type-focused).

## Current-State Facts (verified — trust these)

- `NewClient(config *Config) (client.Client, error)`; `Config{HostPort, Namespace, OTelConfig *otel.Config \`yaml:"-" mapstructure:"-"\`}`; `DefaultConfig() *Config`.
- `NewWorkerManager(config *Config) (*WorkerManager, error)` — creates its own client internally.
- `NewScheduleManager(clientOrConfig interface{}) (*ScheduleManager, error)` — interface{} over client.Client|*Config, `ownsClient` flag.
- `NewWorkflowManager(clientOrConfig interface{})` + `NewWorkflowManagerWithNamespace(clientOrConfig, ns)` — same interface{} pattern.
- `NewZerologAdapter(zerolog.Logger) *ZerologAdapter` — public bridge to temporal's log.Logger; audit says document or unexport (decision: document).
- `WorkerManager`: Register(taskQueue, worker.Options) worker.Worker, Start(ctx, w), StartAll(ctx), Close(), GetClient(), GetWorkers().
- Backfill targets: logger adapter, `validateQueryParam`, `QueryWorkflow`, `ListFailedWorkflows` (unit-testable via go.temporal.io/sdk/mocks).
- Integration tests exist (testcontainer package) — they call these constructors and must be converted.

---

### Task 1: Typed constructors + options (breaking)

**Files:**
- Modify: `temporal/client.go`, `temporal/worker.go`, `temporal/schedule.go`, `temporal/workflow.go`, `temporal/config.go`
- Modify callers: `temporal/*_test.go`, `temporal/testcontainer/`, `examples/temporal/`, `examples/fullstack-otel/`

**Interfaces:**
- Produces:
  - `type Option func(*Config)`; `WithConfig(c Config) Option`; `WithHostPort(addr string) Option`; `WithNamespace(ns string) Option`; `WithOTelConfig(cfg *otel.Config) Option`
  - `NewClient(opts ...Option) (client.Client, error)` — starts from DefaultConfig(), applies opts
  - `NewWorkerManager(client client.Client) (*WorkerManager, error)` — caller owns the client; `Close(ctx context.Context)` (was Close())
  - `NewScheduleManager(client client.Client) (*ScheduleManager, error)` — caller owns; `Close(ctx context.Context)`
  - `NewWorkflowManager(client client.Client)` + `NewWorkflowManagerWithNamespace(client client.Client, namespace string)`
  - archtest: `_ func(*otel.Config) temporal.Option = temporal.WithOTelConfig` in options_test.go; `"temporal": reflect.TypeOf(temporal.Config{})` already registered — verify it stays.
- REMOVED: `NewWorkerManager(*Config)`, `NewScheduleManager(interface{})`, `NewWorkflowManager(interface{})`, `NewWorkflowManagerWithNamespace(interface{}, string)`, `NewClient(*Config)`, `Close()` without ctx, `ownsClient` machinery.
- Migration: `temporal.NewWorkerManager(&cfg.Temporal)` → `c, _ := temporal.NewClient(temporal.WithConfig(cfg.Temporal)); wm, _ := temporal.NewWorkerManager(c)`.

- [ ] **Step 1: Write the failing tests**

Create `temporal/options_test.go`:
1. `TestNewClientOptions` — `NewClient(WithHostPort("x:1"), WithNamespace("ns"), WithOTelConfig(c))` — assert config assembly (error return from dial is fine/expected; assert the error is about the unreachable host, proving opts applied — or factor config assembly so it's testable without dialing).
2. `TestNewScheduleManagerTyped` — `NewScheduleManager(mocks.NewClient(t))` works; no interface{} anywhere (`grep 'interface{}' temporal/*.go | grep -v _test` → 0).

Run: FAIL — new signatures undefined.

- [ ] **Step 2: Implement**

- config.go: Option + the four options.
- client.go: `NewClient(opts ...Option)`.
- worker.go/schedule.go/workflow.go: typed client params, drop ownsClient (Close no longer closes the client — document: caller closes their client), `Close(ctx)`.
- Convert ALL callers: temporal tests, testcontainer setup (check testcontainer.Setup signature — it creates clients; update to new API), examples/temporal, examples/fullstack-otel, temporal/job if it references these (check).
- `grep -rn 'interface{}' temporal/*.go | grep -v _test` → 0.

- [ ] **Step 3: Verify**

```bash
nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./temporal/... ./internal/archtest/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git add temporal/ examples/ internal/archtest/
git commit -m "feat(temporal)!: typed constructors and functional options

BREAKING CHANGE: NewClient now takes options (WithConfig/WithHostPort/WithNamespace/WithOTelConfig); NewWorkerManager/NewScheduleManager/NewWorkflowManager take a caller-owned client.Client instead of config/interface{}; Close now accepts ctx and no longer closes the client."
```

---

### Task 2: Unit test backfill (SDK mocks)

**Files:**
- Test: `temporal/logger_test.go` (extend or new), `temporal/workflow_unit_test.go` (new), `temporal/schedule_unit_test.go` (new)

**Interfaces:**
- Produces: unit coverage for `ZerologAdapter` (Debug/Info/Warn/Error pass-through with fields), `validateQueryParam` (injection rejection — the security fix from #46), `QueryWorkflow`, `ListFailedWorkflows` — using `go.temporal.io/sdk/mocks.Client`.

- [ ] **Step 1: Write the tests**

- ZerologAdapter: capture zerolog output via a bytes.Buffer writer; assert levels + keyvals land.
- validateQueryParam: table test — valid inputs pass, injection attempts (quotes, operators per the regex from #46) rejected.
- QueryWorkflow: mock client `On("QueryWorkflow", ...)` returns a value; assert decoding + error paths.
- ListFailedWorkflows: mock ListWorkflow responses; assert filtering/pagination logic (read the implementation first — test its actual behavior).

- [ ] **Step 2: Verify** — `nix develop -c go test ./temporal/ -count=1` green.

- [ ] **Step 3: Commit**

```bash
git add temporal/
git commit -m "test(temporal): backfill unit tests for logger adapter, query validation, workflow queries"
```

---

### Task 3: README rewrite (SDK-integration posture) + Example tests

**Files:**
- Modify: `temporal/README.md`, `examples/temporal/README.md`
- Test: `temporal/example_test.go` (new)

- [ ] **Step 1: Rewrite temporal/README.md**

- /v3 paths; new typed constructors; options API.
- Explicit SDK-integration posture section: this package intentionally exposes go.temporal.io/sdk types (client.Client, worker.Worker, client.ScheduleHandle) — the managers are convenience lifecycle wrappers, not an abstraction layer; use temporal/job's Definition for typed per-workflow handles.
- Document ZerologAdapter (bridging zerolog into the Temporal SDK logger).
- Document Close(ctx) + caller-owned client semantics.

- [ ] **Step 2: Example tests**

`temporal/example_test.go`: `ExampleNewClient` (compile-checked w/ non-deterministic comment), `ExampleNewScheduleManager` (compile-checked).

- [ ] **Step 3: Sweep examples/temporal/README.md** — typed constructors, run instructions.

- [ ] **Step 4: Verify** — `nix develop -c go test ./temporal/ -count=1` green.

- [ ] **Step 5: Commit**

```bash
git add temporal/ examples/temporal/
git commit -m "docs(temporal): rewrite README for typed constructors and SDK-integration posture"
```

---

### Task 4: Phase verification and push

- [ ] **Step 1: Full gate**

```bash
task check
nix develop -c go build -tags=example,integration ./...
```

- [ ] **Step 2: Integration sanity** — `nix develop -c go test -tags=integration -count=1 -timeout=15m ./temporal/...` (testcontainers; must be green after constructor conversion).

- [ ] **Step 3: Push** — `git push origin next`
