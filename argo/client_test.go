package argo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientWithOptions_AppliesOptions(t *testing.T) {
	// This test verifies that options are properly applied
	// We can't test actual client creation without a real cluster,
	// but we can verify option application by checking errors

	t.Run("invalid kubeconfig path", func(t *testing.T) {
		ctx := context.Background()
		_, _, err := NewClientWithOptions(ctx,
			WithKubeConfig("/nonexistent/kubeconfig"),
		)
		// Should fail trying to load nonexistent kubeconfig
		assert.Error(t, err)
	})
}

func TestBuildClientConfig(t *testing.T) {
	t.Run("in-cluster config", func(t *testing.T) {
		cfg := InClusterConfig()
		clientCfg := buildClientConfig(cfg)

		assert.NotNil(t, clientCfg)
		_, ok := clientCfg.(*inClusterClientConfig)
		assert.True(t, ok, "should return inClusterClientConfig")
	})

	t.Run("with explicit kubeconfig path", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.KubeConfigPath = "/custom/path/kubeconfig"

		clientCfg := buildClientConfig(cfg)
		assert.NotNil(t, clientCfg)
	})

	t.Run("with context override", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Context = "production"

		clientCfg := buildClientConfig(cfg)
		assert.NotNil(t, clientCfg)
	})

	t.Run("default config", func(t *testing.T) {
		cfg := DefaultConfig()
		clientCfg := buildClientConfig(cfg)
		assert.NotNil(t, clientCfg)
	})
}

func TestInClusterClientConfig_RawConfig(t *testing.T) {
	icc := &inClusterClientConfig{}
	_, err := icc.RawConfig()
	assert.Error(t, err, "RawConfig should not be supported")
}

func TestInClusterClientConfig_ConfigAccess(t *testing.T) {
	icc := &inClusterClientConfig{}
	access := icc.ConfigAccess()
	assert.NotNil(t, access)
}

func TestInClusterClientConfig_Namespace(t *testing.T) {
	icc := &inClusterClientConfig{}

	// This will fail in non-k8s environment, which is expected
	namespace, overridden, err := icc.Namespace()

	// If we're not in a k8s pod, expect error
	if err != nil {
		assert.False(t, overridden)
		assert.Equal(t, "default", namespace)
	} else {
		// If somehow we are in a pod, verify the result
		assert.True(t, overridden)
		assert.NotEmpty(t, namespace)
	}
}

func TestInClusterClientConfig_ClientConfig(t *testing.T) {
	icc := &inClusterClientConfig{}

	// This will fail outside of a k8s cluster, which is expected
	config, err := icc.ClientConfig()

	// In CI/local dev, this should fail
	if err != nil {
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to load in-cluster config")
	} else {
		// If we're actually in a cluster, verify config
		assert.NotNil(t, config)
	}
}

func TestNewClient_WithArgoServer(t *testing.T) {
	ctx := context.Background()

	// Client creation should succeed even with nonexistent server
	// It only fails when trying to actually communicate with the server
	_, client, err := NewClientWithOptions(ctx,
		WithArgoServer("http://nonexistent:2746", "Bearer token"),
		WithArgoServerInsecure(true),
	)

	// Client creation should succeed
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClient_OptionErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid kubeconfig triggers error", func(t *testing.T) {
		_, _, err := NewClient(ctx, &Config{
			KubeConfigPath: "/definitely/does/not/exist/kubeconfig",
		})
		require.Error(t, err)
	})
}
