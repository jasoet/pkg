# v3 Phase 9: server Lifecycle + OTel Alignment

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give `server` programmatic lifecycle control (grpc-style `New`/`Start`/`Shutdown`), auto-installed OTel request instrumentation, a corrected health-endpoint comment, and truthful docs.

**Architecture:** A `Server` type replaces the signal-blocking package functions, mirroring `grpc.New(opts...) (*Server, error)` + `Start`/`Stop`. OTel middleware (tracing + basic metrics) is implemented locally using `otel.Config` providers — no new dependencies.

**Tech Stack:** Go 1.26, Echo, OTel, testify, httptest.

## Global Constraints

- Work on `next`, module `github.com/jasoet/pkg/v3`. Conventional Commits; NEVER AI attribution. Breaking commits carry `!` + `BREAKING CHANGE:` footer.
- Verification per task: `nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...` plus focused tests; `task check` green at phase end.
- Backlog: `docs/plans/2026-07-22-v3-audit-backlog.md` (server section).

## Current-State Facts (verified — trust these)

- Current API: `DefaultConfig(port, op, shut) Config`, `StartWithConfig(Config) error`, `Start(port, op, shutdown, mw...) error` — all signal-blocking; no programmatic shutdown. `Config` + `Option` + `With*` options exist (WithPort/WithOperation/WithShutdown/WithShutdownTimeout/WithEchoConfigurer/WithOTelConfig/WithMiddleware).
- `Config.OTelConfig` is used ONLY for startup/shutdown log helpers — no request instrumentation.
- `server/server.go:133` comment claims health routes are "registered before user middleware … intentionally unauthenticated" — FALSE: they register after `e.Use(m...)`; user middleware (incl. auth) DOES apply.
- `server.Config` archtest-registered; `server.WithOTelConfig` signature asserted.
- ErrServerClosed already filtered at server.go:185 (pattern to reuse).
- No otelecho/otelhttp dependency available — implement middleware locally.
- README points at nonexistent example paths and omits the options API.

---

### Task 1: Server type with programmatic lifecycle (breaking)

**Files:**
- Modify: `server/server.go`
- Test: `server/lifecycle_test.go` (new)
- Modify callers: `server/server_test.go`, `examples/server/`, `examples/fullstack-otel/main.go` (if it uses the removed funcs)

**Interfaces:**
- Produces:
  ```go
  type Server struct { /* unexported: config Config, echo *echo.Echo, mu sync.Mutex, shutdown chan */ }
  func New(opts ...Option) (*Server, error)      // validates config (port range etc.)
  func (s *Server) Start() error                  // blocking; nil on clean Shutdown; filters ErrServerClosed
  func (s *Server) Shutdown(ctx context.Context) error
  func (s *Server) Addr() string                  // bound address (useful with Port: 0)
  func (s *Server) Echo() *echo.Echo              // access for tests/route registration pre-Start
  ```
- REMOVED: `Start(port, op, shutdown, mw...)`, `StartWithConfig(Config)`, `DefaultConfig(port, op, shut)` — migration: `server.New(server.WithPort(8080), server.WithOperation(op), server.WithShutdown(shut))`.
- Config struct + all existing `With*` options KEPT unchanged.

- [ ] **Step 1: Write the failing tests**

Create `server/lifecycle_test.go`:
1. `TestServerStartShutdown`: `New(WithPort(0))`, `go srv.Start()`, poll `Addr()` until listening, GET `/health` → 200, `srv.Shutdown(ctx)` → nil, `Start()` returns nil.
2. `TestServerShutdownTimeout`: Shutdown callback invoked on Shutdown.
3. `TestServerStartTwiceFails`: second `Start()` returns an error while running.
4. `TestNewInvalidPort`: `New(WithPort(-1))` (or 70000) returns error.

Run: FAIL — `server.New` undefined.

- [ ] **Step 2: Implement**

- `server/server.go`: add the `Server` type. Reuse existing setupEcho/health/ErrServerClosed logic. `Start` binds the listener explicitly (`net.Listen` so `Addr()` works with port 0), runs Operation before serving (current semantics), blocks until Shutdown/listener error, filters `http.ErrServerClosed`. `Shutdown(ctx)` triggers graceful stop honoring `ShutdownTimeout`, invokes the `Shutdown` callback.
- Delete `Start`, `StartWithConfig`, `DefaultConfig`. Update all callers (`grep -rn 'server\.Start\|StartWithConfig\|DefaultConfig' --include='*.go' . | grep -v vendor | grep -v grpc/`): server tests, examples/server, examples/fullstack-otel, README-adjacent test code.

- [ ] **Step 3: Verify**

```bash
nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./server/ -count=1 -race
```

- [ ] **Step 4: Commit**

```bash
git add server/ examples/
git commit -m "feat(server)!: Server type with programmatic Start/Shutdown lifecycle

BREAKING CHANGE: removed Start, StartWithConfig, DefaultConfig package functions; use server.New(opts...) + srv.Start()/srv.Shutdown(ctx)."
```

---

### Task 2: OTel request instrumentation (tracing + metrics)

**Files:**
- Create: `server/otel_middleware.go`
- Modify: `server/server.go` (install when OTelConfig set)
- Test: `server/otel_middleware_test.go` (new)

**Interfaces:**
- Produces: when `OTelConfig` is set, Start auto-installs (before user middleware):
  - Tracing middleware (if `IsTracingEnabled()`): one span per request named `{method} {route}` with attrs `http.request.method`, `url.full`, `http.response.status_code`, `http.route`; scope `http.server`.
  - Metrics middleware (if `IsMetricsEnabled()`): `http.server.request.count` counter + `http.server.request.duration` histogram (attrs method + status_code); scope `http.server`.
  - Logging via existing startup/shutdown LogHelpers (unchanged).

- [ ] **Step 1: Write the failing tests**

1. Tracing: httptest-driven Server with tracetest exporter — GET /health produces one span with the right name + attrs.
2. Metrics: ManualReader — one request increments count and records duration with method/status attrs.
3. Nil OTelConfig: no middleware, no panic (already-covered pattern, add explicit test).

Run: FAIL — no spans/metrics today.

- [ ] **Step 2: Implement**

`server/otel_middleware.go`: local echo middlewares using `cfg.GetTracer("http.server")` / `cfg.GetMeter("http.server")` (no new deps; no-op-safe). Install in setupEcho after BodyLimit, before user middleware. Extract `http.route` from `c.Path()` post-handler (set span name + attr in a deferred end).

- [ ] **Step 3: Verify**

```bash
nix develop -c go test ./server/ -count=1
nix develop -c go build -tags=example,integration ./...
```

- [ ] **Step 4: Commit**

```bash
git add server/
git commit -m "feat(server): auto-install OTel tracing and metrics middleware when OTelConfig is set"
```

---

### Task 3: README + examples + health comment fix

**Files:**
- Modify: `server/README.md`, `server/server.go` (comment at ~133)
- Test: `server/example_test.go` (new)
- Modify: `examples/server/` if present (check run instructions)

- [ ] **Step 1: Fix the health comment**

At server/server.go:133, replace with an accurate comment: health routes are registered AFTER user middleware, so user middleware (including auth) applies to them; callers needing unauthenticated K8s probes should not register global auth middleware or must exempt these paths themselves.

- [ ] **Step 2: Example tests**

`server/example_test.go`: `ExampleNew` (deterministic with port 0 + httptest GET to /health with `// Output:`), `ExampleServer_Shutdown` (compile-checked).

- [ ] **Step 3: Rewrite server/README.md**

/v3 paths; document `New`/`Start`/`Shutdown`/`Addr`/`Echo`; full options list; OTel instrumentation behavior (spans + metrics names — only the real ones from Task 2); correct example paths (`examples/server/`); remove signal-blocking API references.

- [ ] **Step 4: Verify** — `nix develop -c go test ./server/ -count=1 -v | grep -E 'Example|ok'`

- [ ] **Step 5: Commit**

```bash
git add server/ examples/server/
git commit -m "docs(server): rewrite README for Server API; correct health-endpoint middleware comment"
```

---

### Task 4: Phase verification and push

- [ ] **Step 1: Full gate**

```bash
task check
nix develop -c go build -tags=example,integration ./...
```

- [ ] **Step 2: Push** — `git push origin next`
