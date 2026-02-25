package docker_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogOptions_WithStdout(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test stdout"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx, docker.WithStdout(true), docker.WithStderr(false))
	require.NoError(t, err)
	assert.Contains(t, logs, "test stdout")
}

func TestLogOptions_WithStderr(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo test stderr >&2"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx, docker.WithStdout(false), docker.WithStderr(true))
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestLogOptions_WithTimestamps(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx, docker.WithTimestamps())
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestLogOptions_WithTail(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in 1 2 3 4 5; do echo Line $i; done"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx, docker.WithTail("2"))
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestLogOptions_WithSince(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	// Use a wide window so the log (generated ~2s ago) is included
	logs, err := exec.Logs(ctx, docker.WithSince("1m"))
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestLogOptions_WithUntil(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	// Docker Until is relative to daemon time: "1s" means "until 1 second ago"
	// so we must NOT use it to capture recent logs. Use an RFC3339 timestamp
	// far in the future, or verify that a narrow window correctly excludes logs.
	// Here we verify that Until("1m") excludes logs generated >1 minute ago
	// while including recent ones (generated ~2s ago, within the 1-minute window).
	logs, err := exec.Logs(ctx, docker.WithSince("1m"))
	require.NoError(t, err)
	assert.NotEmpty(t, logs)

	// Verify Until with very small window excludes recent logs
	logsExcluded, err := exec.Logs(ctx, docker.WithUntil("1m"))
	require.NoError(t, err)
	// Until("1m") means "until 1 minute ago" — recent logs should be excluded
	assert.Empty(t, logsExcluded)
}

func TestLogOptions_Combined(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in 1 2 3; do echo Line $i; done"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx,
		docker.WithStdout(true),
		docker.WithStderr(true),
		docker.WithTimestamps(),
		docker.WithTail("10"),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestLogMethods_GetStdout(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "stdout message"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.GetStdout(ctx)
	require.NoError(t, err)
	assert.Contains(t, logs, "stdout")
}

func TestLogMethods_GetStderr(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo stderr message >&2"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.GetStderr(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestFollowLogs_ToWriter(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo line1; sleep 1; echo line2"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(context.Background())

	var buf strings.Builder
	err = exec.FollowLogs(ctx, &buf)
	// May get context deadline exceeded, which is expected
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("FollowLogs error (expected): %v", err)
	}

	output := buf.String()
	assert.NotEmpty(t, output)
}
