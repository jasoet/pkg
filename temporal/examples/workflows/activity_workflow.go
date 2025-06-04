//go:build example

package workflows

import (
	"time"

	"github.com/jasoet/pkg/temporal/examples/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ActivityWorkflow demonstrates a workflow that calls activities.
// It executes a series of activities and combines their results.
func ActivityWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("ActivityWorkflow started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID,
		"Name", name)

	// Step 1: Set activity options
	// These options apply to all activities executed in this workflow
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

	// Step 2: Execute the Greeting activity
	logger.Info("Executing Greeting activity")
	var greetingResult activities.GreetingResult
	greetingInput := activities.GreetingInput{
		Name: name,
	}
	err := workflow.ExecuteActivity(ctx, activities.Greeting, greetingInput).Get(ctx, &greetingResult)
	if err != nil {
		logger.Error("Greeting activity failed", "error", err)
		return "", err
	}
	logger.Info("Greeting activity completed", "result", greetingResult.Greeting)

	// Step 3: Execute the ProcessData activity
	logger.Info("Executing ProcessData activity")
	var processResult activities.ProcessDataResult
	processInput := activities.ProcessDataInput{
		Data:     "Process this data",
		Multiply: 3,
	}
	err = workflow.ExecuteActivity(ctx, activities.ProcessData, processInput).Get(ctx, &processResult)
	if err != nil {
		logger.Error("ProcessData activity failed", "error", err)
		return "", err
	}
	logger.Info("ProcessData activity completed",
		"processedData", processResult.ProcessedData,
		"count", processResult.Count)

	// Step 4: Execute the FetchExternalData activity
	logger.Info("Executing FetchExternalData activity")
	var fetchResult activities.FetchExternalDataResult
	fetchInput := activities.FetchExternalDataInput{
		URL: "https://example.com/api/data",
	}
	err = workflow.ExecuteActivity(ctx, activities.FetchExternalData, fetchInput).Get(ctx, &fetchResult)
	if err != nil {
		logger.Error("FetchExternalData activity failed", "error", err)
		return "", err
	}
	logger.Info("FetchExternalData activity completed",
		"data", fetchResult.Data,
		"statusCode", fetchResult.StatusCode)

	// Step 5: Combine the results
	logger.Info("Combining activity results")
	result := greetingResult.Greeting + "\n" +
		"Processed data: " + processResult.ProcessedData + "\n" +
		"External data: " + fetchResult.Data

	logger.Info("Workflow completed", "result", result)
	return result, nil
}

// ActivityWorkflowWithChildWorkflow demonstrates a workflow that calls activities and a child workflow.
// It executes activities and then calls a child workflow to process the results.
func ActivityWorkflowWithChildWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("ActivityWorkflowWithChildWorkflow started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID,
		"Name", name)

	// Step 1: Set activity options
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

	// Step 2: Execute the Greeting activity
	logger.Info("Executing Greeting activity")
	var greetingResult activities.GreetingResult
	greetingInput := activities.GreetingInput{
		Name: name,
	}
	err := workflow.ExecuteActivity(ctx, activities.Greeting, greetingInput).Get(ctx, &greetingResult)
	if err != nil {
		logger.Error("Greeting activity failed", "error", err)
		return "", err
	}
	logger.Info("Greeting activity completed", "result", greetingResult.Greeting)

	// Step 3: Execute a child workflow to process the greeting
	logger.Info("Executing child workflow")
	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID: "child-" + workflowInfo.WorkflowExecution.ID,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)

	var childResult string
	err = workflow.ExecuteChildWorkflow(childCtx, SimpleWorkflowWithParams, greetingResult.Greeting).Get(ctx, &childResult)
	if err != nil {
		logger.Error("Child workflow failed", "error", err)
		return "", err
	}
	logger.Info("Child workflow completed", "result", childResult)

	// Step 4: Combine the results
	result := "Parent workflow result:\n" + greetingResult.Greeting + "\n\n" +
		"Child workflow result:\n" + childResult

	logger.Info("Workflow completed", "result", result)
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
//         ID:        "activity-workflow",
//         TaskQueue: "example-task-queue",
//     }
//
//     // Execute the workflow
//     we, err := client.ExecuteWorkflow(context.Background(), options, ActivityWorkflow, "John Doe")
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
