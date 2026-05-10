//go:build integration

package job

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/jasoet/pkg/v2/temporal/testcontainer"
)

func TestIntegration_Schedule_FullLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tc, c, cleanup, err := testcontainer.Setup(ctx, testcontainer.ClientConfig{Namespace: "default"}, testcontainer.Options{})
	require.NoError(t, err)
	defer cleanup()
	_ = tc

	wf := func(workflow.Context, string) error { return nil }

	d, err := New("sched-test", "sched-tq",
		WithRegister(func(w worker.Worker) {
			RegisterWorkflowOnce(w, "sched-test", wf, workflow.RegisterOptions{Name: "sched-test"})
		}),
		WithExecute(func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, in any) (client.WorkflowRun, error) {
			return c.ExecuteWorkflow(ctx, opts, "sched-test", in)
		}),
		WithNewInput(func() any { var s string; return &s }),
		WithSchedule(&ScheduleSpec{Interval: time.Hour, Paused: true, Note: "initial"}),
	)
	require.NoError(t, err)

	w := worker.New(c, "sched-tq", worker.Options{})
	d.Register(w)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Apply
	require.NoError(t, d.ApplySchedule(ctx, c))
	defer d.DeleteSchedule(ctx, c) //nolint:errcheck

	// Describe
	desc, err := d.DescribeSchedule(ctx, c)
	require.NoError(t, err)
	assert.True(t, desc.Paused)
	assert.Equal(t, "initial", desc.Note)

	// Resume
	require.NoError(t, d.ResumeSchedule(ctx, c, "resumed"))
	desc, err = d.DescribeSchedule(ctx, c)
	require.NoError(t, err)
	assert.False(t, desc.Paused)

	// Pause
	require.NoError(t, d.PauseSchedule(ctx, c, "paused again"))
	desc, err = d.DescribeSchedule(ctx, c)
	require.NoError(t, err)
	assert.True(t, desc.Paused)

	// Trigger (action runs once even though paused)
	require.NoError(t, d.TriggerSchedule(ctx, c))

	// Delete
	require.NoError(t, d.DeleteSchedule(ctx, c))
	_, err = d.DescribeSchedule(ctx, c)
	assert.Error(t, err)
}
