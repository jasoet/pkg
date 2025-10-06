//go:build example

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/logging"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// Initialize logging
	logging.Initialize("argo-example", false)
	log.Info().Msg("Starting Argo Workflows client example")

	// Create context
	ctx := context.Background()

	// Run examples
	if err := runExamples(ctx); err != nil {
		log.Fatal().Err(err).Msg("Example failed")
	}

	log.Info().Msg("Examples completed successfully")
}

func runExamples(ctx context.Context) error {
	// Example 1: Default configuration (uses ~/.kube/config)
	if err := example1DefaultConfig(ctx); err != nil {
		return fmt.Errorf("example1 failed: %w", err)
	}

	// Example 2: Functional options
	if err := example2FunctionalOptions(ctx); err != nil {
		return fmt.Errorf("example2 failed: %w", err)
	}

	// Example 3: In-cluster configuration
	// Commented out as it requires running inside a Kubernetes cluster
	// if err := example3InCluster(ctx); err != nil {
	//     return fmt.Errorf("example3 failed: %w", err)
	// }

	// Example 4: Argo Server mode
	// Commented out as it requires an Argo Server
	// if err := example4ArgoServer(ctx); err != nil {
	//     return fmt.Errorf("example4 failed: %w", err)
	// }

	// Example 5: With OpenTelemetry
	if err := example5WithOTel(ctx); err != nil {
		return fmt.Errorf("example5 failed: %w", err)
	}

	return nil
}

// Example 1: Using default configuration
func example1DefaultConfig(ctx context.Context) error {
	log.Info().Msg("=== Example 1: Default Configuration ===")

	// Create client with default config (uses ~/.kube/config)
	ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
	if err != nil {
		// This is expected to fail if no kubeconfig is available
		log.Warn().Err(err).Msg("Failed to create default client (expected if no kubeconfig)")
		return nil
	}

	// List workflows
	if err := listWorkflows(ctx, client); err != nil {
		log.Warn().Err(err).Msg("Failed to list workflows")
	}

	log.Info().Msg("Example 1 completed")
	return nil
}

// Example 2: Using functional options
func example2FunctionalOptions(ctx context.Context) error {
	log.Info().Msg("=== Example 2: Functional Options ===")

	// Get kubeconfig path from environment or use default
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}

	// Create client with functional options
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithContext(""), // Use current context
	)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create client with options (expected if no kubeconfig)")
		return nil
	}

	// List workflows
	if err := listWorkflows(ctx, client); err != nil {
		log.Warn().Err(err).Msg("Failed to list workflows")
	}

	log.Info().Msg("Example 2 completed")
	return nil
}

// Example 3: In-cluster configuration
func example3InCluster(ctx context.Context) error {
	log.Info().Msg("=== Example 3: In-Cluster Configuration ===")

	// Create client for in-cluster use
	ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())
	if err != nil {
		return fmt.Errorf("failed to create in-cluster client: %w", err)
	}

	// List workflows
	if err := listWorkflows(ctx, client); err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	log.Info().Msg("Example 3 completed")
	return nil
}

// Example 4: Argo Server mode
func example4ArgoServer(ctx context.Context) error {
	log.Info().Msg("=== Example 4: Argo Server Mode ===")

	// Get Argo Server URL and token from environment
	serverURL := os.Getenv("ARGO_SERVER_URL")
	authToken := os.Getenv("ARGO_AUTH_TOKEN")

	if serverURL == "" {
		serverURL = "https://argo-server:2746"
	}

	// Create client for Argo Server
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithArgoServer(serverURL, authToken),
		argo.WithArgoServerInsecure(false), // Set to true for HTTP
	)
	if err != nil {
		return fmt.Errorf("failed to create Argo Server client: %w", err)
	}

	// List workflows
	if err := listWorkflows(ctx, client); err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	log.Info().Msg("Example 4 completed")
	return nil
}

// Example 5: With OpenTelemetry instrumentation
func example5WithOTel(ctx context.Context) error {
	log.Info().Msg("=== Example 5: With OpenTelemetry ===")

	// Create OTel config
	otelConfig := otel.NewConfig("argo-client-example")

	// Create client with OTel
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create client with OTel (expected if no kubeconfig)")
		return nil
	}

	// List workflows (with tracing)
	if err := listWorkflows(ctx, client); err != nil {
		log.Warn().Err(err).Msg("Failed to list workflows")
	}

	log.Info().Msg("Example 5 completed")
	return nil
}

// Example 6: Create and submit a workflow
func example6CreateWorkflow(ctx context.Context) error {
	log.Info().Msg("=== Example 6: Create and Submit Workflow ===")

	// Create client
	ctx, client, err := argo.NewClientWithOptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create a simple workflow
	wf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hello-world-",
			Namespace:    "argo",
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "hello",
			Templates: []v1alpha1.Template{
				{
					Name: "hello",
					Container: &corev1.Container{
						Image:   "alpine:latest",
						Command: []string{"echo"},
						Args:    []string{"Hello, Argo Workflows!"},
					},
				},
			},
		},
	}

	// Submit workflow
	wfClient := client.NewWorkflowServiceClient()
	created, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
		Namespace: "argo",
		Workflow:  wf,
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	log.Info().
		Str("name", created.Name).
		Str("namespace", created.Namespace).
		Msg("Workflow created successfully")

	log.Info().Msg("Example 6 completed")
	return nil
}

// Helper function to list workflows
func listWorkflows(ctx context.Context, client apiclient.Client) error {
	wfClient := client.NewWorkflowServiceClient()

	// List workflows in the 'argo' namespace
	resp, err := wfClient.ListWorkflows(ctx, &workflow.WorkflowListRequest{
		Namespace: "argo",
		ListOptions: &metav1.ListOptions{
			Limit: 10,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	log.Info().
		Int("count", len(resp.Items)).
		Msg("Listed workflows")

	// Print workflow names
	for i, wf := range resp.Items {
		log.Info().
			Int("index", i+1).
			Str("name", wf.Name).
			Str("namespace", wf.Namespace).
			Str("status", string(wf.Status.Phase)).
			Msg("Workflow")
	}

	return nil
}
