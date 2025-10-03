package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/jasoet/pkg/v2/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitStrategy_WaitForLog(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker processes").
				WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	running, _ := exec.IsRunning(ctx)
	assert.True(t, running)
}

func TestWaitStrategy_WaitForPort(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:8889"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForPort("80").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	port, _ := exec.MappedPort(ctx, "80/tcp")
	assert.Equal(t, "8889", port)
}

func TestWaitStrategy_WaitForHTTP(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForHTTP("80", "/", 200).
				WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	running, _ := exec.IsRunning(ctx)
	assert.True(t, running)
}

func TestWaitStrategy_ForListeningPort(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:8891"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.ForListeningPort("80/tcp").
				WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	port, _ := exec.MappedPort(ctx, "80/tcp")
	assert.Equal(t, "8891", port)
}

func TestWaitStrategy_WaitForFunc(t *testing.T) {
	ctx := context.Background()

	customWait := docker.WaitForFunc(func(ctx context.Context, cli *client.Client, containerID string) error {
		// Custom wait logic - just wait 1 second
		time.Sleep(1 * time.Second)
		return nil
	}).WithStartupTimeout(10 * time.Second)

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(customWait),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	running, _ := exec.IsRunning(ctx)
	assert.True(t, running)
}

func TestWaitStrategy_WaitForAll(t *testing.T) {
	ctx := context.Background()

	multiWait := docker.WaitForAll(
		docker.WaitForLog("start worker"),
		docker.WaitForPort("80"),
	).WithStartupTimeout(60 * time.Second)

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(multiWait),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	running, _ := exec.IsRunning(ctx)
	assert.True(t, running)
}

func TestWaitStrategy_Timeout(t *testing.T) {
	ctx := context.Background()

	// Wait for a log that will never appear with very short timeout
	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "10"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("this-will-never-appear").
				WithStartupTimeout(1*time.Second),
		),
	)

	err := exec.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to become ready")
}

func TestWaitStrategy_PortTimeout(t *testing.T) {
	ctx := context.Background()

	// Wait for a port on a container that doesn't expose it
	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "10"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForPort("8888/tcp").
				WithStartupTimeout(2*time.Second),
		),
	)

	err := exec.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to become ready")
}

func TestWaitStrategy_HTTPTimeout(t *testing.T) {
	ctx := context.Background()

	// Wait for HTTP on a container that doesn't serve HTTP
	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "10"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForHTTP("8080", "/", 200).
				WithStartupTimeout(2*time.Second),
		),
	)

	err := exec.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to become ready")
}
