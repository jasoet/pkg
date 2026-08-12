package rest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRestConfig(t *testing.T) {
	config := DefaultRestConfig()

	require.NotNil(t, config)
	assert.Equal(t, 1, config.RetryCount)
	assert.Equal(t, 2*time.Second, config.RetryWaitTime)
	assert.Equal(t, 10*time.Second, config.RetryMaxWaitTime)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestDefaultRestConfig_MaxResponseBodyLog(t *testing.T) {
	config := DefaultRestConfig()
	assert.Equal(t, 1024, config.MaxResponseBodyLog)
}

func TestDefaultRestConfig_RetryNonIdempotent(t *testing.T) {
	config := DefaultRestConfig()
	assert.False(t, config.RetryNonIdempotent, "non-idempotent retries must be off by default")
}

func TestConfigStructFields(t *testing.T) {
	config := Config{
		RetryCount:       3,
		RetryWaitTime:    5 * time.Second,
		RetryMaxWaitTime: 60 * time.Second,
		Timeout:          10 * time.Second,
	}

	assert.Equal(t, 3, config.RetryCount)
	assert.Equal(t, 5*time.Second, config.RetryWaitTime)
	assert.Equal(t, 60*time.Second, config.RetryMaxWaitTime)
	assert.Equal(t, 10*time.Second, config.Timeout)
}
