# Code Review: `argo` Package

**Date:** 2026-03-21

## Package Summary

Go utility library wrapping the Argo Workflows API client. Provides client creation (3 connection modes), workflow operations (submit, wait, status, list, delete), a fluent workflow builder with OTel instrumentation, template types (container, script, HTTP, noop), and higher-level CI/CD, parallel, and map-reduce patterns.

---

## Issues Found

### Critical

**C1 — `mustParseQuantity` panics on invalid input** (`builder/template/script.go:344-346`)

`Script.Templates()` calls `resource.MustParse` which panics on invalid quantity string. `Container.Templates()` correctly uses `resource.ParseQuantity` with error return.

**Fix:** Replace `mustParseQuantity` with `resource.ParseQuantity` returning error.

**C2 — `BuildWithEntrypoint` mutates shared builder state** (`builder/builder.go:524`)

```go
b.templates = append(b.templates, exitHandler)
```

Unlike `Build()` which creates a fresh copy, `BuildWithEntrypoint` mutates `b.templates` in place. Calling it twice duplicates the exit handler.

### High

- H1: Hardcoded default service account `"argo-workflow"` — magic string with no validation
- H2: `SubmitAndWait` polling swallows transient errors entirely — no backoff, no max error count
- H3: `inClusterClientConfig.ClientConfig()` creates logger with nil OTel config and `context.Background()`
- H4: Workflow injection via unvalidated `when` conditions — arbitrary Argo expressions accepted

### Medium

- M1: `context.Background()` used throughout builder — trace spans disconnected from caller
- M2: `Script.source` field set but never used — `.Source()` method is silently broken
- M3: Token stored as plain string in `Config` — no `Secret` type or zero-on-free
- M4: `WithConfig` performs shallow copy — shared `OTelConfig` pointer mutation risk
- M5: Exit handler prioritization by name substring match (`"destroy"`, `"cleanup"`) is fragile
- M6: `ParallelDataProcessing`/`MapReduce` build args via `fmt.Sprintf` without shell escaping
- M7: `BuildWithEntrypoint` not reentrant-safe (test only covers `Build()`)

### Low

- L1: `otelInstrumentation.incrementCounter` uses string-switch dispatch
- L2: `buildClientConfig` creates logger with `context.Background()`
- L3: Factory functions create logger just for one debug log
- L4: `SubmitWorkflow` logs `GenerateName` not `Name`
- L5: `FanOutFanIn` hardcodes `sleep 2` in pattern code
- L6: Integration tests use `time.Sleep`
- L7: Test accesses unexported `retryStrategy` field directly

### Security

| Finding | Severity |
|---------|----------|
| Shell injection via unsanitized `fmt.Sprintf` in patterns | High |
| Workflow expression injection via `When()` | Medium |
| Auth token as plain string | Medium |
| `InsecureSkipVerify` can be enabled with no warning | Low |
| No namespace isolation enforcement | Low |
| Default service account hardcoded | Low |

### Recommendations

1. Replace `mustParseQuantity` with error-returning variant
2. Fix `BuildWithEntrypoint` to create fresh template slice
3. Add shell escaping in pattern functions
4. Document/enforce `When()` strings must not come from untrusted input
5. Fix `Script.Source()` to actually populate `ScriptTemplate.Source`
6. Add context parameter path through builder
7. Replace `"argo-workflow"` with exported constant
