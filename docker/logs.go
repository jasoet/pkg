package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// LogEntry represents a single log entry from the container.
type LogEntry struct {
	// Stream identifies the source (stdout or stderr)
	Stream string

	// Content is the log line content
	Content string

	// Timestamp is when the log was generated (if timestamps enabled)
	Timestamp time.Time
}

// logOptions configures how logs are retrieved.
type logOptions struct {
	stdout     bool
	stderr     bool
	follow     bool
	timestamps bool
	tail       string
	since      string
	until      string
}

// LogOption is a functional option for log retrieval.
type LogOption func(*logOptions)

// defaultLogOptions returns default log options.
func defaultLogOptions() *logOptions {
	return &logOptions{
		stdout:     true,
		stderr:     true,
		follow:     false,
		timestamps: false,
		tail:       "all",
	}
}

// WithStdout includes stdout in the logs (default: true).
func WithStdout(include bool) LogOption {
	return func(o *logOptions) {
		o.stdout = include
	}
}

// WithStderr includes stderr in the logs (default: true).
func WithStderr(include bool) LogOption {
	return func(o *logOptions) {
		o.stderr = include
	}
}

// WithFollow streams logs in real-time (default: false).
func WithFollow() LogOption {
	return func(o *logOptions) {
		o.follow = true
	}
}

// WithTimestamps includes timestamps in log entries (default: false).
func WithTimestamps() LogOption {
	return func(o *logOptions) {
		o.timestamps = true
	}
}

// WithTail limits the number of lines from the end of the logs.
// Use "all" for all logs, or a number like "100" for last 100 lines.
func WithTail(lines string) LogOption {
	return func(o *logOptions) {
		o.tail = lines
	}
}

// WithSince shows logs since a timestamp (RFC3339) or duration (e.g., "10m").
func WithSince(since string) LogOption {
	return func(o *logOptions) {
		o.since = since
	}
}

// WithUntil shows logs until a timestamp (RFC3339) or duration.
func WithUntil(until string) LogOption {
	return func(o *logOptions) {
		o.until = until
	}
}

// FollowLogs streams container logs to the provided writer.
// This is useful for piping logs to stdout or a file.
//
// Example:
//
//	err := exec.FollowLogs(ctx, os.Stdout)
func (e *Executor) FollowLogs(ctx context.Context, w io.Writer, opts ...LogOption) error {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	logOpts := defaultLogOptions()
	logOpts.follow = true // Force follow mode
	for _, opt := range opts {
		opt(logOpts)
	}

	options := container.LogsOptions{
		ShowStdout: logOpts.stdout,
		ShowStderr: logOpts.stderr,
		Timestamps: logOpts.timestamps,
		Follow:     true,
		Tail:       logOpts.tail,
		Since:      logOpts.since,
		Until:      logOpts.until,
	}

	logs, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	// Copy logs to writer (handles Docker's multiplexed stream format)
	_, err = stdcopy.StdCopy(w, w, logs)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error streaming logs: %w", err)
	}

	return nil
}

// GetLogsSince retrieves logs since a specific time.
// Time can be RFC3339 timestamp or duration string (e.g., "10m", "1h").
func (e *Executor) GetLogsSince(ctx context.Context, since string) (string, error) {
	return e.Logs(ctx, WithSince(since))
}

// GetLastNLines retrieves the last N lines of logs.
func (e *Executor) GetLastNLines(ctx context.Context, n int) (string, error) {
	return e.Logs(ctx, WithTail(fmt.Sprintf("%d", n)))
}

// GetStdout retrieves only stdout logs (excludes stderr).
func (e *Executor) GetStdout(ctx context.Context) (string, error) {
	return e.Logs(ctx, WithStdout(true), WithStderr(false))
}

// GetStderr retrieves only stderr logs (excludes stdout).
func (e *Executor) GetStderr(ctx context.Context) (string, error) {
	return e.Logs(ctx, WithStdout(false), WithStderr(true))
}
