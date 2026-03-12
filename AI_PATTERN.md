# AI Pattern Guide

Guide for AI assistants working on projects that **use** `github.com/jasoet/pkg/v2`. This file is an index — read the linked READMEs and examples for full details.

## Quick Start

```go
import "github.com/jasoet/pkg/v2/<package>"
```

**Go Version:** 1.26+ (generics required)
**Install:** `go get github.com/jasoet/pkg/v2@latest`
**v1 (no OTel):** `go get github.com/jasoet/pkg@v1.6.0` — preserved on [`release/v1`](https://github.com/jasoet/pkg/tree/release/v1) branch, unmaintained.
**Project Template:** See [PROJECT_TEMPLATE.md](PROJECT_TEMPLATE.md) for recommended project structure, wiring patterns, test tiers (E2E), Swagger/OpenAPI setup, and Taskfile targets.

## Core Patterns

### Functional Options

All packages use functional options for flexible, backward-compatible configuration. Add `With*()` option functions rather than expanding struct fields directly.

```go
exec, _ := docker.New(
    docker.WithImage("nginx:alpine"),
    docker.WithPorts("80:8080"),
    docker.WithOTelConfig(otelConfig),
)
```

> Every module's README documents its available options.

### OpenTelemetry Integration

All packages follow the same OTel pattern:

1. Config struct has `OTelConfig *otel.Config` field with `yaml:"-" mapstructure:"-"` (never serialized)
2. Runtime injection via `WithOTelConfig()` functional option
3. Access providers: `cfg.GetTracer()`, `cfg.GetMeter()`, `cfg.GetLogger()`
4. Context propagation: `otel.ContextWithConfig(ctx, cfg)` / `otel.ConfigFromContext(ctx)`
5. All providers default to no-op when nil (zero overhead, no nil checks needed)

> **Details:** [otel/README.md](otel/README.md) | [otel/instrumentation_example_test.go](otel/instrumentation_example_test.go)

### LayerContext (Unified Span + Logger)

Use `otel.Layers.Start*()` for automatic span + logger correlation:

```go
lc := otel.Layers.StartService(ctx, "user", "Create", otel.F("user_id", id))
defer lc.End()

lc.Logger.Info("Creating user")

if err := repo.Save(lc.Context(), user); err != nil {
    return lc.Error(err, "failed to save user")
}
return lc.Success("User created successfully")
```

Available layers: `Handler`, `Service`, `Repository`, `Operations`, `Middleware`
Span naming: `{layer}.{component}.{operation}`

> **Details:** [otel/README.md](otel/README.md)

### Configuration Layers

Three-layer strategy: YAML files -> environment variable overrides -> functional options at runtime.

```go
cfg, err := config.LoadString[AppConfig](yamlContent, "APP")
// Override via env: APP_SERVER_PORT=9090
```

> **Details:** [config/README.md](config/README.md)

## Package Reference

| Package | Description | README | Examples |
|---------|-------------|--------|----------|
| [otel](./otel/) | OpenTelemetry unified config (tracing, metrics, logging) | [README](otel/README.md) | [examples_test.go](otel/examples_test.go), [instrumentation_example_test.go](otel/instrumentation_example_test.go) |
| [config](./config/) | Type-safe YAML config with env overrides and validation | [README](config/README.md) | [examples/](config/examples/) |
| [logging](./logging/) | Structured logging with zerolog + OTel LoggerProvider | [README](logging/README.md) | [examples/](logging/examples/) |
| [db](./db/) | Multi-database (PostgreSQL, MySQL, MSSQL) with GORM + migrations | [README](db/README.md) | [examples/](db/examples/) |
| [docker](./docker/) | Container executor with dual API (functional + struct) | [README](docker/README.md) | [examples/](docker/examples/) |
| [server](./server/) | HTTP server with Echo, health checks, graceful shutdown | [README](server/README.md) | [examples/](server/examples/) |
| [grpc](./grpc/) | gRPC server with Echo gateway, H2C + separate modes | [README](grpc/README.md) | [examples/](grpc/examples/) |
| [rest](./rest/) | HTTP client with retries, middleware, OTel tracing | [README](rest/README.md) | [examples/](rest/examples/) |
| [concurrent](./concurrent/) | Type-safe parallel execution with generics | [README](concurrent/README.md) | [examples/](concurrent/examples/) |
| [temporal](./temporal/) | Temporal workflows, workers, scheduling, monitoring | [README](temporal/README.md) | [examples/](temporal/examples/) |
| [ssh](./ssh/) | SSH tunneling and port forwarding | [README](ssh/README.md) | [examples/](ssh/examples/) |
| [compress](./compress/) | File compression (gzip, tar.gz) with security validation | [README](compress/README.md) | [examples/](compress/examples/) |
| [argo](./argo/) | Argo Workflows client with builder API and patterns | [README](argo/README.md) | [examples/](argo/examples/) |
| [retry](./retry/) | Retry with exponential backoff, OTel, permanent errors | [README](retry/README.md) | [examples/](retry/examples/) |
| [base32](./base32/) | Crockford Base32 encoding with CRC-10 checksums | [README](base32/README.md) | [examples/](base32/examples/) |

## Common Tasks

### Connect to a Database

```go
pool, _ := db.ConnectionConfig{
    DbType: db.Postgresql, Host: "localhost", Port: 5432,
    Username: "user", Password: "pass", DbName: "mydb",
    OTelConfig: otelConfig,
}.Pool()
```

> [db/README.md](db/README.md) for migrations, multi-DB, connection pooling.

### Start an HTTP Server

```go
cfg := server.DefaultConfig(8080, operation, shutdown)
server.StartWithConfig(cfg)
```

> [server/README.md](server/README.md) for health checks, middleware, EchoConfigurer.

### Add Retry Logic

```go
cfg := retry.DefaultConfig().WithName("db.connect").WithOTel(otelConfig)
err := retry.Do(ctx, cfg, func(ctx context.Context) error { return db.Ping(ctx) })
```

> [retry/README.md](retry/README.md) for permanent errors, notifications, config options.

### Run Concurrent Tasks

```go
results, err := concurrent.ExecuteConcurrently(ctx, funcs)
```

> [concurrent/README.md](concurrent/README.md) for typed results, error aggregation.

### Create gRPC Server with Gateway

```go
srv, _ := grpc.New(
    grpc.WithGRPCPort("9090"),
    grpc.WithServiceRegistrar(registrar),
    grpc.WithOTelConfig(otelConfig),
)
```

> [grpc/README.md](grpc/README.md) for H2C mode, separate mode, health checks.
