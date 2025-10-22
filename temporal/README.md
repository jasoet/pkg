# Temporal Package

A comprehensive Go library for working with Temporal workflows, providing high-level abstractions for client management, worker orchestration, scheduling, and workflow monitoring.

## Features

### 🔧 Core Components

- **Client Management** (`client.go`) - Create and configure Temporal clients with metrics integration
- **Worker Management** (`worker.go`) - Manage workflow workers with lifecycle controls
- **Schedule Management** (`schedule.go`) - Create and manage workflow schedules (cron, intervals)
- **Workflow Management** (`workflow.go`) - **NEW!** Query, monitor, and control workflow executions

### 📊 Workflow Query & Monitoring

The WorkflowManager provides powerful capabilities for monitoring and managing workflows:

- **Query Operations**: List, search, and filter workflows by status, type, or custom criteria
- **Workflow Details**: Get detailed execution information, history, and results
- **Lifecycle Control**: Cancel, terminate, signal, and query running workflows
- **Dashboard Support**: Aggregated statistics and real-time monitoring
- **Search Capabilities**: Find workflows by ID prefix, type, or advanced queries

### 🎯 Use Cases

- Build custom workflow dashboards
- Monitor production workflow health
- Implement workflow automation and orchestration
- Create admin tools for workflow management
- Integrate workflow data with external systems

## Quick Start

### Installing

```bash
go get github.com/jasoet/pkg/v2/temporal
```

### Basic Usage

#### 1. Create a Temporal Client

```go
package main

import (
    "github.com/jasoet/pkg/v2/temporal"
)

func main() {
    config := &temporal.Config{
        HostPort:  "localhost:7233",
        Namespace: "default",
        MetricsListenAddress: "0.0.0.0:9090",
    }

    client, err := temporal.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()
}
```

#### 2. Manage Workers

```go
// Create worker manager
wm, err := temporal.NewWorkerManager(config)
if err != nil {
    panic(err)
}
defer wm.Close()

// Register a worker
worker := wm.Register("my-task-queue", worker.Options{})
worker.RegisterWorkflow(MyWorkflow)
worker.RegisterActivity(MyActivity)

// Start all workers
err = wm.StartAll(ctx)
```

#### 3. Query and Monitor Workflows

```go
// Create workflow manager
wfm, err := temporal.NewWorkflowManager(config)
if err != nil {
    panic(err)
}
defer wfm.Close()

// Get dashboard statistics
stats, err := wfm.GetDashboardStats(ctx)
fmt.Printf("Running: %d, Completed: %d, Failed: %d\n",
    stats.TotalRunning, stats.TotalCompleted, stats.TotalFailed)

// List running workflows
workflows, err := wfm.ListRunningWorkflows(ctx, 100)
for _, wf := range workflows {
    fmt.Printf("Workflow: %s (%s)\n", wf.WorkflowID, wf.WorkflowType)
}

// Search by workflow type
orderWorkflows, err := wfm.SearchWorkflowsByType(ctx, "OrderProcessingWorkflow", 50)

// Get specific workflow details
details, err := wfm.DescribeWorkflow(ctx, "order-123", "")
fmt.Printf("Status: %s, Duration: %v\n", details.Status, details.ExecutionTime)

// Cancel a workflow
err = wfm.CancelWorkflow(ctx, "problematic-workflow-id", "")
```

#### 4. Schedule Workflows

```go
// Create schedule manager
sm := temporal.NewScheduleManager(config)
defer sm.Close()

// Schedule a workflow to run every hour
handle, err := sm.CreateWorkflowSchedule(ctx, "hourly-report", temporal.WorkflowScheduleOptions{
    WorkflowID: "report-workflow",
    Workflow:   ReportWorkflow,
    TaskQueue:  "reports",
    Interval:   time.Hour,
    Args:       []any{"daily-report"},
})
```

## Examples

Check out the [examples](../examples/temporal/) directory for complete, runnable examples:

- **[Dashboard Example](../examples/temporal/dashboard/)** - HTTP dashboard for monitoring workflows
- **[Basic Worker](../examples/temporal/worker/)** - Setting up workers
- **[Workflow Examples](../examples/temporal/workflows/)** - Sample workflow implementations
- **[Scheduler Example](../examples/temporal/scheduler/)** - Scheduling workflows

### Running the Dashboard Example

```bash
cd examples/temporal/dashboard
go run main.go

# Or with custom configuration
TEMPORAL_HOST=temporal.example.com:7233 \
TEMPORAL_NAMESPACE=production \
go run main.go
```

Then open http://localhost:8080 in your browser.

## API Reference

### WorkflowManager Methods

#### Query Operations
- `ListWorkflows(ctx, pageSize, query)` - List workflows with optional filtering
- `ListRunningWorkflows(ctx, pageSize)` - Get all running workflows
- `ListCompletedWorkflows(ctx, pageSize)` - Get completed workflows
- `ListFailedWorkflows(ctx, pageSize)` - Get failed workflows
- `DescribeWorkflow(ctx, workflowID, runID)` - Get detailed workflow information
- `GetWorkflowStatus(ctx, workflowID, runID)` - Get current workflow status
- `GetWorkflowHistory(ctx, workflowID, runID)` - Get workflow event history

#### Search Operations
- `SearchWorkflowsByType(ctx, workflowType, pageSize)` - Find workflows by type
- `SearchWorkflowsByID(ctx, idPrefix, pageSize)` - Find workflows by ID prefix
- `CountWorkflows(ctx, query)` - Count workflows matching a query

#### Lifecycle Operations
- `CancelWorkflow(ctx, workflowID, runID)` - Cancel a running workflow
- `TerminateWorkflow(ctx, workflowID, runID, reason)` - Terminate a workflow
- `SignalWorkflow(ctx, workflowID, runID, signalName, data)` - Send signal to workflow
- `QueryWorkflow(ctx, workflowID, runID, queryType, args)` - Query workflow state

#### Dashboard Operations
- `GetDashboardStats(ctx)` - Get aggregated workflow statistics
- `GetRecentWorkflows(ctx, limit)` - Get most recent workflows
- `GetWorkflowResult(ctx, workflowID, runID, valuePtr)` - Get workflow result

## Testing

This package includes comprehensive integration tests using testcontainers to automatically manage Temporal server instances.

### Testcontainer Package

The `temporal/testcontainer` package provides reusable utilities for running Temporal server in Docker containers for integration testing. This package can be used in your own projects for testing Temporal workflows.

#### Installing the Testcontainer Package

```bash
go get github.com/jasoet/pkg/v2/temporal/testcontainer
```

#### Quick Start with Testcontainer

**Simple Setup (Recommended):**

```go
import (
    "context"
    "testing"
    "github.com/jasoet/pkg/v2/temporal/testcontainer"
)

func TestMyWorkflow(t *testing.T) {
    ctx := context.Background()

    // Setup container and client with cleanup
    _, client, cleanup, err := testcontainer.Setup(
        ctx,
        testcontainer.ClientConfig{
            Namespace: "default",
        },
        testcontainer.Options{Logger: t},
    )
    if err != nil {
        t.Fatalf("Setup failed: %v", err)
    }
    defer cleanup()

    // Use client for your tests...
}
```

**Advanced Setup:**

```go
import (
    "go.temporal.io/sdk/client"
)

func TestAdvanced(t *testing.T) {
    ctx := context.Background()

    // Start container with custom options
    container, err := testcontainer.Start(ctx, testcontainer.Options{
        Image:          "temporalio/temporal:1.22.0",
        StartupTimeout: 120 * time.Second,
        Logger:         t,
    })
    if err != nil {
        t.Fatalf("Failed to start: %v", err)
    }
    defer container.Terminate(ctx)

    // Create client using Temporal SDK directly
    temporalClient, err := client.Dial(client.Options{
        HostPort:  container.HostPort(),
        Namespace: "default",
    })
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer temporalClient.Close()

    // Run tests...
}
```

#### Configuration Options

```go
testcontainer.Options{
    Image:           "temporalio/temporal:latest", // Docker image
    StartupTimeout:  60 * time.Second,            // Startup timeout
    Logger:          t,                            // *testing.T or custom logger
    ExtraPorts:      []string{"8080/tcp"},        // Additional ports
    InitialWaitTime: 3 * time.Second,             // Wait after startup
}
```

See the [testcontainer package documentation](./testcontainer/doc.go) and [examples](./testcontainer/example_test.go) for more details.

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

### 4. Workflow Integration Tests (`workflow_integration_test.go`)

Tests the WorkflowManager query and monitoring functionality:
- **WorkflowManager Creation**: Tests manager initialization with client and config
- **List Operations**: Tests listing workflows by status (running, completed, failed)
- **Describe Operations**: Tests getting workflow details, status, and history
- **Search Operations**: Tests searching workflows by type, ID prefix, and counting
- **Lifecycle Operations**: Tests canceling, terminating, and signaling workflows
- **Dashboard Operations**: Tests statistics aggregation and recent workflow retrieval

### 5. End-to-End Integration Tests (`e2e_integration_test.go`)

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