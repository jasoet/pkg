package temporal_test

import (
	"context"
	"fmt"
	"time"

	"github.com/jasoet/pkg/v3/temporal"
)

// ExampleNewClient demonstrates creating a Temporal client with the options
// API. Without options, NewClient dials localhost:7233 in the "default"
// namespace. The caller owns the returned client and must close it.
//
// This example has no Output comment because the result depends on a running
// Temporal server; it is compile-checked but not executed by go test.
func ExampleNewClient() {
	c, err := temporal.NewClient(
		temporal.WithHostPort("localhost:7233"),
		temporal.WithNamespace("default"),
	)
	if err != nil {
		// No Temporal server is listening; handle the dial error.
		fmt.Println("dial failed:", err != nil)
		return
	}
	defer c.Close()

	fmt.Println("connected:", c != nil)
}

// ExampleNewScheduleManager demonstrates creating a ScheduleManager from a
// caller-owned client. ScheduleManager.Close(ctx) does not close the client;
// the caller closes it.
//
// This example has no Output comment because the result depends on a running
// Temporal server; it is compile-checked but not executed by go test.
func ExampleNewScheduleManager() {
	ctx := context.Background()

	c, err := temporal.NewClient()
	if err != nil {
		fmt.Println("dial failed:", err != nil)
		return
	}
	defer c.Close()

	sm, err := temporal.NewScheduleManager(c)
	if err != nil {
		fmt.Println("manager failed:", err != nil)
		return
	}
	defer sm.Close(ctx)

	// Create an interval schedule (requires a running Temporal server).
	_, err = sm.CreateWorkflowSchedule(ctx, "hourly-report", temporal.WorkflowScheduleOptions{
		WorkflowID: "report-workflow",
		Workflow:   "ReportWorkflow", // or a workflow function reference
		TaskQueue:  "reports",
		Interval:   time.Hour,
		Args:       []any{"daily-report"},
	})
	if err != nil {
		fmt.Println("schedule failed:", err != nil)
		return
	}

	fmt.Println("schedule created")
}
