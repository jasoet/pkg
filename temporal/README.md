# Temporal Package Integration Tests

This directory contains comprehensive integration tests for the Temporal package. These tests require a running Temporal server and are designed to test the full functionality of the Temporal client, worker manager, and schedule manager.

## Prerequisites

- Docker and Docker Compose
- Go 1.23+
- Temporal server (will be started automatically via Docker Compose)

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

### Option 1: Using Mage (Recommended)

The project includes a Mage build system with predefined tasks:

```bash
# Start Temporal services and run integration tests
mage temporalTest

# Just start Temporal services (for manual testing)
mage temporal:up

# View Temporal service logs
mage temporal:logs

# Stop Temporal services
mage temporal:down

# Restart Temporal services
mage temporal:restart
```

### Option 2: Manual Docker Setup

1. **Start Temporal Server**:
   ```bash
   cd scripts/compose
   docker compose -f temporal-compose.yml up -d
   ```

2. **Wait for Services to Initialize**:
   ```bash
   # Wait about 30 seconds for Temporal to fully start
   sleep 30
   ```

3. **Run Integration Tests**:
   ```bash
   go test -tags=integration -timeout=10m ./temporal/...
   ```

4. **Clean Up**:
   ```bash
   docker compose -f temporal-compose.yml down -v
   ```

### Option 3: Using External Temporal Server

If you have Temporal running elsewhere:

1. **Update Configuration**:
   ```go
   config := &temporal.Config{
       HostPort:             "your-temporal-host:7233",
       Namespace:            "your-namespace",
       MetricsListenAddress: "0.0.0.0:9090",
   }
   ```

2. **Run Tests**:
   ```bash
   go test -tags=integration ./temporal/...
   ```

## Test Configuration

The integration tests use the following default configuration:

- **Temporal Server**: `localhost:7233`
- **Namespace**: `default`
- **Database**: PostgreSQL (started via Docker Compose)
- **UI**: Available at `http://localhost:8233`

### Docker Compose Services

The `temporal-compose.yml` includes:

- **PostgreSQL**: Database for Temporal (port 5434)
- **Temporal Server**: Core Temporal service (port 7233)
- **Temporal UI**: Web interface (port 8233)

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
# Check Temporal server status
docker compose -f scripts/compose/temporal-compose.yml logs temporal-server

# Check PostgreSQL status
docker compose -f scripts/compose/temporal-compose.yml logs postgresql-temporal

# List running containers
docker ps

# Check Temporal CLI connectivity
docker exec temporal-server tctl namespace describe default
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
3. **Include proper cleanup** in test teardown
4. **Add realistic error scenarios** where appropriate
5. **Document any new configuration requirements**

### Test Naming Convention

- Test functions: `TestFeatureName`
- Workflow IDs: `test-feature-timestamp`
- Task queues: `test-feature-queue`
- Schedule IDs: `test-feature-schedule-timestamp`

## Monitoring and Observability

### Temporal UI

Access the Temporal UI at `http://localhost:8233` to:
- View workflow executions
- Monitor worker activity
- Debug failed workflows
- Inspect workflow history

### Metrics

The tests expose Prometheus metrics on various ports:
- 9091-9102: Different test configurations
- Metrics include workflow counts, activity durations, worker status

### Logs

All components provide structured logging:
- Temporal server logs
- Worker manager logs  
- Individual workflow and activity logs
- Integration test logs

This comprehensive test suite ensures the Temporal package works correctly in real-world scenarios and provides confidence when making changes to the codebase.