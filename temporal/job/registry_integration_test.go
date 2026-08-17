//go:build integration

package job

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/jasoet/pkg/v3/temporal/testcontainer"
)

func TestIntegration_Registry_RegisterAll_Deduplicates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tc, c, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{Namespace: "default"}, testcontainer.Options{})
	require.NoError(t, err)
	defer cleanup()
	_ = tc

	sharedWf := func(workflow.Context, string) (string, error) { return "ok", nil }

	mk := func(name string) *Definition {
		d, err := New(name, "shared-tq",
			WithRegister(func(w worker.Worker) {
				RegisterWorkflowOnce(w, "shared", sharedWf, workflow.RegisterOptions{Name: "shared"})
			}),
			WithExecute(func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, in any) (client.WorkflowRun, error) {
				return c.ExecuteWorkflow(ctx, opts, "shared", in)
			}),
			WithNewInput(func() any { var s string; return &s }),
		)
		require.NoError(t, err)
		return d
	}

	r := NewRegistry(mk("alpha"), mk("beta"))
	w := worker.New(c, "shared-tq", worker.Options{})
	r.RegisterAll(w) // would panic on duplicate workflow type without dedup

	require.NoError(t, w.Start())
	defer w.Stop()

	for _, name := range []string{"alpha", "beta"} {
		run, err := r.MustGet(name).Execute(ctx, c, "in")
		require.NoError(t, err)
		var out string
		require.NoError(t, run.Get(ctx, &out))
		assert.Equal(t, "ok", out)
	}

	time.Sleep(2 * time.Second)

	alphaPage, err := r.MustGet("alpha").ListRuns(ctx, c, ListOpts{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(alphaPage.Runs), 1)
	for _, run := range alphaPage.Runs {
		assert.True(t, strings.HasPrefix(run.WorkflowID, "alpha-"), "expected alpha- prefix, got %s", run.WorkflowID)
	}
}
