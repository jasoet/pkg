package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/docker"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestExecutor_ContainerID(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	// Before starting, should be empty
	assert.Empty(t, exec.ContainerID())

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// After starting, should have ID
	assert.NotEmpty(t, exec.ContainerID())
}

func TestExecutor_Close(t *testing.T) {
	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	err := exec.Close()
	assert.NoError(t, err)
}

func TestStatus_WaitForState(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Wait for running state
	err = exec.WaitForState(ctx, "running", 10*time.Second)
	assert.NoError(t, err)
}

func TestStatus_WaitForStateTimeout(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Wait for a state that won't happen
	err = exec.WaitForState(ctx, "paused", 2*time.Second)
	assert.Error(t, err)
	// Error can be timeout or context deadline exceeded
	assert.True(t, err != nil, "should have an error")
}

func TestStatus_ExitCode(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "exit 42"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Wait for container to exit
	time.Sleep(2 * time.Second)

	exitCode, err := exec.ExitCode(ctx)
	require.NoError(t, err)
	assert.Equal(t, 42, exitCode)
}

func TestStatus_ExitCodeWhileRunning(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Try to get exit code while running
	_, err = exec.ExitCode(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still running")
}

func TestStatus_Inspect(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	inspect, err := exec.Inspect(ctx)
	require.NoError(t, err)
	assert.NotNil(t, inspect)
	assert.NotEmpty(t, inspect.ID)
	assert.Equal(t, "running", inspect.State.Status)
}

func TestStatus_GetStats(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	stats, err := exec.GetStats(ctx)
	require.NoError(t, err)
	defer stats.Body.Close()
	assert.NotNil(t, stats)
}

func TestNetwork_ConnectionString(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:8892"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForPort("80").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	connStr, err := exec.ConnectionString(ctx, "80/tcp", "http://%s/api")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8892/api", connStr)
}

func TestExecutor_NewFromRequest(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "alpine:latest",
		Cmd:   []string{"echo", "hello"},
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestExecutor_NewFromRequest_WithOptions(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "alpine:latest",
		Name:  "struct-name",
		Env: map[string]string{
			"VAR1": "value1",
		},
	}

	// Add additional options that override struct fields
	exec, err := docker.NewFromRequest(req,
		docker.WithName("option-name"), // Override name
		docker.WithCmd("echo", "test"),
		docker.WithEnv("VAR2=value2"), // Add another env var
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)

	// Note: We can't directly access config fields to verify,
	// but we can verify it doesn't error and creates executor
}

func TestExecutor_NewFromRequest_WithOTel(t *testing.T) {
	ctx := context.Background()

	req := docker.ContainerRequest{
		Image: "alpine:latest",
		Cmd:   []string{"echo", "otel-test"},
	}

	// Create OTel config
	tp := trace.NewTracerProvider()
	mp := metric.NewMeterProvider()

	otelCfg := &otel.Config{
		TracerProvider: tp,
		MeterProvider:  mp,
	}

	exec, err := docker.NewFromRequest(req,
		docker.WithOTelConfig(otelCfg),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)

	// Verify it works with OTel
	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)
	assert.NotEmpty(t, exec.ContainerID())
}

func TestExecutor_StatusBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	_, err := exec.Status(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestExecutor_LogsBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	_, err := exec.Logs(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestExecutor_StopBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	err := exec.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestExecutor_TerminateBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	err := exec.Terminate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestExecutor_RestartBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	err := exec.Restart(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestExecutor_WaitBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
	)

	_, err := exec.Wait(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestNetwork_EndpointBeforeStart(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
	)

	_, err := exec.Endpoint(ctx, "80/tcp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not started")
}

func TestNetwork_MappedPortNotFound(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	_, err = exec.MappedPort(ctx, "9999/tcp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExecutor_PullImageError(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("this-image-definitely-does-not-exist-12345:latest"),
	)

	err := exec.Start(ctx)
	assert.Error(t, err)
}

func TestExecutor_WaitExitCode(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "sleep 1; exit 0"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	exitCode, err := exec.Wait(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), exitCode)
}

func TestStatus_HealthCheckNotConfigured(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	_, err = exec.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestStatus_WaitForHealthyNotConfigured(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	err = exec.WaitForHealthy(ctx, 5*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestNetwork_GetNetworksEmpty(t *testing.T) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	networks, err := exec.GetNetworks(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, networks) // At least bridge network
}
