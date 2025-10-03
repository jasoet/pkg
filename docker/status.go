package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
)

// Status represents the container status information.
type Status struct {
	// ID is the container ID
	ID string

	// Name is the container name
	Name string

	// Image is the container image
	Image string

	// State is the container state (running, exited, etc.)
	State string

	// Status is the detailed status string
	Status string

	// Running indicates if the container is running
	Running bool

	// Paused indicates if the container is paused
	Paused bool

	// Restarting indicates if the container is restarting
	Restarting bool

	// ExitCode is the exit code (only valid if container has exited)
	ExitCode int

	// Error contains any error message from the container
	Error string

	// StartedAt is when the container started
	StartedAt time.Time

	// FinishedAt is when the container finished (only if stopped)
	FinishedAt time.Time

	// Health is the health check status (if configured)
	Health *HealthStatus
}

// HealthStatus represents container health check status.
type HealthStatus struct {
	// Status is the health status (healthy, unhealthy, starting)
	Status string

	// FailingStreak is the number of consecutive failures
	FailingStreak int

	// Log contains recent health check results
	Log []HealthLog
}

// HealthLog represents a single health check result.
type HealthLog struct {
	// Start is when the check started
	Start time.Time

	// End is when the check completed
	End time.Time

	// ExitCode is the health check exit code
	ExitCode int

	// Output is the health check output
	Output string
}

// Status retrieves the current container status.
func (e *Executor) Status(ctx context.Context) (*Status, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	inspect, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	status := &Status{
		ID:         inspect.ID,
		Name:       inspect.Name,
		Image:      inspect.Config.Image,
		State:      inspect.State.Status,
		Status:     inspect.State.Status,
		Running:    inspect.State.Running,
		Paused:     inspect.State.Paused,
		Restarting: inspect.State.Restarting,
		ExitCode:   inspect.State.ExitCode,
		Error:      inspect.State.Error,
	}

	// Parse timestamps
	if startedAt, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
		status.StartedAt = startedAt
	}
	if finishedAt, err := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt); err == nil {
		status.FinishedAt = finishedAt
	}

	// Parse health status
	if inspect.State.Health != nil {
		status.Health = &HealthStatus{
			Status:        inspect.State.Health.Status,
			FailingStreak: inspect.State.Health.FailingStreak,
			Log:           make([]HealthLog, len(inspect.State.Health.Log)),
		}

		for i, log := range inspect.State.Health.Log {
			status.Health.Log[i] = HealthLog{
				Start:    log.Start,
				End:      log.End,
				ExitCode: log.ExitCode,
				Output:   log.Output,
			}
		}
	}

	return status, nil
}

// IsRunning checks if the container is currently running.
func (e *Executor) IsRunning(ctx context.Context) (bool, error) {
	status, err := e.Status(ctx)
	if err != nil {
		return false, err
	}
	return status.Running, nil
}

// ExitCode retrieves the container exit code.
// Returns an error if the container hasn't exited yet.
func (e *Executor) ExitCode(ctx context.Context) (int, error) {
	status, err := e.Status(ctx)
	if err != nil {
		return 0, err
	}

	if status.Running {
		return 0, fmt.Errorf("container is still running")
	}

	return status.ExitCode, nil
}

// Inspect returns the full container inspection details.
// This provides access to all container metadata.
func (e *Executor) Inspect(ctx context.Context) (*container.InspectResponse, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	inspect, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &inspect, nil
}

// HealthCheck retrieves the current health status.
// Returns an error if health check is not configured for the container.
func (e *Executor) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status, err := e.Status(ctx)
	if err != nil {
		return nil, err
	}

	if status.Health == nil {
		return nil, fmt.Errorf("health check not configured for this container")
	}

	return status.Health, nil
}

// WaitForState waits for the container to reach a specific state.
// Valid states: running, paused, restarting, removing, exited, dead
func (e *Executor) WaitForState(ctx context.Context, targetState string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for state %s", targetState)
		case <-ticker.C:
			status, err := e.Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			if status.State == targetState {
				return nil
			}
		}
	}
}

// WaitForHealthy waits for the container to become healthy.
// Returns an error if the container doesn't have health checks configured.
func (e *Executor) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to be healthy")
		case <-ticker.C:
			health, err := e.HealthCheck(ctx)
			if err != nil {
				return err
			}

			if health.Status == "healthy" {
				return nil
			}

			if health.Status == "unhealthy" {
				return fmt.Errorf("container became unhealthy")
			}
		}
	}
}

// GetStats retrieves container resource usage statistics.
// This includes CPU, memory, network, and disk I/O stats.
// Returns the stats response reader which can be read and decoded by the caller.
// Remember to close the response body after reading.
func (e *Executor) GetStats(ctx context.Context) (container.StatsResponseReader, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	var emptyStats container.StatsResponseReader
	if containerID == "" {
		return emptyStats, fmt.Errorf("container not started")
	}

	stats, err := e.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return emptyStats, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}
