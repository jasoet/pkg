# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a comprehensive Go utility library (`github.com/jasoet/pkg`) that provides production-ready infrastructure components designed to eliminate boilerplate code and standardize common patterns across Go applications.

### Purpose
- **Accelerate Go application development** with battle-tested components
- **Standardize common patterns** across Go projects
- **Reduce boilerplate code** through generic, type-safe utilities
- **Provide a cohesive ecosystem** where packages integrate naturally
- **Maintain production-grade quality** with extensive testing

### Target Users
- Go developers building microservices
- Teams needing standardized infrastructure components
- Projects requiring database, HTTP, and workflow capabilities
- Applications needing production-grade utilities

### Key Design Principles
1. **Type Safety**: Extensive use of Go 1.23+ generics
2. **Configuration-First**: YAML-based with environment overrides
3. **Consistent Lifecycle**: Start/Stop patterns with graceful shutdown
4. **Integration-Ready**: All packages work seamlessly together
5. **Production-Grade**: Built-in health checks, metrics, error handling

## Development Commands

This project uses [Task](https://taskfile.dev/) for build automation. Common commands:

### Basic Commands
```bash
# Run unit tests
task test

# Run database integration tests (starts PostgreSQL/MySQL/MSSQL automatically)
task integration-test

# Run Temporal integration tests (starts Temporal server automatically)
task temporal-test

# Run all integration tests (database + temporal)
task all-integration-tests

# Run linter (installs golangci-lint if not present)
task lint

# Clean build artifacts
task clean
```

### Development Tools & Quality Checks
```bash
# Install all development tools (golangci-lint, gosec, nancy, etc.)
task tools

# Run security analysis with gosec
task security

# Check dependencies for known vulnerabilities
task dependencies

# Generate test coverage report (creates coverage.html)
task coverage

# Generate API documentation (if swagger annotations exist)
task docs

# Run all quality checks (test, lint, security, dependencies, coverage)
task checkall
```

### Docker Service Management
```bash
task docker:up        # Start PostgreSQL and other services
task docker:down      # Stop services and remove volumes
task docker:logs      # View service logs
task docker:restart   # Restart all services

# Temporal Service Management
task temporal:up      # Start Temporal server with PostgreSQL
task temporal:down    # Stop Temporal services and remove volumes
task temporal:logs    # View Temporal service logs
task temporal:restart # Restart Temporal services
```

## Development Environment

- **PostgreSQL**: localhost:5439 (user: jasoet, password: localhost, database: pkg_db)
- **Temporal Server**: localhost:7233 (for integration tests)
- **Temporal UI**: localhost:8233 (when running temporal tests)
- **Docker Compose**: Services defined in `scripts/compose/docker-compose.yml`
- **Temporal Compose**: Services defined in `scripts/compose/temporal-compose.yml`
- **Database Integration Tests**: Use `-tags=integration` build tag
- **Temporal Integration Tests**: Use `-tags=temporal` build tag

## Examples and Documentation

### Package Examples Structure

Each package has comprehensive examples and documentation in its `examples/` directory:

- **[config/examples/README.md](config/examples/README.md)** - YAML configuration with environment overrides
- **[logging/examples/README.md](logging/examples/README.md)** - Structured logging with zerolog
- **[db/examples/README.md](db/examples/README.md)** - Multi-database support and migrations
- **[server/examples/README.md](server/examples/README.md)** - HTTP server with Echo framework
- **[rest/examples/README.md](rest/examples/README.md)** - HTTP client with retries and middleware
- **[concurrent/examples/README.md](concurrent/examples/README.md)** - Type-safe concurrent execution
- **[temporal/examples/README.md](temporal/examples/README.md)** - Temporal workflow integration
- **[ssh/examples/README.md](ssh/examples/README.md)** - SSH tunneling utilities
- **[compress/examples/README.md](compress/examples/README.md)** - File compression with security

### Example Features

Each package's examples README contains:
- **📍 Example Code Location** - Direct GitHub links to example files
- **🚀 Quick Reference for LLMs/Coding Agents** - Copy-paste code snippets
- **Step-by-step tutorials** - Detailed usage instructions
- **Integration patterns** - How packages work together
- **Best practices** - Recommended patterns and anti-patterns

### Running Examples

```bash
# Run specific package examples
go run -tags=example ./logging/examples
go run -tags=example ./db/examples

# Build all examples
go build -tags=example ./...
```

## Commit and Development Guidelines

### Commit Guidelines
- **Workflow Rule**: Do not automatically commit and push code
- **Approval Process**: Wait for explicit confirmation before committing
- **Commit Type Selection**:
  - Use `chore` for maintenance tasks
  - Use `docs` for documentation updates
  - Use `fix` for bug fixes
  - Use `feat` only for significant functionality changes

### Code Review Process
- Every time a task is completed, do not commit and push the code
- Prepare the changes for review
- Wait for explicit approval before committing
- When approved, use conventional commit messages
- Select the most appropriate commit type based on the nature of changes

## Architecture Overview

[Rest of the existing file content remains unchanged...]