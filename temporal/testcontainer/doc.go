//go:build integration

// Package testcontainer provides utilities for running Temporal server in Docker containers for integration testing.
//
// This package makes it easy to start a Temporal server container, connect to it, and clean up resources
// after testing. It's designed to be framework-agnostic and can be used in any Go testing scenario.
//
// # Basic Usage
//
// The simplest way to use this package is with the Setup function, which handles container startup,
// client creation, and cleanup:
//
//	import (
//	    "context"
//	    "testing"
//	    "github.com/jasoet/pkg/v2/temporal"
//	    "github.com/jasoet/pkg/v2/temporal/testcontainer"
//	)
//
//	func TestMyWorkflow(t *testing.T) {
//	    ctx := context.Background()
//
//	    // Start container and create client
//	    container, client, cleanup, err := testcontainer.Setup(
//	        ctx,
//	        temporal.DefaultConfig(),
//	        testcontainer.Options{Logger: t},
//	    )
//	    if err != nil {
//	        t.Fatalf("Setup failed: %v", err)
//	    }
//	    defer cleanup()
//
//	    // Use client for your tests...
//	    // client.ExecuteWorkflow(...)
//	}
//
// # Advanced Usage
//
// For more control, you can start the container and create the client separately:
//
//	func TestAdvanced(t *testing.T) {
//	    ctx := context.Background()
//
//	    // Start container with custom options
//	    container, err := testcontainer.Start(ctx, testcontainer.Options{
//	        Image: "temporalio/temporal:1.22.0",
//	        StartupTimeout: 120 * time.Second,
//	        Logger: t,
//	    })
//	    if err != nil {
//	        t.Fatalf("Failed to start container: %v", err)
//	    }
//	    defer container.Terminate(ctx)
//
//	    // Create client manually
//	    config := temporal.DefaultConfig()
//	    config.HostPort = container.HostPort()
//	    config.MetricsListenAddress = "0.0.0.0:0"
//
//	    client, closer, err := temporal.NewClient(config)
//	    if err != nil {
//	        t.Fatalf("Failed to create client: %v", err)
//	    }
//	    defer client.Close()
//	    if closer != nil {
//	        defer closer.Close()
//	    }
//
//	    // Run tests...
//	}
//
// # Configuration Options
//
// The Options struct allows you to customize the container:
//
//   - Image: Docker image to use (default: "temporalio/temporal:latest")
//   - StartupTimeout: How long to wait for container startup (default: 60s)
//   - Logger: Optional logger for container events (can be *testing.T)
//   - ExtraPorts: Additional ports to expose
//   - InitialWaitTime: Extra time to wait after startup (default: 3s)
//
// # Logging
//
// You can pass any type that implements the Logger interface (including *testing.T):
//
//	opts := testcontainer.Options{
//	    Logger: t, // *testing.T implements Logger
//	}
//
// Or implement your own:
//
//	type MyLogger struct{}
//	func (l *MyLogger) Logf(format string, args ...interface{}) {
//	    fmt.Printf(format+"\n", args...)
//	}
//
// # Using in Other Projects
//
// This package is designed to be imported and used in any Go project:
//
//	go get github.com/jasoet/pkg/v2/temporal/testcontainer
//
// Then import and use in your tests:
//
//	import "github.com/jasoet/pkg/v2/temporal/testcontainer"
package testcontainer
