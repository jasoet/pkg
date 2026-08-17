# v3.0.0 Audit Backlog

**Date:** 2026-07-22
**Source:** 15-agent audit of all packages at v2.13.1 (swarm report, session of 2026-07-22)
**Status:** Backlog for the v3.0.0 big-bang release

Decisions driving this backlog (agreed 2026-07-22):

- Library is a **product for external users**; docs accuracy, semver, green-at-tag are obligations.
- v2 frozen at **v2.13.1** on `release/v2` (emergency patches only).
- v3 developed on `next` (`v3.0.0-next.N` prereleases), merged to `main` with BREAKING CHANGE → v3.0.0.
- v3 scope: **unify conventions + selective de-leak** (resty, viper advanced API, docker client in `WaitStrategy`). `temporal`/`argo` are documented SDK-integration packages — vendor types there are by design.
- `logging` merges into `otel` → 14 packages.
- Process teeth: integration-test release gate, `gorelease` API-diff CI gate, `internal/archtest` convention tests, `Example*` tests as docs-of-record.

## Cross-Cutting Conventions (v3 contract)

Every package with configuration MUST have:

1. Functional options constructor: `New(opts ...Option) (T, error)` where construction can fail.
2. Config structs carry `OTelConfig *otel.Config` tagged exactly `yaml:"-" mapstructure:"-"`.
3. `WithOTelConfig(cfg *otel.Config) Option` as the OTel injection point.
4. Instrumentation via `otel.Layers.Start*()` at layer boundaries.
5. README snippets backed by `Example*` tests (compile-checked docs).
6. testify for tests; unit (no tag) + integration (`//go:build integration`) tiers.

Enforced mechanically by `internal/archtest` (Phase 1).

## Per-Package Backlog

### otel (foundation — do first, absorbs logging)

- Violates functional-options convention: mutating builders (`NewConfig(...).WithTracerProvider(...)`) with a thread-safety story that contradicts itself between code and README. Decide mutation contract; make options consistent.
- Reassignable global `Layers` and raw third-party provider types are the public contract.
- README/doc-comment examples largely **do not compile** (`logging.NewLoggerProvider`, `grpc.NewServer`, `rest.ClientConfig` references).
- Absorb `logging`: move `LogLevel`, file-output support; `logging.Initialize` becomes deprecated shim or drops. Kill the inverted logging↔otel dependency.
- Add real behavioral tests for `SpanHelper`.

### logging (merge into otel)

- Merge `LogLevel` enum into otel (it exists only to serve otel).
- Provide non-global logger factory; document global `Initialize` as deprecated shim or remove.
- README: nonexistent `otel_example.go`, wrong example path, `ContextLogger` claim contradicts code.

### config

- `*viper.Viper` leaks into every advanced signature — wrap in library-owned type (selective de-leak).
- Variadic `envPrefix` silently ignores extra args — document or fix.
- `NestedEnvVars` is fiddly, non-goroutine-safe, env/YAML precedence contradicts `AutomaticEnv`.
- Docs: broken example links, wrong `/v2`-less import path, fabricated benchmark, stale Go version, contradictory YAML naming guidance.

### rest

- Leaks resty types everywhere (`*resty.Response` from `MakeRequest`/`MakeRequestWithTrace`) — wrap in library-owned `Response` (selective de-leak).
- Exported internal-only helpers: `HandleResponse`, five error constructors; `IsUnauthorized` folds 403 into 401 (misnamed).
- Retry metric is dead code; headline retry feature has no end-to-end test.
- README observability docs largely fabricated (wrong span attributes, nonexistent gauge, phantom benchmarks, broken link).

### retry

- Convention deviations: no functional options (builder methods on `Config`), `WithOTel` instead of `WithOTelConfig`, `OTelConfig` field missing tags.
- Optional setters panic on invalid input while exported fields are unguarded — pick one validation strategy.
- README omits a config field; example README "expected output" not reproducible.

### db

- Clearest convention-breaker: no functional options, no `WithOTelConfig()`, no `otel.Layers`.
- Migration API duplicated four ways — deprecate either `*WithGorm` wrappers or raw variants.
- **Bug:** pool metrics gated behind tracing (`pool.go`) — un-gate.
- **Bug:** `RedactedDsn` naive string replacement (password substring elsewhere in DSN leaks).
- `SQLDB()` surprising resource semantics — document or fix.

### docker

- `WaitStrategy` interface leaks docker client type into consumer code — wrap (selective de-leak, v3).
- `ContainerRequest.OTelConfig` tag deviation (`yaml:"-"` only).
- Docs bug: `%s` vs `{{endpoint}}` drift breaks README and runnable database example.
- Surface clutter: `New`/`NewFromRequest`/`WithRequest` overlap, `WaitForHealthy` name collision, unused `nat.*` helpers, dead `LogEntry.Timestamp`.

### grpc

- ~12 dead/misleading exported symbols: no-op `SetupGatewayForH2C`, `SetupGatewayForSeparate` ignores dial options, unused stdlib-handler health API — remove.
- **Bugs:** unstoppable restarted server, sticky `running` flag, `ErrServerClosed` returned on clean shutdown.
- README documents a Config-struct API (`DefaultConfig`, `StartWithConfig`, `New(config)`) that no longer exists; wrong import paths; nonexistent `logging.NewLoggerProvider`.

### server

- Weakest citizen: options API barely consumed; `WithOTelConfig` delivers a fraction of grpc's; no programmatic lifecycle control (signal-blocking start only) — add `StartContext`/constructor alignment with `grpc.New(opts...) (T, error)`, auto-install Echo OTel middleware.
- Incorrect "health endpoints unauthenticated" comment — security-relevant doc bug (own test disproves it).
- READMEs point to nonexistent example paths, omit options API.

### ssh

- No functional options, no `OTelConfig`/`WithOTelConfig`, hardcodes nil otel configs — add OTel plumbing.
- README overstates: nonexistent "auto reconnection", YAML examples silently drop password, wrong `Start()` signature, unmatchable error-matching guidance.
- Untested exported `LocalAddr`; integration test doesn't assert actual forwarding; error contract built on string matching.

### temporal (SDK-integration package — leak by design, document it)

- No functional options, no `WithOTelConfig()`, no `otel.Layers`; `interface{}` constructors (`NewScheduleManager(clientOrConfig)`).
- One deliberate breaking pass: typed constructors or options, ctx-accepting `NewClient`/`Close`, injectable logger, document or unexport `ZerologAdapter`.
- Backfill unit tests: logger adapter, query validation, `QueryWorkflow`, `ListFailedWorkflows`.

### argo (SDK-integration package — leak by design, document it)

- Split-brain Options: `argo.Option = func(*Config) error` (nothing can fail) vs `builder.Option = func(*WorkflowBuilder)`.
- OTel threading: operations take `cfg *otel.Config` positionally, ignoring client config — unify.
- `argo.Config.OTelConfig` tag deviation (`yaml:"-"` only).
- **Bug:** `Namespace()` untrimmed newline breaks in-cluster mode.
- README: three identifiers don't compile (`ArgoServerConfig`, `WithActiveDeadline`, run command), `ServerOpts` misnamed, "generics" feature claimed that doesn't exist, stale "v2.0.0" instrumentation version.
- Hard-coded poll intervals; non-sentinel errors.

### compress

- API asymmetries: stream-in/path-out for gzip; absolute-path required for `UnGz` not `UnTar`; sentinel errors on only half the guard rails; an option silently ignored by `UnGz`.
- README: undocumented options, quick-start fails at runtime, fabricated benchmark and file-mode claims, error-matching advice matches no real error string.

### concurrent

- No config/options/OTel hook (unlike `retry`) — decide if in scope for conventions (probably exempt: pure utility, stateless).
- `ExecuteConcurrentlyTyped` flipped parameter order; thin duplicate wrapper function.
- Docs: fabricated benchmarks, false 100%-coverage claim, broken links, wrong import path, run instructions fail due to build tag.

### base32

- Implementation healthy; docs layer broken: systematically wrong encoded values in README/doc comments, fabricated checksum example, wrong run instructions, example sections producing empty output (dashed input rejected).
- Add golden checksum regression tests.
- `AppendChecksum`/`ValidateChecksum` should normalize input or loudly document caller must.

## Resolved Process Items

Kept for provenance; each records where the fix landed.

- ~~**gorelease blocks the next→main v3 merge.**~~ RESOLVED — `github.head_ref != 'next'` added to the blocking condition and `head_ref == 'next'` to the informational one (PR #59). Baseline story: gorelease reports "Inferred base version: none / Suggested version: v3.0.0" until the first non-prerelease `/v3` tag exists on main, which is why the release PR must not gate on it.
- ~~**gorelease is unpinned (`@latest`).**~~ RESOLVED — pinned to the `golang.org/x/exp` pseudo-version already in go.mod, via a workflow-level `GORELEASE_VERSION` env var (PR #59).
- ~~**`.releaserc.json` headerPartial hardcodes `/v2`.**~~ RESOLVED — now emits `go get github.com/jasoet/pkg/v3@{{currentTag}}`.
- ~~**Per-commit gate misses build-tagged files.**~~ RESOLVED — ci.yml runs `go vet -tags='example integration argo' ./...` as its own step.
- ~~**Docs phase named checkbox:** sweep examples/db and examples/rest for deleted-logging references.~~ RESOLVED — verified: the remaining "logging" hits in both files refer to the activity, not the deleted package. The real stale reference was `docker/README.md`'s Related Packages link to `../logging/`, now removed.
- ~~**Conventions doc:** constructor naming split.~~ RESOLVED as **not a defect** — see [ADR 0003](../adr/0003-constructor-naming.md). `New` returns the package's primary type; `New<Thing>` names it when a package builds several. `retry.New` returns a `Config` because a `Config` *is* retry's primary artifact.
- ~~**Migration guide** items (rest, temporal, grpc, server, ssh, argo, db, retry, config sections).~~ RESOLVED — all folded into `MIGRATION.md`, including the breaks that shipped without `BREAKING CHANGE` footers. Two backlog notes described a planned API that never shipped and are documented as built: `temporal.NewClient` takes no ctx, and argo operations dropped the namespace parameter as well as the `*otel.Config` one.
- ~~**Final docs sweep must cover stale db APIs**~~ RESOLVED — verified no `cfg.Pool()` or `RunPostgresMigrationsWithGorm` references remain outside the historical plan documents.
- ~~**Root README coverage figures are stale.**~~ RESOLVED — regenerated from a real unit+integration run; `argo`, `retry` and `base32` gained the figures they never had.
- ~~**docker NetworkSettings parity:** unguarded derefs in network.go.~~ RESOLVED — guarded via pure projection helpers with unit tests for the nil case (PR #61).
- ~~**Shared gap (server + grpc):** neither HTTP tracing middleware extracts W3C traceparent.~~ RESOLVED — both now extract, with tests pinning both the joined-trace and root-span cases (PR #60).
- ~~**argo v3.x (1) hard-coded 5s poll interval.**~~ RESOLVED — `argo.WithPollInterval(d)`.
- ~~**argo v3.x (2) non-sentinel errors.**~~ RESOLVED — `ErrWaitTimeout` and `ErrWorkflowFailed`.

## Open Process Items (from Phase 1 review)

- **Post-v3 consideration:** seal config.Option (interface with unexported apply) to fully hide viper from godoc, or explicitly accept the leak; add archtest ratchet for third-party types in public signatures.
- ~~**docker remaining leaks (decision needed).**~~ DECIDED — `Executor.Inspect()` and `GetStats()` stay as documented escape hatches; see [ADR 0004](../adr/0004-selective-de-leak-of-third-party-types.md). Wrapping them would mean tracking the docker API for information that has no natural library-owned shape.
- **docker ContainerTarget limits:** Logs() hardcodes Follow/Timestamps off; exec-based readiness strategies (pg_isready via ContainerExec) are no longer expressible via WaitForFunc — migration guide must note such consumers construct their own client. Consider Exec-capable target in v3.x.
- ~~**Conventions writeup:** grpc restart cycles vs server single-use; `http.server.*` attribute-set divergence.~~ DOCUMENTED — [ADR 0005](../adr/0005-lifecycle-divergence-between-server-and-grpc.md) and the telemetry table in `MIGRATION.md`. Both divergences are deliberate and stay.
- **ssh fix-wave items (Phase 10)** — (1) half-close: RESOLVED, `forward()` propagates `CloseWrite`. (2) nil-config LogHelper: PARTIALLY RESOLVED — `forward()`/`acceptLoop()` are now config-aware, but still build their context from `context.Background()`, so logs carry no trace correlation (see v3.x item 4 below). (3) Close no-op path span status: still open. (4) Close error wrapping: RESOLVED and noted in `MIGRATION.md`.
- **ssh v3.x items (from Phase 10 final review):** (1) accept-loop races — PARTIALLY RESOLVED. The busy-spin half is fixed (capped exponential backoff, exit on `net.ErrClosed`). **Still open:** `acceptLoop` reads `t.stopCh` unlocked at three points while `Start` reassigns it under `t.mu` (tunnel.go:302,328,340 vs :168), and `t.wg.Add` after Accept still races `Close`'s `wg.Wait`. Verified still present as of this sweep — capture `stopCh` locally at loop entry and guard the `Add`. (2) Residual Close hang for peers ignoring FIN — consider tracking active local conns and force-closing them in Close. (3) Pin the ssh testcontainer image digest (currently :latest + unconditional 5s sleep). (4) forward() shipped config-aware logging (context.Background) instead of Start-span ctx propagation — descoped decision, logs carry no trace correlation.
- **argo:** all Phase 12 follow-ups resolved (configurable poll interval, sentinel errors, migration notes). Nothing open.
