package docker

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jasoet/pkg/v2/otel"
)

// ContainerRequest represents a declarative container configuration,
// similar to testcontainers.ContainerRequest.
// This allows users to configure containers using a struct-based approach.
type ContainerRequest struct {
	// Image is the container image to use (e.g., "nginx:latest")
	Image string

	// Name is the container name (optional)
	Name string

	// Hostname sets the container hostname
	Hostname string

	// ExposedPorts are ports to expose (e.g., []string{"80/tcp", "443/tcp"})
	ExposedPorts []string

	// Env is a map of environment variables
	Env map[string]string

	// Cmd overrides the default command
	Cmd []string

	// Entrypoint overrides the default entrypoint
	Entrypoint []string

	// WorkingDir sets the working directory
	WorkingDir string

	// User sets the user (e.g., "1000:1000" or "username")
	User string

	// Labels are container labels
	Labels map[string]string

	// Volumes maps host paths to container paths
	// Key: host path, Value: container path
	Volumes map[string]string

	// BindMounts provides detailed volume configuration
	// Key: container path, Value: bind mount spec
	BindMounts map[string]string

	// Networks to attach the container to
	Networks []string

	// NetworkMode sets the network mode (bridge, host, none, container:<name>)
	NetworkMode string

	// PortBindings maps container ports to host ports
	// Key: container port (e.g., "80/tcp"), Value: host port (e.g., "8080")
	PortBindings map[string]string

	// AutoRemove automatically removes the container when it stops
	AutoRemove bool

	// Privileged runs the container in privileged mode
	Privileged bool

	// CapAdd adds Linux capabilities
	CapAdd []string

	// CapDrop drops Linux capabilities
	CapDrop []string

	// Tmpfs mounts tmpfs filesystems
	// Key: container path, Value: options
	Tmpfs map[string]string

	// ShmSize sets the size of /dev/shm
	ShmSize int64

	// WaitingFor specifies the wait strategy for container readiness
	WaitingFor WaitStrategy

	// Timeout for container operations (default: 30s)
	Timeout time.Duration

	// OTelConfig enables OpenTelemetry instrumentation (optional)
	OTelConfig *otel.Config
}

// config is the internal configuration used by the executor.
// It's built from either functional options or ContainerRequest.
type config struct {
	// Container configuration
	image           string
	name            string
	hostname        string
	cmd             []string
	entrypoint      []string
	env             map[string]string
	exposedPorts    nat.PortSet
	portBindings    nat.PortMap
	volumes         map[string]struct{}
	binds           []string
	labels          map[string]string
	workDir         string
	user            string
	networks        []string
	networkMode     string
	autoRemove      bool
	privileged      bool
	capAdd          []string
	capDrop         []string
	tmpfs           map[string]string
	shmSize         int64

	// Operational configuration
	waitStrategy WaitStrategy
	timeout      time.Duration

	// Observability
	otelConfig *otel.Config
}

// Option is a functional option for configuring the executor.
type Option func(*config) error

// validate checks if the configuration is valid.
func (c *config) validate() error {
	if c.image == "" {
		return fmt.Errorf("image is required")
	}

	if c.timeout == 0 {
		c.timeout = 30 * time.Second
	}

	return nil
}

// newConfig creates a new config from functional options.
func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		env:          make(map[string]string),
		exposedPorts: make(nat.PortSet),
		portBindings: make(nat.PortMap),
		volumes:      make(map[string]struct{}),
		labels:       make(map[string]string),
		tmpfs:        make(map[string]string),
		timeout:      30 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WithRequest creates an option from a ContainerRequest struct.
// This allows combining struct-based and functional options:
//
//	req := docker.ContainerRequest{Image: "nginx:latest", ...}
//	exec := docker.New(
//	    docker.WithRequest(req),
//	    docker.WithOTelConfig(otelCfg), // Additional options
//	)
func WithRequest(req ContainerRequest) Option {
	return func(c *config) error {
		c.image = req.Image
		c.name = req.Name
		c.hostname = req.Hostname
		c.cmd = req.Cmd
		c.entrypoint = req.Entrypoint
		c.workDir = req.WorkingDir
		c.user = req.User
		c.autoRemove = req.AutoRemove
		c.privileged = req.Privileged
		c.capAdd = req.CapAdd
		c.capDrop = req.CapDrop
		c.shmSize = req.ShmSize
		c.waitStrategy = req.WaitingFor
		c.networkMode = req.NetworkMode

		if req.Timeout > 0 {
			c.timeout = req.Timeout
		}

		if req.OTelConfig != nil {
			c.otelConfig = req.OTelConfig
		}

		// Environment variables
		for k, v := range req.Env {
			c.env[k] = v
		}

		// Labels
		for k, v := range req.Labels {
			c.labels[k] = v
		}

		// Exposed ports
		for _, port := range req.ExposedPorts {
			// Parse port format: "8080" or "8080/tcp" or "8080/udp"
			protocol := "tcp"
			portNum := port

			// Check if protocol is specified
			if strings.Contains(port, "/") {
				parts := strings.Split(port, "/")
				if len(parts) == 2 {
					portNum = parts[0]
					protocol = parts[1]
				}
			}

			natPort, err := nat.NewPort(protocol, portNum)
			if err != nil {
				return fmt.Errorf("invalid exposed port %s: %w", port, err)
			}
			c.exposedPorts[natPort] = struct{}{}
		}

		// Port bindings
		for containerPort, hostPort := range req.PortBindings {
			// Parse port format: "8080" or "8080/tcp" or "8080/udp"
			protocol := "tcp"
			portNum := containerPort

			// Check if protocol is specified
			if strings.Contains(containerPort, "/") {
				parts := strings.Split(containerPort, "/")
				if len(parts) == 2 {
					portNum = parts[0]
					protocol = parts[1]
				}
			}

			natPort, err := nat.NewPort(protocol, portNum)
			if err != nil {
				return fmt.Errorf("invalid container port %s: %w", containerPort, err)
			}
			c.portBindings[natPort] = []nat.PortBinding{
				{HostPort: hostPort},
			}
		}

		// Volumes
		for hostPath, containerPath := range req.Volumes {
			c.volumes[containerPath] = struct{}{}
			c.binds = append(c.binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
		}

		// Bind mounts
		for containerPath, bindSpec := range req.BindMounts {
			c.volumes[containerPath] = struct{}{}
			c.binds = append(c.binds, bindSpec)
		}

		// Tmpfs
		for k, v := range req.Tmpfs {
			c.tmpfs[k] = v
		}

		// Networks
		c.networks = append(c.networks, req.Networks...)

		return nil
	}
}

// WithImage sets the container image.
func WithImage(image string) Option {
	return func(c *config) error {
		c.image = image
		return nil
	}
}

// WithName sets the container name.
func WithName(name string) Option {
	return func(c *config) error {
		c.name = name
		return nil
	}
}

// WithHostname sets the container hostname.
func WithHostname(hostname string) Option {
	return func(c *config) error {
		c.hostname = hostname
		return nil
	}
}

// WithCmd sets the container command.
func WithCmd(cmd ...string) Option {
	return func(c *config) error {
		c.cmd = cmd
		return nil
	}
}

// WithEntrypoint sets the container entrypoint.
func WithEntrypoint(entrypoint ...string) Option {
	return func(c *config) error {
		c.entrypoint = entrypoint
		return nil
	}
}

// WithEnv adds an environment variable in KEY=VALUE format.
func WithEnv(env string) Option {
	return func(c *config) error {
		// Parse KEY=VALUE
		for i, ch := range env {
			if ch == '=' {
				key := env[:i]
				value := env[i+1:]
				c.env[key] = value
				return nil
			}
		}
		return fmt.Errorf("invalid env format, expected KEY=VALUE: %s", env)
	}
}

// WithEnvMap sets environment variables from a map.
func WithEnvMap(env map[string]string) Option {
	return func(c *config) error {
		for k, v := range env {
			c.env[k] = v
		}
		return nil
	}
}

// WithPorts adds port mappings in "containerPort:hostPort" format (e.g., "80:8080").
// Protocol defaults to TCP. Use "80:8080/udp" for UDP.
func WithPorts(portMapping string) Option {
	return func(c *config) error {
		// Parse "containerPort:hostPort" or "containerPort:hostPort/protocol"
		var containerPort, hostPort, protocol string
		protocol = "tcp"

		// Check for protocol suffix
		for i := len(portMapping) - 1; i >= 0; i-- {
			if portMapping[i] == '/' {
				protocol = portMapping[i+1:]
				portMapping = portMapping[:i]
				break
			}
		}

		// Parse container:host
		for i, ch := range portMapping {
			if ch == ':' {
				containerPort = portMapping[:i]
				hostPort = portMapping[i+1:]
				break
			}
		}

		if containerPort == "" || hostPort == "" {
			return fmt.Errorf("invalid port mapping format, expected containerPort:hostPort: %s", portMapping)
		}

		// Create port binding
		natPort, err := nat.NewPort(protocol, containerPort)
		if err != nil {
			return fmt.Errorf("invalid port %s: %w", containerPort, err)
		}

		c.exposedPorts[natPort] = struct{}{}
		c.portBindings[natPort] = []nat.PortBinding{
			{HostPort: hostPort},
		}

		return nil
	}
}

// WithPortBindings sets detailed port bindings.
// Key: container port with protocol (e.g., "80/tcp")
// Value: host port (e.g., "8080")
func WithPortBindings(bindings map[string]string) Option {
	return func(c *config) error {
		for containerPort, hostPort := range bindings {
			// Parse port format: "8080" or "8080/tcp" or "8080/udp"
			protocol := "tcp"
			portNum := containerPort

			// Check if protocol is specified
			if strings.Contains(containerPort, "/") {
				parts := strings.Split(containerPort, "/")
				if len(parts) == 2 {
					portNum = parts[0]
					protocol = parts[1]
				}
			}

			natPort, err := nat.NewPort(protocol, portNum)
			if err != nil {
				return fmt.Errorf("invalid container port %s: %w", containerPort, err)
			}

			c.exposedPorts[natPort] = struct{}{}
			c.portBindings[natPort] = []nat.PortBinding{
				{HostPort: hostPort},
			}
		}
		return nil
	}
}

// WithExposedPorts exposes ports without binding to host.
func WithExposedPorts(ports ...string) Option {
	return func(c *config) error {
		for _, port := range ports {
			// Parse port format: "8080" or "8080/tcp" or "8080/udp"
			protocol := "tcp"
			portNum := port

			// Check if protocol is specified
			if strings.Contains(port, "/") {
				parts := strings.Split(port, "/")
				if len(parts) == 2 {
					portNum = parts[0]
					protocol = parts[1]
				}
			}

			natPort, err := nat.NewPort(protocol, portNum)
			if err != nil {
				return fmt.Errorf("invalid port %s: %w", port, err)
			}
			c.exposedPorts[natPort] = struct{}{}
		}
		return nil
	}
}

// WithVolume mounts a volume.
// Format: "hostPath:containerPath" or "hostPath:containerPath:ro"
func WithVolume(hostPath, containerPath string) Option {
	return func(c *config) error {
		c.volumes[containerPath] = struct{}{}
		c.binds = append(c.binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
		return nil
	}
}

// WithVolumeRO mounts a read-only volume.
func WithVolumeRO(hostPath, containerPath string) Option {
	return func(c *config) error {
		c.volumes[containerPath] = struct{}{}
		c.binds = append(c.binds, fmt.Sprintf("%s:%s:ro", hostPath, containerPath))
		return nil
	}
}

// WithVolumes sets multiple volume mounts.
func WithVolumes(volumes map[string]string) Option {
	return func(c *config) error {
		for hostPath, containerPath := range volumes {
			c.volumes[containerPath] = struct{}{}
			c.binds = append(c.binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
		}
		return nil
	}
}

// WithLabel adds a container label.
func WithLabel(key, value string) Option {
	return func(c *config) error {
		c.labels[key] = value
		return nil
	}
}

// WithLabels sets multiple container labels.
func WithLabels(labels map[string]string) Option {
	return func(c *config) error {
		for k, v := range labels {
			c.labels[k] = v
		}
		return nil
	}
}

// WithWorkDir sets the working directory.
func WithWorkDir(workDir string) Option {
	return func(c *config) error {
		c.workDir = workDir
		return nil
	}
}

// WithUser sets the user.
func WithUser(user string) Option {
	return func(c *config) error {
		c.user = user
		return nil
	}
}

// WithNetwork attaches the container to a network.
func WithNetwork(network string) Option {
	return func(c *config) error {
		c.networks = append(c.networks, network)
		return nil
	}
}

// WithNetworks attaches the container to multiple networks.
func WithNetworks(networks ...string) Option {
	return func(c *config) error {
		c.networks = append(c.networks, networks...)
		return nil
	}
}

// WithNetworkMode sets the network mode (bridge, host, none, container:<name>).
func WithNetworkMode(mode string) Option {
	return func(c *config) error {
		c.networkMode = mode
		return nil
	}
}

// WithAutoRemove automatically removes the container when it stops.
func WithAutoRemove(autoRemove bool) Option {
	return func(c *config) error {
		c.autoRemove = autoRemove
		return nil
	}
}

// WithPrivileged runs the container in privileged mode.
func WithPrivileged(privileged bool) Option {
	return func(c *config) error {
		c.privileged = privileged
		return nil
	}
}

// WithCapAdd adds Linux capabilities.
func WithCapAdd(caps ...string) Option {
	return func(c *config) error {
		c.capAdd = append(c.capAdd, caps...)
		return nil
	}
}

// WithCapDrop drops Linux capabilities.
func WithCapDrop(caps ...string) Option {
	return func(c *config) error {
		c.capDrop = append(c.capDrop, caps...)
		return nil
	}
}

// WithTmpfs mounts a tmpfs filesystem.
func WithTmpfs(path, options string) Option {
	return func(c *config) error {
		c.tmpfs[path] = options
		return nil
	}
}

// WithShmSize sets the size of /dev/shm in bytes.
func WithShmSize(size int64) Option {
	return func(c *config) error {
		c.shmSize = size
		return nil
	}
}

// WithWaitStrategy sets the wait strategy for container readiness.
func WithWaitStrategy(strategy WaitStrategy) Option {
	return func(c *config) error {
		c.waitStrategy = strategy
		return nil
	}
}

// WithTimeout sets the timeout for container operations.
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.timeout = timeout
		return nil
	}
}

// WithOTelConfig enables OpenTelemetry instrumentation.
func WithOTelConfig(otelConfig *otel.Config) Option {
	return func(c *config) error {
		c.otelConfig = otelConfig
		return nil
	}
}
