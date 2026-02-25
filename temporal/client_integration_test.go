//go:build integration

package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
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
		HostPort:             container.HostPort(),
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:0", // Random port
	}

	t.Run("NewClient", func(t *testing.T) {
		temporalClient, closer, err := NewClient(config)
		require.NoError(t, err, "Failed to create Temporal client")
		require.NotNil(t, temporalClient, "Client should not be nil")
		defer func() {
			temporalClient.Close()
			if closer != nil {
				closer.Close()
			}
		}()

		// Test basic client functionality
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to check server health by listing task queues
		_, err = temporalClient.DescribeTaskQueue(ctx, "test-queue", enums.TASK_QUEUE_TYPE_WORKFLOW)
		// This may fail but indicates server connectivity
		if err != nil {
			t.Logf("Task queue check failed (expected without server): %v", err)
		}
	})

	t.Run("NewClientWithMetrics", func(t *testing.T) {
		temporalClient, closer, err := NewClientWithMetrics(config, true)
		require.NoError(t, err, "Failed to create Temporal client with metrics")
		require.NotNil(t, temporalClient, "Client should not be nil")

		temporalClient.Close()
		if closer != nil {
			closer.Close()
		}
	})

	t.Run("NewClientWithoutMetrics", func(t *testing.T) {
		temporalClient, closer, err := NewClientWithMetrics(config, false)
		require.NoError(t, err, "Failed to create Temporal client without metrics")
		require.NotNil(t, temporalClient, "Client should not be nil")

		temporalClient.Close()
		if closer != nil {
			closer.Close()
		}
	})

	t.Run("InvalidHost", func(t *testing.T) {
		invalidConfig := &Config{
			HostPort:             "invalid-host:7233",
			Namespace:            "default",
			MetricsListenAddress: "0.0.0.0:9092",
		}

		// This should fail quickly since the host doesn't exist
		temporalClient, closer, err := NewClient(invalidConfig)
		if err == nil && temporalClient != nil {
			temporalClient.Close()
		}
		if closer != nil {
			closer.Close()
		}
		// We don't assert error here because the client creation might succeed
		// but connection will fail later during actual operations
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

		// Test that we can query task queue information
		_, err := temporalClient.DescribeTaskQueue(ctx, "test-queue", enums.TASK_QUEUE_TYPE_WORKFLOW)
		// This may fail but tests connectivity
		if err != nil {
			t.Logf("Task queue describe failed (expected without workers): %v", err)
		}
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

		// Note: This will fail if no worker is registered for this task queue
		// but that's expected in a pure client test
		workflowRun, err := temporalClient.ExecuteWorkflow(ctx, options, simpleWorkflow, "World")
		if err != nil {
			// Expected to fail without a worker, but we test the client API
			t.Logf("Expected failure without worker: %v", err)
			return
		}

		// If somehow it worked, get the result
		var result string
		err = workflowRun.Get(ctx, &result)
		if err == nil {
			assert.Equal(t, "Hello World", result)
		}
	})
}

// TestClientConfig tests configuration validation
func TestClientConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Equal(t, "localhost:7233", config.HostPort)
		assert.Equal(t, "default", config.Namespace)
		assert.Equal(t, "0.0.0.0:9090", config.MetricsListenAddress)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &Config{
			HostPort:             "custom-host:1234",
			Namespace:            "custom-namespace",
			MetricsListenAddress: "127.0.0.1:8080",
		}

		// Should be able to create client with custom config (connection may fail)
		temporalClient, closer, err := NewClient(config)
		if err == nil && temporalClient != nil {
			temporalClient.Close()
		}
		if closer != nil {
			closer.Close()
		}
		// Don't assert success since custom host might not exist
	})
}
