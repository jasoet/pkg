package temporal

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// WorkflowManager provides workflow query and management operations
type WorkflowManager struct {
	client        client.Client
	ownsClient    bool
	metricsCloser io.Closer
	namespace     string
}

// WorkflowFilter defines criteria for filtering workflows
type WorkflowFilter struct {
	WorkflowType   string
	WorkflowID     string
	Status         []enums.WorkflowExecutionStatus
	StartTimeRange *TimeRange
	Query          string // Advanced visibility query
}

// TimeRange defines a time range for filtering
type TimeRange struct {
	StartTime time.Time
	EndTime   time.Time
}

// WorkflowDetails contains detailed information about a workflow execution
type WorkflowDetails struct {
	WorkflowID    string
	RunID         string
	WorkflowType  string
	Status        enums.WorkflowExecutionStatus
	StartTime     time.Time
	CloseTime     time.Time
	ExecutionTime time.Duration
	HistoryLength int64
}

// DashboardStats provides aggregated workflow statistics
type DashboardStats struct {
	TotalRunning    int64
	TotalCompleted  int64
	TotalFailed     int64
	TotalCanceled   int64
	TotalTerminated int64
	AverageDuration time.Duration
}

// safeIdentifier validates that a string is safe for use in visibility queries.
// It allows only alphanumeric characters, hyphens, underscores, and dots.
var safeIdentifier = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validateQueryParam validates a string is safe for use in visibility queries.
func validateQueryParam(param string) error {
	if !safeIdentifier.MatchString(param) {
		return fmt.Errorf("invalid query parameter %q: must contain only alphanumeric characters, hyphens, underscores, and dots", param)
	}
	return nil
}

// NewWorkflowManager creates a new WorkflowManager instance
// Accepts either a client.Client or *Config
func NewWorkflowManager(clientOrConfig interface{}) (*WorkflowManager, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.NewWorkflowManager")

	var temporalClient client.Client
	var ownsClient bool
	var metricsCloser io.Closer
	var namespace string

	switch v := clientOrConfig.(type) {
	case client.Client:
		// If passed a client directly, use it (caller retains ownership)
		temporalClient = v
		ownsClient = false
		namespace = "default" // Default namespace when client is provided
		logger.Debug("Using provided Temporal client for Workflow Manager")
	case *Config:
		// If passed a config, create a new client (we own it)
		logger.Debug("Creating new Workflow Manager with config",
			otel.F("hostPort", v.HostPort),
			otel.F("namespace", v.Namespace))

		namespace = v.Namespace
		var err error
		temporalClient, metricsCloser, err = NewClient(v)
		if err != nil {
			logger.Error(err, "Failed to create Temporal client for Workflow Manager")
			return nil, err
		}
		ownsClient = true
	default:
		logger.Error(nil, "Invalid argument type for NewWorkflowManager")
		return nil, fmt.Errorf("invalid argument type: expected client.Client or *Config")
	}

	logger.Debug("Workflow Manager created successfully")
	return &WorkflowManager{
		client:        temporalClient,
		ownsClient:    ownsClient,
		metricsCloser: metricsCloser,
		namespace:     namespace,
	}, nil
}

// Close closes the Workflow Manager and its client if it was created by the manager
func (wm *WorkflowManager) Close() {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.Close")

	logger.Debug("Closing Workflow Manager")

	if wm.ownsClient && wm.client != nil {
		logger.Debug("Closing Temporal client")
		wm.client.Close()
	}

	if wm.metricsCloser != nil {
		wm.metricsCloser.Close()
	}

	logger.Debug("Workflow Manager closed")
}

// GetClient returns the underlying Temporal client
func (wm *WorkflowManager) GetClient() client.Client {
	return wm.client
}

// ListWorkflows lists workflows with pagination and optional query filter
func (wm *WorkflowManager) ListWorkflows(ctx context.Context, pageSize int, query string) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.ListWorkflows")

	logger.Debug("Listing workflows",
		otel.F("pageSize", pageSize),
		otel.F("query", query))

	request := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: wm.namespace,
		PageSize:  int32(pageSize),
		Query:     query,
	}

	response, err := wm.client.WorkflowService().ListWorkflowExecutions(ctx, request)
	if err != nil {
		logger.Error(err, "Failed to list workflow executions")
		return nil, err
	}

	workflows := make([]*WorkflowDetails, 0, len(response.Executions))
	for _, exec := range response.Executions {
		details := &WorkflowDetails{
			WorkflowID:    exec.Execution.WorkflowId,
			RunID:         exec.Execution.RunId,
			WorkflowType:  exec.Type.Name,
			Status:        exec.Status,
			StartTime:     exec.StartTime.AsTime(),
			HistoryLength: exec.HistoryLength,
		}

		if exec.CloseTime != nil {
			details.CloseTime = exec.CloseTime.AsTime()
			details.ExecutionTime = details.CloseTime.Sub(details.StartTime)
		}

		workflows = append(workflows, details)
	}

	logger.Debug("Workflows listed successfully", otel.F("count", len(workflows)))
	return workflows, nil
}

// DescribeWorkflow retrieves detailed information about a specific workflow execution
func (wm *WorkflowManager) DescribeWorkflow(ctx context.Context, workflowID, runID string) (*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.DescribeWorkflow")

	logger.Debug("Describing workflow",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID))

	response, err := wm.client.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		logger.Error(err, "Failed to describe workflow execution",
			otel.F("workflowID", workflowID),
			otel.F("runID", runID))
		return nil, err
	}

	details := &WorkflowDetails{
		WorkflowID:    response.WorkflowExecutionInfo.Execution.WorkflowId,
		RunID:         response.WorkflowExecutionInfo.Execution.RunId,
		WorkflowType:  response.WorkflowExecutionInfo.Type.Name,
		Status:        response.WorkflowExecutionInfo.Status,
		StartTime:     response.WorkflowExecutionInfo.StartTime.AsTime(),
		HistoryLength: response.WorkflowExecutionInfo.HistoryLength,
	}

	if response.WorkflowExecutionInfo.CloseTime != nil {
		details.CloseTime = response.WorkflowExecutionInfo.CloseTime.AsTime()
		details.ExecutionTime = details.CloseTime.Sub(details.StartTime)
	}

	logger.Debug("Workflow described successfully",
		otel.F("workflowID", workflowID),
		otel.F("status", details.Status))
	return details, nil
}

// GetWorkflowStatus returns the current status of a workflow execution
func (wm *WorkflowManager) GetWorkflowStatus(ctx context.Context, workflowID, runID string) (enums.WorkflowExecutionStatus, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.GetWorkflowStatus")

	logger.Debug("Getting workflow status",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID))

	details, err := wm.DescribeWorkflow(ctx, workflowID, runID)
	if err != nil {
		logger.Error(err, "Failed to get workflow status",
			otel.F("workflowID", workflowID))
		return enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED, err
	}

	logger.Debug("Workflow status retrieved",
		otel.F("workflowID", workflowID),
		otel.F("status", details.Status))
	return details.Status, nil
}

// GetWorkflowHistory retrieves the event history of a workflow execution
func (wm *WorkflowManager) GetWorkflowHistory(ctx context.Context, workflowID, runID string) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.GetWorkflowHistory")

	logger.Debug("Getting workflow history",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID))

	request := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: wm.namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	}

	response, err := wm.client.WorkflowService().GetWorkflowExecutionHistory(ctx, request)
	if err != nil {
		logger.Error(err, "Failed to get workflow history",
			otel.F("workflowID", workflowID))
		return nil, err
	}

	logger.Debug("Workflow history retrieved successfully",
		otel.F("workflowID", workflowID),
		otel.F("eventCount", len(response.History.Events)))
	return response, nil
}

// CancelWorkflow cancels a running workflow execution
func (wm *WorkflowManager) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.CancelWorkflow")

	logger.Debug("Canceling workflow",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID))

	err := wm.client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		logger.Error(err, "Failed to cancel workflow",
			otel.F("workflowID", workflowID))
		return err
	}

	logger.Debug("Workflow canceled successfully",
		otel.F("workflowID", workflowID))
	return nil
}

// TerminateWorkflow terminates a workflow execution with a reason
func (wm *WorkflowManager) TerminateWorkflow(ctx context.Context, workflowID, runID, reason string) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.TerminateWorkflow")

	logger.Debug("Terminating workflow",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID),
		otel.F("reason", reason))

	err := wm.client.TerminateWorkflow(ctx, workflowID, runID, reason)
	if err != nil {
		logger.Error(err, "Failed to terminate workflow",
			otel.F("workflowID", workflowID))
		return err
	}

	logger.Debug("Workflow terminated successfully",
		otel.F("workflowID", workflowID))
	return nil
}

// SignalWorkflow sends a signal to a running workflow
func (wm *WorkflowManager) SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.SignalWorkflow")

	logger.Debug("Signaling workflow",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID),
		otel.F("signalName", signalName))

	err := wm.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)
	if err != nil {
		logger.Error(err, "Failed to signal workflow",
			otel.F("workflowID", workflowID),
			otel.F("signalName", signalName))
		return err
	}

	logger.Debug("Workflow signaled successfully",
		otel.F("workflowID", workflowID),
		otel.F("signalName", signalName))
	return nil
}

// QueryWorkflow queries a running workflow for custom data
func (wm *WorkflowManager) QueryWorkflow(ctx context.Context, workflowID, runID, queryType string, args ...interface{}) (interface{}, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.QueryWorkflow")

	logger.Debug("Querying workflow",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID),
		otel.F("queryType", queryType))

	result, err := wm.client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	if err != nil {
		logger.Error(err, "Failed to query workflow",
			otel.F("workflowID", workflowID),
			otel.F("queryType", queryType))
		return nil, err
	}

	logger.Debug("Workflow queried successfully",
		otel.F("workflowID", workflowID),
		otel.F("queryType", queryType))
	return result, nil
}

// ListWorkflowsByStatus lists workflows filtered by execution status
func (wm *WorkflowManager) ListWorkflowsByStatus(ctx context.Context, status enums.WorkflowExecutionStatus, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.ListWorkflowsByStatus")

	logger.Debug("Listing workflows by status",
		otel.F("status", status.String()),
		otel.F("pageSize", pageSize))

	statusStr := status.String()
	if err := validateQueryParam(statusStr); err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}
	query := fmt.Sprintf("ExecutionStatus='%s'", statusStr)
	return wm.ListWorkflows(ctx, pageSize, query)
}

// ListRunningWorkflows returns all currently running workflows
func (wm *WorkflowManager) ListRunningWorkflows(ctx context.Context, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.ListRunningWorkflows")

	logger.Debug("Listing running workflows", otel.F("pageSize", pageSize))
	return wm.ListWorkflowsByStatus(ctx, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, pageSize)
}

// ListCompletedWorkflows returns completed workflows
func (wm *WorkflowManager) ListCompletedWorkflows(ctx context.Context, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.ListCompletedWorkflows")

	logger.Debug("Listing completed workflows", otel.F("pageSize", pageSize))
	return wm.ListWorkflowsByStatus(ctx, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, pageSize)
}

// ListFailedWorkflows returns failed workflows
func (wm *WorkflowManager) ListFailedWorkflows(ctx context.Context, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.ListFailedWorkflows")

	logger.Debug("Listing failed workflows", otel.F("pageSize", pageSize))
	return wm.ListWorkflowsByStatus(ctx, enums.WORKFLOW_EXECUTION_STATUS_FAILED, pageSize)
}

// SearchWorkflowsByType searches workflows by workflow type name
func (wm *WorkflowManager) SearchWorkflowsByType(ctx context.Context, workflowType string, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.SearchWorkflowsByType")

	logger.Debug("Searching workflows by type",
		otel.F("workflowType", workflowType),
		otel.F("pageSize", pageSize))

	if err := validateQueryParam(workflowType); err != nil {
		return nil, fmt.Errorf("invalid workflow type: %w", err)
	}
	query := fmt.Sprintf("WorkflowType='%s'", workflowType)
	return wm.ListWorkflows(ctx, pageSize, query)
}

// SearchWorkflowsByID searches for workflows matching a workflow ID pattern
func (wm *WorkflowManager) SearchWorkflowsByID(ctx context.Context, workflowIDPrefix string, pageSize int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.SearchWorkflowsByID")

	logger.Debug("Searching workflows by ID",
		otel.F("workflowIDPrefix", workflowIDPrefix),
		otel.F("pageSize", pageSize))

	if err := validateQueryParam(workflowIDPrefix); err != nil {
		return nil, fmt.Errorf("invalid workflow ID prefix: %w", err)
	}
	query := fmt.Sprintf("WorkflowId STARTS_WITH '%s'", workflowIDPrefix)
	return wm.ListWorkflows(ctx, pageSize, query)
}

// CountWorkflows counts workflows matching a query
func (wm *WorkflowManager) CountWorkflows(ctx context.Context, query string) (int64, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.CountWorkflows")

	logger.Debug("Counting workflows", otel.F("query", query))

	request := &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: wm.namespace,
		Query:     query,
	}

	response, err := wm.client.WorkflowService().CountWorkflowExecutions(ctx, request)
	if err != nil {
		logger.Error(err, "Failed to count workflow executions")
		return 0, err
	}

	logger.Debug("Workflows counted successfully", otel.F("count", response.Count))
	return response.Count, nil
}

// GetDashboardStats retrieves aggregated statistics for all workflows
func (wm *WorkflowManager) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.GetDashboardStats")

	logger.Debug("Getting dashboard statistics")

	stats := &DashboardStats{}

	// Count workflows by status
	runningCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Running'")
	if err != nil {
		logger.Error(err, "Failed to count running workflows")
		return nil, err
	}
	stats.TotalRunning = runningCount

	completedCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Completed'")
	if err != nil {
		logger.Error(err, "Failed to count completed workflows")
		return nil, err
	}
	stats.TotalCompleted = completedCount

	failedCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Failed'")
	if err != nil {
		logger.Error(err, "Failed to count failed workflows")
		return nil, err
	}
	stats.TotalFailed = failedCount

	canceledCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Canceled'")
	if err != nil {
		logger.Error(err, "Failed to count canceled workflows")
		return nil, err
	}
	stats.TotalCanceled = canceledCount

	terminatedCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Terminated'")
	if err != nil {
		logger.Error(err, "Failed to count terminated workflows")
		return nil, err
	}
	stats.TotalTerminated = terminatedCount

	// Calculate average duration from completed workflows
	completedWorkflows, err := wm.ListCompletedWorkflows(ctx, 100)
	if err != nil {
		logger.Error(err, "Failed to list completed workflows for duration calculation")
		// Don't fail, just skip average duration
	} else if len(completedWorkflows) > 0 {
		var totalDuration time.Duration
		for _, wf := range completedWorkflows {
			totalDuration += wf.ExecutionTime
		}
		stats.AverageDuration = totalDuration / time.Duration(len(completedWorkflows))
	}

	logger.Debug("Dashboard statistics retrieved successfully",
		otel.F("running", stats.TotalRunning),
		otel.F("completed", stats.TotalCompleted),
		otel.F("failed", stats.TotalFailed))
	return stats, nil
}

// GetRecentWorkflows retrieves the most recent workflow executions
func (wm *WorkflowManager) GetRecentWorkflows(ctx context.Context, limit int) ([]*WorkflowDetails, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.GetRecentWorkflows")

	logger.Debug("Getting recent workflows", otel.F("limit", limit))

	workflows, err := wm.ListWorkflows(ctx, limit, "")
	if err != nil {
		return nil, err
	}

	// Sort by StartTime descending (most recent first)
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].StartTime.After(workflows[j].StartTime)
	})

	return workflows, nil
}

// GetWorkflowResult retrieves the result of a completed workflow
func (wm *WorkflowManager) GetWorkflowResult(ctx context.Context, workflowID, runID string, valuePtr interface{}) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkflowManager.GetWorkflowResult")

	logger.Debug("Getting workflow result",
		otel.F("workflowID", workflowID),
		otel.F("runID", runID))

	run := wm.client.GetWorkflow(ctx, workflowID, runID)
	err := run.Get(ctx, valuePtr)
	if err != nil {
		logger.Error(err, "Failed to get workflow result",
			otel.F("workflowID", workflowID))
		return err
	}

	logger.Debug("Workflow result retrieved successfully",
		otel.F("workflowID", workflowID))
	return nil
}
