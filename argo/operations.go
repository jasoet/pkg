package argo

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubmitWorkflow submits a workflow to Argo with OpenTelemetry tracing.
// This is a convenience wrapper around the Argo API client with better error handling
// and automatic observability.
//
// Example:
//
//	wf, err := builder.NewWorkflowBuilder("deploy", "argo").
//	    Add(deployStep).
//	    Build()
//	if err != nil {
//	    return err
//	}
//
//	created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Workflow %s submitted\n", created.Name)
func SubmitWorkflow(ctx context.Context, client apiclient.Client, wf *v1alpha1.Workflow, cfg *otel.Config) (*v1alpha1.Workflow, error) {
	// Start span
	var span trace.Span
	if cfg != nil && cfg.TracerProvider != nil {
		tracer := cfg.TracerProvider.Tracer("github.com/jasoet/pkg/v2/argo")
		ctx, span = tracer.Start(ctx, "argo.SubmitWorkflow")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/argo", "argo.SubmitWorkflow")
	logger.Info("Submitting workflow",
		otel.F("workflow_name", wf.GenerateName),
		otel.F("namespace", wf.Namespace))

	wfClient := client.NewWorkflowServiceClient()
	created, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
		Namespace: wf.Namespace,
		Workflow:  wf,
	})

	if err != nil {
		logger.Error(err, "Failed to submit workflow",
			otel.F("workflow_name", wf.GenerateName))
		return nil, fmt.Errorf("failed to submit workflow: %w", err)
	}

	logger.Info("Workflow submitted successfully",
		otel.F("workflow_name", created.Name),
		otel.F("workflow_uid", created.UID))

	// Add span attributes
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.String("workflow.name", created.Name),
			attribute.String("workflow.namespace", created.Namespace),
			attribute.String("workflow.uid", string(created.UID)),
		)
	}

	return created, nil
}

// SubmitAndWait submits a workflow and waits for it to complete.
// It polls the workflow status at regular intervals and returns when the workflow
// reaches a terminal state (Succeeded, Failed, or Error).
//
// Example:
//
//	wf, err := builder.NewWorkflowBuilder("backup", "argo").
//	    Add(backupStep).
//	    Build()
//	if err != nil {
//	    return err
//	}
//
//	completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 10*time.Minute)
//	if err != nil {
//	    return err
//	}
//	if completed.Status.Phase == v1alpha1.WorkflowSucceeded {
//	    fmt.Println("Workflow completed successfully")
//	}
func SubmitAndWait(ctx context.Context, client apiclient.Client, wf *v1alpha1.Workflow, cfg *otel.Config, timeout time.Duration) (*v1alpha1.Workflow, error) {
	// Start span for entire operation
	var span trace.Span
	if cfg != nil && cfg.TracerProvider != nil {
		tracer := cfg.TracerProvider.Tracer("github.com/jasoet/pkg/v2/argo")
		ctx, span = tracer.Start(ctx, "argo.SubmitAndWait")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/argo", "argo.SubmitAndWait")

	startTime := time.Now()

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Waiting for workflow completion",
		otel.F("workflow_name", created.Name),
		otel.F("timeout", timeout.String()))

	// Wait for completion with polling
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wfClient := client.NewWorkflowServiceClient()

	for {
		select {
		case <-timeoutCtx.Done():
			err := fmt.Errorf("timeout waiting for workflow: %s", created.Name)
			logger.Error(err, "Workflow timed out",
				otel.F("workflow_name", created.Name),
				otel.F("duration", time.Since(startTime).String()))
			return created, err

		case <-ticker.C:
			result, err := wfClient.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
				Namespace: created.Namespace,
				Name:      created.Name,
			})
			if err != nil {
				logger.Warn("Failed to get workflow status", otel.F("error", err.Error()))
				continue
			}

			// Check if workflow is complete
			if result.Status.Phase == v1alpha1.WorkflowSucceeded {
				duration := time.Since(startTime)
				logger.Info("Workflow succeeded",
					otel.F("workflow_name", created.Name),
					otel.F("duration", duration.String()))

				if span != nil && span.IsRecording() {
					span.SetAttributes(
						attribute.String("workflow.status", "succeeded"),
						attribute.Float64("workflow.duration_seconds", duration.Seconds()),
					)
				}

				return result, nil
			}

			if result.Status.Phase == v1alpha1.WorkflowFailed || result.Status.Phase == v1alpha1.WorkflowError {
				duration := time.Since(startTime)
				err := fmt.Errorf("workflow failed with phase: %s, message: %s", result.Status.Phase, result.Status.Message)
				logger.Error(err, "Workflow failed",
					otel.F("workflow_name", created.Name),
					otel.F("phase", string(result.Status.Phase)),
					otel.F("duration", duration.String()))

				if span != nil && span.IsRecording() {
					span.SetAttributes(
						attribute.String("workflow.status", "failed"),
						attribute.String("workflow.phase", string(result.Status.Phase)),
						attribute.Float64("workflow.duration_seconds", duration.Seconds()),
					)
				}

				return result, err
			}

			logger.Debug("Workflow still running",
				otel.F("workflow_name", created.Name),
				otel.F("phase", string(result.Status.Phase)))
		}
	}
}

// GetWorkflowStatus retrieves the current status of a workflow.
//
// Example:
//
//	status, err := argo.GetWorkflowStatus(ctx, client, "argo", "my-workflow-abc123", otelConfig)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Workflow phase: %s\n", status.Phase)
func GetWorkflowStatus(ctx context.Context, client apiclient.Client, namespace, name string, cfg *otel.Config) (*v1alpha1.WorkflowStatus, error) {
	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/argo", "argo.GetWorkflowStatus")
	logger.Debug("Getting workflow status",
		otel.F("namespace", namespace),
		otel.F("name", name))

	wfClient := client.NewWorkflowServiceClient()
	wf, err := wfClient.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
		Namespace: namespace,
		Name:      name,
	})

	if err != nil {
		logger.Error(err, "Failed to get workflow",
			otel.F("namespace", namespace),
			otel.F("name", name))
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	logger.Debug("Retrieved workflow status",
		otel.F("name", name),
		otel.F("phase", string(wf.Status.Phase)))

	return &wf.Status, nil
}

// ListWorkflows lists workflows in a namespace with optional label selector.
//
// Example:
//
//	// List all workflows
//	workflows, err := argo.ListWorkflows(ctx, client, "argo", "", otelConfig)
//
//	// List workflows with label
//	workflows, err := argo.ListWorkflows(ctx, client, "argo", "app=myapp", otelConfig)
func ListWorkflows(ctx context.Context, client apiclient.Client, namespace, labelSelector string, cfg *otel.Config) ([]v1alpha1.Workflow, error) {
	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/argo", "argo.ListWorkflows")
	logger.Debug("Listing workflows",
		otel.F("namespace", namespace),
		otel.F("label_selector", labelSelector))

	wfClient := client.NewWorkflowServiceClient()

	listOpts := &metav1.ListOptions{}
	if labelSelector != "" {
		listOpts.LabelSelector = labelSelector
	}

	resp, err := wfClient.ListWorkflows(ctx, &workflow.WorkflowListRequest{
		Namespace:   namespace,
		ListOptions: listOpts,
	})

	if err != nil {
		logger.Error(err, "Failed to list workflows",
			otel.F("namespace", namespace))
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	logger.Info("Listed workflows",
		otel.F("namespace", namespace),
		otel.F("count", len(resp.Items)))

	return resp.Items, nil
}

// DeleteWorkflow deletes a workflow by name.
//
// Example:
//
//	err := argo.DeleteWorkflow(ctx, client, "argo", "my-workflow-abc123", otelConfig)
//	if err != nil {
//	    return err
//	}
func DeleteWorkflow(ctx context.Context, client apiclient.Client, namespace, name string, cfg *otel.Config) error {
	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v2/argo", "argo.DeleteWorkflow")
	logger.Info("Deleting workflow",
		otel.F("namespace", namespace),
		otel.F("name", name))

	wfClient := client.NewWorkflowServiceClient()
	_, err := wfClient.DeleteWorkflow(ctx, &workflow.WorkflowDeleteRequest{
		Namespace: namespace,
		Name:      name,
	})

	if err != nil {
		logger.Error(err, "Failed to delete workflow",
			otel.F("namespace", namespace),
			otel.F("name", name))
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	logger.Info("Workflow deleted successfully",
		otel.F("namespace", namespace),
		otel.F("name", name))

	return nil
}
