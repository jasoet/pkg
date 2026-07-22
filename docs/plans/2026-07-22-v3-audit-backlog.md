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

## Open Process Items (from Phase 1 review)

- **gorelease blocks the next→main v3 merge.** The blocking API check (ci.yml) fires on a `next`→`main` PR (base_ref=main) and will report the intended v3 breaks as incompatibilities; after the module path becomes `/v3`, gorelease has no prior v3 baseline. The final phase must deliberately handle this: add `&& github.head_ref != 'next'` to the blocking condition as part of the v3 merge PR, and decide the gorelease baseline story for `/v3`.
- **gorelease is unpinned (`@latest`).** A blocking gate floating on latest is non-reproducible. Pin a version or add gorelease to flake.nix and run the flake-provided binary.
- **`.releaserc.json` headerPartial hardcodes `/v2`** in the `go get` line — must become `/v3` when the module path bumps.
- **Per-commit gate misses build-tagged files.** task check compiles only untagged code; a tagged-only break (example/integration) slipped through Phase 3 Task 1. Remaining phases: include `go build -tags=example,integration ./...` in verification.
- **Docs phase named checkbox:** sweep `examples/db/README.md` and `examples/rest/README.md` for deleted-logging references (lines ~38, ~392, ~499-504).
- **Migration guide must note:** retry RandomizationFactor range widened from [0,1) to [0,1] (1.0 now valid); config keyDepth is prefix-relative (see config/README.md migration note).
- **Post-v3 consideration:** seal config.Option (interface with unexported apply) to fully hide viper from godoc, or explicitly accept the leak; add archtest ratchet for third-party types in public signatures.
- **Conventions doc:** constructor naming split — otel.NewConfig/server.NewConfig vs retry.New/grpc.New/docker.New. Pick one in the v3 conventions writeup.
- **Migration guide (rest section) must disclose:** Client.HandleResponse was unexported in Phase 5 (commit 6cc5af1) without a BREAKING CHANGE footer mention. Guide text: typed errors for non-2xx now come from MakeRequest/MakeRequestWithTrace directly (the returned *rest.Response is non-nil on HTTP errors, so status/body remain inspectable); GetRestClient escape-hatch users who relied on HandleResponse must write their own status mapping.
