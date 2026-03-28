# Go Utility Packages (v2)

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/jasoet/pkg/actions/workflows/release.yml/badge.svg)](https://github.com/jasoet/pkg/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-85%25-brightgreen.svg)](https://github.com/jasoet/pkg)
[![Go Report Card](https://goreportcard.com/badge/github.com/jasoet/pkg/v2)](https://goreportcard.com/report/github.com/jasoet/pkg/v2)

Production-ready Go utility packages with **OpenTelemetry** instrumentation, comprehensive testing, and battle-tested components for building modern cloud-native applications.

## Versioning

**Current Release:** `v2.9.0` (GA)
**Status:** Production Ready
**Test Coverage:** 85%

> **v2 Highlights:** OpenTelemetry instrumentation across all packages, 85% test coverage, modernized dependencies
>
> **Breaking Change:** v1 does not include OpenTelemetry. v2 adds optional OTel support with minimal API changes.

### v1 Availability

The v1 release is preserved on the [`release/v1`](https://github.com/jasoet/pkg/tree/release/v1) branch (final version: v1.6.0). Use v1 if you do not need OpenTelemetry instrumentation:

```bash
go get github.com/jasoet/pkg@v1.6.0
```

**Note:** v1 is no longer actively maintained. All new development targets v2.

## Packages

Production-ready components with comprehensive observability, testing, and examples:

| Package | Description | Key Features |
|---------|-------------|--------------|
| **[otel](./otel/)** | OpenTelemetry integration | Tracing, metrics, logging, unified config |
| **[config](./config/)** | YAML configuration with env overrides | Type-safe, validation, hot-reload |
| **[logging](./logging/)** | Structured logging with zerolog | Context-aware, OTel integration |
| **[db](./db/)** | Multi-database support | PostgreSQL, MySQL, MSSQL, migrations, OTel |
| **[docker](./docker/)** | Docker container executor | Lifecycle management, wait strategies, dual API |
| **[argo](./argo/)** | Argo Workflows client | Kubernetes API, Argo Server, OTel, flexible config |
| **[server](./server/)** | HTTP server with Echo | Health checks, metrics, graceful shutdown |
| **[grpc](./grpc/)** | gRPC server with Echo gateway | H2C mode, dual protocol, observability |
| **[rest](./rest/)** | HTTP client framework | Retries, timeouts, OTel tracing |
| **[retry](./retry/)** | Retry with exponential backoff | Context-aware, OTel tracing, permanent errors |
| **[concurrent](./concurrent/)** | Type-safe concurrent execution | Generics, error handling, cancellation |
| **[temporal](./temporal/)** | Temporal workflow integration | Workers, scheduling, monitoring |
| **[ssh](./ssh/)** | SSH tunneling utilities | Secure connections, port forwarding |
| **[base32](./base32/)** | Crockford Base32 encoding | Human-readable IDs, CRC-10 checksums, error correction |
| **[compress](./compress/)** | File compression utilities | ZIP, tar.gz, security validation |

## Quick Start

### Installation

```bash
go get github.com/jasoet/pkg/v2@latest
```

### Basic Usage

```go
package main

import (
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/server"
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"
)

type AppConfig struct {
    Port int `yaml:"port"`
}

func main() {
    // Setup logging
    if err := logging.Initialize("my-service", false); err != nil {
        log.Fatal().Err(err).Msg("failed to initialize logging")
    }

    // Load configuration
    cfg, _ := config.LoadString[AppConfig](`port: 8080`)

    // Define routes
    operation := func(e *echo.Echo) {
        e.GET("/", func(c echo.Context) error {
            return c.String(200, "Hello!")
        })
    }

    shutdown := func(e *echo.Echo) {
        // cleanup
    }

    // Start HTTP server
    serverCfg := server.DefaultConfig(cfg.Port, operation, shutdown)
    if err := server.StartWithConfig(serverCfg); err != nil {
        log.Fatal().Err(err).Msg("server failed")
    }
}
```

### Examples

Each package includes comprehensive examples:

```bash
# Run specific package examples
go run -tags=example ./logging/examples
go run -tags=example ./db/examples
go run -tags=example ./server/examples

# Build all examples
go build -tags=example ./...
```

Each package's examples live in its own `examples/` subdirectory (e.g. `./logging/examples/`).

## Test Coverage

**Overall Coverage: 85%**

### Package Coverage
- concurrent (100%), otel (97%), config (95%), rest (93%), compress (86%), temporal (86%), docker (84%), server (83%), grpc (82%), logging (82%), db (79%), ssh (77%)

### Run Tests

```bash
# Unit tests
task test

# Integration tests (Docker required)
task test:integration

# Complete test suite with coverage report
task test:complete
open output/coverage-all.html
```

## Key Features

### OpenTelemetry Integration
- **Unified Configuration:** Single config for tracing, metrics, and logging
- **Automatic Instrumentation:** Built-in for HTTP, gRPC, database operations
- **Context Propagation:** Distributed tracing across services
- **Metrics Collection:** Prometheus-compatible metrics

### Database Support
- **Multi-Database:** PostgreSQL, MySQL, MSSQL with GORM
- **Migrations:** Automated schema management with golang-migrate
- **Connection Pooling:** Configurable with health monitoring
- **OTel Tracing:** Automatic query tracing and metrics

### HTTP & gRPC Servers
- **Echo Framework:** Modern HTTP server with middleware support
- **gRPC Gateway:** Dual HTTP/gRPC protocol support with H2C mode
- **Health Checks:** Built-in health endpoints
- **Graceful Shutdown:** Proper resource cleanup

### Resilient REST Client
- **Retry Logic:** Configurable exponential backoff
- **Circuit Breaking:** Fail-fast patterns
- **Request Tracing:** Automatic distributed tracing
- **Middleware Support:** Custom request/response handlers

### Type-Safe Concurrency
- **Go 1.26+ Generics:** Type-safe parallel execution
- **Error Handling:** Aggregate errors from concurrent operations
- **Context Support:** Cancellation and timeout handling
- **Resource Management:** Automatic goroutine cleanup

## Development

### Development Setup

**Prerequisites:**

- **Nix** (with flakes enabled): install via [Determinate Nix Installer](https://install.determinate.systems/nix)
- **go-task** (global): `nix profile install nixpkgs#go-task`
- **gh** (optional): for PR management

**Quick start:**

```bash
task nix:check    # verify environment
task test          # run tests
task lint          # run linter
```

All `task` commands execute through `nix develop -c` automatically, so you do not need to manually enter a Nix shell.

### Development Commands

```bash
# Testing
# All integration tests use testcontainers, docker engine required
task test               # Unit tests
task test:integration   # Integration tests (Docker required)
task test:complete      # All tests with coverage

# Quality
task lint               # Run golangci-lint
```

## AI-Assisted Development

For AI assistants working on projects that **use** this library, see **[AI_PATTERN.md](AI_PATTERN.md)** — it covers core patterns, package reference with links to each module's README and examples, and common usage snippets.

For AI scaffolding new projects, see **[PROJECT_TEMPLATE.md](PROJECT_TEMPLATE.md)**.

## Package Documentation

### Core Infrastructure

#### [otel](./otel/) - OpenTelemetry Integration
Unified configuration for tracing, metrics, and logging.

```go
config := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider).
    WithLoggerProvider(loggerProvider)

tracer := config.GetTracer()
meter := config.GetMeter()
logger := config.GetLogger()
```

**Features:** Automatic instrumentation, context propagation, graceful shutdown
**Coverage:** 97.1% | **[Examples](./otel/examples/)** | **[Documentation](./otel/README.md)**

#### [config](./config/) - Configuration Management
Type-safe YAML configuration with environment variable overrides.

```go
type AppConfig struct {
    Server ServerConfig `yaml:"server"`
    DB     DBConfig     `yaml:"database"`
}

// Load from string with environment variable overrides (prefix: APP)
cfg, _ := config.LoadString[AppConfig](yamlContent, "APP")
// Override via env: APP_SERVER_PORT=9090
```

**Features:** Hot-reload, validation, environment overrides
**Coverage:** 94.7% | **[Examples](./config/examples/)** | **[Documentation](./config/README.md)**

#### [logging](./logging/) - Structured Logging
Zerolog-based OTel LoggerProvider with automatic trace correlation.

```go
// Create LoggerProvider
loggerProvider := logging.NewLoggerProvider("my-service", false)

// Use with OTel config
otelCfg := &otel.Config{
    LoggerProvider: loggerProvider,
    // ... other config
}

// Or use legacy zerolog
_ = logging.Initialize("my-service", false)
log.Info().Str("user", "john").Msg("User logged in")
```

**Features:** Context-aware, OTel log provider, performance optimized
**Coverage:** 82.0% | **[Examples](./logging/examples/)** | **[Documentation](./logging/README.md)**

### Data Access

#### [db](./db/) - Multi-Database Support
PostgreSQL, MySQL, MSSQL support with GORM and migrations.

```go
pool, _ := db.ConnectionConfig{
    DbType:     db.Postgresql,
    Host:       "localhost",
    Port:       5432,
    Username:   "user",
    Password:   "pass",
    DbName:     "mydb",
    OTelConfig: otelConfig,
}.Pool()

// Automatic query tracing and metrics
pool.Find(&users)
```

**Features:** Connection pooling, migrations, OTel tracing, health monitoring
**Coverage:** 79.1% | **[Examples](./db/examples/)** | **[Documentation](./db/README.md)**

#### [docker](./docker/) - Docker Container Executor
Production-ready Docker container management with dual API styles.

```go
// Functional options style
exec, _ := docker.New(
    docker.WithImage("nginx:alpine"),
    docker.WithPorts("80:8080"),
    docker.WithWaitStrategy(
        docker.WaitForLog("start worker processes"),
    ),
)

exec.Start(ctx)
defer exec.Terminate(ctx)

endpoint, _ := exec.Endpoint(ctx, "80/tcp")
// Use: http://localhost:8080

// Or struct-based (testcontainers-like)
req := docker.ContainerRequest{
    Image:        "postgres:18-alpine",
    ExposedPorts: []string{"5432/tcp"},
    Env: map[string]string{
        "POSTGRES_PASSWORD": "secret",
    },
    WaitingFor: docker.WaitForLog("ready to accept connections"),
}
exec, _ := docker.NewFromRequest(req)
```

**Features:** Lifecycle management, wait strategies, log streaming, dual API (functional + struct)
**Coverage:** 83.9% | **[Examples](./docker/examples/)** | **[Documentation](./docker/README.md)**

#### [argo](./argo/) - Argo Workflows Client
Production-ready Argo Workflows client with flexible configuration.

```go
// Default kubeconfig
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
if err != nil {
    return err
}
defer client.Close()

// Or with functional options
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithContext("production"),
    argo.WithOTelConfig(otelConfig),
)

// In-cluster mode
ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())

// Argo Server mode
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
)
```

**Features:** Multiple connection modes, functional options, OTel support, proper error handling
**[Examples](./argo/examples/)** | **[Documentation](./argo/README.md)**

#### [retry](./retry/) - Retry with Exponential Backoff
Production-ready retry mechanism using `cenkalti/backoff/v4` with OTel instrumentation.

```go
cfg := retry.DefaultConfig().
    WithName("db.connect").
    WithMaxRetries(3).
    WithOTel(otelConfig)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return db.Ping(ctx)
})

// For non-retryable errors:
return retry.Permanent(fmt.Errorf("invalid config"))
```

**Features:** Exponential backoff, context-aware, OTel tracing, permanent error marking
**[Documentation](./retry/README.md)**

#### [base32](./base32/) - Crockford Base32 Encoding
Crockford Base32 encoding with CRC-10 checksums for human-readable, error-correcting identifiers.

```go
// Encode a value to fixed-length Base32
id := base32.EncodeBase32(12345, 8) // "0000C1P9"

// Add checksum for error detection
idWithChecksum := base32.AppendChecksum(id)

// Validate and decode
if base32.ValidateChecksum(idWithChecksum) {
    value, _ := base32.DecodeBase32(base32.StripChecksum(idWithChecksum))
}

// Normalize user input (handles case, dashes, common typos)
normalized := base32.NormalizeBase32("ab-CD iL o9") // "ABCD1109"
```

**Features:** URL-safe alphabet, automatic error correction, CRC-10 checksums, compact encoding
**[Examples](./base32/examples/)** | **[Documentation](./base32/README.md)**

### HTTP & gRPC

#### [server](./server/) - HTTP Server
Echo-based HTTP server with built-in observability.

```go
operation := func(e *echo.Echo) {
    e.GET("/health", healthHandler)
}

shutdown := func(e *echo.Echo) {
    // cleanup
}

config := server.DefaultConfig(8080, operation, shutdown)
if err := server.StartWithConfig(config); err != nil {
    log.Fatal().Err(err).Msg("server failed")
}
```

**Features:** Health checks, graceful shutdown, middleware
**Coverage:** 83.0% | **[Examples](./server/examples/)** | **[Documentation](./server/README.md)**

#### [grpc](./grpc/) - gRPC Server
Production-ready gRPC with Echo gateway integration.

```go
server, _ := grpc.New(
    grpc.WithGRPCPort("9090"),
    grpc.WithOTelConfig(otelConfig),
    grpc.WithServiceRegistrar(func(s *grpc.Server) {
        // Register your gRPC services
        pb.RegisterYourServiceServer(s, &YourService{})
    }),
)

server.Start()
```

**Features:** H2C mode, dual HTTP/gRPC, gateway, observability
**Coverage:** 82.0% | **[Examples](./grpc/examples/)** | **[Documentation](./grpc/README.md)**

#### [rest](./rest/) - HTTP Client
Resilient REST client with OTel tracing.

```go
config := rest.Config{
    RetryCount:       3,
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 10 * time.Second,
    Timeout:          30 * time.Second,
}

client := rest.NewClient(
    rest.WithRestConfig(config),
    rest.WithOTelConfig(otelConfig),
)

response, _ := client.MakeRequestWithTrace(ctx, "GET", url, "", headers)
```

**Features:** Retries, circuit breaking, tracing, middleware support
**Coverage:** 92.9% | **[Examples](./rest/examples/)** | **[Documentation](./rest/README.md)**

### Utilities

#### [concurrent](./concurrent/) - Type-Safe Concurrency
Generics-based parallel execution with error handling.

```go
funcs := map[string]concurrent.Func[string]{
    "task1": func(ctx context.Context) (string, error) {
        return "result1", nil
    },
    "task2": func(ctx context.Context) (string, error) {
        return "result2", nil
    },
}

results, _ := concurrent.ExecuteConcurrently(ctx, funcs)
```

**Features:** Go 1.26+ generics, error aggregation, context support
**Coverage:** 100% | **[Examples](./concurrent/examples/)** | **[Documentation](./concurrent/README.md)**

#### [temporal](./temporal/) - Workflow Orchestration
Temporal workflow integration with observability.

```go
config := &temporal.Config{
    HostPort:  "localhost:7233",
    Namespace: "default",
}

client, _ := temporal.NewClient(config)
manager := temporal.NewScheduleManager(client)

manager.CreateWorkflowSchedule(ctx, scheduleID, workflow, schedule)
```

**Features:** Schedule management, workers, monitoring
**Coverage:** 86.4% | **[Examples](./temporal/examples/)** | **[Documentation](./temporal/README.md)**

#### [ssh](./ssh/) - SSH Tunneling
Secure SSH tunneling and port forwarding.

```go
config := ssh.Config{
    Host:       "remote-host",
    Port:       22,
    User:       "user",
    Password:   "password",
    RemoteHost: "db.internal",
    RemotePort: 5432,
    LocalPort:  15432,
}

tunnel := ssh.New(config)
tunnel.Start()
defer tunnel.Close()
```

**Features:** Port forwarding, connection pooling, error handling
**Coverage:** 76.7% | **[Examples](./ssh/examples/)** | **[Documentation](./ssh/README.md)**

#### [compress](./compress/) - File Compression
Secure file compression with validation.

```go
// Gzip compression
sourceFile, _ := os.Open("input.txt")
outputFile, _ := os.Create("output.gz")
compress.Gz(sourceFile, outputFile)

// Tar.gz archive
outputFile, _ := os.Create("archive.tar.gz")
compress.TarGz("/path/to/directory", outputFile)
```

**Features:** ZIP, tar.gz, security validation, path traversal protection
**Coverage:** 86.3% | **[Examples](./compress/examples/)** | **[Documentation](./compress/README.md)**

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Workflow

1. **Fork & Clone**
   ```bash
   git clone https://github.com/your-username/pkg.git
   cd pkg
   ```

2. **Setup Development Environment**
   ```bash
   task nix:check     # verify Nix tools
   task docker:check  # verify Docker/Podman
   task test
   ```

3. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes & Test**
   ```bash
   task test           # Unit tests
   task test:complete  # Full coverage
   task lint           # Code quality
   ```

5. **Commit with Conventional Commits**
   ```bash
   git commit -m "feat(package): add new feature"
   git commit -m "fix(package): resolve issue"
   git commit -m "docs: update README"
   ```

6. **Push & Create PR**
   ```bash
   git push origin feature/your-feature-name
   # Create pull request on GitHub
   ```

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `ci`
**Breaking Changes:** Add `!` after type or `BREAKING CHANGE:` in footer

For v1 availability, see the [Versioning](#versioning) section above.

## Roadmap

### Planned
- [ ] **Temporal Docker Workflows** - Reusable Temporal workflows for Docker container execution
  - Pre-built workflow templates for containerized jobs
  - Integration with docker executor package
  - Observability and error handling patterns

## Links

- **Documentation:** [Browse Package Docs](./docs/)
- **Examples:** [All Examples](./examples/)
- **v1 Branch:** [`release/v1`](https://github.com/jasoet/pkg/tree/release/v1) (v1.6.0, unmaintained)
- **Contributing:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Changelog:** [Releases](https://github.com/jasoet/pkg/releases)
- **Issues:** [GitHub Issues](https://github.com/jasoet/pkg/issues)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⬆ Back to Top](#go-utility-packages-v2)**

Made with ❤️ by [Jasoet](https://github.com/jasoet)

</div>
