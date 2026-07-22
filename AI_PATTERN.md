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
3. Access providers: `cfg.GetTracer(scope)`, `cfg.GetMeter(scope)`, `cfg.GetLogger(scope)`
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
lc.Success("User created successfully")
return nil
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
| [config](./config/) | Type-safe YAML config with env overrides and validation | [README](config/README.md) | [examples/](examples/config/) |
| [logging](./logging/) | Structured logging with zerolog + OTel LoggerProvider | [README](logging/README.md) | [examples/](examples/logging/) |
| [db](./db/) | Multi-database (PostgreSQL, MySQL, MSSQL) with GORM + migrations | [README](db/README.md) | [examples/](examples/db/) |
| [docker](./docker/) | Container executor with dual API (functional + struct) | [README](docker/README.md) | [examples/](examples/docker/) |
| [server](./server/) | HTTP server with Echo, health checks, graceful shutdown | [README](server/README.md) | [examples/](examples/server/) |
| [grpc](./grpc/) | gRPC server with Echo gateway, H2C + separate modes | [README](grpc/README.md) | [examples/](examples/grpc/) |
| [rest](./rest/) | HTTP client with retries, middleware, OTel tracing | [README](rest/README.md) | [examples/](examples/rest/) |
| [concurrent](./concurrent/) | Type-safe parallel execution with generics | [README](concurrent/README.md) | [examples/](examples/concurrent/) |
| [temporal](./temporal/) | Temporal workflows, workers, scheduling, monitoring | [README](temporal/README.md) | [examples/](examples/temporal/) |
| [ssh](./ssh/) | SSH tunneling and port forwarding | [README](ssh/README.md) | [examples/](examples/ssh/) |
| [compress](./compress/) | File compression (gzip, tar.gz) with security validation | [README](compress/README.md) | [examples/](examples/compress/) |
| [argo](./argo/) | Argo Workflows client with builder API and patterns | [README](argo/README.md) | [examples/](examples/argo/) |
| [retry](./retry/) | Retry with exponential backoff, OTel, permanent errors | [README](retry/README.md) | [examples/](examples/retry/) |
| [base32](./base32/) | Crockford Base32 encoding with CRC-10 checksums | [README](base32/README.md) | [examples/](examples/base32/) |

## Common Tasks

### Connect to a Database

```go
pool, _ := db.ConnectionConfig{
    DBType: db.Postgresql, Host: "localhost", Port: 5432,
    Username: "user", Password: "pass", DBName: "mydb",
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
cfg := retry.New(retry.WithName("db.connect"), retry.WithOTelConfig(otelConfig))
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
