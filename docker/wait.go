package docker

import (
	"bufio"
	"context"
	"fmt"
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
	pattern    *regexp.Regexp
	compileErr error
	timeout    time.Duration
}

// WaitForLog creates a wait strategy that waits for a log pattern.
// Pattern can be a simple string or regex pattern.
// If the pattern is invalid, WaitUntilReady will return an error instead of panicking.
func WaitForLog(pattern string) *waitForLog {
	compiled, err := regexp.Compile(pattern)
	return &waitForLog{
		pattern:    compiled,
		compileErr: err,
		timeout:    60 * time.Second,
	}
}

// WithStartupTimeout sets the timeout for the wait strategy.
func (w *waitForLog) WithStartupTimeout(timeout time.Duration) *waitForLog {
	w.timeout = timeout
	return w
}

// WaitUntilReady implements WaitStrategy.
func (w *waitForLog) WaitUntilReady(ctx context.Context, cli *client.Client, containerID string) error {
	if w.compileErr != nil {
		return fmt.Errorf("invalid regex pattern: %w", w.compileErr)
	}

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
	defer func() { _ = logs.Close() }()

	// Use bufio.Scanner to read complete lines, avoiding chunk-boundary false negatives
	// where a pattern could be split across two Read calls.
	// ContainerLogs respects context cancellation, so Scan() will unblock when the timeout fires.
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		if w.pattern.MatchString(scanner.Text()) {
			return nil
		}
	}

	if ctx.Err() != nil {
		return fmt.Errorf("timeout waiting for log pattern: %s", w.pattern.String())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}
	return fmt.Errorf("container stopped before log pattern found: %s", w.pattern.String())
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

					conn, err := (&net.Dialer{Timeout: 1 * time.Second}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", host, hostPort))
					if err == nil {
						_ = conn.Close()
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
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						continue
					}
					resp, err := httpClient.Do(req)
					if err == nil {
						_ = resp.Body.Close()
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
// Note on timeout behavior: multiWait sets an outer deadline via WithStartupTimeout (default 120s).
// Each child strategy also has its own independent timeout. The effective timeout for each child
// is min(child timeout, remaining time on the outer deadline). To avoid unexpected early expiration,
// set the outer timeout to be at least as large as the sum of all child timeouts, or rely solely on
// the outer timeout by setting child timeouts to a large value.
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
