package docker_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v2/docker"
)

// Fix 1 (H6): WaitForLog must return error instead of panicking on invalid regex.
func TestWaitForLog_InvalidRegex_ReturnsError(t *testing.T) {
	// An invalid regex pattern — regexp.MustCompile would panic on this.
	const badPattern = `[invalid`

	// Construction must not panic.
	w := docker.WaitForLog(badPattern)
	require.NotNil(t, w)

	// WaitUntilReady must return an error, not panic.
	err := w.WaitUntilReady(context.Background(), nil, "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestWaitForLog_ValidRegex_NoError(t *testing.T) {
	// Valid patterns must still compile without error (the error field stays nil).
	w := docker.WaitForLog(`start worker processes`)
	require.NotNil(t, w)
	// We cannot call WaitUntilReady without a live container, but construction
	// must succeed without panic — that is sufficient here.
}

// Fix 2 (H7): Start() must guard against double-start to prevent container leaks.
func TestExecutor_DoubleStart_ReturnsError(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "30"),
		docker.WithAutoRemove(true),
	)
	require.NoError(t, err)

	// First start must succeed.
	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Second start on the same executor must return an error.
	err = exec.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}

// Fix 3 (H8): ConnectionString must use {{endpoint}} placeholder, not %s.
func TestConnectionString_PlaceholderReplacement(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:18765"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker").WithStartupTimeout(30*1000000000),
		),
	)
	require.NoError(t, err)

	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// New convention: {{endpoint}} as placeholder.
	connStr, err := exec.ConnectionString(ctx, "80/tcp", "http://{{endpoint}}/api")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:18765/api", connStr)

	// The old %s placeholder must NOT be treated as a format verb any more —
	// it should appear literally in the output (no substitution).
	connStrLiteral, err := exec.ConnectionString(ctx, "80/tcp", "http://%s/api")
	require.NoError(t, err)
	assert.Equal(t, "http://%s/api", connStrLiteral)
}

func TestConnectionString_NoInjection(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:18766"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker").WithStartupTimeout(30*1000000000),
		),
	)
	require.NoError(t, err)

	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// A template that contains format verbs other than the placeholder must
	// not cause a format-string injection or a runtime panic.
	connStr, err := exec.ConnectionString(ctx, "80/tcp", "dsn://user:p%40ss@{{endpoint}}/db?sslmode=disable")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(connStr, "dsn://user:p%40ss@localhost:"), "unexpected connStr: %s", connStr)
	assert.True(t, strings.HasSuffix(connStr, "/db?sslmode=disable"), "unexpected connStr: %s", connStr)
}
