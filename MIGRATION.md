# Migrating from v2 to v3

v3 is a single big-bang release. It unifies conventions across all packages, removes
the `logging` package, and de-leaks selected third-party types from public signatures.

**Everything in this guide is a compile-time or behavioural break.** Work through the
[module path](#1-module-path) change first — it is mechanical — then only the sections
for packages you actually import.

- **v2 is frozen at v2.13.1** on the `release/v2` branch. It receives emergency patches
  only. There is no deprecation window; v2 and v3 can be imported side by side during a
  migration because their module paths differ.
- **v1** remains available at `github.com/jasoet/pkg@v1.6.0` for projects that do not
  want OpenTelemetry.

## Contents

- [1. Module path](#1-module-path)
- [2. `logging` package removed](#2-logging-package-removed)
- [3. Conventions that changed everywhere](#3-conventions-that-changed-everywhere)
- [Per-package migration](#per-package-migration)
  - [otel](#otel) · [config](#config) · [db](#db) · [docker](#docker) · [server](#server)
  - [grpc](#grpc) · [rest](#rest) · [retry](#retry) · [temporal](#temporal) · [argo](#argo)
  - [ssh](#ssh) · [compress](#compress) · [concurrent](#concurrent) · [base32](#base32)
- [4. Telemetry changes that need no code edit](#4-telemetry-changes-that-need-no-code-edit)

---

## 1. Module path

The module path is now `github.com/jasoet/pkg/v3`.

```bash
go get github.com/jasoet/pkg/v3@v3.0.0

# rewrite imports across the tree
find . -name '*.go' -not -path './vendor/*' \
  -exec sed -i '' 's|github.com/jasoet/pkg/v2|github.com/jasoet/pkg/v3|g' {} +

go mod tidy
```

On Linux use `sed -i` without the `''`.

Do this first and in its own commit — the remaining sections assume it is done, and
mixing a path rewrite with semantic changes makes the diff unreviewable.

## 2. `logging` package removed

`logging` existed only to serve `otel`, and the dependency ran backwards. It is now part
of `otel` with identical signatures, so this is an import-and-qualifier change only:

| v2 | v3 |
|---|---|
| `logging.Initialize` | `otel.Initialize` |
| `logging.InitializeWithFile` | `otel.InitializeWithFile` |
| `logging.ContextLogger` | `otel.ContextLogger` |
| `logging.LogLevel` | `otel.LogLevel` |

```go
// v2
import "github.com/jasoet/pkg/v2/logging"
err := logging.Initialize("my-service", false)
log := logging.ContextLogger(ctx, "handler")

// v3
import "github.com/jasoet/pkg/v3/otel"
err := otel.Initialize("my-service", false)
log := otel.ContextLogger(ctx, "handler")
```

## 3. Conventions that changed everywhere

v3 settles on one shape for configurable packages. If you have wrapped these packages,
expect the same three edits in each:

1. **Functional options replace mutating builders and config structs.**
   `New(opts ...Option) (T, error)` where construction can fail.
2. **`WithOTelConfig(*otel.Config)` is the single OTel injection point.** `retry`'s
   `WithOTel` was the odd one out and is renamed.
3. **`OTelConfig` is never serialized.** It is tagged `yaml:"-" mapstructure:"-"` on every
   config struct, so it must be injected via code, never loaded from YAML.

`internal/archtest` enforces all three mechanically, so they will not drift back.

---

# Per-package migration

## otel

**Config construction is options-based.** The mutating builder methods are gone.

```go
// v2 — mutating builders on *Config
cfg := otel.NewConfig("my-service").
    WithServiceVersion("1.0.0").
    WithTracerProvider(tp).
    DisableMetrics()

// v3 — package-level options
cfg := otel.NewConfig("my-service",
    otel.WithServiceVersion("1.0.0"),
    otel.WithTracerProvider(tp),
    otel.WithoutMetrics(),
)
```

`DisableTracing`/`DisableMetrics` are renamed `WithoutTracing`/`WithoutMetrics`
(and `WithoutLogging` joins them) — "Disable" read like an imperative action on an
already-built config, which is exactly the mutation model being removed.

**`WithOTLPEndpoint("")` now returns an error** instead of silently disabling OTLP
export. If you were passing a possibly-empty endpoint from configuration, branch on it:

```go
opts := []otel.LoggerProviderOption{otel.WithConsoleOutput(true)}
if endpoint != "" {
    opts = append(opts, otel.WithOTLPEndpoint(endpoint, insecure))
}
```

Silently exporting nothing because an env var was unset is the failure mode this
prevents — it is worth the explicit branch.

**Nil-config zerolog fallback** now defaults to `Info` (was unset) and labels the emitter
as `scope` rather than `service`. Log-parsing rules keyed on `service` need updating.

**Duplicate span exception events are deduped** — a recorded error no longer appears twice
on the same span.

## config

`*viper.Viper` no longer leaks into public signatures.

```go
// v2
cfg, err := config.LoadStringWithConfig[MyConfig](yamlStr, func(v *viper.Viper) {
    v.SetDefault("port", 8080)
})

// v3
cfg, err := config.LoadStringWithOptions[MyConfig](yamlStr,
    config.WithDefaults(map[string]any{"port": 8080}),
    config.WithEnvPrefix("APP"),
)
```

- `LoadStringWithConfig` → `LoadStringWithOptions`
- `NestedEnvVars` → `config.WithNestedEnvVars(prefix, keyDepth, configPath)`

`LoadString[T](s, envPrefix ...string)` is unchanged for the simple case.

**`keyDepth` is prefix-relative.** It counts segments *after* the env prefix, not from the
start of the variable name. See `config/README.md` for the worked example — this is the
one parameter likely to be silently wrong after the move.

## db

**`(*ConnectionConfig).Pool()` is removed.**

```go
// v2
pool, err := cfg.Pool()

// v3
pool, err := db.NewPool(db.WithConnectionConfig(cfg))
```

**Gorm migration wrappers are removed.** The API had four ways to do one thing.

```go
// v2
err := db.RunPostgresMigrationsWithGorm(ctx, gormDB, migrationsFS, ".")

// v3
sqlDB, err := gormDB.DB()
if err != nil {
    return err
}
err = db.RunPostgresMigrations(ctx, sqlDB, migrationsFS, ".")
```

`RunPostgresMigrationsDownWithGorm` → `RunPostgresMigrationsDown` the same way.

**No code change, but watch your dashboards:** pool metrics were gated behind
`IsTracingEnabled()`, so a metrics-only config emitted none. Fixed — a metrics-only
consumer will now start emitting `db.client.connections.*` series that never appeared
before. Alerts with "no data" conditions on those series may fire on first deploy.

**`RedactedDsn` is now structural** rather than a naive string replacement, so a password
that also appeared as a substring elsewhere in the DSN no longer leaks. Output changes only
for those pathological cases, and only in the safe direction.

## docker

**`WaitStrategy` no longer takes a docker client.** This is the selective de-leak: your
custom strategies stop depending on `docker/docker` types.

```go
// v2
func (s MyStrategy) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error

// v3
func (s MyStrategy) WaitUntilReady(ctx context.Context, target docker.ContainerTarget) error
```

`WaitForFunc` changes the same way. `ContainerTarget` exposes `State(ctx)`, `Logs(ctx)`
and the container ID.

**Limitation to plan for:** exec-based readiness checks (e.g. `pg_isready` via
`ContainerExec`) are no longer expressible through `WaitForFunc`, because `ContainerTarget`
has no exec capability. Such consumers must construct their own client for now. An
Exec-capable target is under consideration for v3.x. `ContainerTarget.Logs()` also
hardcodes `Follow`/`Timestamps` off.

**Renames and removals:**

| v2 | v3 |
|---|---|
| `Executor.WaitForHealthy` | `Executor.WaitHealthy` |
| `NatPort`, `PortBindings`, `ExposedPorts` | removed (unused helpers) |
| `LogEntry.Timestamp` | removed (never populated) |

**`Executor.Inspect()` and `Executor.GetStats()` still return `docker/docker` types.** This
is deliberate — see [ADR 0004](docs/adr/0004-selective-de-leak-of-third-party-types.md).
They are documented escape hatches, not an oversight.

## server

This package changed the most. **Signal handling was removed** — that is the change most
likely to break a production deployment silently, so start there.

```go
// v2 — blocked until SIGINT/SIGTERM, then drained gracefully
server.Start(server.Config{Port: 8080, ...})

// v3 — you own the lifecycle, and therefore the signal handling
srv, err := server.New(
    server.WithPort(8080),
    server.WithOTelConfig(otelCfg),
)
if err != nil {
    return err
}

go func() {
    if err := srv.Start(); err != nil {
        log.Error().Err(err).Msg("server stopped")
    }
}()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
<-sigCh

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
return srv.Shutdown(ctx)
```

**If you skip the `signal.Notify` block your service loses graceful termination** and will
drop in-flight requests on deploy. Nothing will fail to compile to tell you.

Removed: `Start`, `StartWithConfig`, `DefaultConfig` package functions.

Other behavioural changes:

- **Port validation now fails at `New`**, not at start.
- **Restart after shutdown returns an error.** A `Server` is single-use.
- **`Shutdown` is idempotent** — calling it twice is safe.
- **`WithOTelConfig` now auto-installs Echo OTel middleware**, so `http.server.*` spans and
  metrics appear for consumers who previously passed a config and got only partial
  instrumentation. New series, no code change.
- A doc comment claiming health endpoints were unauthenticated was **wrong** and is
  corrected; behaviour is unchanged (the package's own test always disproved it).

## grpc

**Ten dead or misleading exported symbols were removed.** All had zero non-test callers —
the `Server` wires the gateway and health endpoints itself:

`SetupGatewayForH2C`, `SetupGatewayForSeparate`, `GatewayRoute`,
`MountGatewayWithStripPrefix`, `GatewayHealthMiddleware`, `LogGatewayRoutes`,
`CreateHealthHandlers`, `EchoHealthCheckMiddleware`, `CreateEchoHealthHandler`,
`RegisterEchoIndividualHealthChecks`.

**Behavioural changes that shipped without a `BREAKING CHANGE` footer** — these will not
appear in the generated release notes, so they are listed here deliberately:

1. **`MountGatewayOnEcho` now strips the base path.** The mux sees proto http-rule paths
   verbatim. If you registered mux patterns *including* the prefix, switch to
   proto-relative ones:

   ```go
   // v2: pattern had to include the base path
   mux.HandlePath("GET", "/api/v1/users", handler)

   // v3: base path is stripped before the mux sees the request
   mux.HandlePath("GET", "/users", handler)
   ```

2. **`Start`/`StartH2C`/`StartSeparate` return `nil` on clean shutdown**, not
   `http.ErrServerClosed`. Drop the special-casing:

   ```go
   // v2
   if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
       return err
   }

   // v3
   if err := srv.Start(); err != nil {
       return err
   }
   ```

3. **`GetGRPCServer()` returns nil after `Stop`**, until the next `Start`. The server is
   rebuilt per start cycle; a long-lived cached reference will be stale.

**Fixed bugs** (no action needed): a restarted server could not be stopped, the `running`
flag was sticky, and H2C timeouts were applied to gRPC streams.

Note the deliberate divergence from `server`: **grpc supports `Start`→`Stop`→`Start`
cycles, `server` does not.** See [ADR 0005](docs/adr/0005-lifecycle-divergence-between-server-and-grpc.md).

## rest

**`*resty.Response` no longer leaks.** `MakeRequest`/`MakeRequestWithTrace` return
`*rest.Response`:

```go
// v2
resp, err := client.MakeRequest(ctx, "GET", url, nil, nil)
body := resp.Body()          // resty
code := resp.StatusCode()

// v3
resp, err := client.MakeRequest(ctx, "GET", url, nil, nil)
body := resp.Body()          // rest.Response
code := resp.StatusCode()
```

The shape is close enough that most call sites compile unchanged; the type in your own
signatures is what needs editing.

**Status helpers moved onto `Response`,** and one was misnamed:

| v2 (package func) | v3 (method) |
|---|---|
| `rest.IsUnauthorized(resp)` | `resp.IsAuthError()` |
| `rest.IsNotFound(resp)` | `resp.IsNotFound()` |

`IsUnauthorized` folded 403 into 401, so the name was a lie — hence `IsAuthError`.

**`RequestInfo.TraceInfo` is now `rest.TraceInfo`** (was resty's).

**Error constructors are unexported:** `NewUnauthorizedError`, `NewExecutionError`,
`NewServerError`, `NewResponseError`, `NewResourceNotFoundError` and `RecordRetry`. The
error *types* remain exported, so type switches and `errors.As` keep working.

**`Client.HandleResponse` was unexported** (this also shipped without a footer). Typed
errors for non-2xx responses now come from `MakeRequest`/`MakeRequestWithTrace` directly,
and the returned `*rest.Response` is non-nil on HTTP errors, so status and body stay
inspectable. If you used `GetRestClient()` as an escape hatch and called `HandleResponse`
yourself, you must write your own status mapping.

**Retry policy changed — this one is easy to miss:**

- **POST and PATCH are no longer retried by default.** Only idempotent methods are. If you
  relied on automatic POST retries, either make the endpoint idempotent (preferred) or
  handle retries at the call site.
- `Retry-After` response headers are now honoured.
- `ExecutionError` and `UnauthorizedError` messages now include the cause / response body.
  **Do not match on error strings** — use `errors.As` with the exported types.

## retry

```go
// v2 — builder methods
cfg := retry.NewConfig().
    SetOperationName("db.connect").
    SetMaxRetries(3).
    WithOTel(otelCfg)

// v3 — functional options
cfg := retry.New(
    retry.WithName("db.connect"),
    retry.WithMaxRetries(3),
    retry.WithOTelConfig(otelCfg),
)
```

- `Config.OperationName` → `Config.Name`
- `WithOTel` → `WithOTelConfig` (aligning with every other package)
- **Invalid configuration now returns an error from `Do`** instead of panicking inside a
  setter. Setters that panicked while exported fields went unguarded was an inconsistent
  contract; validation now happens in one place.
- **`RandomizationFactor` accepts `[0, 1]`** — the range widened from `[0, 1)`, so `1.0` is
  now valid. Strictly more permissive; nothing that worked stops working.

## temporal

`temporal` is an SDK-integration package: SDK types in signatures are by design
(see [ADR 0004](docs/adr/0004-selective-de-leak-of-third-party-types.md)). What changed is
the constructor shape and **client ownership**.

```go
// v2 — interface{} constructors, manager owned the client
cfg := &temporal.Config{HostPort: "localhost:7233", Namespace: "default"}
wm, err := temporal.NewWorkerManager(cfg)
defer wm.Close()

// v3 — typed options, caller owns the client
c, err := temporal.NewClient(
    temporal.WithHostPort("localhost:7233"),
    temporal.WithNamespace("default"),
    temporal.WithOTelConfig(otelCfg),
)
if err != nil {
    return err
}
defer c.Close()               // <- you close it now

wm, err := temporal.NewWorkerManager(c)
if err != nil {
    return err
}
defer wm.Close(ctx)           // <- takes ctx, does NOT close the client
```

`temporal.WithConfig(cfg)` accepts a whole `Config` if you already load one from YAML.

**The ownership change is the dangerous one.** Managers now *borrow* a caller-owned
`client.Client`. If you previously relied on `wm.Close()` to close the client, you now leak
the connection unless you close it yourself.

Also:

- `Close()` → `Close(ctx)` on `WorkerManager` and `ScheduleManager`.
- **`WorkflowManager.Close` was removed entirely.** It had become a no-op. The commit's
  `BREAKING CHANGE` footer omits this, so it would not otherwise reach the release notes.
  `WorkflowManager` now has no `Close` — close the client you passed in instead.
- `NewWorkflowManagerWithNamespace`'s `namespace` parameter is **now always authoritative**.
  It was previously ignored when a `*Config` was also passed — if you were relying on the
  config value winning, you will now get the parameter's value.
- `NewClient`/`Close` accept a context.
- Namespace handling is consistent across the package; workflow history attribution and
  TLS/auth support were fixed and added respectively.

## argo

Like `temporal`, an SDK-integration package by design.

**`argo.Option` no longer returns an error** — no option ever failed:

```go
// v2
type Option func(*Config) error

// v3
type Option func(*Config)
```

**Operations no longer take a positional `*otel.Config`.** They read it from the context,
which is what `NewClient` returns:

```go
// v2
created, err := argo.SubmitWorkflow(ctx, client, wf, otelCfg)

// v3 — NewClient returns the ctx carrying the OTel config; thread it through
ctx, client, err := argo.NewClientWithOptions(ctx, argo.WithOTelConfig(otelCfg))
if err != nil {
    return err
}
created, err := argo.SubmitWorkflow(ctx, client, wf)
```

**Thread the `ctx` returned by `NewClient` through to every operation.** Passing a fresh
`context.Background()` silently disables instrumentation — no error, just no telemetry.
This applies to `SubmitWorkflow`, `SubmitAndWait`, `GetWorkflowStatus`, `ListWorkflows`
and `DeleteWorkflow`.

**Fixed:** `Namespace()` returned an untrimmed newline in in-cluster mode, which broke it
outright — if in-cluster mode never worked for you on v2, this is why.

`SubmitAndWait`'s poll interval is now configurable via `argo.WithPollInterval(d)`
(was hardcoded at 5s), and timeout/failure errors are sentinels (`argo.ErrWaitTimeout`,
`argo.ErrWorkflowFailed`) usable with `errors.Is`.

## ssh

**Source-compatible for construction** — `New` is variadic — but several behaviours changed:

- **`Close` now tears down in-flight forwarded connections immediately** instead of
  draining them. Finish your work before calling `Close`. In exchange, `Close` no longer
  blocks for ~90s against keep-alive clients.
- **Half-close propagation**: peers now see EOF promptly, which is observable for streaming
  protocols.
- **`Close` error text is wrapped** as `"SSH client close error: %w"`. Use `errors.Is`/`As`
  rather than string matching — the package's old error contract was string-based and the
  README's matching guidance did not match any real error string.
- **Secrets are `yaml:"-"` by design**: `Password`, `PrivateKey` and `PrivateKeyPassphrase`
  are never loaded from YAML. **A `password:` key in your YAML is silently dropped** —
  inject secrets via env or code. If SSH auth breaks after upgrading, check this first.
- `Config` gains an `OTelConfig` field with `WithOTelConfig()` plumbing and
  `otel.Layers` instrumentation.

The accept loop no longer busy-spins on persistent errors, and `Start`→`Close`→`Start`
cycles no longer race.

## compress

- **`UnGz` now enforces `WithMaxArchiveSize`.** It was silently ignored. Extractions over
  the limit fail with `ErrSizeLimitExceeded` — if you set a limit expecting it to apply to
  gzip and it did not, extractions that used to succeed will now correctly fail.
- **Error message texts changed** for non-directory source/destination and tar guard-rail
  rejections. They now wrap `ErrNotDirectory`, `ErrPathTraversal` and
  `ErrSizeLimitExceeded` — match with `errors.Is`, not on strings.
- Symlink writes are refused, overwrites truncate, and the size cap is hard.

## concurrent

**`ExecuteConcurrentlyTyped` type parameters are now `[R, T]` (result first).**

```go
// v2
results, err := concurrent.ExecuteConcurrentlyTyped[Input, Output](ctx, funcs)

// v3
results, err := concurrent.ExecuteConcurrentlyTyped[Output, Input](ctx, funcs)
```

If you relied on inference this compiles unchanged. **If you specified the parameters
explicitly and both are the same type, it still compiles and is now wrong** — check these
call sites by hand.

`ExecuteConcurrently` gains nil-`resultBuilder` validation and captures panic stacks.

## base32

No API breaks. `AppendChecksum` and `ValidateChecksum` now normalize their input like the
other entry points, so dashed and lowercase input is accepted consistently rather than
rejected. Sentinel errors were added.

The README and doc comments contained systematically wrong encoded values in v2 — if you
built expectations from those examples rather than from running the code, re-check them.

---

## 4. Telemetry changes that need no code edit

These require no source change but will alter what your observability backend receives.
Check dashboards and alerts after deploying:

| Change | Effect |
|---|---|
| **`server` and `grpc` gateway now continue inbound W3C traces** | Requests carrying `traceparent` produce spans parented to the caller instead of new roots. Traces that appeared as separate roots will now join. |
| **`db` pool metrics no longer gated on tracing** | Metrics-only configs start emitting `db.client.connections.*`. |
| **`server` `WithOTelConfig` auto-installs Echo middleware** | New `http.server.*` spans and metrics. |
| **`ssh` gains OTel instrumentation** | New spans/metrics where there were none. |
| **`otel` dedupes span exception events** | Error counts derived from span events drop (they were double-counted). |
| **`rest` retry counter wired into the resty hook** | The retry metric was dead code in v2 and now reports real values. |

`server` and the `grpc` gateway both emit metrics named `http.server.*` but with different
attribute sets — `server` uses method + status; the gateway adds `http.route` and
`active_requests`. If you aggregate across both, account for the difference. This is
documented rather than aligned; see [ADR 0005](docs/adr/0005-lifecycle-divergence-between-server-and-grpc.md).

---

## Getting help

- Per-package detail lives in each package's `README.md`.
- Runnable examples are under `examples/<package>/`.
- The full audit that produced this release: `docs/plans/2026-07-22-v3-audit-backlog.md`.
- Architectural decisions: `docs/adr/`.
