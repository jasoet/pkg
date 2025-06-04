//go:build example

package workflows

import (
	"errors"
	"time"

	"github.com/jasoet/pkg/temporal/examples/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ErrorHandlingWorkflow demonstrates a workflow with error handling and retries.
// It shows how to handle activity errors and implement retry strategies.
func ErrorHandlingWorkflow(ctx workflow.Context, url string) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("ErrorHandlingWorkflow started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID,
		"URL", url)

	// Step 1: Set up activity options with retry policy
	// This demonstrates how to configure retries for activities
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
			// Non-retryable errors - errors matching these types will not be retried
			NonRetryableErrorTypes: []string{"InvalidArgumentError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 2: Execute the FetchExternalData activity which might fail
	logger.Info("Executing FetchExternalData activity with retry policy")
	var fetchResult activities.FetchExternalDataResult
	fetchInput := activities.FetchExternalDataInput{
		URL: url,
	}

	// Execute the activity and handle potential errors
	err := workflow.ExecuteActivity(ctx, activities.FetchExternalData, fetchInput).Get(ctx, &fetchResult)
	if err != nil {
		// Check if it's a specific error type
		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) {
			logger.Error("Application error from activity",
				"error", err,
				"type", applicationErr.Type())

			// Handle specific error types differently
			if applicationErr.Type() == "InvalidArgumentError" {
				return "Invalid URL provided: " + url, nil
			}
		}

		// For other errors, we'll return the error to fail the workflow
		logger.Error("FetchExternalData activity failed after retries", "error", err)
		return "", err
	}

	logger.Info("FetchExternalData activity completed successfully",
		"data", fetchResult.Data,
		"statusCode", fetchResult.StatusCode)

	// Step 3: Demonstrate error handling pattern
	var processResult activities.ProcessDataResult

	// Set up different options for the next activity
	processActivityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		// No retry policy - we'll handle errors manually
	}
	processCtx := workflow.WithActivityOptions(ctx, processActivityOptions)

	logger.Info("Demonstrating error handling pattern")

	// Process the data we fetched, with error handling
	processInput := activities.ProcessDataInput{
		Data:     fetchResult.Data,
		Multiply: 2,
	}

	// Execute the activity and handle errors
	err = workflow.ExecuteActivity(processCtx, activities.ProcessData, processInput).Get(ctx, &processResult)
	if err != nil {
		logger.Error("Caught error from ProcessData activity", "error", err)
		// Create a default result if the activity fails
		processResult = activities.ProcessDataResult{
			ProcessedData: "Default processed data (activity failed)",
			Count:         0,
		}
		// Note: We're not returning the error, we're handling it and continuing
	}

	// Step 4: Demonstrate custom retry logic
	logger.Info("Demonstrating custom retry logic")

	// This demonstrates how to implement custom retry logic
	var greetingResult activities.GreetingResult
	greetingInput := activities.GreetingInput{
		Name: "Retry Example",
	}

	// Custom retry logic with backoff
	retryCount := 0
	maxRetries := 3
	var retryErr error

	for retryCount < maxRetries {
		logger.Info("Executing Greeting activity", "attempt", retryCount+1)

		// Execute activity with a shorter timeout
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Second,
			// No retry policy - we're implementing our own retry logic
		}
		retryCtx := workflow.WithActivityOptions(ctx, activityOptions)

		err := workflow.ExecuteActivity(retryCtx, activities.Greeting, greetingInput).Get(ctx, &greetingResult)
		if err == nil {
			// Success!
			logger.Info("Greeting activity succeeded", "result", greetingResult.Greeting)
			retryErr = nil
			break
		}

		retryErr = err
		retryCount++

		if retryCount < maxRetries {
			// Calculate backoff duration (exponential backoff)
			backoffDuration := time.Duration(1<<uint(retryCount)) * time.Second
			logger.Info("Greeting activity failed, retrying after backoff",
				"error", err,
				"attempt", retryCount,
				"backoff", backoffDuration)

			// Sleep before retry
			if err := workflow.Sleep(ctx, backoffDuration); err != nil {
				logger.Error("Failed to sleep before retry", "error", err)
				return "", err
			}
		} else {
			logger.Error("Greeting activity failed after max retries", "error", err)
		}
	}

	if retryErr != nil {
		return "", retryErr
	}

	// Step 5: Combine results and return
	result := "Error Handling Workflow Results:\n" +
		"Fetched data: " + fetchResult.Data + "\n" +
		"Processed data: " + processResult.ProcessedData + "\n" +
		"Greeting: " + greetingResult.Greeting

	logger.Info("Workflow completed successfully", "result", result)
	return result, nil
}

// To run this workflow, you need to:
// 1. Register it with a worker (see examples in the worker directory)
// 2. Register all the activities used by the workflow
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
//         ID:        "error-handling-workflow",
//         TaskQueue: "example-task-queue",
//     }
//
//     // Execute the workflow with a valid URL
//     we, err := client.ExecuteWorkflow(context.Background(), options, ErrorHandlingWorkflow, "https://example.com/api/data")
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
//
//     // Try with an error URL to see error handling in action
//     options.ID = "error-handling-workflow-with-error"
//     we, err = client.ExecuteWorkflow(context.Background(), options, ErrorHandlingWorkflow, "https://example.com/error")
//     if err != nil {
//         log.Fatal().Err(err).Msg("Failed to execute workflow")
//     }
//
//     if err := we.Get(context.Background(), &result); err != nil {
//         log.Error().Err(err).Msg("Workflow failed as expected")
//     } else {
//         log.Info().Str("result", result).Msg("Workflow completed with error handling")
//     }
// }
// ```
