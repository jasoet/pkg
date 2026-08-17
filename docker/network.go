package docker

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// InspectResponse.NetworkSettings is a pointer and the daemon leaves it nil for
// containers without networking (e.g. --network=none). The projection helpers
// below tolerate that instead of panicking, mirroring the guard in
// ContainerTarget.State. They are pure functions of an inspect response so the
// nil-handling is unit-testable without a live daemon.

// portBinding returns the first host port bound to containerPort.
func portBinding(inspect container.InspectResponse, containerPort string) (string, bool) {
	if inspect.NetworkSettings == nil {
		return "", false
	}

	for port, bindings := range inspect.NetworkSettings.Ports {
		if string(port) == containerPort && len(bindings) > 0 {
			return bindings[0].HostPort, true
		}
	}

	return "", false
}

// allPortBindings maps every bound container port to its first host port.
// Exposed-but-unbound ports are omitted. The result is never nil.
func allPortBindings(inspect container.InspectResponse) map[string]string {
	ports := make(map[string]string)
	if inspect.NetworkSettings == nil {
		return ports
	}

	for port, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			ports[string(port)] = bindings[0].HostPort
		}
	}

	return ports
}

// networkNames lists the networks the container is attached to. Never nil.
func networkNames(inspect container.InspectResponse) []string {
	if inspect.NetworkSettings == nil {
		return []string{}
	}

	names := make([]string, 0, len(inspect.NetworkSettings.Networks))
	for name := range inspect.NetworkSettings.Networks {
		names = append(names, name)
	}

	return names
}

// networkIPAddress returns the container's IP on the named network, or on the
// first network that has one when network is empty.
func networkIPAddress(inspect container.InspectResponse, network string) (string, bool) {
	if inspect.NetworkSettings == nil {
		return "", false
	}

	if network != "" {
		settings, ok := inspect.NetworkSettings.Networks[network]
		if !ok || settings == nil {
			return "", false
		}
		return settings.IPAddress, true
	}

	for _, settings := range inspect.NetworkSettings.Networks {
		if settings != nil && settings.IPAddress != "" {
			return settings.IPAddress, true
		}
	}

	return "", false
}

// deriveHost extracts a reachable host from a Docker daemon host URL.
// For remote transports (tcp://, ssh://, http(s)://) it returns the hostname;
// for local transports (unix, npipe) or an empty/unparseable value it falls back
// to defaultHost ("localhost"). This ensures Host()/MappedPort()/Endpoint() point
// at the real daemon when DOCKER_HOST targets a remote engine (e.g. podman-remote).
func deriveHost(daemonHost string) string {
	if daemonHost == "" {
		return defaultHost
	}

	u, err := url.Parse(daemonHost)
	if err != nil {
		return defaultHost
	}

	switch u.Scheme {
	case "tcp", "ssh", "http", "https":
		if h := u.Hostname(); h != "" {
			return h
		}
	}

	return defaultHost
}

// Host returns the container host address.
// For local Docker/Podman this is "localhost"; for a remote daemon (DOCKER_HOST
// set to tcp:// or ssh://) it is the daemon's hostname, since published ports are
// reachable on the daemon host, not the client.
func (e *Executor) Host(_ context.Context) (string, error) {
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

	return deriveHost(cli.DaemonHost()), nil
}

// MappedPort returns the host port mapped to a container port.
// Port format: "8080/tcp" or "8080" (defaults to tcp).
//
// Example:
//
//	hostPort, err := exec.MappedPort(ctx, "80/tcp")
//	// hostPort might be "32768" (randomly assigned by Docker)
func (e *Executor) MappedPort(ctx context.Context, containerPort string) (string, error) {
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

	// Ensure port has protocol
	if !strings.Contains(containerPort, "/") {
		containerPort = containerPort + "/tcp"
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if hostPort, ok := portBinding(inspect, containerPort); ok {
		return hostPort, nil
	}

	return "", fmt.Errorf("port %s not found or not bound", containerPort)
}

// Endpoint returns the full endpoint address (host:port) for a container port.
// This is a convenience method combining Host() and MappedPort().
//
// Example:
//
//	endpoint, err := exec.Endpoint(ctx, "80/tcp")
//	// endpoint might be "localhost:32768"
//
//	// Use it directly with HTTP client
//	resp, err := http.Get("http://" + endpoint + "/health")
func (e *Executor) Endpoint(ctx context.Context, containerPort string) (string, error) {
	host, err := e.Host(ctx)
	if err != nil {
		return "", err
	}

	port, err := e.MappedPort(ctx, containerPort)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}

// GetAllPorts returns all exposed ports and their mappings.
// Returns a map of container ports to host ports.
//
// Example output:
//
//	{
//	    "80/tcp": "8080",
//	    "443/tcp": "8443",
//	}
func (e *Executor) GetAllPorts(ctx context.Context) (map[string]string, error) {
	e.mu.RLock()
	cli := e.client
	containerID := e.containerID
	e.mu.RUnlock()

	if cli == nil {
		return nil, fmt.Errorf("executor is closed")
	}
	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return allPortBindings(inspect), nil
}

// GetNetworks returns all networks the container is connected to.
func (e *Executor) GetNetworks(ctx context.Context) ([]string, error) {
	e.mu.RLock()
	cli := e.client
	containerID := e.containerID
	e.mu.RUnlock()

	if cli == nil {
		return nil, fmt.Errorf("executor is closed")
	}
	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return networkNames(inspect), nil
}

// GetIPAddress returns the container's IP address in a specific network.
// If network is empty, returns the IP from the first available network.
func (e *Executor) GetIPAddress(ctx context.Context, network string) (string, error) {
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

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	ip, ok := networkIPAddress(inspect, network)
	if ok {
		return ip, nil
	}

	if network != "" {
		return "", fmt.Errorf("network %s not found", network)
	}

	return "", fmt.Errorf("no IP address found")
}

// ConnectionString builds a connection string for the container.
// This is useful for database containers.
// Use {{endpoint}} as the placeholder for the host:port value.
//
// Example:
//
//	// For PostgreSQL
//	connStr, err := exec.ConnectionString(ctx, "5432/tcp", "postgres://user:pass@{{endpoint}}/db")
//	// Result: "postgres://user:pass@localhost:32768/db"
func (e *Executor) ConnectionString(ctx context.Context, containerPort, template string) (string, error) {
	endpoint, err := e.Endpoint(ctx, containerPort)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(template, "{{endpoint}}", endpoint), nil
}
