# Temporal Package

A thin integration layer over the [Temporal Go SDK](https://github.com/temporalio/sdk-go) (`go.temporal.io/sdk`), providing client construction with functional options, convenience managers for workers, schedules, and workflow queries, and a zerolog logger adapter.

## SDK-Integration Posture

This package **intentionally exposes `go.temporal.io/sdk` types** in its public API — `client.Client`, `worker.Worker`, `client.ScheduleHandle`, and so on. It is **not** an abstraction layer over the Temporal SDK:

- `NewClient` returns a real `client.Client`; every SDK capability stays available.
- The managers (`WorkerManager`, `ScheduleManager`, `WorkflowManager`) are **convenience lifecycle wrappers** — they collect workers/schedules, add structured logging, and offer ready-made query helpers. You can always drop down to the SDK client directly (each manager exposes `GetClient()`).
- For **typed, per-workflow handles** (register, execute, describe, cancel, schedule — all scoped to one workflow definition), use the [`temporal/job`](./job) package's `Definition` instead of the generic managers.

If the SDK can do it, you can do it through the client this package hands you.

## Features

- **Client** (`client.go`) — `NewClient(opts ...Option)` builds a `client.Client` from `DefaultConfig()` with functional options; optional OTel tracing interceptor and metrics handler.
- **WorkerManager** (`worker.go`) — register and start `worker.Worker`s on a shared client; `Close(ctx)` stops all workers.
- **ScheduleManager** (`schedule.go`) — create, list, update, and delete workflow schedules (cron, intervals).
- **WorkflowManager** (`workflow.go`) — query, monitor, and control workflow executions (list/search/describe, cancel/terminate/signal/query, dashboard stats).
- **ZerologAdapter** (`logger.go`) — bridges a `zerolog.Logger` into the Temporal SDK's `log.Logger` interface.
- **job** (`./job`) — typed per-workflow `Definition` with register/execute/query/schedule operations.
- **testcontainer** (`./testcontainer`) — spin up a Temporal server in Docker for integration tests.

## Installing

```bash
go get github.com/jasoet/pkg/v3/temporal
```

## Quick Start

### 1. Create a Client

`NewClient` starts from `DefaultConfig()` (`localhost:7233`, namespace `default`) and applies options in order. **The caller owns the returned client** — close it with `client.Close()`.

```go
package main

import (
    "github.com/jasoet/pkg/v3/temporal"
)

func main() {
    // Defaults: localhost:7233, namespace "default"
    c, err := temporal.NewClient()
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // Or with options:
    c, err = temporal.NewClient(
        temporal.WithHostPort("temporal.example.com:7233"),
        temporal.WithNamespace("production"),
        // temporal.WithOTelConfig(otelCfg),  // attach OTel tracing/metrics
        // temporal.WithConfig(myConfig),     // or replace the whole Config
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()
}
```

#### Options

| Option | Effect |
|---|---|
| `WithConfig(c Config)` | Replace the entire configuration with `c` |
| `WithHostPort(addr)` | Set the Temporal frontend address (`host:port`) |
| `WithNamespace(ns)` | Set the Temporal namespace |
| `WithOTelConfig(otelCfg)` | Attach OTel tracing interceptor and metrics handler |

### 2. Manage Workers

`NewWorkerManager` takes an existing `client.Client`. The caller retains ownership of the client: **`WorkerManager.Close(ctx)` stops the registered workers but never closes the client.**

```go
c, err := temporal.NewClient()
if err != nil {
    panic(err)
}
defer c.Close()

wm, err := temporal.NewWorkerManager(c)
if err != nil {
    panic(err)
}
defer wm.Close(ctx) // stops workers; does NOT close c

w := wm.Register("my-task-queue", worker.Options{})
w.RegisterWorkflow(MyWorkflow)
w.RegisterActivity(MyActivity)

if err := wm.StartAll(ctx); err != nil {
    panic(err)
}
```

### 3. Query and Monitor Workflows

`NewWorkflowManager(c)` uses the `default` namespace; use `NewWorkflowManagerWithNamespace(c, ns)` for another one. The manager has no `Close` — there is nothing to release; the client is caller-owned.

```go
wfm, err := temporal.NewWorkflowManagerWithNamespace(c, "production")
if err != nil {
    panic(err)
}

// Dashboard statistics
stats, err := wfm.GetDashboardStats(ctx)
fmt.Printf("Running: %d, Completed: %d, Failed: %d\n",
    stats.TotalRunning, stats.TotalCompleted, stats.TotalFailed)

// List / search
running, err := wfm.ListRunningWorkflows(ctx, 100)
orders, err := wfm.SearchWorkflowsByType(ctx, "OrderProcessingWorkflow", 50)

// Details and lifecycle
details, err := wfm.DescribeWorkflow(ctx, "order-123", "")
err = wfm.CancelWorkflow(ctx, "problematic-workflow-id", "")
```

### 4. Schedule Workflows

`NewScheduleManager` also takes a caller-owned client; `Close(ctx)` only logs — it never closes the client.

```go
sm, err := temporal.NewScheduleManager(c)
if err != nil {
    panic(err)
}
defer sm.Close(ctx)

handle, err := sm.CreateWorkflowSchedule(ctx, "hourly-report", temporal.WorkflowScheduleOptions{
    WorkflowID: "report-workflow",
    Workflow:   ReportWorkflow,
    TaskQueue:  "reports",
    Interval:   time.Hour,
    Args:       []any{"daily-report"},
})
```

### 5. Typed Per-Workflow Handles (`temporal/job`)

For application workflows, prefer a `job.Definition`: it binds a workflow type to its name, task queue, registration, and execution, and gives you typed operations scoped to that workflow — including schedules whose ID equals the definition name.

```go
import "github.com/jasoet/pkg/v3/temporal/job"

def, err := job.New("report", "reports",
    job.WithRegister(func(w worker.Worker) {
        w.RegisterWorkflow(ReportWorkflow)
    }),
    job.WithExecute(func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, input any) (client.WorkflowRun, error) {
        return c.ExecuteWorkflow(ctx, opts, ReportWorkflow, input)
    }),
    job.WithNewInput(func() any { return ReportInput{} }),
    job.WithSchedule(&job.ScheduleSpec{Interval: time.Hour}),
)
```

See the [job package](./job) for `Register`, `Execute`, `Describe`, `Cancel`, `ApplySchedule`, `ListRuns`, and more.

## ZerologAdapter

`ZerologAdapter` adapts a `zerolog.Logger` to the Temporal SDK's `log.Logger` interface, so SDK internal logs flow into your zerolog pipeline. `NewClient` wires one up automatically; construct your own when you dial the SDK directly:

```go
import (
    "github.com/rs/zerolog"
    "go.temporal.io/sdk/client"

    "github.com/jasoet/pkg/v3/temporal"
)

zlog := zerolog.New(os.Stderr).With().Timestamp().Logger()
c, err := client.Dial(client.Options{
    HostPort:  "localhost:7233",
    Namespace: "default",
    Logger:    temporal.NewZerologAdapter(zlog),
})
```

It supports `Debug/Info/Warn/Error` with key-value pairs (odd keyvals are tolerated), `With(...)` for derived loggers, and `WithCallerSkip(skip)`.

## Client Ownership and Lifecycle

- `NewClient` returns a `client.Client` that **you** own — always `defer c.Close()`.
- `NewWorkerManager(c)`, `NewScheduleManager(c)`, `NewWorkflowManager(c)` / `NewWorkflowManagerWithNamespace(c, ns)` borrow the client; they never close it.
- `WorkerManager.Close(ctx)` stops all registered workers; `ScheduleManager.Close(ctx)` is a logging-only no-op. Both take a `context.Context` used for logging only.
- `WorkflowManager` has no `Close` — it holds no resources beyond the borrowed client.

## Examples

Runnable examples live in [examples/temporal](../examples/temporal/):

- **[Dashboard](../examples/temporal/dashboard/)** — HTTP dashboard for monitoring workflows
- **[Worker](../examples/temporal/worker/)** — worker setup patterns
- **[Workflows](../examples/temporal/workflows/)** — sample workflow implementations
- **[Scheduler](../examples/temporal/scheduler/)** — scheduling workflows

The examples are guarded by the `example` build tag:

```bash
go build -tags=example ./examples/temporal/...
```

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
- `ListWorkflows(ctx, pageSize, query)` — list workflows with optional filtering
- `ListRunningWorkflows(ctx, pageSize)` / `ListCompletedWorkflows(ctx, pageSize)` / `ListFailedWorkflows(ctx, pageSize)` — list by status
- `ListWorkflowsByStatus(ctx, status, pageSize)` — list by an explicit status
- `DescribeWorkflow(ctx, workflowID, runID)` — detailed workflow information
- `GetWorkflowStatus(ctx, workflowID, runID)` — current status
- `GetWorkflowHistory(ctx, workflowID, runID)` — workflow event history

#### Search Operations
- `SearchWorkflowsByType(ctx, workflowType, pageSize)` — find by workflow type
- `SearchWorkflowsByID(ctx, idPrefix, pageSize)` — find by workflow ID prefix
- `CountWorkflows(ctx, query)` — count workflows matching a query

#### Lifecycle Operations
- `CancelWorkflow(ctx, workflowID, runID)` — cancel a running workflow
- `TerminateWorkflow(ctx, workflowID, runID, reason)` — terminate a workflow
- `SignalWorkflow(ctx, workflowID, runID, signalName, data)` — send a signal
- `QueryWorkflow(ctx, workflowID, runID, queryType, args)` — query workflow state

#### Dashboard Operations
- `GetDashboardStats(ctx)` — aggregated workflow statistics
- `GetRecentWorkflows(ctx, limit)` — most recent workflows
- `GetWorkflowResult(ctx, workflowID, runID, valuePtr)` — workflow result

Visibility-query parameters passed to `ListWorkflowsByStatus`, `SearchWorkflowsByType`, and `SearchWorkflowsByID` are validated against a safe-identifier pattern (alphanumerics, hyphens, underscores, dots).

## Testing

The package has two test tiers:

- **Unit tests** (no tag) — config/options, zerolog adapter, query validation, and manager behavior with mock clients:

  ```bash
  go test ./temporal/ -count=1
  ```

- **Integration tests** (`//go:build integration`) — full suites against a real Temporal server managed by testcontainers (client, worker, schedule, workflow, e2e):

  ```bash
  go test -tags=integration -timeout=10m ./temporal/...
  # or via Taskfile
  task test:integration
  ```

Prerequisites for integration tests: Docker. Each suite gets its own isolated `temporalio/temporal` container; no manual server setup is required.

### Testcontainer Package

The `temporal/testcontainer` package provides reusable utilities for running a Temporal server in Docker containers for integration testing. It can be used in your own projects too.

```bash
go get github.com/jasoet/pkg/v3/temporal/testcontainer
```

**Simple setup (recommended):**

```go
import (
    "context"
    "testing"

    "github.com/jasoet/pkg/v3/temporal/testcontainer"
)

func TestMyWorkflow(t *testing.T) {
    ctx := context.Background()

    _, c, cleanup, err := testcontainer.Setup(
        ctx,
        testcontainer.ClientConfig{Namespace: "default"},
        testcontainer.Options{Logger: t},
    )
    if err != nil {
        t.Fatalf("Setup failed: %v", err)
    }
    defer cleanup()

    // Use c (a client.Client) for your tests...
}
```

**Advanced setup** (manual container management, dial the SDK yourself):

```go
container, err := testcontainer.Start(ctx, testcontainer.Options{
    Image:          "temporalio/temporal:1.22.0",
    StartupTimeout: 120 * time.Second,
    Logger:         t,
})
if err != nil {
    t.Fatalf("Failed to start: %v", err)
}
defer container.Terminate(ctx)

temporalClient, err := client.Dial(client.Options{
    HostPort:  container.HostPort(),
    Namespace: "default",
})
```

**Configuration options:**

```go
testcontainer.Options{
    Image:           "temporalio/temporal:latest", // Docker image
    StartupTimeout:  60 * time.Second,             // Startup timeout
    Logger:          t,                            // *testing.T or custom logger
    ExtraPorts:      []string{"8080/tcp"},         // Additional ports
    InitialWaitTime: 3 * time.Second,              // Wait after startup
}
```

See the [testcontainer package documentation](./testcontainer/doc.go) and [examples](./testcontainer/example_test.go) for more details.

## Contributing

When adding new integration tests:

1. Use the `//go:build integration` tag.
2. Create unique identifiers (workflow IDs, task queues, schedule IDs — timestamps work well).
3. Use the testcontainer package (`testcontainer.Setup()`).
4. Construct clients with `temporal.NewClient(...)` and managers with the typed constructors (`NewWorkerManager(c)`, etc.).
5. Document any new configuration requirements.
