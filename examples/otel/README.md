# OTel Package Examples

Runnable examples for the [`otel`](../../otel/) package — the configuration hub every other
package takes its telemetry from.

**Example code:** [`example.go`](./example.go)

## Running

```bash
go run -tags=example ./examples/otel
```

The `example` build tag is required; without it the file is excluded from the build.

## Quick reference

```go
import "github.com/jasoet/pkg/v3/otel"

// Configuration is built with functional options.
cfg := otel.NewConfig("my-service",
    otel.WithServiceVersion("1.0.0"),
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider),
    otel.WithLoggerProvider(loggerProvider),
)

// Ask before you use — but the getters are safe either way.
if cfg.IsTracingEnabled() {
    tracer := cfg.GetTracer("scope-name")
}

// LogHelper correlates logs with the active span.
logger := otel.NewLogHelper(ctx, cfg, "github.com/myorg/myapp", "myFunc")
logger.Info("Processing request", otel.F("request_id", reqID))
logger.Error(err, "Failed to process", otel.F("user_id", userID))
```

> **`NewConfig` installs a default zerolog-backed logger provider** unless you opt out with
> `otel.WithoutLogging()`. A config with no options is therefore *not* fully no-op — logging
> is on. Use `WithoutTracing()` / `WithoutMetrics()` / `WithoutLogging()` to turn pillars off
> explicitly.

## What the example covers

| # | Example | Shows |
|---|---|---|
| 1 | Basic configuration | All three pillars wired from SDK providers |
| 2 | No-op configuration | `WithoutLogging()` to get a genuinely inert config |
| 3 | LogHelper usage | Zerolog fallback vs. OTel-backed, and `otel.F()` fields |
| 4 | Telemetry pillars | Traces-only, metrics-only, and all-on configurations |
| 5 | Configuration validation | `Is*Enabled()` checks and safe no-op getters |

## Telemetry pillars

The three pillars are independent — enable only what you need.

```go
// Traces only
cfg := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithoutLogging(),
)

// Metrics only
cfg := otel.NewConfig("my-service",
    otel.WithMeterProvider(meterProvider),
    otel.WithoutLogging(),
)
```

When a provider is absent, `Is*Enabled()` returns `false` and `GetTracer`/`GetMeter`/
`GetLogger` return no-op implementations. Calling them is always safe — the `Is*Enabled()`
check is for skipping *your* work, not for guarding the getters.

## Passing the config to other packages

One config, injected everywhere via each package's `WithOTelConfig` option:

```go
otelCfg := otel.NewConfig("my-service",
    otel.WithServiceVersion(version),
    otel.WithTracerProvider(tp),
    otel.WithMeterProvider(mp),
)

srv, err := server.New(server.WithPort(8080), server.WithOTelConfig(otelCfg))
grpcSrv, err := grpc.New(grpc.WithGRPCPort("50051"), grpc.WithOTelConfig(otelCfg))
client := rest.NewClient(rest.WithOTelConfig(otelCfg))
pool, err := db.NewPool(db.WithConnectionConfig(dbCfg), db.WithOTelConfig(otelCfg))
```

`OTelConfig` is never read from YAML — it is tagged `yaml:"-" mapstructure:"-"` on every
config struct, so it must be injected in code. See
[ADR 0002](../../docs/adr/0002-otel-config-is-injected-never-serialized.md).

## LogHelper

```go
logger := otel.NewLogHelper(ctx, cfg, "github.com/myorg/myapp", "functionName")

logger.Debug("Cache lookup", otel.F("cache_key", "user:123"), otel.F("hit", true))
logger.Info("User logged in", otel.F("user_id", 123))
logger.Warn("Rate limit approaching", otel.F("current_rate", 95))
logger.Error(err, "Failed to process payment", otel.F("payment_id", "PAY-123"))
```

Passing `nil` for the config falls back to zerolog, so helper code works with or without
telemetry configured. With a config, logs carry `trace_id` and `span_id` for correlation.

## Related

- [`otel` package documentation](../../otel/README.md)
- [fullstack-otel example](../fullstack-otel/) — complete app with Jaeger and Prometheus
- [OpenTelemetry Go docs](https://opentelemetry.io/docs/languages/go/)
