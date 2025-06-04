package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// SimpleWorkflow demonstrates a basic sequential workflow.
// It executes a series of steps in sequence and returns a result.
func SimpleWorkflow(ctx workflow.Context) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("SimpleWorkflow started", "WorkflowID", workflowInfo.WorkflowExecution.ID)

	// Step 1: Log the start of the workflow
	logger.Info("Executing Step 1: Starting the workflow")

	// Step 2: Simulate some processing time
	logger.Info("Executing Step 2: Processing data")
	// In a real workflow, you might do some actual processing here
	// For this example, we'll just sleep to simulate work
	if err := workflow.Sleep(ctx, 1*time.Second); err != nil {
		logger.Error("Failed to sleep", "error", err)
		return "", err
	}

	// Step 3: Make a decision
	logger.Info("Executing Step 3: Making a decision")
	currentTime := workflow.Now(ctx)
	var result string
	if currentTime.Hour() < 12 {
		result = "Good morning! Workflow completed successfully."
	} else {
		result = "Good afternoon! Workflow completed successfully."
	}

	// Step 4: Complete the workflow
	logger.Info("Executing Step 4: Completing the workflow", "result", result)

	// Return the result
	return result, nil
}

// SimpleWorkflowWithParams demonstrates a workflow that accepts parameters.
// It takes a name parameter and returns a personalized greeting.
func SimpleWorkflowWithParams(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	logger.Info("SimpleWorkflowWithParams started",
		"WorkflowID", workflowInfo.WorkflowExecution.ID,
		"Name", name)

	// Step 1: Validate input
	logger.Info("Executing Step 1: Validating input")
	if name == "" {
		name = "World" // Default value if no name is provided
	}

	// Step 2: Simulate some processing time
	logger.Info("Executing Step 2: Processing data")
	if err := workflow.Sleep(ctx, 1*time.Second); err != nil {
		logger.Error("Failed to sleep", "error", err)
		return "", err
	}

	// Step 3: Generate personalized message
	logger.Info("Executing Step 3: Generating personalized message")
	result := fmt.Sprintf("Hello, %s! Your workflow completed at %s.",
		name,
		workflow.Now(ctx).Format(time.RFC3339))

	// Step 4: Complete the workflow
	logger.Info("Executing Step 4: Completing the workflow", "result", result)

	// Return the result
	return result, nil
}

// To run these workflows, you need to:
// 1. Register them with a worker (see examples in the worker directory)
// 2. Start the worker
// 3. Execute the workflow using a client (example):
//
// ```go
// import (
//     "context"
//     "github.com/rs/zerolog/log"
//     "github.com/amanata-dev/twc-report-backend/pkg/temporal"
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
//         ID:        "simple-workflow",
//         TaskQueue: "example-task-queue",
//     }
//
//     // Execute the workflow
//     we, err := client.ExecuteWorkflow(context.Background(), options, SimpleWorkflow)
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
