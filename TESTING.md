# Testing Guide

This document explains the testing strategy and commands for this Go utility library.

## Test Categories

### 1. Unit Tests
**Purpose**: Test individual functions and components in isolation  
**Requirements**: No external dependencies  
**Command**: `mage test`  
**Coverage**: All packages have unit tests

```bash
# Run all unit tests
mage test

# Run unit tests for specific package
go test ./config/...
go test ./logging/...
```

### 2. Database Integration Tests  
**Purpose**: Test database connectivity and operations  
**Requirements**: PostgreSQL, MySQL, MSSQL (via Docker)  
**Command**: `mage integrationTest`  
**Build Tag**: `integration`

```bash
# Run database integration tests (starts Docker services automatically)
mage integrationTest

# Manual execution
go test -tags=integration ./db/...
```

**Covered packages**:
- `compress/` - File compression with real files
- `concurrent/` - Concurrent execution patterns  
- `config/` - Configuration loading with real files
- `db/` - Database connections and migrations
- `logging/` - Structured logging functionality
- `rest/` - HTTP client operations
- `server/` - HTTP server startup and health checks

### 3. Temporal Integration Tests
**Purpose**: Test Temporal workflow engine integration  
**Requirements**: Temporal server + PostgreSQL (via Docker)  
**Command**: `mage temporalTest`  
**Build Tag**: `temporal`

```bash
# Run Temporal integration tests (starts Temporal server automatically)
mage temporalTest

# Manual execution  
go test -tags=temporal ./temporal/...
```

**Test scenarios**:
- Client connectivity and configuration
- Worker registration and lifecycle management
- Workflow execution with activities
- Schedule creation and management
- End-to-end order processing workflows
- Error handling and compensation patterns

### 4. All Integration Tests
**Purpose**: Run both database and temporal integration tests  
**Requirements**: All Docker services  
**Command**: `mage allIntegrationTests`

```bash
# Run all integration tests sequentially
mage allIntegrationTests
```

## Service Management

### Database Services
```bash
mage docker:up        # Start PostgreSQL, MySQL, MSSQL
mage docker:down      # Stop database services  
mage docker:logs      # View database logs
mage docker:restart   # Restart database services
```

### Temporal Services  
```bash
mage temporal:up      # Start Temporal server + PostgreSQL
mage temporal:down    # Stop Temporal services
mage temporal:logs    # View Temporal logs
mage temporal:restart # Restart Temporal services
```

## Test Isolation Strategy

### Build Tags
- **No tag**: Unit tests only
- **`integration`**: Database integration tests  
- **`temporal`**: Temporal integration tests
- **`example`**: Example code (excluded from normal builds)
- **`template`**: Template projects (excluded from normal builds)

### Why Separate Tags?

1. **Fast Development Feedback**  
   - `mage test` runs quickly without external dependencies
   - Developers get immediate feedback on code changes

2. **Focused Integration Testing**
   - `mage integrationTest` tests database functionality without requiring Temporal
   - `mage temporalTest` tests workflow functionality with proper Temporal setup

3. **CI/CD Flexibility**
   - Different CI stages can run different test suites
   - Parallel execution of independent test categories
   - Graceful handling of missing services

4. **Resource Optimization**  
   - Only start required services for specific test categories
   - Avoid unnecessary service startup time and resource usage

## Environment Configuration

### Database Integration Tests
```bash
# Environment variables
AUTOMATION=true          # Enable integration test mode
DB_HOST=localhost       # Database host override
DB_PORT=5439           # Database port override
```

### Temporal Integration Tests  
```bash
# Environment variables
TEMPORAL_INTEGRATION=true    # Enable temporal test mode
TEMPORAL_ADDRESS=localhost:7233  # Temporal server address
DEBUG=true                   # Enable debug logging
```

## Example Workflows

### Development Workflow
```bash
# 1. Quick validation during development
mage test

# 2. Test database integration before commit
mage integrationTest

# 3. Full validation before release
mage allIntegrationTests
```

### CI/CD Pipeline
```bash
# Stage 1: Fast feedback
mage test
mage lint

# Stage 2: Database integration (parallel)
mage integrationTest

# Stage 3: Temporal integration (parallel) 
mage temporalTest

# Stage 4: Quality checks
mage security
mage dependencies
mage coverage
```

### Debugging Failed Tests

#### Database Connection Issues
```bash
# Check if services are running
docker ps

# View database logs
mage docker:logs

# Test direct connection
psql -h localhost -p 5439 -U jasoet -d pkg_db
```

#### Temporal Connection Issues  
```bash
# Check Temporal services
docker ps | grep temporal

# View Temporal logs
mage temporal:logs

# Check Temporal UI
open http://localhost:8233

# Test connectivity
go test -tags=temporal -v ./temporal/client_integration_test.go
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
- **Database Integration**: < 30 seconds  
- **Temporal Integration**: < 2 minutes
- **All Integration Tests**: < 3 minutes

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   - Check for services using ports 5439, 7233, 8233
   - Use `lsof -i :PORT` to identify conflicts

2. **Docker Issues**  
   - Ensure Docker is running: `docker info`
   - Clean up containers: `docker system prune`
   - Check disk space: `docker system df`

3. **Build Tag Confusion**
   - Unit tests: no tag needed
   - Database: `-tags=integration`
   - Temporal: `-tags=temporal`  
   - Examples: `-tags=example`

4. **Service Startup Timing**
   - Database services: Wait 2-5 seconds
   - Temporal services: Wait 10-30 seconds
   - Check health with `docker compose logs`

This testing strategy ensures reliable, maintainable tests while providing flexibility for different development and deployment scenarios.