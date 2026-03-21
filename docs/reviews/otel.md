# Code Review: `otel` Package

**Date:** 2026-03-21

## Package Summary

The `otel` package provides OpenTelemetry instrumentation wrappers for the `github.com/jasoet/pkg/v2` ecosystem. It has four source files:

- **`config.go`** — `Config` struct with builder/accessor methods, context storage, and graceful shutdown.
- **`logging.go`** — `NewLoggerProviderWithOptions` factory; custom `consoleExporter` backed by zerolog.
- **`helper.go`** — `LogHelper` struct that routes to either the OTel `log.Logger` or zerolog, plus the `Field`/`F()` types.
- **`instrumentation.go`** — `SpanHelper`, `LayerContext`, `LayeredSpanHelper` (the `Layers` singleton) for layer-aware span + log correlation.

---

## Issues Found

### High

**H1 — Slice aliasing / append race in `LayeredSpanHelper.Start*` methods** (`instrumentation.go:355,388,421,464,500`)

```go
allFields := append([]Field{F("layer", "handler")}, fields...)
```

Uses `append` on a single-element literal slice. If the caller's `fields` variadic argument has backing capacity beyond its length (resliced slice), the `append` writes the prepended element into the caller's memory.

**Fix:** Use `append(make([]Field, 0, 1+len(fields)), F("layer", "handler"))` followed by `append(..., fields...)`.

**H2 — `GetTracer`, `GetMeter`, `GetLogger` instantiate new no-op providers on every call** (`config.go:222,231,240`)

```go
return noopt.NewTracerProvider().Tracer(scopeName, opts...)
```

Each call when the respective pillar is disabled creates and discards a new provider object. In hot-path code this generates non-trivial garbage pressure.

**Fix:** Store package-level no-op singletons:
```go
var noopTracer = noopt.NewTracerProvider()
```

### Medium

**M1 — `Config` documented as "immutable after creation" but With*/Disable* methods mutate in-place** (`config.go:75-114`)

Builder methods mutate the receiver and return `self`. This breaks the stated contract when a shared `*Config` is used concurrently.

**M2 — `defaultLoggerProvider` silently swallows creation errors** (`config.go:159-164`)

Error is discarded entirely. Falls back to no-op with no indication.

**M3 — OTLP endpoint not validated** (`logging.go:109-123`)

Empty string endpoint with `otlpInsecure: true` is never validated — could cause subtle misconfiguration.

**M4 — `consoleExporter` uses `SimpleProcessor`, blocking the hot path** (`logging.go:106`)

Console exporter blocks on `os.Stderr` write synchronously. OTLP correctly uses batch.

**M5 — `StartSpan` falls back to global OTel tracer silently** (`instrumentation.go:96-100`)

When no config is in context, uses the global tracer provider — surprising hidden side-effect.

**M6 — `LayerContext.fields` stored but never used after construction** (`instrumentation.go:278-280`)

Dead code that adds confusion. The `fields` slice is populated in all `Start*` methods but never read.

### Low

- L1: `contextKey` type leaks string value in context dumps
- L2: `LogHelper.ctx` stored at construction time, never updated — stale span correlation
- L3: `toString` has unreachable first branch
- L4: `emitOTel` does not set `Timestamp` on log records
- L5: `addAttributeToEvent` default case uses `AsString()` instead of `fmt.Sprint()`
- L6: Example logs email without PII warning
- L7: `WithOTLPEndpoint` doesn't validate endpoint format
- L8: `consoleExporter.Shutdown/ForceFlush` are no-ops ignoring context

### Security

- **SEC-1** (Medium): No TLS certificate pinning for OTLP endpoint
- **SEC-2** (Low): Error messages may leak internal paths/config
- **SEC-3** (Low): PII may be logged in error fields via `F("error", err.Error())`
- **SEC-4** (Low): Global OTel tracer fallback enables confused-deputy behavior

### Recommendations

1. Fix slice aliasing in all five `Start*` methods
2. Add package-level no-op singletons
3. Clarify or fix mutability contract on `Config`
4. Replace global tracer fallback with no-op
5. Remove dead `LayerContext.fields` field
6. Add `record.SetTimestamp(time.Now())` in `emitOTel`
