# Temporal Workflow Examples

This directory contains examples demonstrating how to use Temporal workflows in Go with `github.com/jasoet/pkg/v3/temporal`. These examples show practical usage patterns and best practices for implementing Temporal workflows, activities, workers, and schedulers.

## 📍 Example Code Locations

**Workflow examples:**
- [Simple workflow](./workflows/simple_workflow.go)
- [Activity workflow](./workflows/activity_workflow.go)
- [Error handling workflow](./workflows/error_handling_workflow.go)
- [Timer workflow](./workflows/timer_workflow.go)

**Other examples:**
- [Basic activities](./activities/basic_activities.go)
- [Basic worker](./worker/basic_worker.go)
- [Basic scheduler](./scheduler/basic_scheduler.go)
- [Dashboard](./dashboard/main.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import (
    "github.com/jasoet/pkg/v3/temporal"
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"
)

// 1. Define workflow
func MyWorkflow(ctx workflow.Context, input string) (string, error) {
    // Workflow logic here
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Second,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var result string
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &result)
    return result, err
}

// 2. Define activities
func MyActivity(ctx context.Context, input string) (string, error) {
    // Activity logic here
    return "processed: " + input, nil
}

// 3. Create client (caller owns it; defaults are localhost:7233 / "default")
ctx := context.Background()
c, err := temporal.NewClient(
    temporal.WithHostPort("localhost:7233"),
    temporal.WithNamespace("default"),
)
if err != nil {
    log.Fatal().Err(err).Msg("Failed to create Temporal client")
}
defer c.Close()

// 4. Create a worker manager and register a worker
wm, err := temporal.NewWorkerManager(c)
if err != nil {
    log.Fatal().Err(err).Msg("Failed to create worker manager")
}
defer wm.Close(ctx) // stops workers; does NOT close c

w := wm.Register("my-task-queue", worker.Options{})
w.RegisterWorkflow(MyWorkflow)
w.RegisterActivity(MyActivity)

// 5. Start all workers (blocks the process via signal handling in real apps)
if err := wm.StartAll(ctx); err != nil {
    log.Fatal().Err(err).Msg("Worker failed")
}

// 6. Trigger workflow manually (from another process, using the SDK client)
workflowOptions := client.StartWorkflowOptions{
    ID:        "my-workflow-id",
    TaskQueue: "my-task-queue",
}
we, err := c.ExecuteWorkflow(ctx, workflowOptions, MyWorkflow, "input-data")
if err != nil {
    log.Error().Err(err).Msg("Failed to start workflow")
}
log.Info().Str("workflow_id", we.GetID()).Str("run_id", we.GetRunID()).Msg("Workflow started")

// Get workflow result
var result string
err = we.Get(ctx, &result)

// 7. Or create a schedule for periodic execution
scheduleManager, err := temporal.NewScheduleManager(c)
if err != nil {
    log.Fatal().Err(err).Msg("failed to create schedule manager")
}
defer scheduleManager.Close(ctx) // does NOT close c

scheduleID := "my-schedule-id"
scheduleOptions := temporal.WorkflowScheduleOptions{
    WorkflowID: "scheduled-workflow-id",
    Workflow:   MyWorkflow,
    TaskQueue:  "my-task-queue",
    Interval:   1 * time.Hour, // Run every hour
    Args:       []any{"scheduled-input"},
}

scheduleHandle, err := scheduleManager.CreateWorkflowSchedule(ctx, scheduleID, scheduleOptions)
if err != nil {
    log.Error().Err(err).Msg("Failed to create schedule")
}
log.Info().Str("schedule_id", scheduleID).Msg("Schedule created")

// Delete schedule when done
err = scheduleHandle.Delete(ctx)
```

**Key features:**
- Typed constructors: `temporal.NewClient(opts ...Option)`, `temporal.NewWorkerManager(c)`, `temporal.NewScheduleManager(c)`, `temporal.NewWorkflowManager(c)` / `temporal.NewWorkflowManagerWithNamespace(c, ns)`
- Caller-owned client: managers never close the client; you do
- Durable execution with automatic retries
- Built-in error handling and compensation
- Timer and scheduling support
- Direct access to the `go.temporal.io/sdk` client for anything the managers don't cover
- Scheduled workflows with cron-like intervals
- Async execution with result retrieval

## Directory Structure

```
examples/temporal/
├── workflows/     # Example workflow implementations
├── activities/    # Example activity implementations
├── worker/        # Example worker setup
├── scheduler/     # Example scheduler setup
├── dashboard/     # HTTP dashboard for monitoring workflows
└── README.md      # This file
```

## Prerequisites

- Go 1.23 or later
- Temporal server running (default: localhost:7233)
- Understanding of basic Temporal concepts

## Examples Overview

### Workflows

The `workflows/` directory contains examples of different workflow patterns:

1. **Simple Sequential Workflow**: A basic workflow that executes steps in sequence
2. **Workflow with Activities**: A workflow that calls activities
3. **Workflow with Error Handling**: A workflow demonstrating error handling and retry patterns
4. **Workflow with Timers**: A workflow using timers and delays

### Activities

The `activities/` directory contains examples of different activity patterns:

1. **Basic Activities**: Simple activity implementations
2. **Activities with Error Handling**: Activities demonstrating error handling and retries
3. **External Service Activities**: Activities that interact with external services

### Worker

The `worker/` directory contains examples of worker setup:

1. **Basic Worker**: A complete worker setup example
2. **Worker with Multiple Task Queues**: A worker handling multiple task queues
3. **Worker with Graceful Shutdown**: A worker with proper shutdown handling

### Scheduler

The `scheduler/` directory contains examples of scheduler setup:

1. **Interval Scheduler**: A scheduler using interval-based scheduling
2. **Cron Scheduler**: A scheduler using cron-based scheduling
3. **One-time Scheduler**: A scheduler for one-time execution

### Dashboard

The `dashboard/` directory contains a runnable HTTP dashboard (`main.go`) built on `temporal.WorkflowManager`.

## How to Run the Examples

The `workflows/`, `activities/`, `worker/`, and `scheduler/` packages are guarded by the `example` build tag and expose `Run*` functions rather than `main` programs:

```bash
# Compile-check all examples
go build -tags=example ./examples/temporal/...

# Run the dashboard (a real main package, no build tag)
cd examples/temporal/dashboard
go run main.go

# Or with custom configuration
TEMPORAL_HOST=temporal.example.com:7233 \
TEMPORAL_NAMESPACE=production \
go run main.go
```

Then open http://localhost:8080 in your browser.

To run a worker or scheduler example, call its `Run*` function (e.g. `worker.RunBasicWorker()`, `scheduler.RunIntervalScheduler()`) from your own `main`, with a Temporal server listening on `localhost:7233`.

## Best Practices

- Create the client with `temporal.NewClient(...)` and close it yourself — managers never close the client
- Always handle errors properly in workflows and activities
- Use proper logging with zerolog (the SDK logger can be bridged via `temporal.NewZerologAdapter`)
- Implement proper shutdown handling for workers (`WorkerManager.Close(ctx)` stops all registered workers)
- Use meaningful task queue names
- Structure your code to separate workflows, activities, and worker setup

## Integration with Main Package

These examples use the main temporal package from this project. Key integration points:

- Client creation using `temporal.NewClient()` with functional options (`WithHostPort`, `WithNamespace`, `WithOTelConfig`, `WithConfig`)
- Worker management using `temporal.NewWorkerManager(c)`
- Schedule management using `temporal.NewScheduleManager(c)`
- Workflow queries using `temporal.NewWorkflowManager(c)` / `temporal.NewWorkflowManagerWithNamespace(c, ns)`
- Logging using `temporal.ZerologAdapter`

For more details on the main temporal package, see the [temporal package README](../../temporal/README.md).
