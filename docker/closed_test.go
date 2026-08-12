package docker_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/docker"
)

// TestExecutor_AfterClose_ReturnsErrorNotPanic verifies that once an executor is
// Close()d (client set to nil), the methods that use the Docker client return an
// "executor is closed" error instead of dereferencing a nil client and panicking.
// Constructing the executor only builds a client handle (no daemon connection),
// so this runs without a container runtime. Run under -race to exercise the
// synchronized snapshot of e.client.
func TestExecutor_AfterClose_ReturnsErrorNotPanic(t *testing.T) {
	ctx := context.Background()

	exec, err := docker.New(docker.WithImage("alpine:latest"))
	require.NoError(t, err)

	require.NoError(t, exec.Close())
	// Close is idempotent.
	require.NoError(t, exec.Close())

	t.Run("Stop", func(t *testing.T) {
		err := exec.Stop(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Restart", func(t *testing.T) {
		err := exec.Restart(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Terminate", func(t *testing.T) {
		err := exec.Terminate(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Wait", func(t *testing.T) {
		_, err := exec.Wait(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Logs", func(t *testing.T) {
		_, err := exec.Logs(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Status", func(t *testing.T) {
		_, err := exec.Status(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Inspect", func(t *testing.T) {
		_, err := exec.Inspect(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("Host", func(t *testing.T) {
		_, err := exec.Host(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("MappedPort", func(t *testing.T) {
		_, err := exec.MappedPort(ctx, "80/tcp")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("GetStats", func(t *testing.T) {
		_, err := exec.GetStats(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})

	t.Run("StreamLogs", func(t *testing.T) {
		_, errCh := exec.StreamLogs(ctx)
		err := <-errCh
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is closed")
	})
}

// TestExecutor_ConcurrentCloseAndUse exercises the race detector: one goroutine
// Close()s the executor while others call client-using methods. The methods must
// return an error rather than race on or nil-deref e.client. Run with -race.
func TestExecutor_ConcurrentCloseAndUse(t *testing.T) {
	ctx := context.Background()

	exec, err := docker.New(docker.WithImage("alpine:latest"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// These must never panic regardless of whether Close has run yet.
			_, _ = exec.Status(ctx)
			_ = exec.Stop(ctx)
			_, _ = exec.Host(ctx)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = exec.Close()
	}()

	wg.Wait()
}
