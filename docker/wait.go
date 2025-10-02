package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// WaitStrategy defines how to wait for a container to be ready.
type WaitStrategy interface {
	// WaitUntilReady blocks until the container is ready or timeout occurs.
	WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error
}

// waitForLog waits for a specific log pattern to appear.
type waitForLog struct {
	pattern *regexp.Regexp
	timeout time.Duration
}

// WaitForLog creates a wait strategy that waits for a log pattern.
// Pattern can be a simple string or regex pattern.
func WaitForLog(pattern string) *waitForLog {
	return &waitForLog{
		pattern: regexp.MustCompile(pattern),
		timeout: 60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitForLog) WithStartupTimeout(timeout time.Duration) *waitForLog {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitForLog) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Stream logs and search for pattern
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	}

	logs, err := cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	// Read logs line by line
	buf := make([]byte, 8192)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for log pattern: %s", w.pattern.String())
		default:
			n, err := logs.Read(buf)
			if err != nil {
				if err == io.EOF {
					// Container stopped before pattern found
					return fmt.Errorf("container stopped before log pattern found: %s", w.pattern.String())
				}
				return fmt.Errorf("error reading logs: %w", err)
			}

			// Check if pattern matches
			line := string(buf[:n])
			if w.pattern.MatchString(line) {
				return nil
			}
		}
	}
}

// waitForPort waits for a port to be listening.
type waitForPort struct {
	port    string
	timeout time.Duration
}

// WaitForPort creates a wait strategy that waits for a port to be listening.
// Port format: "8080/tcp" or just "8080" (defaults to tcp).
func WaitForPort(port string) *waitForPort {
	// Ensure port has protocol
	if !strings.Contains(port, "/") {
		port = port + "/tcp"
	}

	return &waitForPort{
		port:    port,
		timeout: 60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitForPort) WithStartupTimeout(timeout time.Duration) *waitForPort {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitForPort) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for port %s", w.port)
		case <-ticker.C:
			// Get container details
			inspect, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			// Check if container is still running
			if !inspect.State.Running {
				return fmt.Errorf("container stopped while waiting for port %s", w.port)
			}

			// Get mapped port
			portBindings := inspect.NetworkSettings.Ports
			for containerPort, bindings := range portBindings {
				if string(containerPort) == w.port && len(bindings) > 0 {
					// Try to connect to the port
					host := "localhost"
					hostPort := bindings[0].HostPort

					conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, hostPort), 1*time.Second)
					if err == nil {
						conn.Close()
						return nil // Port is ready
					}
				}
			}
		}
	}
}

// waitForHTTP waits for an HTTP endpoint to return expected status.
type waitForHTTP struct {
	port           string
	path           string
	expectedStatus int
	timeout        time.Duration
}

// WaitForHTTP creates a wait strategy that waits for an HTTP endpoint.
// Port format: "8080" or "8080/tcp"
// Path: HTTP path (e.g., "/health")
// Expected status: HTTP status code (e.g., 200)
func WaitForHTTP(port, path string, expectedStatus int) *waitForHTTP {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if !strings.Contains(port, "/") {
		port = port + "/tcp"
	}

	return &waitForHTTP{
		port:           port,
		path:           path,
		expectedStatus: expectedStatus,
		timeout:        60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitForHTTP) WithStartupTimeout(timeout time.Duration) *waitForHTTP {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitForHTTP) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for HTTP %s on port %s", w.path, w.port)
		case <-ticker.C:
			// Get container details
			inspect, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			// Check if container is still running
			if !inspect.State.Running {
				return fmt.Errorf("container stopped while waiting for HTTP endpoint")
			}

			// Get mapped port
			portBindings := inspect.NetworkSettings.Ports
			for containerPort, bindings := range portBindings {
				if string(containerPort) == w.port && len(bindings) > 0 {
					host := "localhost"
					hostPort := bindings[0].HostPort

					url := fmt.Sprintf("http://%s:%s%s", host, hostPort, w.path)
					resp, err := httpClient.Get(url)
					if err == nil {
						resp.Body.Close()
						if resp.StatusCode == w.expectedStatus {
							return nil // Endpoint is ready
						}
					}
				}
			}
		}
	}
}

// waitForHealthy waits for container health check to be healthy.
type waitForHealthy struct {
	timeout time.Duration
}

// WaitForHealthy creates a wait strategy that waits for health check to pass.
// Container must have a HEALTHCHECK defined in Dockerfile or via Docker API.
func WaitForHealthy() *waitForHealthy {
	return &waitForHealthy{
		timeout: 60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitForHealthy) WithStartupTimeout(timeout time.Duration) *waitForHealthy {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitForHealthy) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to be healthy")
		case <-ticker.C:
			inspect, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			// Check if container is still running
			if !inspect.State.Running {
				return fmt.Errorf("container stopped before becoming healthy")
			}

			// Check health status
			if inspect.State.Health != nil && inspect.State.Health.Status == "healthy" {
				return nil
			}
		}
	}
}

// waitFunc wraps a custom wait function.
type waitFunc struct {
	fn      func(ctx context.Context, cli *client.Client, containerID string) error
	timeout time.Duration
}

// WaitForFunc creates a wait strategy from a custom function.
// This allows users to implement their own wait logic.
func WaitForFunc(fn func(ctx context.Context, cli *client.Client, containerID string) error) *waitFunc {
	return &waitFunc{
		fn:      fn,
		timeout: 60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitFunc) WithStartupTimeout(timeout time.Duration) *waitFunc {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitFunc) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	return w.fn(ctx, cli, containerID)
}

// multiWait combines multiple wait strategies (ALL must pass).
type multiWait struct {
	strategies []WaitStrategy
	timeout    time.Duration
}

// WaitForAll creates a wait strategy that waits for all strategies to pass.
func WaitForAll(strategies ...WaitStrategy) *multiWait {
	return &multiWait{
		strategies: strategies,
		timeout:    120 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *multiWait) WithStartupTimeout(timeout time.Duration) *multiWait {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *multiWait) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	for i, strategy := range w.strategies {
		if err := strategy.WaitUntilReady(ctx, cli, containerID); err != nil {
			return fmt.Errorf("wait strategy %d failed: %w", i, err)
		}
	}

	return nil
}

// ForListeningPort is an alias for WaitForPort for testcontainers compatibility.
func ForListeningPort(port string) *waitForPort {
	return WaitForPort(port)
}
