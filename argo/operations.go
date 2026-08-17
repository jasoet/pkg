package argo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jasoet/pkg/v3/otel"
)

// Sentinel errors returned by workflow operations. Callers can match them with errors.Is.
var (
	// ErrWorkflowFailed is returned by SubmitAndWait when the workflow reaches a terminal
	// Failed or Error phase.
	ErrWorkflowFailed = errors.New("argo: workflow failed")

	// ErrWaitTimeout is returned by SubmitAndWait when the wait deadline elapses before the
	// workflow completes. The returned error also wraps context.DeadlineExceeded.
	ErrWaitTimeout = errors.New("argo: timed out waiting for workflow to complete")
)

// defaultPollInterval is how often SubmitAndWait polls the workflow status when no
// WithPollInterval option is supplied.
const defaultPollInterval = 5 * time.Second

// waitOptions holds tunables for SubmitAndWait.
type waitOptions struct {
	pollInterval time.Duration
}

// WaitOption configures SubmitAndWait.
type WaitOption func(*waitOptions)

// WithPollInterval overrides how often SubmitAndWait polls the workflow status.
// Non-positive values are ignored (the default interval is used).
func WithPollInterval(d time.Duration) WaitOption {
	return func(o *waitOptions) {
		if d > 0 {
			o.pollInterval = d
		}
	}
}

// isTransientPollError reports whether a GetWorkflow polling error is transient (worth
// retrying) rather than permanent. Permanent errors — the workflow does not exist, or the
// caller lacks permission — would otherwise spin uselessly until the wait deadline.
func isTransientPollError(err error) bool {
	if err == nil {
		return false
	}
	// A per-call deadline/cancellation is handled by the outer select, but be defensive.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status error: treat as transient so genuinely flaky/network errors
		// keep retrying until the deadline.
		return true
	}
	switch st.Code() {
	case codes.NotFound, codes.PermissionDenied, codes.Unauthenticated,
		codes.InvalidArgument, codes.FailedPrecondition, codes.Unimplemented:
		return false
	default:
		return true
	}
}

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
//	created, err := argo.SubmitWorkflow(ctx, client, wf)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Workflow %s submitted\n", created.Name)
func SubmitWorkflow(ctx context.Context, client apiclient.Client, wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	cfg := otel.ConfigFromContext(ctx)

	// Start span
	var span trace.Span
	if cfg != nil && cfg.TracerProvider != nil {
		tracer := cfg.TracerProvider.Tracer("github.com/jasoet/pkg/v3/argo")
		ctx, span = tracer.Start(ctx, "argo.SubmitWorkflow")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v3/argo", "argo.SubmitWorkflow")
	logger.Info("Submitting workflow",
		otel.F("workflow_generate_name", wf.GenerateName),
		otel.F("namespace", wf.Namespace))

	wfClient := client.NewWorkflowServiceClient()
	created, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
		Namespace: wf.Namespace,
		Workflow:  wf,
	})
	if err != nil {
		logger.Error(err, "Failed to submit workflow",
			otel.F("workflow_generate_name", wf.GenerateName))
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
//	completed, err := argo.SubmitAndWait(ctx, client, wf, 10*time.Minute)
//	if err != nil {
//	    return err
//	}
//	if completed.Status.Phase == v1alpha1.WorkflowSucceeded {
//	    fmt.Println("Workflow completed successfully")
//	}
func SubmitAndWait(ctx context.Context, client apiclient.Client, wf *v1alpha1.Workflow, timeout time.Duration, opts ...WaitOption) (*v1alpha1.Workflow, error) {
	cfg := otel.ConfigFromContext(ctx)

	options := waitOptions{pollInterval: defaultPollInterval}
	for _, opt := range opts {
		opt(&options)
	}

	// Start span for entire operation
	var span trace.Span
	if cfg != nil && cfg.TracerProvider != nil {
		tracer := cfg.TracerProvider.Tracer("github.com/jasoet/pkg/v3/argo")
		ctx, span = tracer.Start(ctx, "argo.SubmitAndWait")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v3/argo", "argo.SubmitAndWait")

	startTime := time.Now()

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf)
	if err != nil {
		return nil, err
	}

	logger.Info("Waiting for workflow completion",
		otel.F("workflow_name", created.Name),
		otel.F("timeout", timeout.String()),
		otel.F("poll_interval", options.pollInterval.String()))

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wfClient := client.NewWorkflowServiceClient()

	// poll performs a single status check. It returns (workflowToReturn, terminal, err):
	//   - terminal true with err nil  => workflow succeeded
	//   - err non-nil                 => terminal failure or permanent poll error
	//   - terminal false, err nil     => still running (or a transient poll error to retry)
	poll := func() (*v1alpha1.Workflow, bool, error) {
		result, gErr := wfClient.GetWorkflow(timeoutCtx, &workflow.WorkflowGetRequest{
			Namespace: created.Namespace,
			Name:      created.Name,
		})
		if gErr != nil {
			if isTransientPollError(gErr) {
				logger.Warn("Transient error getting workflow status, will retry",
					otel.F("workflow_name", created.Name),
					otel.F("error", gErr.Error()))
				return nil, false, nil
			}
			// Permanent error (e.g. NotFound/PermissionDenied): abort instead of spinning
			// until the deadline.
			wErr := fmt.Errorf("failed to get workflow status for %q: %w", created.Name, gErr)
			logger.Error(wErr, "Permanent error getting workflow status; aborting wait",
				otel.F("workflow_name", created.Name))
			return created, false, wErr
		}

		switch result.Status.Phase {
		case v1alpha1.WorkflowSucceeded:
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
			return result, true, nil

		case v1alpha1.WorkflowFailed, v1alpha1.WorkflowError:
			duration := time.Since(startTime)
			wErr := fmt.Errorf("%w: %q (phase: %s): %s", ErrWorkflowFailed, created.Name, result.Status.Phase, result.Status.Message)
			logger.Error(wErr, "Workflow failed",
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
			return result, true, wErr

		default:
			logger.Debug("Workflow still running",
				otel.F("workflow_name", created.Name),
				otel.F("phase", string(result.Status.Phase)))
			return result, false, nil
		}
	}

	// Poll immediately so a workflow that is already terminal is detected without waiting a
	// full interval.
	if result, terminal, pErr := poll(); terminal || pErr != nil {
		return result, pErr
	}

	ticker := time.NewTicker(options.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			cause := timeoutCtx.Err()
			duration := time.Since(startTime)
			if errors.Is(cause, context.DeadlineExceeded) {
				// Wrap both the sentinel and context.DeadlineExceeded so callers can match
				// either with errors.Is.
				wErr := fmt.Errorf("%w: %q after %s: %w", ErrWaitTimeout, created.Name, duration, cause)
				logger.Error(wErr, "Workflow wait timed out",
					otel.F("workflow_name", created.Name),
					otel.F("duration", duration.String()))
				return created, wErr
			}
			// Parent context canceled (not a timeout): label it accurately.
			wErr := fmt.Errorf("waiting for workflow %q canceled after %s: %w", created.Name, duration, cause)
			logger.Error(wErr, "Workflow wait canceled",
				otel.F("workflow_name", created.Name),
				otel.F("duration", duration.String()))
			return created, wErr

		case <-ticker.C:
			if result, terminal, pErr := poll(); terminal || pErr != nil {
				return result, pErr
			}
		}
	}
}

// GetWorkflowStatus retrieves the current status of a workflow.
//
// Example:
//
//	status, err := argo.GetWorkflowStatus(ctx, client, "argo", "my-workflow-abc123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Workflow phase: %s\n", status.Phase)
func GetWorkflowStatus(ctx context.Context, client apiclient.Client, namespace, name string) (*v1alpha1.WorkflowStatus, error) {
	cfg := otel.ConfigFromContext(ctx)

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v3/argo", "argo.GetWorkflowStatus")
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
//	workflows, err := argo.ListWorkflows(ctx, client, "argo", "")
//
//	// List workflows with label
//	workflows, err := argo.ListWorkflows(ctx, client, "argo", "app=myapp")
func ListWorkflows(ctx context.Context, client apiclient.Client, namespace, labelSelector string) ([]v1alpha1.Workflow, error) {
	cfg := otel.ConfigFromContext(ctx)

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v3/argo", "argo.ListWorkflows")
	logger.Debug("Listing workflows",
		otel.F("namespace", namespace),
		otel.F("label_selector", labelSelector))

	wfClient := client.NewWorkflowServiceClient()

	// Follow pagination continue tokens so callers get the full result set rather than a
	// truncated first page.
	var all []v1alpha1.Workflow
	continueToken := ""
	for {
		listOpts := &metav1.ListOptions{Continue: continueToken}
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

		all = append(all, resp.Items...)

		continueToken = resp.Continue
		if continueToken == "" {
			break
		}
	}

	logger.Info("Listed workflows",
		otel.F("namespace", namespace),
		otel.F("count", len(all)))

	return all, nil
}

// DeleteWorkflow deletes a workflow by name.
//
// Example:
//
//	err := argo.DeleteWorkflow(ctx, client, "argo", "my-workflow-abc123")
//	if err != nil {
//	    return err
//	}
func DeleteWorkflow(ctx context.Context, client apiclient.Client, namespace, name string) error {
	cfg := otel.ConfigFromContext(ctx)

	logger := otel.NewLogHelper(ctx, cfg, "github.com/jasoet/pkg/v3/argo", "argo.DeleteWorkflow")
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
