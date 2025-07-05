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

This project uses [Mage](https://magefile.org/) for build automation. Common commands:

### Basic Commands
```bash
# Run unit tests
mage test

# Run database integration tests (starts PostgreSQL/MySQL/MSSQL automatically)
mage integrationTest

# Run Temporal integration tests (starts Temporal server automatically)
mage temporalTest

# Run all integration tests (database + temporal)
mage allIntegrationTests

# Run linter (installs golangci-lint if not present)
mage lint

# Clean build artifacts
mage clean
```

### Development Tools & Quality Checks
```bash
# Install all development tools (golangci-lint, gosec, nancy, etc.)
mage tools

# Run security analysis with gosec
mage security

# Check dependencies for known vulnerabilities
mage dependencies

# Generate test coverage report (creates coverage.html)
mage coverage

# Generate API documentation (if swagger annotations exist)
mage docs

# Run all quality checks (test, lint, security, dependencies, coverage)
mage checkall
```

### Docker Service Management
```bash
mage docker:up        # Start PostgreSQL and other services
mage docker:down      # Stop services and remove volumes
mage docker:logs      # View service logs
mage docker:restart   # Restart all services

# Temporal Service Management
mage temporal:up      # Start Temporal server with PostgreSQL
mage temporal:down    # Stop Temporal services and remove volumes
mage temporal:logs    # View Temporal service logs
mage temporal:restart # Restart Temporal services
```

## Development Environment

- **PostgreSQL**: localhost:5439 (user: jasoet, password: localhost, database: pkg_db)
- **Temporal Server**: localhost:7233 (for integration tests)
- **Temporal UI**: localhost:8233 (when running temporal tests)
- **Docker Compose**: Services defined in `scripts/compose/docker-compose.yml`
- **Temporal Compose**: Services defined in `scripts/compose/temporal-compose.yml`
- **Database Integration Tests**: Use `AUTOMATION=true` environment variable and `-tags=integration`
- **Temporal Integration Tests**: Use `TEMPORAL_INTEGRATION=true` environment variable and `-tags=temporal`

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