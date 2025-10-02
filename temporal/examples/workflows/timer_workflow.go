//go:build example

package workflows

import (
	"time"

	"github.com/jasoet/pkg/v2/temporal/examples/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TimerWorkflow demonstrates a workflow that uses timers and delays.
// It shows how to use timers for scheduling activities, implementing timeouts,
// and creating periodic tasks.
func TimerWorkflow(ctx workflow.Context, duration time.Duration) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("TimerWorkflow started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID,
		"Duration", duration)

	// Step 1: Use a timer to delay execution
	logger.Info("Step 1: Delaying execution with a timer", "delay", duration)
	if err := workflow.Sleep(ctx, duration); err != nil {
		logger.Error("Failed to sleep", "error", err)
		return "", err
	}
	logger.Info("Timer completed, continuing execution")

	// Step 2: Execute an activity with a timeout
	logger.Info("Step 2: Executing activity with a timeout")

	// Set up activity options with a short timeout
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var greetingResult activities.GreetingResult
	greetingInput := activities.GreetingInput{
		Name: "Timer Example",
	}

	err := workflow.ExecuteActivity(ctx, activities.Greeting, greetingInput).Get(ctx, &greetingResult)
	if err != nil {
		logger.Error("Activity failed", "error", err)
		return "", err
	}

	logger.Info("Activity completed", "result", greetingResult.Greeting)

	// Step 3: Implement a periodic task using timers
	logger.Info("Step 3: Implementing a periodic task")

	// We'll execute a task every second for 5 iterations
	iterations := 5
	interval := 1 * time.Second

	var results []string
	results = append(results, greetingResult.Greeting)

	for i := 0; i < iterations; i++ {
		// Sleep for the interval
		if err := workflow.Sleep(ctx, interval); err != nil {
			logger.Error("Failed to sleep", "error", err)
			return "", err
		}

		// Execute a periodic task (in this case, just logging)
		currentTime := workflow.Now(ctx)
		logger.Info("Executing periodic task",
			"iteration", i+1,
			"time", currentTime.Format(time.RFC3339))

		// Add a timestamp to our results
		results = append(results, "Iteration "+string('A'+i)+": "+currentTime.Format(time.RFC3339))
	}

	// Step 4: Implement a timer-based selector
	logger.Info("Step 4: Implementing a timer-based selector")

	// Create a future that will be ready after a timeout
	timerFuture := workflow.NewTimer(ctx, 3*time.Second)

	// Set up an activity to run
	fetchInput := activities.FetchExternalDataInput{
		URL: "https://example.com/api/data",
	}

	// Execute the activity
	fetchFuture := workflow.ExecuteActivity(ctx, activities.FetchExternalData, fetchInput)

	// Use selector to wait for either the activity to complete or the timer to fire
	selector := workflow.NewSelector(ctx)
	var fetchResult activities.FetchExternalDataResult
	var timedOut bool

	selector.AddFuture(fetchFuture, func(f workflow.Future) {
		err := f.Get(ctx, &fetchResult)
		if err != nil {
			logger.Error("Activity failed in selector", "error", err)
		} else {
			logger.Info("Activity completed in selector", "data", fetchResult.Data)
		}
	})

	selector.AddFuture(timerFuture, func(f workflow.Future) {
		err := f.Get(ctx, nil)
		if err != nil {
			logger.Error("Timer failed in selector", "error", err)
		} else {
			logger.Info("Timer fired in selector, activity timed out")
			timedOut = true
		}
	})

	// Wait for one of the futures to complete
	selector.Select(ctx)

	// Add the result to our results
	if timedOut {
		results = append(results, "Activity timed out after 3 seconds")
	} else {
		results = append(results, "Activity completed: "+fetchResult.Data)
	}

	// Step 5: Combine results and return
	result := "Timer Workflow Results:\n"
	for i, r := range results {
		result += "- Result " + string('0'+i) + ": " + r + "\n"
	}

	logger.Info("Workflow completed successfully", "result", result)
	return result, nil
}

// ScheduledWorkflow demonstrates a workflow that runs on a schedule.
// It simulates a workflow that would be scheduled to run periodically.
func ScheduledWorkflow(ctx workflow.Context) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("ScheduledWorkflow started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID)

	// Get the current time
	currentTime := workflow.Now(ctx)
	logger.Info("Current time", "time", currentTime.Format(time.RFC3339))

	// Simulate some scheduled work
	logger.Info("Executing scheduled work")

	// Set up activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute an activity as part of the scheduled work
	var processResult activities.ProcessDataResult
	processInput := activities.ProcessDataInput{
		Data:     "Scheduled data processing at " + currentTime.Format(time.RFC3339),
		Multiply: 1,
	}

	err := workflow.ExecuteActivity(ctx, activities.ProcessData, processInput).Get(ctx, &processResult)
	if err != nil {
		logger.Error("Scheduled activity failed", "error", err)
		return "", err
	}

	logger.Info("Scheduled activity completed", "result", processResult.ProcessedData)

	// Return the result
	result := "Scheduled Workflow Results:\n" +
		"- Execution Time: " + currentTime.Format(time.RFC3339) + "\n" +
		"- Processed Data: " + processResult.ProcessedData

	logger.Info("Workflow completed successfully", "result", result)
	return result, nil
}

// To run these workflows, you need to:
// 1. Register them with a worker (see examples in the worker directory)
// 2. Register all the activities used by the workflows
// 3. Start the worker
// 4. Execute the workflow using a client (example):
//
// ```go
// import (
//     "context"
//     "time"
//     "github.com/rs/zerolog/log"
//     "github.com/amanata-dev/twc-report-backend/pkg/temporal"
//     "github.com/amanata-dev/twc-report-backend/pkg/temporal/examples/activities"
//     "go.temporal.io/sdk/client"
// )
//
// func main() {
//     // Create a Temporal client
//     client, err := temporal.NewClient(temporal.DefaultConfig())
//     if err != nil {
//         log.Fatal().Err(err).Msg("Failed to create Temporal client")
//     }
//     defer client.Close()
//
//     // Set workflow options
//     options := client.StartWorkflowOptions{
//         ID:        "timer-workflow",
//         TaskQueue: "example-task-queue",
//     }
//
//     // Execute the workflow with a 5-second delay
//     we, err := client.ExecuteWorkflow(context.Background(), options, TimerWorkflow, 5*time.Second)
//     if err != nil {
//         log.Fatal().Err(err).Msg("Failed to execute workflow")
//     }
//
//     // Get the workflow result
//     var result string
//     if err := we.Get(context.Background(), &result); err != nil {
//         log.Fatal().Err(err).Msg("Failed to get workflow result")
//     }
//
//     log.Info().Str("result", result).Msg("Workflow completed")
// }
// ```
