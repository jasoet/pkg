package testcontainer

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

// nopLogger is a no-op implementation of log.Logger that discards all output.
// Used to suppress the Temporal SDK's "No logger configured" INFO message.
type nopLogger struct{}

func (nopLogger) Debug(string, ...interface{}) {}
func (nopLogger) Info(string, ...interface{})  {}
func (nopLogger) Warn(string, ...interface{})  {}
func (nopLogger) Error(string, ...interface{}) {}

// ClientConfig holds the configuration for creating a Temporal client.
// The connection address is taken from the started container, so no host/port
// is configured here.
type ClientConfig struct {
	// Namespace is the Temporal namespace to use.
	// Default: "default"
	Namespace string
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

	// Create the Temporal client using SDK directly.
	// A nop logger is provided to suppress the SDK's default "No logger configured" message.
	var logger log.Logger = nopLogger{}
	temporalClient, err := client.Dial(client.Options{
		HostPort:  container.HostPort(),
		Namespace: config.Namespace,
		Logger:    logger,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	// Actively wait until the frontend serves RPCs. An open port (the
	// testcontainers wait strategy) plus the fixed InitialWaitTime buffer do
	// not guarantee the server is ready, so poll CheckHealth as an authoritative
	// readiness signal.
	waitForReady(ctx, temporalClient, opts)

	// Create cleanup function. context.Background() is used intentionally here
	// instead of the caller-provided ctx, because the caller's context may
	// already be canceled by the time cleanup runs (e.g. after t.Cleanup or
	// defer fires at the end of a test).
	cleanup := func() {
		temporalClient.Close()
		if err := container.Terminate(context.Background()); err != nil {
			if opts.Logger != nil {
				opts.Logger.Logf("Failed to terminate container: %v", err)
			}
		}
	}

	return container, temporalClient, cleanup, nil
}

// waitForReady polls the Temporal frontend's health check until it passes or
// the startup budget is exhausted. It is best-effort: if the server never
// reports healthy in time, Setup still returns the client and lets the caller's
// own operations surface the failure, preserving prior behavior while removing
// the reliance on a blind sleep.
func waitForReady(ctx context.Context, c client.Client, opts Options) {
	timeout := opts.StartupTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := c.CheckHealth(checkCtx, &client.CheckHealthRequest{})
		cancel()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			if opts.Logger != nil {
				opts.Logger.Logf("Temporal health check did not pass within %s: %v", timeout, err)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
