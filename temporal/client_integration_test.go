//go:build integration

package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"github.com/jasoet/pkg/v3/temporal/testcontainer"
)

func TestClientIntegration(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container once for all subtests
	container, _, containerCleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer containerCleanup()

	// Create config using container's address
	config := &Config{
		HostPort:  container.HostPort(),
		Namespace: "default",
	}

	t.Run("NewClient", func(t *testing.T) {
		temporalClient, err := NewClient(WithConfig(*config))
		require.NoError(t, err, "Failed to create Temporal client")
		require.NotNil(t, temporalClient, "Client should not be nil")
		defer temporalClient.Close()

		// Test basic client functionality
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// The container is up, so describing a (worker-less) task queue must
		// succeed and return an empty poller set rather than error.
		_, err = temporalClient.DescribeTaskQueue(ctx, "test-queue", enums.TASK_QUEUE_TYPE_WORKFLOW)
		require.NoError(t, err, "DescribeTaskQueue against the running container must succeed")
	})

	t.Run("InvalidHost", func(t *testing.T) {
		invalidConfig := &Config{
			HostPort:  "invalid-host:7233",
			Namespace: "default",
		}

		// NewClient dials eagerly with a health check, so an unresolvable host
		// must fail fast.
		temporalClient, err := NewClient(WithConfig(*invalidConfig))
		if temporalClient != nil {
			temporalClient.Close()
		}
		require.Error(t, err, "dialing an invalid host must return an error")
	})
}

func TestClientOperations(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	t.Run("DescribeTaskQueue", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Querying task queue information against the running container must
		// succeed even with no workers registered.
		_, err := temporalClient.DescribeTaskQueue(ctx, "test-queue", enums.TASK_QUEUE_TYPE_WORKFLOW)
		require.NoError(t, err, "DescribeTaskQueue must succeed against the running container")
	})

	t.Run("WorkflowService", func(t *testing.T) {
		// Test that we can get the workflow service
		workflowService := temporalClient.WorkflowService()
		assert.NotNil(t, workflowService, "WorkflowService should not be nil")
	})

	t.Run("ScheduleClient", func(t *testing.T) {
		// Test that we can get the schedule client
		scheduleClient := temporalClient.ScheduleClient()
		assert.NotNil(t, scheduleClient, "ScheduleClient should not be nil")
	})
}

// TestWorkflowExecution tests basic workflow execution functionality
func TestWorkflowExecution(t *testing.T) {
	ctx := context.Background()

	// Start Temporal container and get client
	_, temporalClient, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{Namespace: "default"},
		testcontainer.Options{Logger: t},
	)
	require.NoError(t, err, "Failed to setup temporal container")
	defer cleanup()

	t.Run("ExecuteSimpleWorkflow", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Simple workflow that just returns a string
		simpleWorkflow := func(ctx context.Context, input string) (string, error) {
			return "Hello " + input, nil
		}

		// Start workflow
		options := client.StartWorkflowOptions{
			ID:        "test-simple-workflow-" + time.Now().Format("20060102-150405"),
			TaskQueue: "test-task-queue",
		}

		// Starting a workflow only enqueues it; it succeeds even when no worker
		// is registered for the task queue. (The run itself will not complete
		// without a worker, so we do not block on Get here.)
		workflowRun, err := temporalClient.ExecuteWorkflow(ctx, options, simpleWorkflow, "World")
		require.NoError(t, err, "starting a workflow must succeed even without a worker")
		require.NotNil(t, workflowRun)
		assert.NotEmpty(t, workflowRun.GetID())
		assert.NotEmpty(t, workflowRun.GetRunID())
	})
}

// TestClientConfig tests configuration validation
func TestClientConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Equal(t, "localhost:7233", config.HostPort)
		assert.Equal(t, "default", config.Namespace)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &Config{
			HostPort:  "custom-host:1234",
			Namespace: "custom-namespace",
		}

		// Should be able to create client with custom config (connection may fail)
		temporalClient, err := NewClient(WithConfig(*config))
		if err == nil && temporalClient != nil {
			temporalClient.Close()
		}
		// Don't assert success since custom host might not exist
	})
}
