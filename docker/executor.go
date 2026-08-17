package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// stopTimeoutSeconds converts a stop/restart timeout into whole seconds for the
// Docker API, rounding any positive sub-second duration up to 1s so a small
// timeout never truncates to 0 (which Docker interprets as an immediate SIGKILL).
func stopTimeoutSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	secs := int(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	return secs
}

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
// Additional options can be passed to override or extend the request configuration.
//
// Example with just struct:
//
//	req := docker.ContainerRequest{
//	    Image: "nginx:latest",
//	    Env: map[string]string{"KEY": "value"},
//	}
//	exec, err := docker.NewFromRequest(req)
//
// Example with additional options (options override struct fields):
//
//	req := docker.ContainerRequest{
//	    Image: "nginx:latest",
//	    Name:  "default-name",
//	}
//	exec, err := docker.NewFromRequest(req,
//	    docker.WithName("override-name"),  // Overrides struct name
//	    docker.WithOTelConfig(otelCfg),    // Adds observability
//	    docker.WithPorts("80:8080"),       // Adds port mapping
//	)
func NewFromRequest(req ContainerRequest, opts ...Option) (*Executor, error) {
	// Prepend WithRequest so additional options can override struct fields
	allOpts := append([]Option{WithRequest(req)}, opts...)
	return New(allOpts...)
}

// Start pulls the image (if needed), creates and starts the container.
// If a wait strategy is configured, it blocks until the container is ready.
func (e *Executor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.client == nil {
		return fmt.Errorf("executor has been closed")
	}

	// Guard against double-start which would leak containers.
	if e.containerID != "" {
		return fmt.Errorf("container already started: %s", e.containerID)
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Start")
		defer span.End()
		e.otel.addSpanAttributes(ctx, attribute.String("docker.image", e.config.image))
	}

	// Pull image
	if err := e.pullImage(ctx); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "pull_image_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to pull image")
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container
	containerID, err := e.createContainer(ctx)
	if err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "create_container_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to create container")
		}
		return fmt.Errorf("failed to create container: %w", err)
	}
	e.containerID = containerID

	// Start container
	if err := e.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "start_container_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to start container")
		}
		// The container was created but never started: remove it and clear the
		// ID so the executor is reusable and no dangling container is leaked.
		// Use a detached context so a canceled caller ctx still allows cleanup.
		_ = e.terminate(context.WithoutCancel(ctx)) //nolint:errcheck // best effort cleanup
		e.containerID = ""                          // reset unconditionally so a failed removal cannot wedge the executor
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for readiness if strategy is configured
	if e.config.waitStrategy != nil {
		if err := e.config.waitStrategy.WaitUntilReady(ctx, newContainerTarget(e.client, containerID)); err != nil {
			if e.otel != nil {
				e.otel.recordError(ctx, "wait_strategy_error", err)
				e.otel.setSpanStatus(ctx, 1, "container failed to become ready")
			}
			// Container failed to become ready, clean up. Use a detached context so
			// a canceled/expired caller ctx cannot leave the container running.
			_ = e.terminate(context.WithoutCancel(ctx)) //nolint:errcheck // Best effort cleanup, original error is more important
			return fmt.Errorf("container failed to become ready: %w", err)
		}
	}

	if e.otel != nil {
		e.otel.addSpanAttributes(ctx, attribute.String("docker.container.id", containerID))
		e.otel.setSpanStatus(ctx, 0, "container started")
		e.otel.incrementCounter(ctx, "containers_started", 1)
	}

	return nil
}

// Stop gracefully stops the container (sends SIGTERM).
// The container can still be restarted after stopping.
// Note: There is a small TOCTOU window between the containerID check and the Docker API call.
// Concurrent Terminate() may cause a benign "container not found" error.
func (e *Executor) Stop(ctx context.Context) error {
	e.mu.RLock()
	cli := e.client
	containerID := e.containerID
	e.mu.RUnlock()

	if cli == nil {
		return fmt.Errorf("executor is closed")
	}
	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Stop")
		defer span.End()
	}

	timeout := stopTimeoutSeconds(e.config.timeout)
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "stop_container_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to stop container")
		}
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if e.otel != nil {
		e.otel.setSpanStatus(ctx, 0, "container stopped")
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
	if e.client == nil {
		return fmt.Errorf("executor is closed")
	}
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

	// A container that is already gone (e.g. AutoRemove removed it on exit, or a
	// concurrent Terminate won the race) is treated as success: the desired
	// end-state — no such container — has been reached.
	if err := e.client.ContainerRemove(ctx, e.containerID, removeOptions); err != nil && !cerrdefs.IsNotFound(err) {
		if e.otel != nil {
			e.otel.recordError(ctx, "terminate_container_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to remove container")
		}
		return fmt.Errorf("failed to remove container: %w", err)
	}

	if e.otel != nil {
		e.otel.setSpanStatus(ctx, 0, "container terminated")
		e.otel.incrementCounter(ctx, "containers_terminated", 1)
	}

	e.containerID = ""
	return nil
}

// Restart restarts the container.
// Note: There is a small TOCTOU window between the containerID check and the Docker API call.
// Concurrent Terminate() may cause a benign "container not found" error.
func (e *Executor) Restart(ctx context.Context) error {
	e.mu.RLock()
	cli := e.client
	containerID := e.containerID
	e.mu.RUnlock()

	if cli == nil {
		return fmt.Errorf("executor is closed")
	}
	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Restart")
		defer span.End()
	}

	timeout := stopTimeoutSeconds(e.config.timeout)
	restartOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := cli.ContainerRestart(ctx, containerID, restartOptions); err != nil {
		if e.otel != nil {
			e.otel.recordError(ctx, "restart_container_error", err)
			e.otel.setSpanStatus(ctx, 1, "failed to restart container")
		}
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if e.otel != nil {
		e.otel.setSpanStatus(ctx, 0, "container restarted")
		e.otel.incrementCounter(ctx, "containers_restarted", 1)
	}

	return nil
}

// Wait blocks until the container stops and returns its exit code.
func (e *Executor) Wait(ctx context.Context) (int64, error) {
	e.mu.RLock()
	cli := e.client
	containerID := e.containerID
	autoRemove := e.config.autoRemove
	e.mu.RUnlock()

	if cli == nil {
		return 0, fmt.Errorf("executor is closed")
	}
	if containerID == "" {
		return 0, fmt.Errorf("container not started")
	}

	// Trace with OTel
	if e.otel != nil {
		var span trace.Span
		ctx, span = e.otel.startSpan(ctx, "docker.Wait")
		defer span.End()
	}

	// With AutoRemove the daemon deletes the container as soon as it stops, which
	// races WaitConditionNotRunning and yields a spurious "No such container".
	// Waiting for removal instead observes the exit code before the container vanishes.
	condition := container.WaitConditionNotRunning
	if autoRemove {
		condition = container.WaitConditionRemoved
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerID, condition)
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
// After Close(), any method that uses the Docker client will return an error.
func (e *Executor) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.client == nil {
		return nil
	}
	err := e.client.Close()
	e.client = nil
	return err
}

// pullImage pulls the container image if not already present.
func (e *Executor) pullImage(ctx context.Context) error {
	// Check if image exists locally
	_, err := e.client.ImageInspect(ctx, e.config.image)
	if err == nil {
		return nil // image exists
	}
	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect image %s: %w", e.config.image, err)
	}
	// Image not found, proceed to pull

	// Pull image
	reader, err := e.client.ImagePull(ctx, e.config.image, image.PullOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	// Docker streams pull progress as newline-delimited JSON on a 200 response and
	// reports failures (auth, missing manifest, etc.) as an "errorDetail" message
	// inside that stream rather than via the initial error. Decoding the stream
	// surfaces those in-band errors instead of silently discarding them, which
	// would otherwise only manifest later as a confusing "No such image".
	if err := jsonmessage.DisplayJSONMessagesStream(reader, io.Discard, 0, false, nil); err != nil {
		return err
	}
	return nil
}

// createContainer creates the container with configured options.
func (e *Executor) createContainer(ctx context.Context) (string, error) {
	// Build environment variables slice
	env := make([]string, 0, len(e.config.env))
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
	cli := e.client
	containerID := e.containerID
	e.mu.RUnlock()

	if cli == nil {
		return "", fmt.Errorf("executor is closed")
	}
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
		Since:      logOpts.since,
		Until:      logOpts.until,
	}

	logs, err := cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer func() { _ = logs.Close() }()

	// Read all logs
	var buf strings.Builder
	_, err = stdcopy.StdCopy(&buf, &buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// logChanWriter is an io.Writer that turns Docker's demultiplexed log bytes into
// line-oriented LogEntry values on a channel. It buffers partial lines across
// writes and emits one entry per complete line. Sends respect ctx cancellation so
// an abandoned consumer cannot wedge the writer.
type logChanWriter struct {
	ctx    context.Context
	ch     chan<- LogEntry
	stream string
	buf    bytes.Buffer
}

func (w *logChanWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// No newline yet: retain the partial line for the next write.
			w.buf.Reset()
			w.buf.WriteString(line)
			break
		}
		if err := w.emit(strings.TrimRight(line, "\n")); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// flush emits any buffered trailing content that was not newline-terminated.
func (w *logChanWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	_ = w.emit(strings.TrimRight(w.buf.String(), "\n"))
	w.buf.Reset()
}

func (w *logChanWriter) emit(content string) error {
	select {
	case w.ch <- LogEntry{Stream: w.stream, Content: content}:
		return nil
	case <-w.ctx.Done():
		return w.ctx.Err()
	}
}

// StreamLogs streams container logs to a channel.
//
// The context MUST be cancelable: streaming (especially with WithFollow) blocks
// until the container's log stream ends or ctx is canceled. Cancel ctx when done
// to release the background goroutine and underlying connection; abandoning the
// returned channels without canceling ctx leaks both. The error channel is
// buffered and closed alongside the log channel, so it is safe to ignore.
//
// Docker's multiplexed stream is demultiplexed with stdcopy, so stdout and stderr
// frames are labeled correctly and malformed frame sizes cannot trigger huge
// allocations.
func (e *Executor) StreamLogs(ctx context.Context, opts ...LogOption) (<-chan LogEntry, <-chan error) {
	logCh := make(chan LogEntry, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(logCh)
		defer close(errCh)

		e.mu.RLock()
		cli := e.client
		containerID := e.containerID
		e.mu.RUnlock()

		if cli == nil {
			errCh <- fmt.Errorf("executor is closed")
			return
		}
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
			Since:      logOpts.since,
			Until:      logOpts.until,
		}

		logs, err := cli.ContainerLogs(ctx, containerID, options)
		if err != nil {
			errCh <- fmt.Errorf("failed to get logs: %w", err)
			return
		}
		defer func() { _ = logs.Close() }()

		stdoutW := &logChanWriter{ctx: ctx, ch: logCh, stream: "stdout"}
		stderrW := &logChanWriter{ctx: ctx, ch: logCh, stream: "stderr"}
		_, err = stdcopy.StdCopy(stdoutW, stderrW, logs)
		stdoutW.flush()
		stderrW.flush()
		if err != nil && ctx.Err() == nil {
			errCh <- fmt.Errorf("error streaming logs: %w", err)
		}
	}()

	return logCh, errCh
}
