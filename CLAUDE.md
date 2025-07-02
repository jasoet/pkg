# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

This project uses [Mage](https://magefile.org/) for build automation. Common commands:

```bash
# Run unit tests
mage test

# Run integration tests (starts Docker services automatically)
mage integrationTest

# Run linter (installs golangci-lint if not present)
mage lint

# Clean build artifacts
mage clean

# Docker service management
mage docker:up        # Start PostgreSQL and other services
mage docker:down      # Stop services and remove volumes
mage docker:logs      # View service logs
mage docker:restart   # Restart all services
```

## Development Environment

- **PostgreSQL**: localhost:5439 (user: jasoet, password: localhost, database: pkg_db)
- **Docker Compose**: Services defined in `scripts/compose/docker-compose.yml`
- **Integration Tests**: Use `AUTOMATION=true` environment variable and `-tags=integration`

## Architecture Overview

This is a Go utility library providing reusable infrastructure components. The packages are designed to work together while remaining modular:

### Core Packages

- **config**: YAML configuration loading with environment variable overrides using Viper
- **logging**: Structured logging with zerolog, provides centralized setup and context-aware loggers
- **concurrent**: Type-safe concurrent execution utilities using Go generics
- **db**: Multi-database support (MySQL, PostgreSQL, MSSQL) with GORM and migrations
- **rest**: HTTP client framework with middleware support built on Resty
- **server**: Echo-based HTTP server with health checks, metrics, and graceful shutdown
- **temporal**: Temporal workflow engine integration with workers and scheduling
- **ssh**: SSH tunneling utilities for secure remote connections
- **compress**: File compression and archive utilities with security validations

### Key Patterns

- **Configuration**: YAML-first with environment variable overrides, validation via struct tags
- **Logging**: All packages integrate with the central logging package for consistency
- **Generics**: Extensive use of Go generics for type safety (config loading, concurrent execution)
- **Lifecycle Management**: Consistent Start/Stop patterns with context-based cancellation
- **Error Handling**: Custom error types with context information and error wrapping

### Dependencies

- **logging** package is used by db, temporal, and server packages
- External integrations: GORM (databases), Temporal (workflows), Prometheus (metrics)
- Docker Compose provides PostgreSQL for integration testing

## Testing

- Unit tests: `go test ./...` or `mage test`
- Integration tests: `go test -tags=integration ./...` or `mage integrationTest`
- Integration tests automatically start required Docker services
- Test database: Uses the same PostgreSQL configuration as development

## Code Conventions

- Follow standard Go conventions and idioms
- Use struct tags for configuration validation
- Implement graceful shutdown patterns for services
- Use context-aware logging with component identification
- Prefer composition over inheritance for middleware and configuration
- Use generics for type-safe APIs where appropriate