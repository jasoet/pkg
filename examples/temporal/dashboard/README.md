# Temporal Workflow Dashboard Example

A simple HTTP dashboard and CLI tool for monitoring and managing Temporal workflows using the WorkflowManager.

## Features

- 📊 **Dashboard Statistics** - View workflow counts by status and average execution time
- 📋 **Workflow Listing** - Browse all workflows with filtering by status
- 🔍 **Workflow Details** - View detailed information about specific workflows
- 🏃 **Real-time Monitoring** - Track running workflows with auto-refresh
- ⚡ **Workflow Control** - Cancel workflows directly from the dashboard
- 🌐 **REST API** - Full API for integration with other tools
- 💻 **CLI Mode** - Command-line interface for quick insights

## Prerequisites

- Go 1.23+
- Running Temporal server (local or remote)
- Active workflows (for meaningful dashboard data)

## Quick Start

### 1. Using Default Configuration (localhost)

```bash
cd temporal/examples/dashboard
go run main.go
```

Then open your browser to [http://localhost:8080](http://localhost:8080)

### 2. Using Environment Variables

```bash
# Configure Temporal connection
export TEMPORAL_HOST="temporal.example.com:7233"
export TEMPORAL_NAMESPACE="production"
export PORT="3000"

go run main.go
```

### 3. CLI Demo Mode

Run a one-time CLI report without starting the server:

```bash
go run main.go --cli-demo
```

Output example:
```
=== CLI Demo Mode ===

📊 Dashboard Statistics:
  Running:    3
  Completed:  127
  Failed:     2
  Canceled:   1
  Terminated: 0
  Avg Duration: 2.5s

🏃 Running Workflows:
  • order-123 (OrderProcessingWorkflow) - Started: 2025-10-10T10:15:30Z
  • payment-456 (PaymentWorkflow) - Started: 2025-10-10T10:16:00Z

📋 Recent Workflows (last 10):
  ✅ order-120 (OrderProcessingWorkflow)
     Status: COMPLETED | Duration: 3.2s
  ❌ payment-error-1 (PaymentWorkflow)
     Status: FAILED | Duration: 1.5s
```

## API Endpoints

### Statistics

**GET /api/stats**

Returns aggregated workflow statistics.

```bash
curl http://localhost:8080/api/stats
```

Response:
```json
{
  "TotalRunning": 3,
  "TotalCompleted": 127,
  "TotalFailed": 2,
  "TotalCanceled": 1,
  "TotalTerminated": 0,
  "AverageDuration": 2500000000
}
```

### Workflow Listing

**GET /api/workflows**

List all workflows (paginated, max 100).

```bash
curl http://localhost:8080/api/workflows
```

**GET /api/workflows/running**

List only running workflows.

```bash
curl http://localhost:8080/api/workflows/running
```

**GET /api/workflows/failed**

List only failed workflows.

```bash
curl http://localhost:8080/api/workflows/failed
```

**GET /api/workflows/recent**

Get the 50 most recent workflows.

```bash
curl http://localhost:8080/api/workflows/recent
```

### Workflow Details

**GET /api/workflows/{workflowID}**

Get detailed information about a specific workflow.

```bash
curl http://localhost:8080/api/workflows/order-processing-123
```

Response:
```json
{
  "WorkflowID": "order-processing-123",
  "RunID": "abc123...",
  "WorkflowType": "OrderProcessingWorkflow",
  "Status": "COMPLETED",
  "StartTime": "2025-10-10T10:15:30Z",
  "CloseTime": "2025-10-10T10:15:33Z",
  "ExecutionTime": 3000000000,
  "HistoryLength": 15
}
```

### Workflow Control

**POST /api/workflows/cancel/{workflowID}**

Cancel a running workflow.

```bash
curl -X POST http://localhost:8080/api/workflows/cancel/order-processing-123
```

Response:
```json
{
  "status": "canceled",
  "workflowID": "order-processing-123"
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPORAL_HOST` | `localhost:7233` | Temporal server address |
| `TEMPORAL_NAMESPACE` | `default` | Temporal namespace |
| `PORT` | `8080` | HTTP server port |

### Example .env File

```bash
TEMPORAL_HOST=temporal.mycompany.com:7233
TEMPORAL_NAMESPACE=production
PORT=3000
```

## Using as a Library

You can also use the WorkflowManager in your own applications:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jasoet/pkg/v2/temporal"
)

func main() {
    // Create WorkflowManager
    config := &temporal.Config{
        HostPort:  "localhost:7233",
        Namespace: "default",
        MetricsListenAddress: "0.0.0.0:0",
    }

    wm, err := temporal.NewWorkflowManager(config)
    if err != nil {
        log.Fatalf("Failed to create WorkflowManager: %v", err)
    }
    defer wm.Close()

    ctx := context.Background()

    // Get dashboard statistics
    stats, err := wm.GetDashboardStats(ctx)
    if err != nil {
        log.Fatalf("Failed to get stats: %v", err)
    }
    fmt.Printf("Running workflows: %d\n", stats.TotalRunning)

    // List running workflows
    workflows, err := wm.ListRunningWorkflows(ctx, 10)
    if err != nil {
        log.Fatalf("Failed to list workflows: %v", err)
    }

    for _, wf := range workflows {
        fmt.Printf("- %s (%s)\n", wf.WorkflowID, wf.WorkflowType)

        // Get detailed information
        details, err := wm.DescribeWorkflow(ctx, wf.WorkflowID, wf.RunID)
        if err != nil {
            log.Printf("Failed to describe workflow: %v", err)
            continue
        }
        fmt.Printf("  Status: %s\n", details.Status)
    }

    // Search workflows by type
    orderWorkflows, err := wm.SearchWorkflowsByType(ctx, "OrderProcessingWorkflow", 50)
    if err != nil {
        log.Fatalf("Failed to search workflows: %v", err)
    }
    fmt.Printf("Found %d order processing workflows\n", len(orderWorkflows))

    // Cancel a specific workflow
    err = wm.CancelWorkflow(ctx, "problematic-workflow-id", "")
    if err != nil {
        log.Printf("Failed to cancel workflow: %v", err)
    }
}
```

## Advanced Features

### Filtering Workflows

Use Temporal's query syntax to filter workflows:

```go
// Find workflows by status
workflows, err := wm.ListWorkflows(ctx, 100, "ExecutionStatus='Failed'")

// Find workflows by type
workflows, err := wm.ListWorkflows(ctx, 100, "WorkflowType='OrderWorkflow'")

// Find workflows by ID prefix
workflows, err := wm.ListWorkflows(ctx, 100, "WorkflowId STARTS_WITH 'order-'")

// Complex queries
workflows, err := wm.ListWorkflows(ctx, 100,
    "ExecutionStatus='Running' AND WorkflowType='PaymentWorkflow'")
```

### Workflow Lifecycle Management

```go
ctx := context.Background()

// Cancel a workflow (graceful shutdown)
err := wm.CancelWorkflow(ctx, workflowID, runID)

// Terminate a workflow (force stop)
err := wm.TerminateWorkflow(ctx, workflowID, runID, "Emergency termination")

// Signal a workflow
err := wm.SignalWorkflow(ctx, workflowID, runID, "approval", map[string]interface{}{
    "approved": true,
    "approver": "admin",
})

// Query a workflow for custom data
result, err := wm.QueryWorkflow(ctx, workflowID, runID, "getProgress")
```

### Getting Workflow Results

```go
// Get the result of a completed workflow
var orderResult OrderResult
err := wm.GetWorkflowResult(ctx, "order-123", "", &orderResult)
if err != nil {
    log.Fatalf("Failed to get workflow result: %v", err)
}
fmt.Printf("Order status: %s\n", orderResult.Status)
```

## Troubleshooting

### Connection Issues

```bash
# Test Temporal connection
curl http://localhost:7233/api/v1/namespaces

# Check if Temporal server is running
docker ps | grep temporal
```

### No Workflows Showing

Make sure you have active workflows running:

```bash
# Using tctl (Temporal CLI)
tctl workflow list

# Check namespace
tctl --namespace default workflow list
```

### CORS Issues

If integrating with a frontend app, consider adding CORS middleware:

```go
mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    // ... rest of handler
})
```

## Next Steps

- **Customize the UI**: Modify the HTML template to match your branding
- **Add Authentication**: Implement auth middleware for production use
- **Enhance Filtering**: Add more sophisticated query builders
- **Real-time Updates**: Use WebSockets for live workflow updates
- **Export Data**: Add CSV/JSON export functionality
- **Alerting**: Integrate with monitoring systems for workflow failures

## Related Examples

- [Basic Worker](../worker/basic_worker.go) - Learn how to run workers
- [Workflow Examples](../workflows/) - Sample workflow implementations
- [Scheduler Example](../scheduler/basic_scheduler.go) - Schedule recurring workflows

## License

This example is part of the jasoet/pkg temporal package.
