//go:build integration

package temporal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// RetryPolicy alias for temporal retry policy
type RetryPolicy = temporal.RetryPolicy

// Test workflows and activities for integration tests
func SampleWorkflow(ctx workflow.Context, input string) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("SampleWorkflow started", "input", input)

	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute activity
	var result string
	err := workflow.ExecuteActivity(ctx, SampleActivity, input).Get(ctx, &result)
	if err != nil {
		logger.Error("SampleActivity failed", "error", err)
		return "", err
	}

	logger.Info("SampleWorkflow completed", "result", result)
	return result, nil
}

func SampleActivity(ctx context.Context, input string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SampleActivity started", "input", input)

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	result := fmt.Sprintf("Processed: %s", input)
	logger.Info("SampleActivity completed", "result", result)
	return result, nil
}

func FailingTestActivity(ctx context.Context, input string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FailingTestActivity started", "input", input)

	// Always fail for testing error handling
	return "", errors.New("intentional test failure")
}

func LongRunningTestActivity(ctx context.Context, duration time.Duration) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LongRunningTestActivity started", "duration", duration)

	// Send heartbeat every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			logger.Info("LongRunningTestActivity cancelled")
			return "", ctx.Err()
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "heartbeat")
			if time.Since(start) >= duration {
				logger.Info("LongRunningTestActivity completed")
				return "completed", nil
			}
		}
	}
}

func TestWorkerManager(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9095"

	t.Run("CreateWorkerManager", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err, "Failed to create WorkerManager")
		require.NotNil(t, wm, "WorkerManager should not be nil")
		defer wm.Close()

		assert.NotNil(t, wm.GetClient(), "Client should not be nil")
		assert.Empty(t, wm.GetWorkers(), "Workers list should be empty initially")
	})

	t.Run("RegisterWorker", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err)
		defer wm.Close()

		taskQueue := "test-task-queue-register"
		options := worker.Options{}

		w := wm.Register(taskQueue, options)
		require.NotNil(t, w, "Worker should not be nil")

		workers := wm.GetWorkers()
		assert.Len(t, workers, 1, "Should have one worker registered")
		assert.Equal(t, w, workers[0], "Registered worker should match returned worker")
	})

	t.Run("RegisterMultipleWorkers", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err)
		defer wm.Close()

		// Register multiple workers
		w1 := wm.Register("queue-1", worker.Options{})
		w2 := wm.Register("queue-2", worker.Options{})
		w3 := wm.Register("queue-3", worker.Options{})

		workers := wm.GetWorkers()
		assert.Len(t, workers, 3, "Should have three workers registered")
		assert.Contains(t, workers, w1)
		assert.Contains(t, workers, w2)
		assert.Contains(t, workers, w3)
	})
}

func TestWorkerWorkflowExecution(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9096"

	wm, err := NewWorkerManager(config)
	require.NoError(t, err)
	defer wm.Close()

	taskQueue := "test-workflow-execution"

	t.Run("BasicWorkflowExecution", func(t *testing.T) {
		// Register worker
		w := wm.Register(taskQueue, worker.Options{})
		w.RegisterWorkflow(SampleWorkflow)
		w.RegisterActivity(SampleActivity)

		// Start worker in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		workerErr := make(chan error, 1)
		go func() {
			workerErr <- wm.Start(ctx, w)
		}()

		// Give worker time to start
		time.Sleep(2 * time.Second)

		// Execute workflow
		temporalClient := wm.GetClient()
		options := client.StartWorkflowOptions{
			ID:        "test-basic-workflow-" + time.Now().Format("20060102-150405-000"),
			TaskQueue: taskQueue,
		}

		workflowCtx, workflowCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer workflowCancel()

		workflowRun, err := temporalClient.ExecuteWorkflow(workflowCtx, options, SampleWorkflow, "integration-test")
		require.NoError(t, err, "Failed to start workflow")

		// Get result
		var result string
		err = workflowRun.Get(workflowCtx, &result)
		require.NoError(t, err, "Failed to get workflow result")

		assert.Equal(t, "Processed: integration-test", result)

		// Stop worker
		w.Stop()

		// Check if worker stopped gracefully
		select {
		case err := <-workerErr:
			if err != nil {
				t.Logf("Worker stopped with error (may be expected): %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Log("Worker did not stop within timeout")
		}
	})

	t.Run("WorkflowWithFailingActivity", func(t *testing.T) {
		// Register worker with failing activity
		w := wm.Register(taskQueue+"-failing", worker.Options{})
		w.RegisterWorkflow(SampleWorkflow)
		w.RegisterActivity(FailingTestActivity)

		// Start worker in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = wm.Start(ctx, w)
		}()

		// Give worker time to start
		time.Sleep(2 * time.Second)

		// Get client first
		temporalClient := wm.GetClient()

		// Create a workflow that uses the failing activity
		failingWorkflow := func(ctx workflow.Context, input string) (string, error) {
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 10 * time.Second,
				RetryPolicy: &RetryPolicy{
					MaximumAttempts: 1, // Don't retry to speed up test
				},
			}
			ctx = workflow.WithActivityOptions(ctx, ao)

			var result string
			err := workflow.ExecuteActivity(ctx, FailingTestActivity, input).Get(ctx, &result)
			return result, err
		}

		w.RegisterWorkflow(failingWorkflow)

		// Execute workflow with failing activity
		options := client.StartWorkflowOptions{
			ID:        "test-failing-workflow-" + time.Now().Format("20060102-150405-000"),
			TaskQueue: taskQueue + "-failing",
		}

		workflowCtx, workflowCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer workflowCancel()

		workflowRun, err := temporalClient.ExecuteWorkflow(workflowCtx, options, failingWorkflow, "test-input")
		require.NoError(t, err, "Failed to start workflow")

		// Expect workflow to fail
		var result string
		err = workflowRun.Get(workflowCtx, &result)
		assert.Error(t, err, "Workflow should have failed")

		w.Stop()
	})
}

func TestWorkerManagerLifecycle(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9097"

	t.Run("StartAll", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err)
		defer wm.Close()

		// Register multiple workers
		w1 := wm.Register("queue-1", worker.Options{})
		w2 := wm.Register("queue-2", worker.Options{})

		// Register dummy workflows
		dummyWorkflow := func(ctx workflow.Context) error { return nil }
		w1.RegisterWorkflow(dummyWorkflow)
		w2.RegisterWorkflow(dummyWorkflow)

		// Start all workers
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = wm.StartAll(ctx)
		require.NoError(t, err, "Failed to start all workers")

		// Give workers time to start
		time.Sleep(1 * time.Second)

		// Stop all workers via Close
		wm.Close()
	})

	t.Run("StartAllWithNoWorkers", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err)
		defer wm.Close()

		ctx := context.Background()
		err = wm.StartAll(ctx)
		require.NoError(t, err, "Starting with no workers should not fail")
	})

	t.Run("CloseWithoutWorkers", func(t *testing.T) {
		wm, err := NewWorkerManager(config)
		require.NoError(t, err)

		// Should not panic
		wm.Close()
	})
}

func TestWorkerConfiguration(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9098"

	wm, err := NewWorkerManager(config)
	require.NoError(t, err)
	defer wm.Close()

	t.Run("WorkerOptions", func(t *testing.T) {
		options := worker.Options{
			MaxConcurrentActivityExecutionSize:      10,
			MaxConcurrentLocalActivityExecutionSize: 5,
			MaxConcurrentWorkflowTaskExecutionSize:  2,
		}

		w := wm.Register("test-options-queue", options)
		require.NotNil(t, w)

		// Worker is registered but options are internal to Temporal SDK
		// We can only verify the worker was created successfully
		assert.Len(t, wm.GetWorkers(), 1)
	})

	t.Run("MultipleTaskQueues", func(t *testing.T) {
		taskQueues := []string{
			"queue-orders",
			"queue-payments",
			"queue-inventory",
			"queue-notifications",
		}

		for _, queue := range taskQueues {
			w := wm.Register(queue, worker.Options{})
			require.NotNil(t, w, "Failed to register worker for queue: %s", queue)
		}

		assert.Len(t, wm.GetWorkers(), len(taskQueues))
	})
}
