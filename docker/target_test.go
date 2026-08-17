package docker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/docker"
)

// Compile-level assertions: every built-in strategy satisfies the
// ContainerTarget-based WaitStrategy interface.
var (
	_ docker.WaitStrategy = docker.WaitForLog("")
	_ docker.WaitStrategy = docker.WaitForPort("80")
	_ docker.WaitStrategy = docker.WaitForHTTP("80", "/", 200)
	_ docker.WaitStrategy = docker.WaitForHealthy()
	_ docker.WaitStrategy = docker.WaitForFunc(func(context.Context, docker.ContainerTarget) error { return nil })
	_ docker.WaitStrategy = docker.WaitForAll(docker.WaitForHealthy())
	_ docker.WaitStrategy = docker.ForListeningPort("80/tcp")
)

func TestContainerTarget_ZeroValue(t *testing.T) {
	target := docker.ContainerTarget{}
	assert.Empty(t, target.ID())
}

func TestWaitForFunc_NewSignature(t *testing.T) {
	sentinel := errors.New("boom")
	called := false

	w := docker.WaitForFunc(func(ctx context.Context, target docker.ContainerTarget) error {
		called = true
		return sentinel
	})

	err := w.WaitUntilReady(context.Background(), docker.ContainerTarget{})
	require.Error(t, err)
	assert.True(t, called, "WaitForFunc must invoke the wrapped function")
	assert.ErrorIs(t, err, sentinel, "WaitUntilReady must propagate the function error")
}

func TestWaitForFunc_RespectsContextDeadline(t *testing.T) {
	w := docker.WaitForFunc(func(ctx context.Context, target docker.ContainerTarget) error {
		<-ctx.Done()
		return ctx.Err()
	}).WithStartupTimeout(50 * time.Millisecond)

	start := time.Now()
	err := w.WaitUntilReady(context.Background(), docker.ContainerTarget{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second, "timeout wrapper must cap the wait")
}
