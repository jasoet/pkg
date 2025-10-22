//go:build integration

package testcontainer

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Logger is an interface for logging within the testcontainer package.
// This allows users to provide their own logger implementation (e.g., *testing.T, zerolog, etc.)
// or pass nil for no logging.
type Logger interface {
	Logf(format string, args ...interface{})
}

// Options configures the Temporal test container.
type Options struct {
	// Image is the Docker image to use for Temporal server.
	// Default: "temporalio/temporal:latest"
	Image string

	// StartupTimeout is the maximum time to wait for the container to be ready.
	// Default: 60 seconds
	StartupTimeout time.Duration

	// Logger is an optional logger for container events.
	// If nil, no logging will be performed.
	Logger Logger

	// ExtraPorts are additional ports to expose from the container.
	// By default, ports 7233 (gRPC) and 8233 (Web UI) are exposed.
	ExtraPorts []string

	// InitialWaitTime is the additional time to wait after container starts
	// to ensure Temporal is fully initialized.
	// Default: 3 seconds
	InitialWaitTime time.Duration
}

// Container represents a running Temporal server test container.
type Container struct {
	testcontainers.Container
	hostPort string
}

// Start creates and starts a Temporal server container for testing.
// It returns a Container instance that can be used to connect to the server.
func Start(ctx context.Context, opts Options) (*Container, error) {
	// Apply defaults
	if opts.Image == "" {
		opts.Image = "temporalio/temporal:latest"
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 60 * time.Second
	}
	if opts.InitialWaitTime == 0 {
		opts.InitialWaitTime = 3 * time.Second
	}

	// Build exposed ports list
	exposedPorts := []string{"7233/tcp", "8233/tcp"}
	if len(opts.ExtraPorts) > 0 {
		exposedPorts = append(exposedPorts, opts.ExtraPorts...)
	}

	req := testcontainers.ContainerRequest{
		Image:        opts.Image,
		ExposedPorts: exposedPorts,
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(opts.StartupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start temporal container: %w", err)
	}

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "7233")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	hostPort := fmt.Sprintf("%s:%s", host, mappedPort.Port())

	if opts.Logger != nil {
		opts.Logger.Logf("Temporal container started at %s", hostPort)
	}

	// Wait for Temporal to fully initialize
	time.Sleep(opts.InitialWaitTime)

	return &Container{
		Container: container,
		hostPort:  hostPort,
	}, nil
}

// HostPort returns the host:port address to connect to the Temporal server.
func (c *Container) HostPort() string {
	return c.hostPort
}

// Terminate stops and removes the Temporal container.
func (c *Container) Terminate(ctx context.Context) error {
	return c.Container.Terminate(ctx)
}
