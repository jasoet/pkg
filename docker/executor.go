package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"go.opentelemetry.io/otel/trace"
)

// Executor manages a Docker container lifecycle.
type Executor struct {
	config      *config
	client      *client.Client
	containerID string
	mu          sync.RWMutex
	otel        *otelInstrumentation
}

// New creates a new Docker executor with functional options.
//
// Example with functional options:
//
//	exec := docker.New(
//	    docker.WithImage("nginx:latest"),
//	    docker.WithPorts("80:8080"),
//	    docker.WithEnv("KEY=value"),
//	)
//
// Example with ContainerRequest struct:
//
//	req := docker.ContainerRequest{
//	    Image: "nginx:latest",
//	    ExposedPorts: []string{"80/tcp"},
//	    Env: map[string]string{"KEY": "value"},
//	}
//	exec := docker.New(docker.WithRequest(req))
//
// Example combining both:
//
//	req := docker.ContainerRequest{Image: "nginx:latest"}
//	exec := docker.New(
//	    docker.WithRequest(req),
//	    docker.WithOTelConfig(otelCfg), // Add observability
//	)
func New(opts ...Option) (*Executor, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	exec := &Executor{
		config: cfg,
		client: cli,
	}

	// Initialize OTel if configured
	if cfg.otelConfig != nil {
		exec.otel = newOTelInstrumentation(cfg.otelConfig)
	}

	return exec, nil
}

// NewFromRequest creates a new Docker executor from a ContainerRequest struct.
// This is a convenience function equivalent to: New(WithRequest(req))
func NewFromRequest(req ContainerRequest) (*Executor, error) {
	return New(WithRequest(req))
}

// Start pulls the image (if needed), creates and starts the container.
// If a wait strategy is configured, it blocks until the container is ready.
func (e *Executor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Start")
		defer span.End()
	}

	// Pull image
	if err := e.pullImage(ctx); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "pull_image_error", err)
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container
	containerID, err := e.createContainer(ctx)
	if err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "create_container_error", err)
		}
		return fmt.Errorf("failed to create container: %w", err)
	}
	e.containerID = containerID

	// Start container
	if err := e.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "start_container_error", err)
		}
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for readiness if strategy is configured
	if e.config.waitStrategy != nil {
		if err := e.config.waitStrategy.WaitUntilReady(ctx, e.client, containerID); err != nil {
			if e.otel != nil {
				e.otel.recordError(ctx, "wait_strategy_error", err)
			}
			// Container failed to become ready, clean up
			_ = e.terminate(ctx)
			return fmt.Errorf("container failed to become ready: %w", err)
		}
	}

	if e.otel != nil {
		e.otel.incrementCounter(ctx, "containers_started", 1)
	}

	return nil
}

// Stop gracefully stops the container (sends SIGTERM).
// The container can still be restarted after stopping.
func (e *Executor) Stop(ctx context.Context) error {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Stop")
		defer span.End()
	}

	timeout := int(e.config.timeout.Seconds())
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := e.client.ContainerStop(ctx, containerID, stopOptions); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "stop_container_error", err)
		}
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if e.otel != nil {
		e.otel.incrementCounter(ctx, "containers_stopped", 1)
	}

	return nil
}

// Terminate forcefully stops and removes the container.
// This is a destructive operation and the container cannot be restarted.
func (e *Executor) Terminate(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.terminate(ctx)
}

// terminate is the internal implementation of Terminate (without locking).
func (e *Executor) terminate(ctx context.Context) error {
	if e.containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Terminate")
		defer span.End()
	}

	// Remove container (force stop if running)
	removeOptions := container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	if err := e.client.ContainerRemove(ctx, e.containerID, removeOptions); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "terminate_container_error", err)
		}
		return fmt.Errorf("failed to remove container: %w", err)
	}

	if e.otel != nil {
		e.otel.incrementCounter(ctx, "containers_terminated", 1)
	}

	e.containerID = ""
	return nil
}

// Restart restarts the container.
func (e *Executor) Restart(ctx context.Context) error {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Restart")
		defer span.End()
	}

	timeout := int(e.config.timeout.Seconds())
	restartOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := e.client.ContainerRestart(ctx, containerID, restartOptions); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "restart_container_error", err)
		}
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if e.otel != nil {
		e.otel.incrementCounter(ctx, "containers_restarted", 1)
	}

	return nil
}

// Wait blocks until the container stops and returns its exit code.
func (e *Executor) Wait(ctx context.Context) (int64, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return 0, fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Wait")
		defer span.End()
	}

	statusCh, errCh := e.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if e.otel != nil {
			e.otel.recordError(ctx, "wait_container_error", err)
		}
		return 0, fmt.Errorf("error waiting for container: %w", err)
	case status := <-statusCh:
		return status.StatusCode, nil
	}
}

// ContainerID returns the Docker container ID.
// Returns empty string if container hasn't been started yet.
func (e *Executor) ContainerID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.containerID
}

// Close closes the Docker client connection.
// The container is NOT terminated automatically - call Terminate() first if needed.
func (e *Executor) Close() error {
	return e.client.Close()
}

// pullImage pulls the container image if not already present.
func (e *Executor) pullImage(ctx context.Context) error {
	// Check if image exists locally
	_, _, err := e.client.ImageInspectWithRaw(ctx, e.config.image)
	if err == nil {
		// Image already exists
		return nil
	}

	// Pull image
	reader, err := e.client.ImagePull(ctx, e.config.image, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Consume output to ensure pull completes
	_, err = io.Copy(io.Discard, reader)
	return err
}

// createContainer creates the container with configured options.
func (e *Executor) createContainer(ctx context.Context) (string, error) {
	// Build environment variables slice
	var env []string
	for k, v := range e.config.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        e.config.image,
		Cmd:          e.config.cmd,
		Entrypoint:   e.config.entrypoint,
		Env:          env,
		ExposedPorts: e.config.exposedPorts,
		Labels:       e.config.labels,
		WorkingDir:   e.config.workDir,
		User:         e.config.user,
		Hostname:     e.config.hostname,
		Volumes:      e.config.volumes,
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: e.config.portBindings,
		Binds:        e.config.binds,
		AutoRemove:   e.config.autoRemove,
		Privileged:   e.config.privileged,
		CapAdd:       e.config.capAdd,
		CapDrop:      e.config.capDrop,
		Tmpfs:        e.config.tmpfs,
		ShmSize:      e.config.shmSize,
	}

	// Set network mode if specified
	if e.config.networkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(e.config.networkMode)
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{}
	if len(e.config.networks) > 0 {
		endpoints := make(map[string]*network.EndpointSettings)
		for _, net := range e.config.networks {
			endpoints[net] = &network.EndpointSettings{}
		}
		networkConfig.EndpointsConfig = endpoints
	}

	// Create container
	resp, err := e.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		e.config.name,
	)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// Logs returns all container logs as a string.
// Use LogOptions for more control.
func (e *Executor) Logs(ctx context.Context, opts ...LogOption) (string, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return "", fmt.Errorf("container not started")
	}

	logOpts := defaultLogOptions()
	for _, opt := range opts {
		opt(logOpts)
	}

	options := container.LogsOptions{
		ShowStdout: logOpts.stdout,
		ShowStderr: logOpts.stderr,
		Timestamps: logOpts.timestamps,
		Follow:     false,
		Tail:       logOpts.tail,
	}

	logs, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	// Read all logs
	var buf strings.Builder
	_, err = stdcopy.StdCopy(&buf, &buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// StreamLogs streams container logs to a channel.
// The channel is closed when streaming completes or context is cancelled.
func (e *Executor) StreamLogs(ctx context.Context, opts ...LogOption) (<-chan LogEntry, <-chan error) {
	logCh := make(chan LogEntry, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(logCh)
		defer close(errCh)

		e.mu.RLock()
		containerID := e.containerID
		e.mu.RUnlock()

		if containerID == "" {
			errCh <- fmt.Errorf("container not started")
			return
		}

		logOpts := defaultLogOptions()
		for _, opt := range opts {
			opt(logOpts)
		}

		options := container.LogsOptions{
			ShowStdout: logOpts.stdout,
			ShowStderr: logOpts.stderr,
			Timestamps: logOpts.timestamps,
			Follow:     logOpts.follow,
			Tail:       logOpts.tail,
		}

		logs, err := e.client.ContainerLogs(ctx, containerID, options)
		if err != nil {
			errCh <- fmt.Errorf("failed to get logs: %w", err)
			return
		}
		defer logs.Close()

		// Stream logs line by line
		buf := make([]byte, 8192)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := logs.Read(buf)
				if err != nil {
					if err != io.EOF {
						errCh <- fmt.Errorf("error reading logs: %w", err)
					}
					return
				}

				if n > 0 {
					entry := LogEntry{
						Stream:  "stdout", // Docker API doesn't distinguish in this mode
						Content: string(buf[:n]),
					}
					select {
					case logCh <- entry:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return logCh, errCh
}
