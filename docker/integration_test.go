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

// Integration test with OpenTelemetry
func TestIntegration_WithOTel(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	// Create OTel providers
	tp := trace.NewTracerProvider()
	mp := metric.NewMeterProvider()

	otelCfg := &otel.Config{
		TracerProvider: tp,
		MeterProvider:  mp,
	}

	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test with otel"),
		docker.WithOTelConfig(otelCfg),
	)
	require.NoError(t, err)

	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx)
	require.NoError(t, err)
	assert.Contains(t, logs, "test with otel")
}

// Integration test for complex container lifecycle
func TestIntegration_ComplexLifecycle(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithName("test-lifecycle-complex"),
		docker.WithLabel("test", "integration"),
		docker.WithEnv("NGINX_HOST=localhost"),
	)

	// Start
	err := exec.Start(ctx)
	require.NoError(t, err)

	// Get container ID
	containerID := exec.ContainerID()
	assert.NotEmpty(t, containerID)

	// Check status
	status, err := exec.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Running)

	// Get network info
	host, err := exec.Host(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, host)

	endpoint, err := exec.Endpoint(ctx, "80/tcp")
	require.NoError(t, err)
	assert.Contains(t, endpoint, ":")

	allPorts, err := exec.GetAllPorts(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, allPorts)

	networks, err := exec.GetNetworks(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, networks)

	ip, err := exec.GetIPAddress(ctx, "")
	require.NoError(t, err)
	assert.NotEmpty(t, ip)

	// Inspect
	inspect, err := exec.Inspect(ctx)
	require.NoError(t, err)
	assert.NotNil(t, inspect)

	// Get stats
	stats, err := exec.GetStats(ctx)
	require.NoError(t, err)
	defer stats.Body.Close()

	// Stop
	err = exec.Stop(ctx)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Verify stopped
	running, err := exec.IsRunning(ctx)
	require.NoError(t, err)
	assert.False(t, running)

	// Restart
	err = exec.Restart(ctx)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Verify running again
	running, _ = exec.IsRunning(ctx)
	assert.True(t, running)

	// Terminate
	err = exec.Terminate(ctx)
	require.NoError(t, err)

	// Close client
	err = exec.Close()
	assert.NoError(t, err)
}

// Integration test for error scenarios
func TestIntegration_ErrorScenarios(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	t.Run("OperationsBeforeStart", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("alpine:latest"),
		)

		// All these should fail
		_, err := exec.Status(ctx)
		assert.Error(t, err)

		_, err = exec.Logs(ctx)
		assert.Error(t, err)

		_, err = exec.Host(ctx)
		assert.Error(t, err)

		_, err = exec.MappedPort(ctx, "80/tcp")
		assert.Error(t, err)

		err = exec.Stop(ctx)
		assert.Error(t, err)
	})

	t.Run("InvalidImage", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("this-image-does-not-exist-xyz:latest"),
		)

		err := exec.Start(ctx)
		assert.Error(t, err)
	})
}

// Integration test for volume mounting
func TestIntegration_VolumeMounts(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("ls", "/data"),
		docker.WithVolume("/tmp", "/data"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	// Container should have run successfully
	exitCode, err := exec.ExitCode(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

// Integration test for network modes
func TestIntegration_NetworkModes(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sleep", "3"),
		docker.WithNetworkMode("bridge"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	networks, err := exec.GetNetworks(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, networks)
}

// Integration test for labels
func TestIntegration_Labels(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test"),
		docker.WithLabels(map[string]string{
			"env":     "test",
			"version": "1.0",
			"app":     "integration",
		}),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	inspect, err := exec.Inspect(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test", inspect.Config.Labels["env"])
	assert.Equal(t, "1.0", inspect.Config.Labels["version"])
	assert.Equal(t, "integration", inspect.Config.Labels["app"])
}

// Integration test for environment variables
func TestIntegration_EnvironmentVars(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo VAR1=$VAR1 VAR2=$VAR2 VAR3=$VAR3"),
		docker.WithEnv("VAR1=value1"),
		docker.WithEnvMap(map[string]string{
			"VAR2": "value2",
			"VAR3": "value3",
		}),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.Logs(ctx)
	require.NoError(t, err)
	assert.Contains(t, logs, "value1")
	assert.Contains(t, logs, "value2")
	assert.Contains(t, logs, "value3")
}

// Integration test for port mapping edge cases
func TestIntegration_PortMapping(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPortBindings(map[string]string{
			"80/tcp": "0", // Random port
		}),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForPort("80/tcp").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	port, err := exec.MappedPort(ctx, "80/tcp")
	require.NoError(t, err)
	assert.NotEmpty(t, port)
	assert.NotEqual(t, "80", port) // Should be random
}

// Integration test for wait strategies combinations
func TestIntegration_WaitStrategies(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	t.Run("MultipleStrategies", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("nginx:alpine"),
			docker.WithPorts("80:0"),
			docker.WithAutoRemove(true),
			docker.WithWaitStrategy(
				docker.WaitForAll(
					docker.WaitForLog("start worker"),
					docker.WaitForPort("80/tcp"),
				).WithStartupTimeout(60*time.Second),
			),
		)

		err := exec.Start(ctx)
		require.NoError(t, err)
		defer exec.Terminate(ctx)

		running, _ := exec.IsRunning(ctx)
		assert.True(t, running)
	})

	t.Run("HTTPWaitStrategy", func(t *testing.T) {
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
	})
}
