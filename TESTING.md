# Testing Guide

This document explains the testing strategy and commands for this Go utility library.

## Test Categories

### 1. Unit Tests
**Purpose**: Test individual functions and components in isolation  
**Requirements**: No external dependencies  
**Command**: `task test`  
**Coverage**: All packages have unit tests

```bash
# Run all unit tests
task test

# Run unit tests for specific package
go test ./config/...
go test ./logging/...
```

### 2. Integration Tests
**Purpose**: Test with real dependencies (database, etc.)
**Requirements**: Docker (testcontainers)
**Command**: `task test:integration`
**Build Tag**: `integration`

```bash
# Run integration tests (uses testcontainers)
task test:integration

# Manual execution
go test -tags=integration ./...
```

**Covered packages**:
- `compress/` - File compression with real files
- `concurrent/` - Concurrent execution patterns  
- `config/` - Configuration loading with real files
- `db/` - Database connections and migrations
- `logging/` - Structured logging functionality
- `rest/` - HTTP client operations
- `server/` - HTTP server startup and health checks

### 3. All Tests (Unit + Integration)
**Purpose**: Run complete test suite with coverage
**Requirements**: Docker (testcontainers)
**Command**: `task test:all`

```bash
# Run all tests with coverage report
task test:all

# Opens output/coverage-all.html automatically
```

**Integration test packages**:
- `db/` - Database with testcontainer (PostgreSQL, MySQL, MSSQL)
- `temporal/` - Temporal with testcontainer
- `compress/`, `concurrent/`, `config/`, `logging/`, `rest/`, `server/`

## Additional Test Commands

### Quality Testing
```bash
task lint      # Run linter (golangci-lint)
task check     # Run golangci-lint and tests
```

### Development Tools
```bash
task tools     # Install development tools
task fmt       # Format code
task clean     # Clean build artifacts
```

### Coverage Reports
The test:all task generates detailed HTML reports:
```bash
task test:all
# Opens output/coverage-all.html automatically
```

## Service Management

### Temporal Services (Optional)
For manual testing, you can start Temporal services:

```bash
task temporal:start   # Start Temporal server (docker compose)
task temporal:stop    # Stop Temporal services
task temporal:logs    # View Temporal logs
task temporal:status  # Check Temporal status
```

**Note**: Integration tests use testcontainers and manage their own services automatically.

## Test Isolation Strategy

### Build Tags
- **No tag**: Unit tests only
- **`integration`**: Database integration tests  
- **`temporal`**: Temporal integration tests
- **`example`**: Example code (excluded from normal builds)

### Why Separate Tags?

1. **Fast Development Feedback**
   - `task test` runs quickly without external dependencies
   - Developers get immediate feedback on code changes

2. **Focused Integration Testing**
   - `task test:integration` tests with real dependencies using testcontainers
   - Automatic service management (no manual Docker setup)

3. **CI/CD Flexibility**
   - Different CI stages can run different test suites
   - Parallel execution of independent test categories
   - Testcontainers handle service lifecycle automatically

4. **Resource Optimization**
   - Unit tests run without Docker overhead
   - Integration tests use testcontainers for isolation
   - Services start only when needed and clean up automatically

## Environment Configuration

The test system uses build tags to control which tests run, not environment variables:

- **Unit tests**: No tag required - `go test ./...`
- **Database integration**: `-tags=integration` 
- **Temporal integration**: `-tags=temporal`

### Optional Environment Overrides
```bash
# Optional database connection overrides (if needed)
DB_HOST=localhost       # Database host override
DB_PORT=5439           # Database port override

# Optional Temporal server override (if needed)
TEMPORAL_ADDRESS=localhost:7233  # Temporal server address

# Optional debug logging
DEBUG=true             # Enable debug logging
```

## Example Workflows

### Development Workflow
```bash
# 1. Quick validation during development
task test

# 2. Test with real dependencies before commit
task test:integration

# 3. Full validation with coverage before release
task test:all
```

### CI/CD Pipeline
```bash
# Stage 1: Fast feedback
task test
task lint

# Stage 2: Full test suite with coverage
task test:all

# Stage 3: Code quality
task check
```

### Debugging Failed Tests

#### Integration Test Issues
```bash
# Check if Docker is running
docker info

# View testcontainer logs (during test)
go test -tags=integration -v ./db/...

# Clean up old containers
docker system prune
```

#### Temporal Test Issues
```bash
# Run with verbose output
go test -tags=integration -v ./temporal/...

# Check Docker resources
docker ps
docker system df

# Ensure sufficient Docker resources (memory, disk)
```

## Best Practices

### For Unit Tests
- Mock external dependencies
- Test edge cases and error conditions
- Keep tests fast and deterministic
- Use table-driven tests for multiple scenarios

### For Integration Tests  
- Use unique identifiers to avoid conflicts
- Clean up resources in test teardown
- Handle service startup delays gracefully  
- Test realistic scenarios with real data

### For Temporal Tests
- Use unique workflow and task queue names
- Test both success and failure scenarios
- Include compensation patterns where applicable
- Verify proper resource cleanup

## Performance Targets

- **Unit Tests**: < 5 seconds total
- **Integration Tests**: < 2 minutes (with testcontainer startup)
- **All Tests**: < 3 minutes (unit + integration + coverage)

## Troubleshooting

### Common Issues

1. **Docker Issues**
   - Ensure Docker is running: `docker info`
   - Clean up old containers: `docker system prune`
   - Check disk space: `docker system df`
   - Ensure sufficient memory (4GB+ recommended)

2. **Build Tag Confusion**
   - Unit tests: no tag needed - `task test`
   - Integration tests: `-tags=integration` - `task test:integration`
   - Examples: `-tags=example`

3. **Testcontainer Issues**
   - Testcontainers manages service lifecycle automatically
   - Containers are ephemeral and cleaned up after tests
   - If tests hang, check Docker resources
   - View testcontainer logs with `-v` flag

4. **Test Parallelization**
   - Integration tests may run slower due to container startup
   - Each test suite gets isolated containers
   - Parallel execution is safe (containers don't conflict)

This testing strategy ensures reliable, maintainable tests while providing flexibility for different development and deployment scenarios.