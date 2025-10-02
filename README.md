# Go Utility Packages (v2)

[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/jasoet/pkg/actions/workflows/release.yml/badge.svg)](https://github.com/jasoet/pkg/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-85%25-brightgreen.svg)](https://github.com/jasoet/pkg)
[![Go Report Card](https://goreportcard.com/badge/github.com/jasoet/pkg/v2)](https://goreportcard.com/report/github.com/jasoet/pkg/v2)

Production-ready Go utility packages with **OpenTelemetry v2** instrumentation, comprehensive testing, and battle-tested components for building modern cloud-native applications.

## 🎯 Version 2 Status

**Current Release:** `v2.0.0-beta.2`
**Status:** Release Candidate - Ready for v2.0.0 GA
**Test Coverage:** 85% (excludes generated code)
**Breaking Changes:** OpenTelemetry v2 migration

> **v2 Highlights:** Full OpenTelemetry v2 support, 85% test coverage, modernized dependencies, enhanced observability

See [VERSIONING_GUIDE.md](VERSIONING_GUIDE.md) for migration instructions and versioning workflow.

## 📦 Packages

Production-ready components with comprehensive observability, testing, and examples:

| Package | Description | Coverage | Key Features |
|---------|-------------|----------|--------------|
| **[otel](./otel/)** | OpenTelemetry v2 integration | 97.1% | Tracing, metrics, logging, unified config |
| **[config](./config/)** | YAML configuration with env overrides | 94.7% | Type-safe, validation, hot-reload |
| **[logging](./logging/)** | Structured logging with zerolog | 82.0% | Context-aware, OTel integration |
| **[db](./db/)** | Multi-database support | 79.1% | PostgreSQL, MySQL, MSSQL, migrations, OTel |
| **[server](./server/)** | HTTP server with Echo | 83.0% | Health checks, metrics, graceful shutdown |
| **[grpc](./grpc/)** | gRPC server with Echo gateway | 82.0% | H2C mode, dual protocol, observability |
| **[rest](./rest/)** | HTTP client framework | 92.9% | Retries, timeouts, OTel tracing |
| **[concurrent](./concurrent/)** | Type-safe concurrent execution | 100% | Generics, error handling, cancellation |
| **[temporal](./temporal/)** | Temporal workflow integration | 86.4% | Workers, scheduling, monitoring |
| **[ssh](./ssh/)** | SSH tunneling utilities | 76.7% | Secure connections, port forwarding |
| **[compress](./compress/)** | File compression utilities | 86.3% | ZIP, tar.gz, security validation |

**Overall Coverage:** 85.0% (unit + integration tests, excludes generated protobuf code)

## 🚀 Quick Start

### Installation

```bash
# Latest stable v1 (production)
go get github.com/jasoet/pkg

# v2 Beta (OpenTelemetry v2)
go get github.com/jasoet/pkg/v2@v2.0.0-beta.2
```

### Basic Usage

```go
package main

import (
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/server"
    "github.com/jasoet/pkg/v2/otel"
)

func main() {
    // Load configuration
    cfg := config.NewConfig("config.yml")

    // Setup OpenTelemetry
    otelConfig := otel.NewConfig("my-service").
        WithTracerProvider(/* your tracer */).
        WithMeterProvider(/* your meter */)

    // Setup logging with OTel
    logger := logging.NewLogger(logging.Config{
        Level:      "info",
        OTelConfig: otelConfig,
    })

    // Start HTTP server with observability
    server.Start(server.Config{
        Port:       8080,
        Logger:     logger,
        OTelConfig: otelConfig,
    })
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

📖 **[Browse Package Examples](./examples/)**

## 🔬 Test Coverage

**Overall: 85.0%** (unit + integration tests)

### Coverage by Package
- ✅ **Excellent (80%+):** concurrent (100%), otel (97.1%), config (94.7%), rest (92.9%), compress (86.3%), temporal (86.4%), server (83.0%), grpc (82.0%), logging (82.0%)
- ✅ **Good (70-80%):** db (79.1%), ssh (76.7%)

### Testing Strategy
- **Unit Tests:** Pure logic, no dependencies (`task test`)
- **Integration Tests:** Real databases, SSH servers, Temporal (`task test:integration`)
- **Complete Coverage:** Unit + Integration (`task test:all`)
- **Tools:** testcontainers for real services, assert library for assertions

### Run Tests

```bash
# Unit tests only (no Docker required)
task test

# Integration tests (requires Docker)
task test:integration

# All tests with complete coverage report
task test:all
open output/coverage-all.html
```

## 🎯 Key Features

### OpenTelemetry v2 Integration
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
- **Go 1.25+ Generics:** Type-safe parallel execution
- **Error Handling:** Aggregate errors from concurrent operations
- **Context Support:** Cancellation and timeout handling
- **Resource Management:** Automatic goroutine cleanup

## 🔧 Development

### Prerequisites

- **Go 1.25+** - Latest Go version with enhanced generics
- **[Task](https://taskfile.dev/)** - Modern task runner
- **Docker & Docker Compose** - For integration tests and services

### Development Commands

```bash
# Setup
task docker:up          # Start PostgreSQL and services
task dependencies       # Check and update dependencies

# Testing
task test               # Unit tests
task test:integration   # Integration tests (Docker required)
task test:all           # All tests with coverage
task coverage           # Generate coverage report

# Quality
task lint               # Run golangci-lint
task security           # Security analysis with gosec
task checkall           # Run all quality checks

# Docker Services
task docker:down        # Stop services
task docker:restart     # Restart services
task docker:logs        # View service logs
```

### Database Configuration

PostgreSQL available for testing:
- **Host:** localhost:5439
- **Username:** jasoet
- **Password:** localhost
- **Database:** pkg_db

## 🤖 AI Agent Instructions

**Repository Type:** Go utility library (v2) - production-ready infrastructure components with OpenTelemetry v2

**Critical Setup:**
```bash
task docker:up  # Start PostgreSQL (localhost:5439, user: jasoet, password: localhost, db: pkg_db)
```

**Architecture:**
- **11 core packages:** otel, config, logging, db, server, grpc, rest, concurrent, temporal, ssh, compress
- **Integration-ready:** Packages work seamlessly together
- **Examples:** Each package has runnable examples with `go run -tags=example ./package/examples`
- **Module Path:** `github.com/jasoet/pkg/v2` (Go v2+ semantics)

**Key Development Patterns:**
- **OpenTelemetry:** v2 instrumentation across all packages (otel package)
- **Configuration:** Type-safe YAML with environment variable overrides (config package)
- **Database:** Multi-database support with GORM, migrations, OTel tracing (db package)
- **HTTP Server:** Echo framework with health checks, metrics, graceful shutdown (server package)
- **gRPC:** Production-ready server with Echo gateway, H2C mode, observability (grpc package)
- **REST Client:** Resilient HTTP client with retries, OTel tracing (rest package)
- **Logging:** Zerolog with OTel log provider integration (logging package)
- **Concurrency:** Type-safe parallel execution with Go 1.25+ generics (concurrent package)
- **Workflows:** Temporal integration for distributed workflows (temporal package)

**Testing Strategy:**
- **Coverage:** 85% (unit + integration, excludes generated code)
- **Unit Tests:** `task test` (no Docker, race detection enabled)
- **Integration Tests:** `task test:integration` (testcontainers, Docker required)
- **All Tests:** `task test:all` (complete coverage, generates output/coverage-all.html)
- **Assertion Library:** Use `github.com/stretchr/testify/assert` for all test assertions
- **Test Categories:**
  - Unit: No build tags, no external dependencies
  - Integration: Build tag `integration`, uses testcontainers

**Library Usage Focus:**
- Emphasize zero-configuration startup
- Type safety with Go 1.25+ generics
- Production-grade features: health endpoints, metrics, observability, graceful shutdown
- OpenTelemetry v2 as first-class citizen

**Version Information:**
- **Current:** v2.0.0-beta.2 (OpenTelemetry v2)
- **Stable:** v1.6.0 (OpenTelemetry v1)
- **Migration Guide:** See [VERSIONING_GUIDE.md](VERSIONING_GUIDE.md)

## 📚 Package Documentation

### Core Infrastructure

#### [otel](./otel/) - OpenTelemetry v2 Integration
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

cfg := config.NewConfig("config.yml")
var appCfg AppConfig
cfg.Unmarshal(&appCfg)
```

**Features:** Hot-reload, validation, environment overrides
**Coverage:** 94.7% | **[Examples](./config/examples/)** | **[Documentation](./config/README.md)**

#### [logging](./logging/) - Structured Logging
Zerolog-based structured logging with OTel integration.

```go
logger := logging.NewLogger(logging.Config{
    Level:      "info",
    OTelConfig: otelConfig,
})

logger.Info().Str("user", "john").Msg("User logged in")
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

### HTTP & gRPC

#### [server](./server/) - HTTP Server
Echo-based HTTP server with built-in observability.

```go
server.Start(server.Config{
    Port:       8080,
    Logger:     logger,
    OTelConfig: otelConfig,
})
```

**Features:** Health checks, Prometheus metrics, graceful shutdown, middleware
**Coverage:** 83.0% | **[Examples](./server/examples/)** | **[Documentation](./server/README.md)**

#### [grpc](./grpc/) - gRPC Server
Production-ready gRPC with Echo gateway integration.

```go
cfg := grpc.NewConfig("my-service", 9090).
    WithGatewayPort(8080).
    WithH2CMode(true).
    WithOTelConfig(otelConfig)

srv := grpc.NewServer(cfg)
// Register your services
srv.Start()
```

**Features:** H2C mode, dual HTTP/gRPC, gateway, observability
**Coverage:** 82.0% | **[Examples](./grpc/examples/)** | **[Documentation](./grpc/README.md)**

#### [rest](./rest/) - HTTP Client
Resilient REST client with OTel tracing.

```go
client := rest.NewClient(rest.ClientConfig{
    BaseURL:    "https://api.example.com",
    Timeout:    30 * time.Second,
    RetryCount: 3,
    OTelConfig: otelConfig,
})

var result Response
client.Get("/endpoint", &result)
```

**Features:** Retries, circuit breaking, tracing, middleware support
**Coverage:** 92.9% | **[Examples](./rest/examples/)** | **[Documentation](./rest/README.md)**

### Utilities

#### [concurrent](./concurrent/) - Type-Safe Concurrency
Generics-based parallel execution with error handling.

```go
results, err := concurrent.Map(items, func(item Item) (Result, error) {
    return processItem(item)
})
```

**Features:** Go 1.25+ generics, error aggregation, context support
**Coverage:** 100% | **[Examples](./concurrent/examples/)** | **[Documentation](./concurrent/README.md)**

#### [temporal](./temporal/) - Workflow Orchestration
Temporal workflow integration with observability.

```go
manager := temporal.NewScheduleManager(temporal.Config{
    HostPort:   "localhost:7233",
    Namespace:  "default",
    OTelConfig: otelConfig,
})

manager.CreateWorkflowSchedule(ctx, scheduleID, workflow, schedule)
```

**Features:** Schedule management, workers, monitoring
**Coverage:** 86.4% | **[Examples](./temporal/examples/)** | **[Documentation](./temporal/README.md)**

#### [ssh](./ssh/) - SSH Tunneling
Secure SSH tunneling and port forwarding.

```go
tunnel := ssh.NewTunnel(ssh.Config{
    Host:       "remote-host",
    Port:       22,
    User:       "user",
    PrivateKey: privateKey,
})

tunnel.Start()
defer tunnel.Close()
```

**Features:** Port forwarding, connection pooling, error handling
**Coverage:** 76.7% | **[Examples](./ssh/examples/)** | **[Documentation](./ssh/README.md)**

#### [compress](./compress/) - File Compression
Secure file compression with validation.

```go
compress.ZipFiles(files, "output.zip")
compress.UnzipFile("archive.zip", "output/")
```

**Features:** ZIP, tar.gz, security validation, path traversal protection
**Coverage:** 86.3% | **[Examples](./compress/examples/)** | **[Documentation](./compress/README.md)**

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Workflow

1. **Fork & Clone**
   ```bash
   git clone https://github.com/your-username/pkg.git
   cd pkg
   ```

2. **Setup Development Environment**
   ```bash
   task docker:up
   task test
   ```

3. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes & Test**
   ```bash
   task test           # Unit tests
   task test:all       # Full coverage
   task lint           # Code quality
   task security       # Security check
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

See [VERSIONING_GUIDE.md](VERSIONING_GUIDE.md) for versioning workflow details.

## 📋 Version Support

| Version | Status | OpenTelemetry | Go Version | Support |
|---------|--------|---------------|------------|---------|
| v2.0.0-beta.2 | Beta | v2 (v1.38+) | 1.25+ | Active development |
| v1.6.0 | Stable | v1 (v1.x) | 1.23+ | Critical fixes (6 months) |
| < v1.6.0 | Legacy | v1 (v1.x) | 1.23+ | No support |

**Migration:** See [VERSIONING_GUIDE.md](VERSIONING_GUIDE.md) for v1 to v2 migration instructions.

## 📈 Roadmap

### ✅ Completed
- [x] Core packages (11 components)
- [x] OpenTelemetry v2 integration
- [x] 85% test coverage (unit + integration)
- [x] Integration examples
- [x] Task-based development workflow
- [x] CI/CD pipeline with automated testing
- [x] Comprehensive documentation
- [x] gRPC & Protobuf support with Echo gateway
- [x] Testcontainer-based integration tests

### 🚧 In Progress (v2.0.0 GA)
- [ ] Review and update all package READMEs
- [ ] Review and update all example READMEs
- [ ] Ensure all examples demonstrate OTel integration
- [ ] Create fullstack OTel example application (examples/fullstack-otel)
- [ ] v2.0.0 GA release

### 📝 Planned (Post v2.0 GA)
- [ ] **Docker Executor Package** - Production-ready Docker execution helper inspired by testcontainer simplicity
  - Execute short-lived jobs in containers
  - Use cases: Infrastructure workflows, data pipelines, batch processing
  - Foundation for Temporal workflow integration
- [ ] **Temporal Docker Workflows** - Reusable Temporal workflows for Docker container execution
  - Pre-built workflow templates for containerized jobs
  - Integration with docker executor package
  - Observability and error handling patterns

## 🔗 Links

- **Documentation:** [Browse Package Docs](./docs/)
- **Examples:** [All Examples](./examples/)
- **Versioning Guide:** [VERSIONING_GUIDE.md](VERSIONING_GUIDE.md)
- **Contributing:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Changelog:** [Releases](https://github.com/jasoet/pkg/releases)
- **Issues:** [GitHub Issues](https://github.com/jasoet/pkg/issues)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⬆ Back to Top](#go-utility-packages-v2)**

Made with ❤️ by [Jasoet](https://github.com/jasoet)

</div>
