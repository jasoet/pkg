//go:build integration

package docker_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// uniqueName builds a collision-free container name from the test name and a
// nanosecond timestamp, so parallel or repeated runs never clash on a fixed name.
// Subtest separators ("/") are replaced so the result is a valid container name.
func uniqueName(t *testing.T, prefix string) string {
	t.Helper()
	safe := strings.ReplaceAll(t.Name(), "/", "-")
	return fmt.Sprintf("%s-%s-%d", prefix, safe, time.Now().UnixNano())
}

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
