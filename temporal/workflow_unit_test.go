package temporal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateQueryParam(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		wantErr bool
	}{
		{name: "SimpleAlphanumeric", param: "Running"},
		{name: "Lowercase", param: "completed"},
		{name: "Digits", param: "12345"},
		{name: "HyphensUnderscoresDots", param: "my-workflow_type.v2"},
		{name: "Mixed", param: "OrderFlow-2024.01_abc"},

		{name: "Empty", param: "", wantErr: true},
		{name: "SingleQuote", param: "' OR '1'='1", wantErr: true},
		{name: "DoubleQuote", param: `Running" OR 1=1`, wantErr: true},
		{name: "EqualsOperator", param: "ExecutionStatus=Running", wantErr: true},
		{name: "Space", param: "Running Completed", wantErr: true},
		{name: "Semicolon", param: "Running;DROP", wantErr: true},
		{name: "Parentheses", param: "Running)", wantErr: true},
		{name: "Wildcard", param: "workflow*", wantErr: true},
		{name: "Slash", param: "a/b", wantErr: true},
		{name: "Unicode", param: "workflöw", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryParam(tt.param)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid query parameter")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueryWorkflow(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		c := mocks.NewClient(t)
		value := mocks.NewEncodedValue(t)
		c.On("QueryWorkflow", mock.Anything, "wf-1", "run-1", "state").Return(value, nil)

		wm, err := NewWorkflowManager(c)
		require.NoError(t, err)

		result, err := wm.QueryWorkflow(ctx, "wf-1", "run-1", "state")
		require.NoError(t, err)
		assert.Same(t, value, result, "the encoded value must be returned as-is")

		// The returned value must be decodable through converter.EncodedValue.
		value.On("Get", mock.Anything).Return(nil)
		encoded, ok := result.(converter.EncodedValue)
		require.True(t, ok)
		var decoded string
		assert.NoError(t, encoded.Get(&decoded))
	})

	t.Run("SuccessWithArgs", func(t *testing.T) {
		c := mocks.NewClient(t)
		value := mocks.NewEncodedValue(t)
		c.On("QueryWorkflow", mock.Anything, "wf-2", "run-2", "progress", "arg1", 42).
			Return(value, nil)

		wm, err := NewWorkflowManager(c)
		require.NoError(t, err)

		result, err := wm.QueryWorkflow(ctx, "wf-2", "run-2", "progress", "arg1", 42)
		require.NoError(t, err)
		assert.Same(t, value, result)
	})

	t.Run("Error", func(t *testing.T) {
		c := mocks.NewClient(t)
		c.On("QueryWorkflow", mock.Anything, "wf-missing", "run-1", "state").
			Return(nil, errors.New("workflow not found"))

		wm, err := NewWorkflowManager(c)
		require.NoError(t, err)

		result, err := wm.QueryWorkflow(ctx, "wf-missing", "run-1", "state")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), `query workflow "wf-missing" type "state"`)
		assert.Contains(t, err.Error(), "workflow not found")
	})
}

func TestListFailedWorkflows(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessMapsExecutions", func(t *testing.T) {
		c := mocks.NewClient(t)
		wm, err := NewWorkflowManagerWithNamespace(c, "test-ns")
		require.NoError(t, err)

		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		closed := start.Add(5 * time.Second)

		var captured *workflowservice.ListWorkflowExecutionsRequest
		c.On("ListWorkflow", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*workflowservice.ListWorkflowExecutionsRequest)
			}).
			Return(&workflowservice.ListWorkflowExecutionsResponse{
				Executions: []*workflowpb.WorkflowExecutionInfo{
					{
						Execution:     &commonpb.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
						Type:          &commonpb.WorkflowType{Name: "OrderWorkflow"},
						Status:        enums.WORKFLOW_EXECUTION_STATUS_FAILED,
						StartTime:     timestamppb.New(start),
						CloseTime:     timestamppb.New(closed),
						HistoryLength: 42,
					},
					{
						// Still running: no CloseTime set.
						Execution:     &commonpb.WorkflowExecution{WorkflowId: "wf-2", RunId: "run-2"},
						Type:          &commonpb.WorkflowType{Name: "EmailWorkflow"},
						Status:        enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
						StartTime:     timestamppb.New(start),
						HistoryLength: 7,
					},
				},
			}, nil)

		workflows, err := wm.ListFailedWorkflows(ctx, 10)
		require.NoError(t, err)
		require.Len(t, workflows, 2)

		require.NotNil(t, captured)
		assert.Equal(t, "test-ns", captured.Namespace, "explicit namespace must be sent to the high-level ListWorkflow")
		assert.Equal(t, "ExecutionStatus='Failed'", captured.Query)
		assert.Equal(t, int32(10), captured.PageSize)

		first := workflows[0]
		assert.Equal(t, "wf-1", first.WorkflowID)
		assert.Equal(t, "run-1", first.RunID)
		assert.Equal(t, "OrderWorkflow", first.WorkflowType)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_FAILED, first.Status)
		assert.Equal(t, start, first.StartTime)
		assert.Equal(t, closed, first.CloseTime)
		assert.Equal(t, 5*time.Second, first.ExecutionTime)
		assert.Equal(t, int64(42), first.HistoryLength)

		second := workflows[1]
		assert.Equal(t, "wf-2", second.WorkflowID)
		assert.True(t, second.CloseTime.IsZero(), "CloseTime must be zero when unset")
		assert.Zero(t, second.ExecutionTime, "ExecutionTime must be zero without CloseTime")
	})

	t.Run("SuccessEmptyResult", func(t *testing.T) {
		c := mocks.NewClient(t)
		wm, err := NewWorkflowManagerWithNamespace(c, "test-ns")
		require.NoError(t, err)

		c.On("ListWorkflow", mock.Anything, mock.Anything).
			Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)

		workflows, err := wm.ListFailedWorkflows(ctx, 10)
		require.NoError(t, err)
		assert.NotNil(t, workflows)
		assert.Empty(t, workflows)
	})

	t.Run("Error", func(t *testing.T) {
		c := mocks.NewClient(t)
		wm, err := NewWorkflowManagerWithNamespace(c, "test-ns")
		require.NoError(t, err)

		c.On("ListWorkflow", mock.Anything, mock.Anything).
			Return(nil, errors.New("visibility store unavailable"))

		workflows, err := wm.ListFailedWorkflows(ctx, 10)
		require.Error(t, err)
		assert.Nil(t, workflows)
		assert.Contains(t, err.Error(), "list workflow executions")
		assert.Contains(t, err.Error(), "visibility store unavailable")
	})
}

// TestWorkflowManagerNamespaceAgreement pins the split-brain fix: the default
// constructor must NOT hardcode "default" into visibility queries — it must
// leave the request namespace empty so the SDK client fills in its own
// configured namespace, keeping List/Count in agreement with the client-method
// calls (Describe/Cancel/…). The explicit constructor forwards its namespace.
func TestWorkflowManagerNamespaceAgreement(t *testing.T) {
	ctx := context.Background()

	captureList := func(t *testing.T, c *mocks.Client) func() *workflowservice.ListWorkflowExecutionsRequest {
		t.Helper()
		var captured *workflowservice.ListWorkflowExecutionsRequest
		c.On("ListWorkflow", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*workflowservice.ListWorkflowExecutionsRequest)
			}).
			Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)
		return func() *workflowservice.ListWorkflowExecutionsRequest { return captured }
	}

	t.Run("DefaultConstructorLeavesNamespaceEmpty", func(t *testing.T) {
		c := mocks.NewClient(t)
		get := captureList(t, c)

		wm, err := NewWorkflowManager(c)
		require.NoError(t, err)

		_, err = wm.ListWorkflows(ctx, 5, "")
		require.NoError(t, err)
		require.NotNil(t, get())
		assert.Empty(t, get().Namespace,
			"NewWorkflowManager must not hardcode a namespace; the SDK client fills in its own")
	})

	t.Run("ExplicitConstructorForwardsNamespace", func(t *testing.T) {
		c := mocks.NewClient(t)
		get := captureList(t, c)

		wm, err := NewWorkflowManagerWithNamespace(c, "production")
		require.NoError(t, err)

		_, err = wm.ListWorkflows(ctx, 5, "")
		require.NoError(t, err)
		require.NotNil(t, get())
		assert.Equal(t, "production", get().Namespace)
	})

	t.Run("CountUsesSameNamespaceRule", func(t *testing.T) {
		c := mocks.NewClient(t)
		var captured *workflowservice.CountWorkflowExecutionsRequest
		c.On("CountWorkflow", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*workflowservice.CountWorkflowExecutionsRequest)
			}).
			Return(&workflowservice.CountWorkflowExecutionsResponse{Count: 3}, nil)

		wm, err := NewWorkflowManager(c)
		require.NoError(t, err)

		count, err := wm.CountWorkflows(ctx, "ExecutionStatus='Running'")
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
		require.NotNil(t, captured)
		assert.Empty(t, captured.Namespace, "default manager must leave Count namespace empty")
	})
}
