# Temporal Workflow Examples

This directory contains examples demonstrating how to use Temporal workflows in Go. These examples show practical usage patterns and best practices for implementing Temporal workflows, activities, workers, and schedulers.

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
