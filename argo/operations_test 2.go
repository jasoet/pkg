package argo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mock workflow service client
type mockWorkflowServiceClient struct {
	createWorkflowFunc func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error)
	getWorkflowFunc    func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error)
	listWorkflowsFunc  func(ctx context.Context, req *workflow.WorkflowListRequest) (*v1alpha1.WorkflowList, error)
	deleteWorkflowFunc func(ctx context.Context, req *workflow.WorkflowDeleteRequest) (*workflow.WorkflowDeleteResponse, error)
}

func (m *mockWorkflowServiceClient) CreateWorkflow(ctx context.Context, req *workflow.WorkflowCreateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	if m.createWorkflowFunc != nil {
		return m.createWorkflowFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) GetWorkflow(ctx context.Context, req *workflow.WorkflowGetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	if m.getWorkflowFunc != nil {
		return m.getWorkflowFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) ListWorkflows(ctx context.Context, req *workflow.WorkflowListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowList, error) {
	if m.listWorkflowsFunc != nil {
		return m.listWorkflowsFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) DeleteWorkflow(ctx context.Context, req *workflow.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflow.WorkflowDeleteResponse, error) {
	if m.deleteWorkflowFunc != nil {
		return m.deleteWorkflowFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) WatchWorkflows(ctx context.Context, req *workflow.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflow.WorkflowService_WatchWorkflowsClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) WatchEvents(ctx context.Context, req *workflow.WatchEventsRequest, _ ...grpc.CallOption) (workflow.WorkflowService_WatchEventsClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) SubmitWorkflow(ctx context.Context, req *workflow.WorkflowSubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) ResubmitWorkflow(ctx context.Context, req *workflow.WorkflowResubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) RetryWorkflow(ctx context.Context, req *workflow.WorkflowRetryRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) ResumeWorkflow(ctx context.Context, req *workflow.WorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) SuspendWorkflow(ctx context.Context, req *workflow.WorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) TerminateWorkflow(ctx context.Context, req *workflow.WorkflowTerminateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) StopWorkflow(ctx context.Context, req *workflow.WorkflowStopRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) SetWorkflow(ctx context.Context, req *workflow.WorkflowSetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) LintWorkflow(ctx context.Context, req *workflow.WorkflowLintRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) PodLogs(ctx context.Context, req *workflow.WorkflowLogRequest, _ ...grpc.CallOption) (workflow.WorkflowService_PodLogsClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowServiceClient) WorkflowLogs(ctx context.Context, req *workflow.WorkflowLogRequest, _ ...grpc.CallOption) (workflow.WorkflowService_WorkflowLogsClient, error) {
	return nil, errors.New("not implemented")
}

// Mock Argo client
type mockArgoClient struct {
	workflowServiceClient workflow.WorkflowServiceClient
}

func (m *mockArgoClient) NewWorkflowServiceClient() (workflow.WorkflowServiceClient, error) {
	return m.workflowServiceClient, nil
}

func (m *mockArgoClient) NewArchivedWorkflowServiceClient() (workflowarchive.ArchivedWorkflowServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewCronWorkflowServiceClient() (apiclient.CronWorkflowServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewInfoServiceClient() (apiclient.InfoServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewEventServiceClient() (apiclient.EventServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewEventSourceServiceClient() (apiclient.EventSourceServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewSensorServiceClient() (apiclient.SensorServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewClusterWorkflowTemplateServiceClient() (apiclient.ClusterWorkflowTemplateServiceClient, error) {
	return nil, errors.New("not implemented")
}

func (m *mockArgoClient) NewWorkflowTemplateServiceClient() (apiclient.WorkflowTemplateServiceClient, error) {
	return nil, errors.New("not implemented")
}

func TestSubmitWorkflow(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("test")

	testWf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    "argo",
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "main",
		},
	}

	t.Run("successful submission", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				created := testWf.DeepCopy()
				created.Name = "test-abc123"
				created.UID = "uid-123"
				return created, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		created, err := SubmitWorkflow(ctx, client, testWf, cfg)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "test-abc123", created.Name)
		assert.Equal(t, "argo", created.Namespace)
	})

	t.Run("submission failure", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				return nil, errors.New("submission failed")
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		created, err := SubmitWorkflow(ctx, client, testWf, cfg)
		require.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "failed to submit workflow")
	})

	t.Run("without otel config", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				created := testWf.DeepCopy()
				created.Name = "test-xyz789"
				return created, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		created, err := SubmitWorkflow(ctx, client, testWf, nil)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "test-xyz789", created.Name)
	})
}

func TestSubmitAndWait(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("test")

	testWf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    "argo",
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "main",
		},
	}

	t.Run("workflow succeeds", func(t *testing.T) {
		callCount := 0
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				created := testWf.DeepCopy()
				created.Name = "test-abc123"
				created.Status.Phase = v1alpha1.WorkflowRunning
				return created, nil
			},
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				callCount++
				result := testWf.DeepCopy()
				result.Name = "test-abc123"
				if callCount >= 2 {
					result.Status.Phase = v1alpha1.WorkflowSucceeded
				} else {
					result.Status.Phase = v1alpha1.WorkflowRunning
				}
				return result, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		completed, err := SubmitAndWait(ctx, client, testWf, cfg, 30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, completed)
		assert.Equal(t, v1alpha1.WorkflowSucceeded, completed.Status.Phase)
	})

	t.Run("workflow fails", func(t *testing.T) {
		callCount := 0
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				created := testWf.DeepCopy()
				created.Name = "test-failed"
				return created, nil
			},
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				callCount++
				result := testWf.DeepCopy()
				result.Name = "test-failed"
				if callCount >= 2 {
					result.Status.Phase = v1alpha1.WorkflowFailed
					result.Status.Message = "workflow failed"
				} else {
					result.Status.Phase = v1alpha1.WorkflowRunning
				}
				return result, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		completed, err := SubmitAndWait(ctx, client, testWf, cfg, 30*time.Second)
		require.Error(t, err)
		require.NotNil(t, completed)
		assert.Equal(t, v1alpha1.WorkflowFailed, completed.Status.Phase)
		assert.Contains(t, err.Error(), "workflow failed")
	})

	t.Run("workflow timeout", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			createWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
				created := testWf.DeepCopy()
				created.Name = "test-timeout"
				return created, nil
			},
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				result := testWf.DeepCopy()
				result.Name = "test-timeout"
				result.Status.Phase = v1alpha1.WorkflowRunning
				return result, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		completed, err := SubmitAndWait(ctx, client, testWf, cfg, 1*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestGetWorkflowStatus(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("test")

	t.Run("successful get", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				return &v1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "argo",
					},
					Status: v1alpha1.WorkflowStatus{
						Phase:   v1alpha1.WorkflowSucceeded,
						Message: "completed",
					},
				}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		status, err := GetWorkflowStatus(ctx, client, "argo", "test-workflow", cfg)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		assert.Equal(t, "completed", status.Message)
	})

	t.Run("workflow not found", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				return nil, errors.New("workflow not found")
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		status, err := GetWorkflowStatus(ctx, client, "argo", "nonexistent", cfg)
		require.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "failed to get workflow")
	})

	t.Run("without otel config", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			getWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
				return &v1alpha1.Workflow{
					Status: v1alpha1.WorkflowStatus{
						Phase: v1alpha1.WorkflowRunning,
					},
				}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		status, err := GetWorkflowStatus(ctx, client, "argo", "test", nil)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, v1alpha1.WorkflowRunning, status.Phase)
	})
}

func TestListWorkflows(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("test")

	t.Run("list all workflows", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			listWorkflowsFunc: func(ctx context.Context, req *workflow.WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
				return &v1alpha1.WorkflowList{
					Items: []v1alpha1.Workflow{
						{ObjectMeta: metav1.ObjectMeta{Name: "wf-1"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "wf-2"}},
					},
				}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		workflows, err := ListWorkflows(ctx, client, "argo", "", cfg)
		require.NoError(t, err)
		require.Len(t, workflows, 2)
		assert.Equal(t, "wf-1", workflows[0].Name)
		assert.Equal(t, "wf-2", workflows[1].Name)
	})

	t.Run("list with label selector", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			listWorkflowsFunc: func(ctx context.Context, req *workflow.WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
				assert.Equal(t, "app=myapp", req.ListOptions.LabelSelector)
				return &v1alpha1.WorkflowList{
					Items: []v1alpha1.Workflow{
						{ObjectMeta: metav1.ObjectMeta{Name: "wf-app"}},
					},
				}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		workflows, err := ListWorkflows(ctx, client, "argo", "app=myapp", cfg)
		require.NoError(t, err)
		require.Len(t, workflows, 1)
	})

	t.Run("list failure", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			listWorkflowsFunc: func(ctx context.Context, req *workflow.WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
				return nil, errors.New("list failed")
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		workflows, err := ListWorkflows(ctx, client, "argo", "", cfg)
		require.Error(t, err)
		assert.Nil(t, workflows)
		assert.Contains(t, err.Error(), "failed to list workflows")
	})
}

func TestDeleteWorkflow(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("test")

	t.Run("successful deletion", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			deleteWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowDeleteRequest) (*workflow.WorkflowDeleteResponse, error) {
				assert.Equal(t, "argo", req.Namespace)
				assert.Equal(t, "test-workflow", req.Name)
				return &workflow.WorkflowDeleteResponse{}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		err := DeleteWorkflow(ctx, client, "argo", "test-workflow", cfg)
		require.NoError(t, err)
	})

	t.Run("deletion failure", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			deleteWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowDeleteRequest) (*workflow.WorkflowDeleteResponse, error) {
				return nil, errors.New("delete failed")
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		err := DeleteWorkflow(ctx, client, "argo", "test-workflow", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete workflow")
	})

	t.Run("without otel config", func(t *testing.T) {
		mockWfClient := &mockWorkflowServiceClient{
			deleteWorkflowFunc: func(ctx context.Context, req *workflow.WorkflowDeleteRequest) (*workflow.WorkflowDeleteResponse, error) {
				return &workflow.WorkflowDeleteResponse{}, nil
			},
		}

		client := &mockArgoClient{workflowServiceClient: mockWfClient}

		err := DeleteWorkflow(ctx, client, "argo", "test-workflow", nil)
		require.NoError(t, err)
	})
}
