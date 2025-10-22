//go:build integration

package temporal

import (
	"context"
	"testing"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
	"go.temporal.io/sdk/client"
)

// TemporalContainer represents a Temporal server test container.
// Deprecated: Use testcontainer.Container instead.
type TemporalContainer struct {
	*testcontainer.Container
	HostPort string
}

// StartTemporalContainer starts a Temporal server container for testing.
// Deprecated: Use testcontainer.Start() instead for more flexibility.
func StartTemporalContainer(ctx context.Context, t *testing.T) (*TemporalContainer, error) {
	// Use the new testcontainer package
	container, err := testcontainer.Start(ctx, testcontainer.Options{
		Logger: t,
	})
	if err != nil {
		return nil, err
	}

	return &TemporalContainer{
		Container: container,
		HostPort:  container.HostPort(),
	}, nil
}

// Terminate stops and removes the Temporal container.
// Deprecated: Use Container.Terminate() directly.
func (tc *TemporalContainer) Terminate(ctx context.Context) error {
	return tc.Container.Terminate(ctx)
}

// setupTemporalContainerForTest is a helper that starts a Temporal container and returns a client.
// Deprecated: Use testcontainer.Setup() instead.
func setupTemporalContainerForTest(ctx context.Context, t *testing.T) (*TemporalContainer, client.Client, func()) {
	// Use the new testcontainer.Setup function
	container, client, cleanup, err := testcontainer.Setup(
		ctx,
		testcontainer.ClientConfig{
			Namespace: "default",
		},
		testcontainer.Options{Logger: t},
	)
	if err != nil {
		t.Fatalf("Failed to setup temporal container: %v", err)
	}

	// Wrap in old TemporalContainer type for backward compatibility
	wrappedContainer := &TemporalContainer{
		Container: container,
		HostPort:  container.HostPort(),
	}

	return wrappedContainer, client, cleanup
}
