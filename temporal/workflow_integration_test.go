//go:build integration

package temporal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Test workflows for integration testing
func SimpleTestWorkflow(ctx workflow.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s!", name), nil
}

func LongRunningWorkflow(ctx workflow.Context, duration int) (string, error) {
	err := workflow.Sleep(ctx, time.Duration(duration)*time.Second)
	if err != nil {
		return "", err
	}
	return "completed", nil
}

func SignalTestWorkflow(ctx workflow.Context) (string, error) {
	var signal string
	signalChan := workflow.GetSignalChannel(ctx, "test-signal")
	signalChan.Receive(ctx, &signal)
	return fmt.Sprintf("received: %s", signal), nil
}

func TestWorkflowManagerCreation(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container
	container, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	config := &Config{
		HostPort:             container.HostPort(),
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:0",
	}

	t.Run("NewWorkflowManagerWithClient", func(t *testing.T) {
		wm, err := NewWorkflowManager(temporalClient)
		require.NoError(t, err)
		require.NotNil(t, wm)
		assert.NotNil(t, wm.GetClient())
		// Don't close the client here as it's shared
	})

	t.Run("NewWorkflowManagerWithConfig", func(t *testing.T) {
		wm, err := NewWorkflowManager(config)
		require.NoError(t, err)
		require.NotNil(t, wm)
		assert.NotNil(t, wm.GetClient())
		wm.Close()
	})

	t.Run("NewWorkflowManagerInvalidType", func(t *testing.T) {
		wm, err := NewWorkflowManager("invalid")
		assert.Error(t, err)
		assert.Nil(t, wm)
	})
}

func TestWorkflowManagerListOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	// Create workflow manager
	wm, err := NewWorkflowManager(temporalClient)
	require.NoError(t, err)
	defer func() {
		// Don't close the client as it's shared with temporalClient
	}()

	// Create a worker to execute test workflows
	taskQueue := "test-workflow-list-queue"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterWorkflow(SimpleTestWorkflow)
	w.RegisterWorkflow(LongRunningWorkflow)

	err = w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	t.Run("ListWorkflows", func(t *testing.T) {
		// Start some test workflows
		workflowID1 := fmt.Sprintf("test-list-workflow-1-%d", time.Now().UnixNano())
		workflowID2 := fmt.Sprintf("test-list-workflow-2-%d", time.Now().UnixNano())

		options1 := client.StartWorkflowOptions{
			ID:        workflowID1,
			TaskQueue: taskQueue,
		}
		_, err := temporalClient.ExecuteWorkflow(ctx, options1, SimpleTestWorkflow, "Alice")
		require.NoError(t, err)

		options2 := client.StartWorkflowOptions{
			ID:        workflowID2,
			TaskQueue: taskQueue,
		}
		_, err = temporalClient.ExecuteWorkflow(ctx, options2, SimpleTestWorkflow, "Bob")
		require.NoError(t, err)

		// Wait for workflows to complete
		time.Sleep(3 * time.Second)

		// List workflows
		workflows, err := wm.ListWorkflows(ctx, 100, "")
		require.NoError(t, err)
		assert.NotEmpty(t, workflows)

		// Verify our workflows are in the list
		foundIDs := make(map[string]bool)
		for _, wf := range workflows {
			foundIDs[wf.WorkflowID] = true
		}
		assert.True(t, foundIDs[workflowID1], "Should find workflow 1")
		assert.True(t, foundIDs[workflowID2], "Should find workflow 2")
	})

	t.Run("ListRunningWorkflows", func(t *testing.T) {
		// Start a long-running workflow
		workflowID := fmt.Sprintf("test-running-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}
		_, err := temporalClient.ExecuteWorkflow(ctx, options, LongRunningWorkflow, 30)
		require.NoError(t, err)

		// Wait a bit for workflow to start
		time.Sleep(2 * time.Second)

		// List running workflows
		runningWorkflows, err := wm.ListRunningWorkflows(ctx, 100)
		require.NoError(t, err)

		// Verify our workflow is in the running list
		found := false
		for _, wf := range runningWorkflows {
			if wf.WorkflowID == workflowID {
				found = true
				assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, wf.Status)
				break
			}
		}
		assert.True(t, found, "Should find the running workflow")

		// Cancel the long-running workflow
		err = wm.CancelWorkflow(ctx, workflowID, "")
		require.NoError(t, err)
	})

	t.Run("ListCompletedWorkflows", func(t *testing.T) {
		// Start and wait for a workflow to complete
		workflowID := fmt.Sprintf("test-completed-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}
		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Charlie")
		require.NoError(t, err)

		// Wait for completion
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// List completed workflows
		completedWorkflows, err := wm.ListCompletedWorkflows(ctx, 100)
		require.NoError(t, err)

		// Verify our workflow is in the completed list
		found := false
		for _, wf := range completedWorkflows {
			if wf.WorkflowID == workflowID {
				found = true
				assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, wf.Status)
				break
			}
		}
		assert.True(t, found, "Should find the completed workflow")
	})
}

func TestWorkflowManagerDescribeOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	// Create workflow manager
	wm, err := NewWorkflowManager(temporalClient)
	require.NoError(t, err)

	// Create a worker to execute test workflows
	taskQueue := "test-workflow-describe-queue"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterWorkflow(SimpleTestWorkflow)

	err = w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	t.Run("DescribeWorkflow", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-describe-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "David")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// Describe the workflow
		details, err := wm.DescribeWorkflow(ctx, workflowID, "")
		require.NoError(t, err)
		assert.Equal(t, workflowID, details.WorkflowID)
		assert.Equal(t, "SimpleTestWorkflow", details.WorkflowType)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, details.Status)
		assert.False(t, details.StartTime.IsZero())
		assert.False(t, details.CloseTime.IsZero())
		assert.Greater(t, details.ExecutionTime, time.Duration(0))
	})

	t.Run("GetWorkflowStatus", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-status-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Eve")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// Get workflow status
		status, err := wm.GetWorkflowStatus(ctx, workflowID, "")
		require.NoError(t, err)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, status)
	})

	t.Run("GetWorkflowHistory", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-history-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Frank")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// Get workflow history
		history, err := wm.GetWorkflowHistory(ctx, workflowID, "")
		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.NotNil(t, history.History)
		assert.NotEmpty(t, history.History.Events)
	})
}

func TestWorkflowManagerSearchOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	// Create workflow manager
	wm, err := NewWorkflowManager(temporalClient)
	require.NoError(t, err)

	// Create a worker to execute test workflows
	taskQueue := "test-workflow-search-queue"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterWorkflow(SimpleTestWorkflow)

	err = w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	t.Run("SearchWorkflowsByType", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-search-type-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Grace")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// Search by workflow type
		workflows, err := wm.SearchWorkflowsByType(ctx, "SimpleTestWorkflow", 100)
		require.NoError(t, err)
		assert.NotEmpty(t, workflows)

		// Verify our workflow is in the results
		found := false
		for _, wf := range workflows {
			if wf.WorkflowID == workflowID {
				found = true
				assert.Equal(t, "SimpleTestWorkflow", wf.WorkflowType)
				break
			}
		}
		assert.True(t, found, "Should find the workflow by type")
	})

	t.Run("SearchWorkflowsByID", func(t *testing.T) {
		prefix := fmt.Sprintf("test-search-id-%d", time.Now().UnixNano())
		workflowID := fmt.Sprintf("%s-workflow-1", prefix)
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Henry")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)

		// Wait a bit for indexing
		time.Sleep(2 * time.Second)

		// Search by workflow ID prefix
		workflows, err := wm.SearchWorkflowsByID(ctx, prefix, 100)
		require.NoError(t, err)
		assert.NotEmpty(t, workflows)

		// Verify our workflow is in the results
		found := false
		for _, wf := range workflows {
			if wf.WorkflowID == workflowID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find the workflow by ID prefix")
	})

	t.Run("CountWorkflows", func(t *testing.T) {
		// Count all workflows
		count, err := wm.CountWorkflows(ctx, "")
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))

		// Count running workflows
		runningCount, err := wm.CountWorkflows(ctx, "ExecutionStatus='Running'")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, runningCount, int64(0))
	})
}

func TestWorkflowManagerLifecycleOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	// Create workflow manager
	wm, err := NewWorkflowManager(temporalClient)
	require.NoError(t, err)

	// Create a worker to execute test workflows
	taskQueue := "test-workflow-lifecycle-queue"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterWorkflow(LongRunningWorkflow)
	w.RegisterWorkflow(SignalTestWorkflow)

	err = w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	t.Run("CancelWorkflow", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-cancel-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		_, err := temporalClient.ExecuteWorkflow(ctx, options, LongRunningWorkflow, 60)
		require.NoError(t, err)

		// Wait for workflow to start
		time.Sleep(2 * time.Second)

		// Cancel the workflow
		err = wm.CancelWorkflow(ctx, workflowID, "")
		require.NoError(t, err)

		// Wait for cancellation to take effect
		time.Sleep(2 * time.Second)

		// Verify workflow is canceled
		status, err := wm.GetWorkflowStatus(ctx, workflowID, "")
		require.NoError(t, err)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_CANCELED, status)
	})

	t.Run("TerminateWorkflow", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-terminate-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		_, err := temporalClient.ExecuteWorkflow(ctx, options, LongRunningWorkflow, 60)
		require.NoError(t, err)

		// Wait for workflow to start
		time.Sleep(2 * time.Second)

		// Terminate the workflow
		err = wm.TerminateWorkflow(ctx, workflowID, "", "Test termination")
		require.NoError(t, err)

		// Wait for termination to take effect
		time.Sleep(2 * time.Second)

		// Verify workflow is terminated
		status, err := wm.GetWorkflowStatus(ctx, workflowID, "")
		require.NoError(t, err)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, status)
	})

	t.Run("SignalWorkflow", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-signal-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SignalTestWorkflow)
		require.NoError(t, err)

		// Wait for workflow to start
		time.Sleep(2 * time.Second)

		// Send signal to workflow
		err = wm.SignalWorkflow(ctx, workflowID, "", "test-signal", "Hello from signal!")
		require.NoError(t, err)

		// Wait for workflow to complete
		var result string
		err = run.Get(ctx, &result)
		require.NoError(t, err)
		assert.Equal(t, "received: Hello from signal!", result)
	})
}

func TestWorkflowManagerDashboardOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	// Create workflow manager
	wm, err := NewWorkflowManager(temporalClient)
	require.NoError(t, err)

	// Create a worker to execute test workflows
	taskQueue := "test-workflow-dashboard-queue"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterWorkflow(SimpleTestWorkflow)
	w.RegisterWorkflow(LongRunningWorkflow)

	err = w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	// Start some test workflows
	for i := 0; i < 3; i++ {
		workflowID := fmt.Sprintf("test-dashboard-workflow-%d-%d", i, time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}
		_, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, fmt.Sprintf("User%d", i))
		require.NoError(t, err)
	}

	// Wait for workflows to complete
	time.Sleep(5 * time.Second)

	t.Run("GetDashboardStats", func(t *testing.T) {
		stats, err := wm.GetDashboardStats(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)

		// We should have some completed workflows
		assert.GreaterOrEqual(t, stats.TotalCompleted, int64(3))

		t.Logf("Dashboard Stats - Running: %d, Completed: %d, Failed: %d, Canceled: %d, Terminated: %d",
			stats.TotalRunning, stats.TotalCompleted, stats.TotalFailed, stats.TotalCanceled, stats.TotalTerminated)
	})

	t.Run("GetRecentWorkflows", func(t *testing.T) {
		recentWorkflows, err := wm.GetRecentWorkflows(ctx, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, recentWorkflows)

		// Verify workflows are ordered by start time (most recent first)
		if len(recentWorkflows) > 1 {
			for i := 0; i < len(recentWorkflows)-1; i++ {
				assert.True(t, recentWorkflows[i].StartTime.After(recentWorkflows[i+1].StartTime) ||
					recentWorkflows[i].StartTime.Equal(recentWorkflows[i+1].StartTime),
					"Workflows should be ordered by start time descending")
			}
		}
	})

	t.Run("GetWorkflowResult", func(t *testing.T) {
		workflowID := fmt.Sprintf("test-result-workflow-%d", time.Now().UnixNano())
		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		run, err := temporalClient.ExecuteWorkflow(ctx, options, SimpleTestWorkflow, "Isabella")
		require.NoError(t, err)

		// Wait for workflow to complete
		var expectedResult string
		err = run.Get(ctx, &expectedResult)
		require.NoError(t, err)

		// Get workflow result using WorkflowManager
		var actualResult string
		err = wm.GetWorkflowResult(ctx, workflowID, "", &actualResult)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.Equal(t, "Hello, Isabella!", actualResult)
	})
}
