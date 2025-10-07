//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/jasoet/pkg/v2/otel"
)

// Example 1: Submit a Simple Workflow
// Demonstrates the basic workflow submission pattern
func exampleSubmitWorkflow() {
	ctx := context.Background()

	// Initialize Argo client
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Build a simple workflow
	step := template.NewContainer("hello", "alpine:3.19",
		template.WithCommand("echo", "Hello, World!"))

	wf, err := builder.NewWorkflowBuilder("hello-world", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"app":  "examples",
			"type": "simple",
		})).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Submit the workflow
	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow submitted successfully: %s\n", created.Name)
	fmt.Printf("View workflow: kubectl get workflow -n argo %s\n", created.Name)
}

// Example 2: Submit Workflow with OpenTelemetry Tracing
// Demonstrates how to use OpenTelemetry for observability
func exampleSubmitWorkflowWithOTel() {
	ctx := context.Background()

	// Initialize OpenTelemetry
	otelConfig := otel.NewConfig("argo-operations-example")
	defer otelConfig.Shutdown(ctx)

	// Initialize Argo client with OTel
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Build workflow
	step := template.NewContainer("traced-step", "alpine:3.19",
		template.WithCommand("echo", "This workflow is being traced!"))

	wf, err := builder.NewWorkflowBuilder("traced-workflow", "argo",
		builder.WithServiceAccount("default")).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Submit with OpenTelemetry tracing
	created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Traced workflow submitted: %s\n", created.Name)
	fmt.Println("Traces will be exported to configured OpenTelemetry collector")
}

// Example 3: Submit and Wait for Completion
// Demonstrates submitting a workflow and waiting for it to complete
func exampleSubmitAndWait() {
	ctx := context.Background()

	// Initialize client
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Build a multi-step workflow
	step1 := template.NewContainer("step1", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Step 1 running'; sleep 5"))
	step2 := template.NewContainer("step2", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Step 2 running'; sleep 5"))
	step3 := template.NewContainer("step3", "alpine:3.19",
		template.WithCommand("echo", "All steps completed!"))

	wf, err := builder.NewWorkflowBuilder("wait-example", "argo",
		builder.WithServiceAccount("default")).
		Add(step1).
		Add(step2).
		Add(step3).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	fmt.Println("Submitting workflow and waiting for completion...")
	startTime := time.Now()

	// Submit and wait with 5 minute timeout
	completed, err := argo.SubmitAndWait(ctx, client, wf, nil, 5*time.Minute)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("Workflow completed in %s\n", duration.Round(time.Second))
	fmt.Printf("Status: %s\n", completed.Status.Phase)
	fmt.Printf("Message: %s\n", completed.Status.Message)
}

// Example 4: Submit and Wait with Custom Timeout and Error Handling
// Demonstrates advanced error handling for long-running workflows
func exampleSubmitAndWaitWithErrorHandling() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	otelConfig := otel.NewConfig("argo-wait-example")
	defer otelConfig.Shutdown(ctx)

	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Build a long-running data processing workflow
	process := template.NewContainer("process-data", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Processing data...'; sleep 30"))

	wf, err := builder.NewWorkflowBuilder("data-process", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"app":  "data-pipeline",
			"env":  "production",
		})).
		Add(process).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Submit and wait with longer timeout for data processing
	fmt.Println("Starting data processing workflow...")
	completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 10*time.Minute)

	if err != nil {
		// Handle different failure scenarios
		if completed != nil {
			fmt.Printf("Workflow failed: %s\n", completed.Status.Phase)
			fmt.Printf("Failure message: %s\n", completed.Status.Message)

			// Log detailed node information
			for nodeName, node := range completed.Status.Nodes {
				if node.Phase == v1alpha1.NodeFailed || node.Phase == v1alpha1.NodeError {
					fmt.Printf("Failed node: %s, Phase: %s, Message: %s\n",
						nodeName, node.Phase, node.Message)
				}
			}
		} else {
			fmt.Printf("Workflow submission or polling failed: %v\n", err)
		}
		return
	}

	fmt.Printf("Workflow completed successfully: %s\n", completed.Name)
}

// Example 5: Get Workflow Status
// Demonstrates retrieving and displaying workflow status
func exampleGetWorkflowStatus() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Replace with an actual workflow name from your cluster
	workflowName := "example-workflow-xxxxx"
	namespace := "argo"

	// Get workflow status
	status, err := argo.GetWorkflowStatus(ctx, client, namespace, workflowName, nil)
	if err != nil {
		log.Fatalf("Failed to get workflow status: %v", err)
	}

	// Display status information
	fmt.Printf("Workflow: %s\n", workflowName)
	fmt.Printf("Phase: %s\n", status.Phase)
	fmt.Printf("Message: %s\n", status.Message)
	fmt.Printf("Started at: %s\n", status.StartedAt.Format(time.RFC3339))

	if !status.FinishedAt.IsZero() {
		fmt.Printf("Finished at: %s\n", status.FinishedAt.Format(time.RFC3339))
		duration := status.FinishedAt.Sub(status.StartedAt.Time)
		fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	} else {
		fmt.Println("Status: Still running")
	}

	// Display node information
	fmt.Printf("\nNodes: %d total\n", len(status.Nodes))
	for nodeName, node := range status.Nodes {
		fmt.Printf("  - %s: %s\n", nodeName, node.Phase)
	}
}

// Example 6: Monitor Workflow Status with Polling
// Demonstrates how to poll workflow status until completion
func exampleMonitorWorkflowStatus() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Submit a workflow first
	step := template.NewContainer("monitor-example", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Running...'; sleep 20; echo 'Done!'"))

	wf, err := builder.NewWorkflowBuilder("monitor-example", "argo",
		builder.WithServiceAccount("default")).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow submitted: %s\n", created.Name)
	fmt.Println("Monitoring workflow status...")

	// Poll status every 3 seconds
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			fmt.Println("Timeout reached while monitoring workflow")
			return

		case <-ticker.C:
			status, err := argo.GetWorkflowStatus(ctx, client, created.Namespace, created.Name, nil)
			if err != nil {
				fmt.Printf("Error getting status: %v\n", err)
				continue
			}

			fmt.Printf("[%s] Phase: %s\n", time.Now().Format("15:04:05"), status.Phase)

			// Check if workflow is complete
			if status.Phase == v1alpha1.WorkflowSucceeded {
				fmt.Println("Workflow completed successfully!")
				return
			}

			if status.Phase == v1alpha1.WorkflowFailed || status.Phase == v1alpha1.WorkflowError {
				fmt.Printf("Workflow failed: %s\n", status.Message)
				return
			}
		}
	}
}

// Example 7: List All Workflows
// Demonstrates listing workflows in a namespace
func exampleListWorkflows() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	namespace := "argo"

	// List all workflows
	workflows, err := argo.ListWorkflows(ctx, client, namespace, "", nil)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}

	fmt.Printf("Found %d workflows in namespace %s:\n\n", len(workflows), namespace)

	// Display workflow information
	for _, wf := range workflows {
		fmt.Printf("Name: %s\n", wf.Name)
		fmt.Printf("  Phase: %s\n", wf.Status.Phase)
		fmt.Printf("  Started: %s\n", wf.Status.StartedAt.Format(time.RFC3339))

		if !wf.Status.FinishedAt.IsZero() {
			duration := wf.Status.FinishedAt.Sub(wf.Status.StartedAt.Time)
			fmt.Printf("  Duration: %s\n", duration.Round(time.Second))
		} else {
			fmt.Printf("  Duration: Running...\n")
		}

		fmt.Println()
	}
}

// Example 8: List Workflows with Label Selector
// Demonstrates filtering workflows using label selectors
func exampleListWorkflowsWithLabels() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	namespace := "argo"

	// Example 1: Filter by single label
	fmt.Println("=== Workflows with label app=myapp ===")
	workflows, err := argo.ListWorkflows(ctx, client, namespace, "app=myapp", nil)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}
	fmt.Printf("Found %d workflows\n\n", len(workflows))

	// Example 2: Filter by multiple labels
	fmt.Println("=== Workflows with labels app=myapp,env=production ===")
	workflows, err = argo.ListWorkflows(ctx, client, namespace, "app=myapp,env=production", nil)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}
	fmt.Printf("Found %d workflows\n\n", len(workflows))

	// Example 3: Filter using label expressions
	fmt.Println("=== Workflows where app exists ===")
	workflows, err = argo.ListWorkflows(ctx, client, namespace, "app", nil)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}
	fmt.Printf("Found %d workflows\n\n", len(workflows))

	// Display filtered workflows
	for _, wf := range workflows {
		fmt.Printf("Name: %s, Labels: %v\n", wf.Name, wf.Labels)
	}
}

// Example 9: Delete a Workflow
// Demonstrates deleting a workflow by name
func exampleDeleteWorkflow() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	namespace := "argo"
	workflowName := "example-workflow-xxxxx" // Replace with actual workflow name

	// Delete the workflow
	err = argo.DeleteWorkflow(ctx, client, namespace, workflowName, nil)
	if err != nil {
		log.Fatalf("Failed to delete workflow: %v", err)
	}

	fmt.Printf("Workflow %s deleted successfully\n", workflowName)
}

// Example 10: Complete Workflow Lifecycle Management
// Demonstrates the full lifecycle: submit, monitor, and cleanup
func exampleCompleteWorkflowLifecycle() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	otelConfig := otel.NewConfig("argo-lifecycle-example")
	defer otelConfig.Shutdown(ctx)

	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Step 1: Build and submit workflow
	fmt.Println("=== Step 1: Submitting Workflow ===")
	step := template.NewContainer("lifecycle-demo", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Running workflow'; sleep 10; echo 'Complete'"))

	wf, err := builder.NewWorkflowBuilder("lifecycle-demo", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"app":  "lifecycle-example",
			"type": "demo",
		})).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}
	fmt.Printf("Workflow submitted: %s\n\n", created.Name)

	// Step 2: Wait for completion
	fmt.Println("=== Step 2: Waiting for Completion ===")
	completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 2*time.Minute)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}
	fmt.Printf("Workflow completed: %s\n\n", completed.Status.Phase)

	// Step 3: Get final status
	fmt.Println("=== Step 3: Getting Final Status ===")
	status, err := argo.GetWorkflowStatus(ctx, client, created.Namespace, created.Name, otelConfig)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	fmt.Printf("Phase: %s\n", status.Phase)
	fmt.Printf("Duration: %s\n\n", status.FinishedAt.Sub(status.StartedAt.Time).Round(time.Second))

	// Step 4: List workflows with our label
	fmt.Println("=== Step 4: Listing Similar Workflows ===")
	workflows, err := argo.ListWorkflows(ctx, client, created.Namespace, "app=lifecycle-example", otelConfig)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}
	fmt.Printf("Found %d workflows with label app=lifecycle-example\n\n", len(workflows))

	// Step 5: Cleanup (optional)
	fmt.Println("=== Step 5: Cleanup ===")
	fmt.Printf("To delete workflow, run: kubectl delete workflow -n %s %s\n", created.Namespace, created.Name)

	// Uncomment to actually delete:
	// err = argo.DeleteWorkflow(ctx, client, created.Namespace, created.Name, otelConfig)
	// if err != nil {
	//     log.Fatalf("Failed to delete workflow: %v", err)
	// }
	// fmt.Printf("Workflow %s deleted\n", created.Name)
}

// Example 11: Batch Operations - Submit Multiple Workflows
// Demonstrates submitting and managing multiple workflows
func exampleBatchOperations() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Submit multiple workflows
	workflowNames := []string{}
	numWorkflows := 3

	fmt.Printf("Submitting %d workflows...\n", numWorkflows)

	for i := 1; i <= numWorkflows; i++ {
		step := template.NewContainer(fmt.Sprintf("batch-%d", i), "alpine:3.19",
			template.WithCommand("sh", "-c", fmt.Sprintf("echo 'Workflow %d'; sleep 5", i)))

		wf, err := builder.NewWorkflowBuilder(fmt.Sprintf("batch-job-%d", i), "argo",
			builder.WithServiceAccount("default"),
			builder.WithLabels(map[string]string{
				"batch": "true",
				"run":   "demo",
			})).
			Add(step).
			Build()
		if err != nil {
			log.Fatalf("Failed to build workflow: %v", err)
		}

		created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
		if err != nil {
			log.Fatalf("Failed to submit workflow: %v", err)
		}

		workflowNames = append(workflowNames, created.Name)
		fmt.Printf("  Submitted: %s\n", created.Name)
	}

	// Monitor all workflows
	fmt.Println("\nMonitoring workflows...")
	time.Sleep(2 * time.Second)

	for _, name := range workflowNames {
		status, err := argo.GetWorkflowStatus(ctx, client, "argo", name, nil)
		if err != nil {
			fmt.Printf("  Error getting status for %s: %v\n", name, err)
			continue
		}
		fmt.Printf("  %s: %s\n", name, status.Phase)
	}

	// List all batch workflows
	fmt.Println("\nListing all batch workflows...")
	workflows, err := argo.ListWorkflows(ctx, client, "argo", "batch=true,run=demo", nil)
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}
	fmt.Printf("Found %d batch workflows\n", len(workflows))
}

// Example 12: Error Handling Patterns
// Demonstrates comprehensive error handling for workflow operations
func exampleErrorHandling() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example 1: Handle workflow submission errors
	fmt.Println("=== Example 1: Submission Error Handling ===")
	invalidWf := &v1alpha1.Workflow{} // Invalid workflow
	_, err = argo.SubmitWorkflow(ctx, client, invalidWf, nil)
	if err != nil {
		fmt.Printf("Expected error caught: %v\n\n", err)
	}

	// Example 2: Handle non-existent workflow
	fmt.Println("=== Example 2: Non-existent Workflow ===")
	_, err = argo.GetWorkflowStatus(ctx, client, "argo", "non-existent-workflow", nil)
	if err != nil {
		fmt.Printf("Expected error caught: %v\n\n", err)
	}

	// Example 3: Handle timeout in SubmitAndWait
	fmt.Println("=== Example 3: Timeout Handling ===")
	longRunning := template.NewContainer("long-task", "alpine:3.19",
		template.WithCommand("sleep", "300")) // 5 minutes

	wf, err := builder.NewWorkflowBuilder("timeout-test", "argo",
		builder.WithServiceAccount("default")).
		Add(longRunning).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Submit with very short timeout
	_, err = argo.SubmitAndWait(ctx, client, wf, nil, 5*time.Second)
	if err != nil {
		fmt.Printf("Expected timeout error caught: %v\n\n", err)
	}

	// Example 4: Handle invalid label selectors
	fmt.Println("=== Example 4: Invalid Label Selector ===")
	_, err = argo.ListWorkflows(ctx, client, "argo", "invalid==selector", nil)
	if err != nil {
		fmt.Printf("Expected error caught: %v\n\n", err)
	}

	// Example 5: Graceful handling of delete on non-existent workflow
	fmt.Println("=== Example 5: Delete Non-existent Workflow ===")
	err = argo.DeleteWorkflow(ctx, client, "argo", "non-existent-workflow", nil)
	if err != nil {
		fmt.Printf("Expected error caught: %v\n", err)
	}
}

func main() {
	fmt.Println("Argo Workflow Operations Examples")
	fmt.Println("===================================")
	fmt.Println()
	fmt.Println("Uncomment the example you want to run:")
	fmt.Println()

	// Uncomment one example at a time to run:

	// exampleSubmitWorkflow()
	// exampleSubmitWorkflowWithOTel()
	// exampleSubmitAndWait()
	// exampleSubmitAndWaitWithErrorHandling()
	// exampleGetWorkflowStatus()
	// exampleMonitorWorkflowStatus()
	// exampleListWorkflows()
	// exampleListWorkflowsWithLabels()
	// exampleDeleteWorkflow()
	// exampleCompleteWorkflowLifecycle()
	// exampleBatchOperations()
	// exampleErrorHandling()

	fmt.Println("Please uncomment one of the example functions in main()")
}
