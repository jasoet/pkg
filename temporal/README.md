# Temporal Package Integration Tests

This directory contains comprehensive integration tests for the Temporal package. These tests use testcontainers to automatically manage Temporal server instances and test the full functionality of the Temporal client, worker manager, and schedule manager.

## Prerequisites

- Docker (for testcontainers)
- Go 1.23+
- No manual Temporal server setup required

## Test Categories

### 1. Client Integration Tests (`client_integration_test.go`)

Tests the Temporal client functionality:
- **NewClient**: Tests client creation with default configuration
- **NewClientWithMetrics**: Tests client creation with metrics enabled/disabled
- **DescribeNamespace**: Tests basic server connectivity
- **WorkflowService**: Tests access to workflow service APIs
- **Configuration Validation**: Tests various client configurations

### 2. Worker Integration Tests (`worker_integration_test.go`)

Tests the WorkerManager and workflow execution:
- **WorkerManager Creation**: Tests worker manager lifecycle
- **Worker Registration**: Tests registering workers with different task queues
- **Workflow Execution**: Tests end-to-end workflow execution with activities
- **Error Handling**: Tests workflow failure scenarios
- **Multiple Workers**: Tests managing multiple workers simultaneously

### 3. Schedule Integration Tests (`schedule_integration_test.go`)

Tests the ScheduleManager functionality:
- **Schedule Creation**: Tests creating cron and interval schedules
- **Schedule Management**: Tests listing, getting, updating, and deleting schedules
- **Error Handling**: Tests various failure scenarios
- **Schedule Types**: Tests different schedule configurations

### 4. End-to-End Integration Tests (`e2e_integration_test.go`)

Tests complex, real-world scenarios:
- **Order Processing Workflow**: Complete e-commerce order processing with compensation patterns
- **Multi-step Workflows**: Tests workflows with multiple activities and error handling
- **Parallel Execution**: Tests processing multiple workflows simultaneously
- **Full Stack Integration**: Tests all components working together

## Running the Tests

### Using Task (Recommended)

The project uses Taskfile for running tests:

```bash
# Run all integration tests (includes temporal + db tests)
task test:integration

# Run all tests with combined coverage
task test:all
```

### Direct Go Test Command

```bash
# Run temporal integration tests only
go test -tags=integration -timeout=10m ./temporal/...

# Run with verbose output
go test -tags=integration -v ./temporal/...

# Run specific test
go test -tags=integration -run TestClientIntegration ./temporal/...
```

### How It Works

The tests use **testcontainers** to automatically:
1. Pull the `temporalio/temporal:latest` Docker image
2. Start a Temporal server container for each test suite
3. Wait for the server to be ready
4. Run the tests against the containerized server
5. Automatically clean up containers when tests complete

No manual server management required!

## Test Configuration

The integration tests use testcontainers with automatic configuration:

- **Temporal Server**: Dynamically assigned port (managed by testcontainers)
- **Namespace**: `default`
- **Database**: Built-in (managed by Temporal container)
- **Container Image**: `temporalio/temporal:latest`

Each test suite gets its own isolated Temporal container instance.

## Test Features

### Realistic Workflows

The e2e tests include a complete order processing workflow that demonstrates:

- **Multi-step Processing**: Validation → Payment → Inventory → Shipping → Confirmation
- **Compensation Patterns**: Automatic rollback on failures (Saga pattern)
- **Error Handling**: Retry policies and graceful degradation
- **Activity Timeouts**: Proper timeout and heartbeat handling

### Test Data

Tests use realistic data patterns:
- Order IDs with timestamps
- Customer information
- Payment amounts and transaction IDs
- Inventory reservations
- Shipping tracking numbers

### Error Simulation

Tests include controlled failure scenarios:
- Random payment failures (5% chance)
- Inventory shortages (3% chance)
- Shipping unavailability (2% chance)
- Network timeouts and connectivity issues

## Debugging Integration Tests

### Common Issues

1. **Connection Refused**:
   - Ensure Temporal server is running: `docker ps`
   - Check if ports are available: `lsof -i :7233`
   - Wait longer for services to start (up to 60 seconds)

2. **Namespace Not Found**:
   - Verify the `default` namespace exists
   - Check Temporal UI at `http://localhost:8233`

3. **Worker Registration Failures**:
   - Ensure task queue names are unique across tests
   - Check for port conflicts on metrics endpoints

### Debugging Commands

```bash
# List running testcontainer instances
docker ps | grep temporalio/temporal

# View logs from a specific container
docker logs <container-id>

# Check Docker status
docker info
```

### Test Logging

The integration tests use structured logging with different levels:

```bash
# Run with verbose output
go test -tags=integration -v ./temporal/...

# Run with debug logging
DEBUG=true go test -tags=integration ./temporal/...
```

## Performance Considerations

### Test Timeouts

- Individual tests: 30-60 seconds
- Full test suite: Up to 10 minutes
- Workflow executions: Usually complete in 2-5 seconds

### Resource Usage

- **Memory**: ~500MB for Temporal server + PostgreSQL
- **CPU**: Moderate during test execution
- **Disk**: ~100MB for Docker volumes
- **Network**: Local Docker networking only

### Parallel Execution

The tests are designed to run safely in parallel:
- Unique workflow IDs with timestamps
- Separate task queues for different test scenarios
- Independent metrics endpoints
- Isolated schedule names

## Contributing

When adding new integration tests:

1. **Use the `//go:build integration` tag**
2. **Create unique identifiers** (workflow IDs, task queues, etc.)
3. **Use the testcontainer helper** (`setupTemporalContainerForTest`)
4. **Add realistic error scenarios** where appropriate
5. **Document any new configuration requirements**

### Test Naming Convention

- Test functions: `TestFeatureName`
- Workflow IDs: `test-feature-timestamp`
- Task queues: `test-feature-queue`
- Schedule IDs: `test-feature-schedule-timestamp`

## Monitoring and Observability

### Testcontainer Logs

View container logs during test execution:
```bash
# Watch test output for container status
go test -tags=integration -v ./temporal/...
```

### Metrics

The tests use dynamic port allocation for metrics:
- Port 0 (random available port) for each test instance
- Metrics include workflow counts, activity durations, worker status

### Logs

All components provide structured logging:
- Temporal container logs (viewable via docker logs)
- Worker manager logs
- Individual workflow and activity logs
- Integration test logs

This comprehensive test suite uses testcontainers to ensure the Temporal package works correctly in isolated, reproducible environments and provides confidence when making changes to the codebase.