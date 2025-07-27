# Temporal Workflow Examples

This directory contains examples demonstrating how to use Temporal workflows in Go. These examples show practical usage patterns and best practices for implementing Temporal workflows, activities, workers, and schedulers.

## 📍 Example Code Locations

**Workflow examples:**
- [Simple workflow](https://github.com/jasoet/pkg/blob/main/temporal/examples/workflows/simple_workflow.go)
- [Activity workflow](https://github.com/jasoet/pkg/blob/main/temporal/examples/workflows/activity_workflow.go)
- [Error handling workflow](https://github.com/jasoet/pkg/blob/main/temporal/examples/workflows/error_handling_workflow.go)
- [Timer workflow](https://github.com/jasoet/pkg/blob/main/temporal/examples/workflows/timer_workflow.go)

**Other examples:**
- [Basic activities](https://github.com/jasoet/pkg/blob/main/temporal/examples/activities/basic_activities.go)
- [Basic worker](https://github.com/jasoet/pkg/blob/main/temporal/examples/worker/basic_worker.go)
- [Basic scheduler](https://github.com/jasoet/pkg/blob/main/temporal/examples/scheduler/basic_scheduler.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/temporal"

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

// 3. Create worker
client, _ := temporal.NewClient(temporal.ClientConfig{
    HostPort: "localhost:7233",
})
worker := temporal.NewWorker(client, "my-task-queue")
worker.RegisterWorkflow(MyWorkflow)
worker.RegisterActivity(MyActivity)

// 4. Start worker (in a goroutine)
go func() {
    err := worker.Run(context.Background())
    if err != nil {
        log.Fatal().Err(err).Msg("Worker failed")
    
}()

// 5. Trigger workflow manually
workflowOptions := client.StartWorkflowOptions{
    ID:        "my-workflow-id",
    TaskQueue: "my-task-queue",
}
we, err := client.ExecuteWorkflow(context.Background(), workflowOptions, MyWorkflow, "input-data")
if err != nil {
    log.Error().Err(err).Msg("Failed to start workflow")
}
log.Info().Str("workflow_id", we.GetID()).Str("run_id", we.GetRunID()).Msg("Workflow started")

// Get workflow result
var result string
err = we.Get(context.Background(), &result)

// 6. Or create a scheduler for periodic execution
scheduleManager := temporal.NewScheduleManager(client)
defer scheduleManager.Close()

scheduleID := "my-schedule-id"
scheduleOptions := temporal.WorkflowScheduleOptions{
    WorkflowID: "scheduled-workflow-id",
    Workflow:   MyWorkflow,
    TaskQueue:  "my-task-queue",
    Interval:   1 * time.Hour, // Run every hour
    Args:       []any{"scheduled-input"},
}

scheduleHandle, err := scheduleManager.CreateWorkflowSchedule(
    context.Background(), 
    scheduleID, 
    scheduleOptions,
)
if err != nil {
    log.Error().Err(err).Msg("Failed to create schedule")
}
log.Info().Str("schedule_id", scheduleID).Msg("Schedule created")

// Delete schedule when done
err = scheduleHandle.Delete(context.Background())
```

**Key features:**
- Durable execution with automatic retries
- Built-in error handling and compensation
- Timer and scheduling support
- Integration with logging package
- Manual workflow invocation with ExecuteWorkflow
- Scheduled workflows with cron-like intervals
- Async execution with result retrieval

## Directory Structure

```
pkg/temporal/examples/
├── workflows/     # Example workflow implementations
├── activities/    # Example activity implementations
├── worker/        # Example worker setup
├── scheduler/     # Example scheduler setup
└── README.md      # This file
```

## Prerequisites

- Go 1.22.2 or later
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

## How to Run the Examples

Each example directory contains a README.md with specific instructions for running that example. In general, you'll need to:

1. Start the Temporal server
2. Run the worker in one terminal
3. Run the workflow starter in another terminal

## Best Practices

- Use the `NewClient()` or `NewClientWithMetrics()` functions from the main temporal package
- Always handle errors properly in workflows and activities
- Use proper logging with zerolog
- Implement proper shutdown handling for workers
- Use meaningful task queue names
- Structure your code to separate workflows, activities, and worker setup

## Integration with Main Package

These examples use the main temporal package from this project. Key integration points:

- Client creation using `temporal.NewClient()` or `temporal.NewClientWithMetrics()`
- Worker management using `temporal.WorkerManager`
- Schedule management using `temporal.ScheduleManager`
- Logging using `temporal.ZerologAdapter`

For more details on the main temporal package, see the source code in the parent directory.
