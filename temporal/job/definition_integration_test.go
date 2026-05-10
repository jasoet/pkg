//go:build integration

package job

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
)

func echoWorkflow(ctx workflow.Context, in string) (string, error) {
	var out string
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	}), echoActivity, in).Get(ctx, &out); err != nil {
		return "", err
	}
	return out, nil
}

func echoActivity(_ context.Context, in string) (string, error) { return in, nil }

func setupTestDef(t *testing.T, c client.Client, w worker.Worker) *Definition {
	t.Helper()
	d, err := New("echo", "echo-tq",
		WithRegister(func(w worker.Worker) {
			RegisterWorkflowOnce(w, "echo", echoWorkflow, workflow.RegisterOptions{Name: "echo"})
			RegisterActivityOnce(w, "echoActivity", echoActivity, activity.RegisterOptions{Name: "echoActivity"})
		}),
		WithExecute(func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, in any) (client.WorkflowRun, error) {
			return c.ExecuteWorkflow(ctx, opts, "echo", in)
		}),
		WithNewInput(func() any { var s string; return &s }),
	)
	require.NoError(t, err)
	d.Register(w)
	return d
}

func TestIntegration_Definition_Execute_Describe_History(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tc, c, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{Namespace: "default"}, testcontainer.Options{})
	require.NoError(t, err)
	defer cleanup()
	_ = tc

	w := worker.New(c, "echo-tq", worker.Options{})
	d := setupTestDef(t, c, w)
	require.NoError(t, w.Start())
	defer w.Stop()

	run, err := d.Execute(ctx, c, "hello world")
	require.NoError(t, err)
	assert.True(t, len(run.WorkflowID) > 5, "workflow ID has prefix")

	var got string
	require.NoError(t, run.Get(ctx, &got))
	assert.Equal(t, "hello world", got)

	detail, err := d.Describe(ctx, c, run.WorkflowID, run.RunID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, detail.Status)
	assert.Equal(t, "echo", detail.Type)

	hist, err := d.History(ctx, c, run.WorkflowID, run.RunID, HistoryOpts{})
	require.NoError(t, err)
	assert.NotEmpty(t, hist.Activities)
}

func TestIntegration_Definition_Cancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tc, c, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{Namespace: "default"}, testcontainer.Options{})
	require.NoError(t, err)
	defer cleanup()
	_ = tc

	longWf := func(ctx workflow.Context, _ string) error {
		return workflow.Sleep(ctx, 1*time.Hour)
	}

	d, err := New("long", "long-tq",
		WithRegister(func(w worker.Worker) {
			RegisterWorkflowOnce(w, "long", longWf, workflow.RegisterOptions{Name: "long"})
		}),
		WithExecute(func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, in any) (client.WorkflowRun, error) {
			return c.ExecuteWorkflow(ctx, opts, "long", in)
		}),
		WithNewInput(func() any { var s string; return &s }),
	)
	require.NoError(t, err)

	w := worker.New(c, "long-tq", worker.Options{})
	d.Register(w)
	require.NoError(t, w.Start())
	defer w.Stop()

	run, err := d.Execute(ctx, c, "x")
	require.NoError(t, err)

	require.NoError(t, d.Cancel(ctx, c, run.WorkflowID, run.RunID))

	// Wait a moment for cancellation to propagate
	time.Sleep(2 * time.Second)
	detail, err := d.Describe(ctx, c, run.WorkflowID, run.RunID)
	require.NoError(t, err)
	assert.Equal(t, StatusCanceled, detail.Status)
}

func TestIntegration_Definition_ListRuns_ScopedByName(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tc, c, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{Namespace: "default"}, testcontainer.Options{})
	require.NoError(t, err)
	defer cleanup()
	_ = tc

	w := worker.New(c, "echo-tq", worker.Options{})
	d := setupTestDef(t, c, w)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Run twice.
	r1, err := d.Execute(ctx, c, "one")
	require.NoError(t, err)
	require.NoError(t, r1.Get(ctx, new(string)))
	r2, err := d.Execute(ctx, c, "two")
	require.NoError(t, err)
	require.NoError(t, r2.Get(ctx, new(string)))

	// Visibility settles asynchronously.
	time.Sleep(2 * time.Second)

	page, err := d.ListRuns(ctx, c, ListOpts{PageSize: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page.Runs), 2, "ListRuns scoped by Name prefix returns both runs")
}
