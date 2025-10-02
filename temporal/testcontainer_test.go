//go:build temporal

package temporal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.temporal.io/sdk/client"
)

// TemporalContainer represents a Temporal server test container
type TemporalContainer struct {
	testcontainers.Container
	HostPort string
}

// StartTemporalContainer starts a Temporal server container for testing
func StartTemporalContainer(ctx context.Context, t *testing.T) (*TemporalContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "temporalio/temporal:latest",
		ExposedPorts: []string{"7233/tcp", "8233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(60 * time.Second),
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
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	hostPort := fmt.Sprintf("%s:%s", host, mappedPort.Port())

	t.Logf("Temporal container started at %s", hostPort)

	// Wait a bit more for Temporal to fully initialize
	time.Sleep(3 * time.Second)

	return &TemporalContainer{
		Container: container,
		HostPort:  hostPort,
	}, nil
}

// Terminate stops and removes the Temporal container
func (tc *TemporalContainer) Terminate(ctx context.Context) error {
	return tc.Container.Terminate(ctx)
}

// setupTemporalContainerForTest is a helper that starts a Temporal container and returns a client
func setupTemporalContainerForTest(ctx context.Context, t *testing.T) (*TemporalContainer, client.Client, func()) {
	container, err := StartTemporalContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to start Temporal container: %v", err)
	}

	// Create client config using the container's address
	config := DefaultConfig()
	config.HostPort = container.HostPort
	config.MetricsListenAddress = "0.0.0.0:0" // Use random port for metrics

	// Create client
	client, err := NewClient(config)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to create Temporal client: %v", err)
	}

	// Return container, client, and cleanup function
	cleanup := func() {
		client.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return container, client, cleanup
}
