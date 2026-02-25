package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/go-connections/nat"
)

// Host returns the container host address.
// For local Docker, this is always "localhost" since containers use port forwarding.
func (e *Executor) Host(_ context.Context) (string, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return "", fmt.Errorf("container not started")
	}

	return defaultHost, nil
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
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return "", fmt.Errorf("container not started")
	}

	// Ensure port has protocol
	if !strings.Contains(containerPort, "/") {
		containerPort = containerPort + "/tcp"
	}

	inspect, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	// Find the port binding
	for port, bindings := range inspect.NetworkSettings.Ports {
		if string(port) == containerPort && len(bindings) > 0 {
			return bindings[0].HostPort, nil
		}
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
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	inspect, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	ports := make(map[string]string)
	for port, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			ports[string(port)] = bindings[0].HostPort
		}
	}

	return ports, nil
}

// GetNetworks returns all networks the container is connected to.
func (e *Executor) GetNetworks(ctx context.Context) ([]string, error) {
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

	networks := make([]string, 0, len(inspect.NetworkSettings.Networks))
	for name := range inspect.NetworkSettings.Networks {
		networks = append(networks, name)
	}

	return networks, nil
}

// GetIPAddress returns the container's IP address in a specific network.
// If network is empty, returns the IP from the first available network.
func (e *Executor) GetIPAddress(ctx context.Context, network string) (string, error) {
	e.mu.RLock()
	containerID := e.containerID
	e.mu.RUnlock()

	if containerID == "" {
		return "", fmt.Errorf("container not started")
	}

	inspect, err := e.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if network != "" {
		// Get IP from specific network
		if netSettings, ok := inspect.NetworkSettings.Networks[network]; ok {
			return netSettings.IPAddress, nil
		}
		return "", fmt.Errorf("network %s not found", network)
	}

	// Return IP from first available network
	for _, netSettings := range inspect.NetworkSettings.Networks {
		if netSettings.IPAddress != "" {
			return netSettings.IPAddress, nil
		}
	}

	return "", fmt.Errorf("no IP address found")
}

// ConnectionString builds a connection string for the container.
// This is useful for database containers.
//
// Example:
//
//	// For PostgreSQL
//	connStr, err := exec.ConnectionString(ctx, "5432/tcp", "postgres://user:pass@%s/db")
//	// Result: "postgres://user:pass@localhost:32768/db"
func (e *Executor) ConnectionString(ctx context.Context, containerPort, template string) (string, error) {
	endpoint, err := e.Endpoint(ctx, containerPort)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(template, endpoint), nil
}

// NatPort is a helper to create a nat.Port from a string.
// Port format: "8080/tcp", "8080/udp", or "8080" (defaults to tcp).
// This is useful when working with Docker API types directly.
func NatPort(port string) (nat.Port, error) {
	if !strings.Contains(port, "/") {
		port = port + "/tcp"
	}
	parts := strings.SplitN(port, "/", 2)
	return nat.NewPort(parts[1], parts[0])
}

// PortBindings is a helper to create port bindings from a map.
// This is useful for programmatically building port configurations.
//
// Example:
//
//	bindings := docker.PortBindings(map[string]string{
//	    "80/tcp": "8080",
//	    "443/tcp": "8443",
//	})
func PortBindings(ports map[string]string) (nat.PortMap, error) {
	portMap := make(nat.PortMap)

	for containerPort, hostPort := range ports {
		natPort, err := NatPort(containerPort)
		if err != nil {
			return nil, fmt.Errorf("invalid container port %s: %w", containerPort, err)
		}

		portMap[natPort] = []nat.PortBinding{
			{HostPort: hostPort},
		}
	}

	return portMap, nil
}

// ExposedPorts creates a nat.PortSet from a slice of port strings.
// This is useful for programmatically building exposed ports.
//
// Example:
//
//	ports := docker.ExposedPorts([]string{"80/tcp", "443/tcp"})
func ExposedPorts(ports []string) (nat.PortSet, error) {
	portSet := make(nat.PortSet)

	for _, port := range ports {
		natPort, err := NatPort(port)
		if err != nil {
			return nil, fmt.Errorf("invalid port %s: %w", port, err)
		}
		portSet[natPort] = struct{}{}
	}

	return portSet, nil
}
