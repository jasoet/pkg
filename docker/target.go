package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ContainerState is a snapshot of a container's runtime state,
// projected from ContainerInspect into a library-owned type.
type ContainerState struct {
	Running      bool
	HealthStatus string              // "" when no healthcheck is defined
	Ports        map[string][]string // containerPort ("80/tcp") → hostPorts
}

// ContainerTarget is the runtime surface a WaitStrategy can inspect.
// It wraps the Docker client and container ID internally so that
// strategies never need to import the docker client.
type ContainerTarget struct {
	cli         *client.Client
	containerID string
}

// newContainerTarget constructs a ContainerTarget for the given container.
func newContainerTarget(cli *client.Client, containerID string) ContainerTarget {
	return ContainerTarget{cli: cli, containerID: containerID}
}

// ID returns the container ID.
func (t ContainerTarget) ID() string {
	return t.containerID
}

// Logs streams the container's stdout and stderr (follow mode).
// The caller is responsible for closing the returned reader.
func (t ContainerTarget) Logs(ctx context.Context) (io.ReadCloser, error) {
	return t.cli.ContainerLogs(ctx, t.containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
}

// State inspects the container and projects the result into a ContainerState.
func (t ContainerTarget) State(ctx context.Context) (ContainerState, error) {
	inspect, err := t.cli.ContainerInspect(ctx, t.containerID)
	if err != nil {
		return ContainerState{}, fmt.Errorf("failed to inspect container: %w", err)
	}

	state := ContainerState{
		Running: inspect.State.Running,
		Ports:   make(map[string][]string, len(inspect.NetworkSettings.Ports)),
	}
	if inspect.State.Health != nil {
		state.HealthStatus = inspect.State.Health.Status
	}
	for containerPort, bindings := range inspect.NetworkSettings.Ports {
		hostPorts := make([]string, 0, len(bindings))
		for _, binding := range bindings {
			hostPorts = append(hostPorts, binding.HostPort)
		}
		state.Ports[string(containerPort)] = hostPorts
	}
	return state, nil
}
