# gRPC Metrics: Remove Legacy Prometheus, Consolidate on OTel

> Date: 2026-04-08
> Status: Pending Review
> Scope: Remove the duplicate Prometheus `MetricsManager` in `grpc/` and consolidate all metrics on the existing OTel instrumentation layer

## Problem

The `grpc/` package runs **two parallel metrics systems** simultaneously:

| Layer | File | What it does |
|-------|------|-------------|
| Legacy | `grpc/metrics.go` | Creates its own `prometheus.NewRegistry()`, manually registers 12 Prometheus metrics (CounterVec, HistogramVec, Gauge), serves them via `promhttp` at `/metrics` |
| OTel | `grpc/otel_instrumentation.go` | Uses `otel.Config.GetMeter()` to create OTel counters, histograms, and gauges following semantic conventions |

Both are wired in `server.go` at `setupEchoServer()` (lines 140-159):

```go
if s.config.otelConfig != nil {
    // OTel path: logging, tracing, metrics middleware
    e.Use(createHTTPGatewayMetricsMiddleware(s.config.otelConfig))
} else {
    // Legacy path: Prometheus MetricsManager
    if s.config.enableMetrics {
        e.Use(s.metricsManager.EchoMetricsMiddleware())
        s.metricsManager.RegisterEchoMetrics(e, s.config.metricsPath)
    }
}
```

Additionally, gRPC interceptors always install OTel metrics when `otelConfig != nil` (line 100), but the `MetricsManager` is always created unconditionally in `New()` (line 53).

### Issues

1. **Double counting**: When OTel is configured, the `MetricsManager` is still created (wasting memory) and the uptime goroutine still runs if `enableMetrics` is true — even though the Prometheus `/metrics` endpoint is never registered in the OTel path.

2. **Inconsistent with other packages**: `docker/`, `rest/`, `db/`, `argo/builder/` all use OTel exclusively. The `grpc/` package is the only one maintaining a parallel Prometheus implementation.

3. **Unnecessary dependency**: `prometheus/client_golang` is a direct dependency used only by `grpc/metrics.go` and `temporal/client.go`. The temporal design doc (2026-04-07) already plans to remove the temporal usage. After both migrations, this dependency can be dropped entirely.

4. **Exposed API surface**: `GetMetricsManager()`, `GetRegistry()`, `MetricsManager` type, and several recording methods are public API that consumers could theoretically depend on. However, no code outside `grpc/` references these (verified via grep).

## Solution

Remove `grpc/metrics.go` entirely. The OTel instrumentation in `grpc/otel_instrumentation.go` already covers all the same metrics with proper semantic conventions. The `enableMetrics` / `metricsPath` config options become legacy — repurpose them or remove them.

### Before
```
OTel config set:
  → OTel interceptors (gRPC unary/stream metrics)      ✅
  → OTel Echo middleware (HTTP gateway metrics)          ✅
  → MetricsManager created but unused                    ❌ (waste)
  → trackUptime goroutine may still run                  ❌ (waste)

OTel config nil:
  → No OTel interceptors                                 ✅
  → Prometheus MetricsManager + /metrics endpoint        ⚠️ (legacy)
  → trackUptime goroutine runs                           ⚠️ (legacy)
```

### After
```
OTel config set:
  → OTel interceptors (gRPC unary/stream metrics)        ✅
  → OTel Echo middleware (HTTP gateway metrics)           ✅
  → Server uptime/start_time as OTel observable gauges   ✅

OTel config nil:
  → No metrics instrumentation                            ✅ (safe, no crash)
```

No Prometheus. No `/metrics` HTTP endpoint. No `MetricsManager`. Metrics flow through the OTel pipeline (OTLP exporter → VictoriaMetrics/etc).

## Changes

### 1. Delete `grpc/metrics.go`

Remove the entire file (379 lines). This removes:
- `MetricsManager` struct and all methods
- `NewMetricsManager()`
- `RecordGRPCRequest()`, `RecordHTTPRequest()`
- `IncrementGRPCConnections()`, `DecrementGRPCConnections()`
- `IncrementHTTPRequests()`, `DecrementHTTPRequests()`
- `UpdateUptime()`
- `HTTPMetricsMiddleware()`, `EchoMetricsMiddleware()`, `EchoUptimeMiddleware()`
- `RegisterEchoMetrics()`
- `CreateMetricsHandler()`, `GetRegistry()`
- `responseWriter` type (used only by `HTTPMetricsMiddleware`)
- Imports: `prometheus`, `prometheus/collectors`, `prometheus/promhttp`

### 2. `grpc/server.go`

**Remove from `Server` struct:**
- `metricsManager *MetricsManager` field (line 36)
- `stopUptime chan struct{}` field (line 40)

**Remove from `New()`:**
- `metricsManager: NewMetricsManager("grpc_server")` (line 53)

**Simplify `setupEchoServer()`:**
Remove the legacy fallback branch entirely. The `else` block at lines 151-159 currently handles `otelConfig == nil` by wiring Prometheus middleware. Remove it:

```go
// Before (lines 140-160):
if s.config.otelConfig != nil {
    if s.config.otelConfig.IsLoggingEnabled() { ... }
    if s.config.otelConfig.IsTracingEnabled() { ... }
    if s.config.otelConfig.IsMetricsEnabled() { ... }
} else {
    // Legacy fallback — REMOVE THIS BLOCK
    if s.config.enableLogging { e.Use(middleware.RequestLogger()) }
    if s.config.enableMetrics {
        e.Use(s.metricsManager.EchoMetricsMiddleware())
        s.metricsManager.RegisterEchoMetrics(e, s.config.metricsPath)
    }
}

// After:
if s.config.otelConfig != nil {
    if s.config.otelConfig.IsLoggingEnabled() { ... }
    if s.config.otelConfig.IsTracingEnabled() { ... }
    if s.config.otelConfig.IsMetricsEnabled() { ... }
}
```

**Remove from `Start()`:**
- The `enableMetrics` block that creates `stopUptime` and starts `trackUptime` goroutine (lines 238-241)
- Log line about metrics endpoint (lines 282-283, 321-323)

**Remove from `Stop()`:**
- The `stopUptime` close block (lines 344-346)

**Remove methods:**
- `GetMetricsManager()` (lines 407-409)
- `trackUptime()` (lines 425-436)

### 3. `grpc/config.go`

**Remove fields from `config` struct:**
- `enableMetrics bool` (line 45)
- `metricsPath string` (line 46)
- `enableLogging bool` (line 49) — only used in the legacy fallback branch

**Remove defaults from `newConfig()`:**
- `enableMetrics: true` (line 91)
- `metricsPath: "/metrics"` (line 92)
- `enableLogging: true` (line 95)

**Remove option functions:**
- `WithMetrics()` (lines 298-302)
- `WithoutMetrics()` (lines 305-309)
- `WithMetricsPath()` (lines 382-386)
- `WithLogging()` (lines 326-330)
- `WithoutLogging()` (lines 333-337)

### 4. `grpc/otel_instrumentation.go` — Add server uptime metrics

The legacy `MetricsManager` tracked `server_uptime_seconds` and `server_start_time_seconds`. Add these as OTel observable gauges to maintain feature parity. Add to the HTTP gateway metrics setup:

```go
func registerServerMetrics(cfg *pkgotel.Config) {
    if cfg == nil || !cfg.IsMetricsEnabled() {
        return
    }
    meter := cfg.GetMeter("grpc.server")
    startTime := time.Now()

    meter.Float64ObservableGauge(
        "server.uptime",
        metric.WithDescription("Server uptime in seconds"),
        metric.WithUnit("s"),
        metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
            o.Observe(time.Since(startTime).Seconds())
            return nil
        }),
    )

    meter.Float64ObservableGauge(
        "server.start_time",
        metric.WithDescription("Server start time as Unix timestamp"),
        metric.WithUnit("s"),
        metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
            o.Observe(float64(startTime.Unix()))
            return nil
        }),
    )
}
```

Call `registerServerMetrics(s.config.otelConfig)` from `setupGRPCServer()` when OTel is configured.

### 5. Delete `grpc/metrics_test.go`

Remove the entire file (386 lines). All tests are for the removed `MetricsManager`.

### 6. Update remaining test files

- `grpc/server_test.go` — remove assertions on `metricsManager` field (line 31), `GetMetricsManager()` calls (line 52), `trackUptime` tests (lines 384-396), `enableMetrics`/`metricsPath` assertions (lines 470, 477)
- `grpc/config_test.go` — remove tests for `WithMetrics()`, `WithoutMetrics()`, `WithMetricsPath()`, `WithLogging()`, `WithoutLogging()`

### 7. `grpc/README.md`

- Remove references to Prometheus `/metrics` endpoint
- Remove `enableMetrics`, `metricsPath` from config examples
- Update examples to show OTel-only metrics configuration

### 8. `go.mod` (after both temporal + grpc migrations)

After both this change and the temporal migration are complete, run `go mod tidy`. Expected removals:

- `github.com/prometheus/client_golang` (direct → removed or demoted to indirect)
- `github.com/uber-go/tally/v4` (from temporal migration)
- `go.temporal.io/sdk/contrib/tally` (from temporal migration)

Whether `prometheus/client_golang` becomes fully removable depends on whether any indirect dependency still pulls it in. `go mod tidy` will determine.

## Backward Compatibility

**Breaking changes:**
- `MetricsManager` type removed (public)
- `NewMetricsManager()` removed (public)
- `GetMetricsManager()` removed from `Server` (public)
- `WithMetrics()`, `WithoutMetrics()`, `WithMetricsPath()` options removed
- `WithLogging()`, `WithoutLogging()` options removed
- Prometheus `/metrics` endpoint no longer served

**Mitigation:**
- No code outside `grpc/` references `MetricsManager` or `GetMetricsManager()` (verified via grep)
- Consumers using the legacy `enableMetrics` path should switch to `WithOTelConfig()` — this is the same migration path that was already required to get proper tracing and logging
- The `/metrics` scrape endpoint is replaced by OTel push-based metrics (OTLP → collector → VictoriaMetrics)

**Migration for consumers:**
```go
// Before:
server, _ := grpc.New(
    grpc.WithMetrics(),
    grpc.WithMetricsPath("/metrics"),
)

// After:
otelCfg := otel.NewConfig(
    otel.WithMeterProvider(mp),
)
server, _ := grpc.New(
    grpc.WithOTelConfig(otelCfg),
)
```

## Files Affected

| File | Change |
|------|--------|
| `grpc/metrics.go` | **Delete** |
| `grpc/metrics_test.go` | **Delete** |
| `grpc/server.go` | Remove `metricsManager`, `stopUptime`, `trackUptime`, legacy fallback, `GetMetricsManager()` |
| `grpc/config.go` | Remove `enableMetrics`, `metricsPath`, `enableLogging`, related options |
| `grpc/otel_instrumentation.go` | Add `registerServerMetrics()` for uptime/start_time gauges |
| `grpc/server_test.go` | Update for removed fields/methods |
| `grpc/config_test.go` | Remove tests for removed options |
| `grpc/README.md` | Update docs |

## Relationship to Temporal Migration

This plan is a companion to [2026-04-07-temporal-otel-metrics-design.md](2026-04-07-temporal-otel-metrics-design.md). Together they:

1. Remove all direct Prometheus metric instrumentation from the codebase
2. Consolidate all metrics on OTel via `otel.Config.MeterProvider`
3. Enable dropping `prometheus/client_golang` and `uber-go/tally` as direct dependencies

The two can be implemented in any order. Running `go mod tidy` after both will yield the cleanest dependency tree.
