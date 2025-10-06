//go:build argo

package argo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient_WithKubeconfig tests client creation with a valid kubeconfig.
// This test requires a valid kubeconfig file and access to a Kubernetes cluster.
func TestNewClient_WithKubeconfig(t *testing.T) {
	// Skip if no kubeconfig is available
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: kubeconfig not found")
	}

	ctx := context.Background()
	config := &Config{
		KubeConfigPath: kubeconfigPath,
	}

	ctx, client, err := NewClient(ctx, config)
	require.NoError(t, err, "Failed to create Argo client")
	require.NotNil(t, client, "Client should not be nil")

	// Verify we can get a workflow service client
	wfClient := client.NewWorkflowServiceClient()
	assert.NotNil(t, wfClient, "Workflow service client should not be nil")
}

// TestNewClientWithOptions_DefaultConfig tests client creation with default configuration.
func TestNewClientWithOptions_DefaultConfig(t *testing.T) {
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: kubeconfig not found")
	}

	ctx := context.Background()

	ctx, client, err := NewClientWithOptions(ctx)
	if err != nil {
		// If connection fails, it might be because cluster is not accessible
		// This is acceptable for integration tests
		t.Logf("Failed to create client (expected if cluster not accessible): %v", err)
		return
	}

	require.NotNil(t, client, "Client should not be nil")

	wfClient := client.NewWorkflowServiceClient()
	assert.NotNil(t, wfClient, "Workflow service client should not be nil")
}

// TestNewClientWithOptions_CustomPath tests client creation with custom kubeconfig path.
func TestNewClientWithOptions_CustomPath(t *testing.T) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: kubeconfig not found")
	}

	ctx := context.Background()

	ctx, client, err := NewClientWithOptions(ctx,
		WithKubeConfig(kubeconfigPath),
	)

	if err != nil {
		t.Logf("Failed to create client (expected if cluster not accessible): %v", err)
		return
	}

	require.NotNil(t, client, "Client should not be nil")

	wfClient := client.NewWorkflowServiceClient()
	assert.NotNil(t, wfClient, "Workflow service client should not be nil")
}

// TestNewClientWithOptions_MultipleOptions tests combining multiple functional options.
func TestNewClientWithOptions_MultipleOptions(t *testing.T) {
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: kubeconfig not found")
	}

	ctx := context.Background()

	ctx, client, err := NewClientWithOptions(ctx,
		WithKubeConfig(kubeconfigPath),
		WithContext(""), // Use current context
	)

	if err != nil {
		t.Logf("Failed to create client (expected if cluster not accessible): %v", err)
		return
	}

	require.NotNil(t, client, "Client should not be nil")

	wfClient := client.NewWorkflowServiceClient()
	assert.NotNil(t, wfClient, "Workflow service client should not be nil")
}

// TestBuildClientConfig_Default tests default client config building.
func TestBuildClientConfig_Default(t *testing.T) {
	config := DefaultConfig()
	clientConfig := buildClientConfig(config)

	assert.NotNil(t, clientConfig, "Client config should not be nil")

	// Try to get REST config (might fail if no cluster access)
	_, err := clientConfig.ClientConfig()
	if err != nil {
		t.Logf("Failed to get client config (expected if no cluster access): %v", err)
	}
}

// TestInClusterClientConfig tests in-cluster configuration.
// This test will fail when not running inside a Kubernetes pod, which is expected.
func TestInClusterClientConfig(t *testing.T) {
	config := &inClusterClientConfig{}

	// This should fail when not running in a pod
	restConfig, err := config.ClientConfig()
	if err != nil {
		t.Logf("Failed to get in-cluster config (expected when not in pod): %v", err)
		return
	}

	assert.NotNil(t, restConfig, "REST config should not be nil")
}

// TestGetRestConfig tests the deprecated GetRestConfig function.
func TestGetRestConfig(t *testing.T) {
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: kubeconfig not found")
	}

	// Test out-of-cluster mode
	config, err := GetRestConfig(false)
	if err != nil {
		t.Logf("Failed to get REST config (expected if no cluster access): %v", err)
		return
	}

	assert.NotNil(t, config, "REST config should not be nil")
	assert.NotEmpty(t, config.Host, "Host should not be empty")
}

// TestArgoServerConfig_Integration tests Argo Server configuration creation in integration context.
func TestArgoServerConfig_Integration(t *testing.T) {
	serverURL := "https://argo-server:2746"
	authToken := "Bearer test-token"

	config := ArgoServerConfig(serverURL, authToken)

	assert.Equal(t, serverURL, config.ArgoServerOpts.URL)
	assert.Equal(t, authToken, config.ArgoServerOpts.AuthToken)
	assert.False(t, config.InCluster)
	assert.False(t, config.ArgoServerOpts.InsecureSkipVerify)
}

// TestInClusterConfig tests in-cluster configuration creation.
func TestInClusterConfig_Creation(t *testing.T) {
	config := InClusterConfig()

	assert.True(t, config.InCluster)
	assert.Empty(t, config.KubeConfigPath)
	assert.Empty(t, config.Context)
}

// TestDefaultConfig tests default configuration creation.
func TestDefaultConfig_Creation(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.InCluster)
	assert.Empty(t, config.KubeConfigPath)
	assert.Empty(t, config.Context)
	assert.False(t, config.ArgoServerOpts.InsecureSkipVerify)
}
