//go:build integration

package testcontainer_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// Example workflow for demonstration
func SampleWorkflow(ctx workflow.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s!", name), nil
}

// ExampleSetup demonstrates the simplest way to use testcontainer
func ExampleSetup() {
	ctx := context.Background()

	// Setup container and client with default configuration
	_, client, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{
			Namespace: "default",
		},
		testcontainer.Options{},
	)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Now you can use the client for your tests
	fmt.Println("Client connected to:", client != nil)
	// Output: Client connected to: true
}

// ExampleSetup_withCustomOptions shows how to customize container options
func ExampleSetup_withCustomOptions() {
	ctx := context.Background()

	// Custom options
	opts := testcontainer.Options{
		Image:          "temporalio/temporal:latest",
		StartupTimeout: 90 * time.Second,
		// Logger can be *testing.T or any custom logger
		Logger: nil, // Set to t in actual tests
	}

	_, client, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{
			Namespace: "default",
		},
		opts,
	)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	fmt.Println("Client ready:", client != nil)
	// Output: Client ready: true
}

// ExampleStart demonstrates manual container management
func ExampleStart() {
	ctx := context.Background()

	// Start container manually
	container, err := testcontainer.Start(ctx, testcontainer.Options{})
	if err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get connection details
	hostPort := container.HostPort()
	fmt.Println("Container running at:", hostPort != "")
	// Output: Container running at: true
}

// TestIntegration_FullWorkflow demonstrates a complete integration test
func TestIntegration_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	_, client, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{
			Namespace: "default",
		},
		testcontainer.Options{Logger: t},
	)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Verify we can connect
	if client == nil {
		t.Fatal("Client is nil")
	}

	// In a real test, you would:
	// 1. Start a worker
	// 2. Execute workflows
	// 3. Verify results
	t.Log("Integration test environment ready")
}

// TestIntegration_CustomConfig demonstrates using custom Temporal configuration
func TestIntegration_CustomConfig(t *testing.T) {
	ctx := context.Background()

	// Create custom config
	config := testcontainer.ClientConfig{
		Namespace: "test-namespace",
	}

	container, client, cleanup, err := testcontainer.Setup(
		ctx,
		config,
		testcontainer.Options{Logger: t},
	)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Verify container is running
	if container.HostPort() == "" {
		t.Fatal("Container host port is empty")
	}

	// Verify client is configured correctly
	if client == nil {
		t.Fatal("Client is nil")
	}

	t.Logf("Container running at: %s", container.HostPort())
}

// TestIntegration_ManualSetup demonstrates manual setup for advanced use cases
func TestIntegration_ManualSetup(t *testing.T) {
	ctx := context.Background()

	// Start container with custom options
	container, err := testcontainer.Start(ctx, testcontainer.Options{
		Image:          "temporalio/temporal:latest",
		StartupTimeout: 120 * time.Second,
		Logger:         t,
	})
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	// Create client directly using Temporal SDK
	// You can also use your own client creation logic here
	temporalClient, err := client.Dial(client.Options{
		HostPort:  container.HostPort(),
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer temporalClient.Close()

	t.Logf("Manual setup complete. Container at: %s", container.HostPort())
}
