package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.False(t, config.InCluster)
	assert.Empty(t, config.KubeConfigPath)
	assert.Empty(t, config.Context)
	assert.Empty(t, config.ArgoServerOpts.URL)
	assert.Empty(t, config.ArgoServerOpts.AuthToken)
	assert.False(t, config.ArgoServerOpts.InsecureSkipVerify)
	assert.False(t, config.ArgoServerOpts.HTTP1)
	assert.Nil(t, config.OTelConfig)
}

func TestInClusterConfig(t *testing.T) {
	config := InClusterConfig()

	assert.NotNil(t, config)
	assert.True(t, config.InCluster)
	assert.Empty(t, config.KubeConfigPath)
	assert.Empty(t, config.Context)
}

func TestArgoServerConfig(t *testing.T) {
	serverURL := "https://argo-server.example.com:2746"
	authToken := "Bearer test-token-123"

	config := ArgoServerConfig(serverURL, authToken)

	assert.NotNil(t, config)
	assert.False(t, config.InCluster)
	assert.Equal(t, serverURL, config.ArgoServerOpts.URL)
	assert.Equal(t, authToken, config.ArgoServerOpts.AuthToken)
	assert.False(t, config.ArgoServerOpts.InsecureSkipVerify)
	assert.False(t, config.ArgoServerOpts.HTTP1)
}

func TestArgoServerConfig_EmptyValues(t *testing.T) {
	config := ArgoServerConfig("", "")

	assert.NotNil(t, config)
	assert.Empty(t, config.ArgoServerOpts.URL)
	assert.Empty(t, config.ArgoServerOpts.AuthToken)
}
