package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// skipIfNoContainerRuntime skips the test if no Docker-compatible container runtime
// (Docker or Podman) is available. It respects DOCKER_HOST for Podman support.
func skipIfNoContainerRuntime(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Container runtime client not available:", err)
	}
	defer cli.Close()
	if _, err := cli.Ping(ctx); err != nil {
		t.Skip("Container runtime not running (set DOCKER_HOST for Podman):", err)
	}
}
