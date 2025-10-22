//go:build integration

package testcontainer

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
)

// ClientConfig holds the configuration for creating a Temporal client.
type ClientConfig struct {
	// Namespace is the Temporal namespace to use.
	// Default: "default"
	Namespace string

	// HostPort will be automatically set to the container's address.
	// Any value provided here will be overridden.
	HostPort string
}

// Setup is a convenience function that:
// 1. Starts a Temporal test container
// 2. Creates a Temporal client configured to connect to the container
// 3. Returns a cleanup function that closes the client and terminates the container
//
// This function is ideal for integration tests where you need both container and client.
//
// Example:
//
//	container, client, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{
//	    Namespace: "default",
//	}, testcontainer.Options{})
//	if err != nil {
//	    t.Fatalf("Setup failed: %v", err)
//	}
//	defer cleanup()
//
//	// Use client for your tests...
func Setup(ctx context.Context, config ClientConfig, opts Options) (*Container, client.Client, func(), error) {
	// Start the container
	container, err := Start(ctx, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Set default namespace
	if config.Namespace == "" {
		config.Namespace = "default"
	}

	// Create the Temporal client using SDK directly
	temporalClient, err := client.Dial(client.Options{
		HostPort:  container.HostPort(),
		Namespace: config.Namespace,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	// Create cleanup function
	cleanup := func() {
		temporalClient.Close()
		if err := container.Terminate(ctx); err != nil {
			if opts.Logger != nil {
				opts.Logger.Logf("Failed to terminate container: %v", err)
			}
		}
	}

	return container, temporalClient, cleanup, nil
}
