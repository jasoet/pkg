package docker_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_FunctionalOptions_Nginx(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	// Create executor with functional options
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"), // Random host port
		docker.WithName("test-nginx-functional"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker processes").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Start container
	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Verify container is running
	running, err := exec.IsRunning(ctx)
	require.NoError(t, err)
	assert.True(t, running)

	// Get endpoint and test HTTP
	endpoint, err := exec.Endpoint(ctx, "80/tcp")
	require.NoError(t, err)
	assert.NotEmpty(t, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+endpoint, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExecutor_StructBased_Nginx(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	// Create executor with ContainerRequest (testcontainers style)
	req := docker.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"80/tcp"},
		Name:         "test-nginx-struct",
		AutoRemove:   true,
		WaitingFor: docker.WaitForLog("start worker processes").
			WithStartupTimeout(30 * time.Second),
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)

	// Start container
	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Verify container is running
	status, err := exec.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Equal(t, "running", status.State)
}

func TestExecutor_Hybrid_Redis(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	// Start with struct, add functional options
	req := docker.ContainerRequest{
		Image: "redis:7-alpine",
		PortBindings: map[string]string{
			"6379/tcp": "0", // Random port
		},
		Env: map[string]string{
			"REDIS_PASSWORD": "test123",
		},
	}

	exec, err := docker.New(
		docker.WithRequest(req),
		docker.WithName("test-redis-hybrid"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	err = exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Verify endpoint
	endpoint, err := exec.Endpoint(ctx, "6379/tcp")
	require.NoError(t, err)
	assert.Contains(t, endpoint, ":")
}

func TestExecutor_WaitStrategies(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	t.Run("WaitForLog", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("nginx:alpine"),
			docker.WithPorts("80:0"),
			docker.WithAutoRemove(true),
			docker.WithWaitStrategy(
				docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
			),
		)

		err := exec.Start(ctx)
		require.NoError(t, err)
		defer exec.Terminate(ctx)

		running, _ := exec.IsRunning(ctx)
		assert.True(t, running)
	})

	t.Run("WaitForPort", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("nginx:alpine"),
			docker.WithPorts("80:8888"),
			docker.WithAutoRemove(true),
			docker.WithWaitStrategy(
				docker.WaitForPort("80/tcp").WithStartupTimeout(30*time.Second),
			),
		)

		err := exec.Start(ctx)
		require.NoError(t, err)
		defer exec.Terminate(ctx)

		port, _ := exec.MappedPort(ctx, "80/tcp")
		assert.Equal(t, "8888", port)
	})

	t.Run("WaitForHTTP", func(t *testing.T) {
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

		endpoint, _ := exec.Endpoint(ctx, "80/tcp")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+endpoint, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestExecutor_Logs(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "Hello from Docker"),
		// Don't use AutoRemove - we need to read logs after container finishes
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Wait for container to finish
	time.Sleep(2 * time.Second)

	// Get logs
	logs, err := exec.Logs(ctx)
	require.NoError(t, err)
	assert.Contains(t, logs, "Hello from Docker")
}

func TestExecutor_LogStreaming(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in 1 2 3; do echo Line $i; sleep 1; done"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Stream logs
	logCh, errCh := exec.StreamLogs(ctx, docker.WithFollow())

	var lines []string
	timeout := time.After(10 * time.Second)

collectLogs:
	for {
		select {
		case log, ok := <-logCh:
			if !ok {
				break collectLogs
			}
			lines = append(lines, log.Content)
		case err := <-errCh:
			if err != nil {
				t.Logf("Log stream error: %v", err)
			}
		case <-timeout:
			break collectLogs
		}
	}

	// Should have received some log lines
	assert.NotEmpty(t, lines)
}

func TestExecutor_Status(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Get status
	status, err := exec.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Equal(t, "running", status.State)
	assert.NotEmpty(t, status.ID)
	assert.NotEmpty(t, status.Image)
	assert.False(t, status.StartedAt.IsZero())
}

func TestExecutor_Network(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:9999"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	t.Run("Host", func(t *testing.T) {
		host, err := exec.Host(ctx)
		require.NoError(t, err)
		assert.Equal(t, "localhost", host)
	})

	t.Run("MappedPort", func(t *testing.T) {
		port, err := exec.MappedPort(ctx, "80/tcp")
		require.NoError(t, err)
		assert.Equal(t, "9999", port)
	})

	t.Run("Endpoint", func(t *testing.T) {
		endpoint, err := exec.Endpoint(ctx, "80/tcp")
		require.NoError(t, err)
		assert.Equal(t, "localhost:9999", endpoint)
	})

	t.Run("GetAllPorts", func(t *testing.T) {
		ports, err := exec.GetAllPorts(ctx)
		require.NoError(t, err)
		assert.Contains(t, ports, "80/tcp")
		assert.Equal(t, "9999", ports["80/tcp"])
	})

	t.Run("GetNetworks", func(t *testing.T) {
		networks, err := exec.GetNetworks(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, networks)
	})

	t.Run("GetIPAddress", func(t *testing.T) {
		ip, err := exec.GetIPAddress(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, ip)
	})
}

func TestExecutor_Lifecycle(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithName("test-lifecycle"),
	)

	// Start
	err := exec.Start(ctx)
	require.NoError(t, err)

	running, _ := exec.IsRunning(ctx)
	assert.True(t, running)

	// Stop
	err = exec.Stop(ctx)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
	running, _ = exec.IsRunning(ctx)
	assert.False(t, running)

	// Restart
	err = exec.Restart(ctx)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
	running, _ = exec.IsRunning(ctx)
	assert.True(t, running)

	// Terminate
	err = exec.Terminate(ctx)
	require.NoError(t, err)
}

func TestExecutor_EnvironmentVariables(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo $MY_VAR $ANOTHER_VAR"),
		docker.WithEnv("MY_VAR=hello"),
		docker.WithEnvMap(map[string]string{
			"ANOTHER_VAR": "world",
		}),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, _ := exec.Logs(ctx)
	assert.Contains(t, logs, "hello")
	assert.Contains(t, logs, "world")
}

func TestExecutor_WorkDir(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithWorkDir("/tmp"),
		docker.WithCmd("pwd"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, _ := exec.Logs(ctx)
	assert.Contains(t, logs, "/tmp")
}

func TestExecutor_ContainerName(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	containerName := fmt.Sprintf("test-name-%d", time.Now().Unix())

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithName(containerName),
		docker.WithCmd("sleep", "5"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	status, _ := exec.Status(ctx)
	// Docker prepends "/" to container names
	assert.Contains(t, status.Name, containerName)
}

func TestExecutor_MultipleContainers(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	// Start nginx
	nginx, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"),
		docker.WithName("test-multi-nginx"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)

	// Start redis
	redis, _ := docker.New(
		docker.WithImage("redis:7-alpine"),
		docker.WithPorts("6379:0"),
		docker.WithName("test-multi-redis"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("Ready to accept").WithStartupTimeout(30*time.Second),
		),
	)

	// Start both
	err := nginx.Start(ctx)
	require.NoError(t, err)
	defer nginx.Terminate(ctx)

	err = redis.Start(ctx)
	require.NoError(t, err)
	defer redis.Terminate(ctx)

	// Verify both running
	nginxRunning, _ := nginx.IsRunning(ctx)
	redisRunning, _ := redis.IsRunning(ctx)
	assert.True(t, nginxRunning)
	assert.True(t, redisRunning)

	// Verify endpoints
	nginxEndpoint, _ := nginx.Endpoint(ctx, "80/tcp")
	redisEndpoint, _ := redis.Endpoint(ctx, "6379/tcp")
	assert.NotEqual(t, nginxEndpoint, redisEndpoint)
}

func TestExecutor_AutoRemove(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)

	// Wait for container to finish
	_, err = exec.Wait(ctx)
	require.NoError(t, err)

	// Container should auto-remove, status check should fail eventually
	time.Sleep(2 * time.Second)
	_, err = exec.Status(ctx)
	// Error expected because container was auto-removed
	assert.Error(t, err)
}

func TestExecutor_FollowLogs(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in 1 2 3; do echo Line $i; sleep 1; done"),
		docker.WithAutoRemove(true),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	// Capture logs
	var buf strings.Builder
	done := make(chan error, 1)

	go func() {
		done <- exec.FollowLogs(ctx, &buf)
	}()

	// Wait for logs or timeout
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}

	logs := buf.String()
	assert.Contains(t, logs, "Line 1")
}

func TestExecutor_ConnectionString(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:8765"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	connStr, err := exec.ConnectionString(ctx, "80/tcp", "http://%s/api")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8765/api", connStr)
}

func TestExecutor_GetLogsSince(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo start; sleep 2; echo end"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(4 * time.Second)

	// Use a wide window (1 minute) to capture logs generated ~2-4s ago
	logs, err := exec.GetLogsSince(ctx, "1m")
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestExecutor_GetLastNLines(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in $(seq 1 10); do echo Line $i; done"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logs, err := exec.GetLastNLines(ctx, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestExecutor_ErrorHandling(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	t.Run("InvalidImage", func(t *testing.T) {
		exec, _ := docker.New(
			docker.WithImage("this-image-does-not-exist:latest"),
		)

		err := exec.Start(ctx)
		assert.Error(t, err)
	})

	t.Run("PortAlreadyInUse", func(t *testing.T) {
		// Start first container
		exec1, _ := docker.New(
			docker.WithImage("nginx:alpine"),
			docker.WithPorts("80:7777"),
			docker.WithAutoRemove(true),
			docker.WithWaitStrategy(
				docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
			),
		)

		err := exec1.Start(ctx)
		require.NoError(t, err)
		defer exec1.Terminate(ctx)

		// Try to start second container on same port
		exec2, _ := docker.New(
			docker.WithImage("nginx:alpine"),
			docker.WithPorts("80:7777"), // Same port
		)

		err = exec2.Start(ctx)
		assert.Error(t, err) // Should fail
	})
}

func TestExecutor_StdoutStderr(t *testing.T) {
	skipIfNoContainerRuntime(t)
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo stdout message; echo stderr message >&2"),
	)

	err := exec.Start(ctx)
	require.NoError(t, err)
	defer exec.Terminate(ctx)

	time.Sleep(2 * time.Second)

	// Get all logs
	allLogs, err := exec.Logs(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, allLogs)
}

// Benchmark tests
func BenchmarkExecutor_StartStop(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec, _ := docker.New(
			docker.WithImage("alpine:latest"),
			docker.WithCmd("sleep", "1"),
			docker.WithAutoRemove(true),
		)

		exec.Start(ctx)
		exec.Terminate(ctx)
	}
}

func BenchmarkExecutor_GetStatus(b *testing.B) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)

	exec.Start(ctx)
	defer exec.Terminate(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.Status(ctx)
	}
}

func BenchmarkExecutor_GetLogs(b *testing.B) {
	ctx := context.Background()

	exec, _ := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "test log"),
		docker.WithAutoRemove(true),
	)

	exec.Start(ctx)
	defer exec.Terminate(ctx)

	time.Sleep(1 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.Logs(ctx)
	}
}
