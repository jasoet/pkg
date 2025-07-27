# 🚀 Go Utility Packages

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/jasoet/pkg/actions/workflows/release.yml/badge.svg)](https://github.com/jasoet/pkg/actions)

A comprehensive collection of production-ready Go utility packages designed to eliminate boilerplate and standardize common patterns across Go applications. These battle-tested components integrate seamlessly to accelerate development while maintaining best practices.

## 📦 What's Inside

| Package | Description | Key Features |
|---------|-------------|--------------|
| **[config](./config/)** | YAML configuration with env overrides | Type-safe, validation, hot-reload |
| **[logging](./logging/)** | Structured logging with zerolog | Context-aware, performance optimized |
| **[db](./db/)** | Multi-database support | PostgreSQL, MySQL, MSSQL, migrations |
| **[server](./server/)** | HTTP server with Echo | Health checks, metrics, graceful shutdown |
| **[rest](./rest/)** | HTTP client framework | Retries, timeouts, middleware support |
| **[concurrent](./concurrent/)** | Type-safe concurrent execution | Generics, error handling, cancellation |
| **[temporal](./temporal/)** | Temporal workflow integration | Workers, scheduling, monitoring |
| **[ssh](./ssh/)** | SSH tunneling utilities | Secure connections, port forwarding |
| **[compress](./compress/)** | File compression utilities | ZIP, tar.gz with security validation |

## 🎯 Quick Start

### Installation

```bash
go get github.com/jasoet/pkg
```

### Basic Usage

This library provides production-ready infrastructure components. Each package has comprehensive examples and documentation:

**🚀 Jump to Examples:**
- [Configuration Examples](config/examples/README.md) - YAML config with environment overrides
- [Logging Examples](logging/examples/README.md) - Structured logging setup
- [Server Examples](server/examples/README.md) - HTTP server with health checks
- [Database Examples](db/examples/README.md) - Multi-database connectivity
- [REST Client Examples](rest/examples/README.md) - HTTP client with retries
- [More examples below...](#-packages-overview)

## 📚 Packages Overview

This library provides 8 core packages, each with comprehensive examples and documentation:

| Package | Description | Key Features | Examples & Documentation |
|---------|-------------|--------------|--------------------------|
| **[config](./config/)** | YAML configuration with env overrides | Type-safe, validation, hot-reload | [📖 Examples & Guide](config/examples/README.md) |
| **[logging](./logging/)** | Structured logging with zerolog | Context-aware, performance optimized | [📖 Examples & Guide](logging/examples/README.md) |
| **[db](./db/)** | Multi-database support | PostgreSQL, MySQL, MSSQL, migrations | [📖 Examples & Guide](db/examples/README.md) |
| **[server](./server/)** | HTTP server with Echo | Health checks, metrics, graceful shutdown | [📖 Examples & Guide](server/examples/README.md) |
| **[rest](./rest/)** | HTTP client framework | Retries, timeouts, middleware support | [📖 Examples & Guide](rest/examples/README.md) |
| **[concurrent](./concurrent/)** | Type-safe concurrent execution | Generics, error handling, cancellation | [📖 Examples & Guide](concurrent/examples/README.md) |
| **[temporal](./temporal/)** | Temporal workflow integration | Workers, scheduling, monitoring | [📖 Examples & Guide](temporal/examples/README.md) |
| **[ssh](./ssh/)** | SSH tunneling utilities | Secure connections, port forwarding | [📖 Examples & Guide](ssh/examples/README.md) |
| **[compress](./compress/)** | File compression utilities | ZIP, tar.gz with security validation | [📖 Examples & Guide](compress/examples/README.md) |

## 🎭 Examples & Usage

### Running Examples

Each package has comprehensive examples isolated with build tags:

```bash
# Run specific package examples
go run -tags=example ./logging/examples
go run -tags=example ./db/examples
go run -tags=example ./server/examples

# Build all examples
go build -tags=example ./...
```

### Example Categories

- **Basic Usage**: Simple getting-started examples
- **Integration Patterns**: Real-world usage with multiple packages
- **Production Scenarios**: Error handling, performance, security
- **Best Practices**: Recommended patterns and configurations

Each package's examples directory contains a comprehensive README with:
- Quick reference for LLMs/coding agents
- Step-by-step tutorials
- Common patterns and anti-patterns
- Integration examples with other packages

## 🔧 Development

### Prerequisites

- Go 1.23+
- [Task](https://taskfile.dev/) for build automation
- Docker & Docker Compose for services

### Development Commands

```bash
# Development environment
task docker:up          # Start PostgreSQL and other services
task test              # Run unit tests
task integration-test  # Run integration tests
task lint              # Run linter
task security          # Security analysis
task coverage          # Generate coverage report

# Docker services management
task docker:down       # Stop services
task docker:restart    # Restart services
task docker:logs       # View service logs

# Quality checks
task checkall          # Run all quality checks
task dependencies      # Check for vulnerabilities
task docs              # Generate documentation
```

### Database Configuration

PostgreSQL is available for testing:
- **Host**: localhost:5439
- **Username**: jasoet
- **Password**: localhost
- **Database**: pkg_db

## 🤝 Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Guide

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
   task test
   task lint
   task integration-test
   ```

5. **Commit with Conventional Commits**
   ```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: resolve issue"
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

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
**Breaking Changes**: Add `!` after type or `BREAKING CHANGE:` in footer

## 📈 Roadmap

- [x] **Core Packages**: All essential utilities implemented
- [x] **Integration Examples**: Real-world usage patterns
- [x] **Build Automation**: Task-based development workflow
- [x] **CI/CD Pipeline**: Automated testing and releases
- [x] **Comprehensive Documentation**: Examples and guides
- [ ] **Performance Benchmarks**: Optimization guides and metrics
- [ ] **Distributed Tracing**: OpenTelemetry integration
- [ ] **Additional Database Drivers**: MongoDB, Redis support
- [ ] **Cloud Provider Integrations**: AWS, GCP, Azure utilities
- [ ] **Kubernetes Helpers**: Service discovery, health checks

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⬆ Back to Top](#-go-utility-packages)**

Made with ❤️ by [Jasoet](https://github.com/jasoet)

</div>