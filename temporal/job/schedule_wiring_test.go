package job

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/worker"
)

func newScheduledDef(t *testing.T, spec *ScheduleSpec, args ...any) *Definition {
	t.Helper()
	opts := []Option{
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
		WithSchedule(spec),
	}
	if len(args) > 0 {
		opts = append(opts, WithScheduleArgs(args...))
	}
	d, err := New("sched-def", "tq", opts...)
	require.NoError(t, err)
	return d
}

// TestScheduleAction_IncludesArgs pins that scheduled runs carry the configured
// arguments — without them, scheduled workflows requiring input fail to decode.
func TestScheduleAction_IncludesArgs(t *testing.T) {
	d := newScheduledDef(t, &ScheduleSpec{Interval: time.Hour}, "hello", 42)
	action := scheduleAction(d)
	assert.Equal(t, "sched-def-scheduled", action.ID)
	assert.Equal(t, "sched-def", action.Workflow)
	assert.Equal(t, "tq", action.TaskQueue)
	assert.Equal(t, []any{"hello", 42}, action.Args)
}

// TestApplySchedule_CreateWiresCatchupWindowAndArgs verifies the create path
// forwards CatchupWindow and the scheduled Args to the SDK.
func TestApplySchedule_CreateWiresCatchupWindowAndArgs(t *testing.T) {
	d := newScheduledDef(t, &ScheduleSpec{
		Interval:      time.Hour,
		CatchupWindow: 2 * time.Minute,
		Overlap:       OverlapSkip,
	}, "hello", 42)

	c := mocks.NewClient(t)
	sc := mocks.NewScheduleClient(t)
	handle := mocks.NewScheduleHandle(t)

	c.On("ScheduleClient").Return(sc)
	sc.On("GetHandle", mock.Anything, "sched-def").Return(handle)
	// No existing schedule => Describe fails => create path.
	handle.On("Describe", mock.Anything).Return(nil, serviceerror.NewNotFound("not found"))

	var captured client.ScheduleOptions
	sc.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(client.ScheduleOptions)
		}).
		Return(handle, nil)

	err := d.ApplySchedule(context.Background(), c)
	require.NoError(t, err)

	assert.Equal(t, 2*time.Minute, captured.CatchupWindow, "CatchupWindow must be wired into the create path")
	action, ok := captured.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok)
	assert.Equal(t, []any{"hello", 42}, action.Args, "scheduled Args must reach the SDK action")
}

// TestStats_TodayOnly verifies StatsOpts.TodayOnly is honored: only when set do
// the completed/failed queries filter by CloseTime.
func TestStats_TodayOnly(t *testing.T) {
	captureQueries := func(t *testing.T) (*mocks.Client, *[]string) {
		t.Helper()
		c := mocks.NewClient(t)
		var queries []string
		c.On("CountWorkflow", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				req := args.Get(1).(*workflowservice.CountWorkflowExecutionsRequest)
				queries = append(queries, req.Query)
			}).
			Return(&workflowservice.CountWorkflowExecutionsResponse{Count: 0}, nil)
		return c, &queries
	}

	d := &Definition{Name: "orders", TaskQueue: "tq"}

	t.Run("AllTimeByDefault", func(t *testing.T) {
		c, queries := captureQueries(t)
		_, err := d.Stats(context.Background(), c, StatsOpts{TodayOnly: false})
		require.NoError(t, err)
		require.Len(t, *queries, 3)
		for _, q := range *queries {
			assert.NotContains(t, q, "CloseTime", "default Stats must not filter by CloseTime: %q", q)
		}
	})

	t.Run("TodayOnlyFiltersClosed", func(t *testing.T) {
		c, queries := captureQueries(t)
		_, err := d.Stats(context.Background(), c, StatsOpts{TodayOnly: true, Location: time.UTC})
		require.NoError(t, err)
		require.Len(t, *queries, 3)

		var closeFiltered int
		for _, q := range *queries {
			if strings.Contains(q, "CloseTime >=") {
				closeFiltered++
			}
		}
		assert.Equal(t, 2, closeFiltered, "TodayOnly must add a CloseTime filter to the completed and failed queries")
	})
}
