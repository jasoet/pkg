package job

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
)

// ApplySchedule creates or updates the Temporal schedule for this Definition.
// Schedule ID equals Definition.Name. If a schedule with that ID already
// exists, it is updated to match the current ScheduleSpec.
func (d *Definition) ApplySchedule(ctx context.Context, c client.Client) error {
	if d.Schedule == nil {
		return ErrNoSchedule
	}
	spec, err := d.Schedule.toSDKSpec()
	if err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	sc := c.ScheduleClient()

	// Check existence by trying to describe.
	handle := sc.GetHandle(ctx, d.Name)
	_, descErr := handle.Describe(ctx)
	if descErr == nil {
		// Update path
		return translateSDKError("schedule-update", handle.Update(ctx, client.ScheduleUpdateOptions{
			DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
				return &client.ScheduleUpdate{
					Schedule: &client.Schedule{
						Spec:   &spec,
						Action: scheduleAction(d),
						Policy: &client.SchedulePolicies{
							Overlap: d.Schedule.Overlap.ToSDK(),
						},
						State: &client.ScheduleState{
							Paused: d.Schedule.Paused,
							Note:   d.Schedule.Note,
						},
					},
				}, nil
			},
		}))
	}

	// Create path
	_, err = sc.Create(ctx, client.ScheduleOptions{
		ID:      d.Name,
		Spec:    spec,
		Action:  scheduleAction(d),
		Overlap: d.Schedule.Overlap.ToSDK(),
		Paused:  d.Schedule.Paused,
		Note:    d.Schedule.Note,
	})
	return translateSDKError("schedule-create", err)
}

func scheduleAction(d *Definition) *client.ScheduleWorkflowAction {
	return &client.ScheduleWorkflowAction{
		ID:        d.Name + "-scheduled",
		Workflow:  d.Name,
		TaskQueue: d.TaskQueue,
	}
}

// PauseSchedule pauses an existing schedule.
func (d *Definition) PauseSchedule(ctx context.Context, c client.Client, note string) error {
	if d.Schedule == nil {
		return ErrNoSchedule
	}
	handle := c.ScheduleClient().GetHandle(ctx, d.Name)
	return translateSDKError("schedule-pause", handle.Pause(ctx, client.SchedulePauseOptions{Note: note}))
}

// ResumeSchedule unpauses an existing schedule.
func (d *Definition) ResumeSchedule(ctx context.Context, c client.Client, note string) error {
	if d.Schedule == nil {
		return ErrNoSchedule
	}
	handle := c.ScheduleClient().GetHandle(ctx, d.Name)
	return translateSDKError("schedule-resume", handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: note}))
}

// TriggerSchedule fires an immediate run of the schedule's action.
func (d *Definition) TriggerSchedule(ctx context.Context, c client.Client) error {
	if d.Schedule == nil {
		return ErrNoSchedule
	}
	handle := c.ScheduleClient().GetHandle(ctx, d.Name)
	return translateSDKError("schedule-trigger", handle.Trigger(ctx, client.ScheduleTriggerOptions{}))
}

// DeleteSchedule removes the schedule from Temporal.
func (d *Definition) DeleteSchedule(ctx context.Context, c client.Client) error {
	handle := c.ScheduleClient().GetHandle(ctx, d.Name)
	return translateSDKError("schedule-delete", handle.Delete(ctx))
}

// DescribeSchedule returns the current schedule state.
func (d *Definition) DescribeSchedule(ctx context.Context, c client.Client) (ScheduleDetail, error) {
	handle := c.ScheduleClient().GetHandle(ctx, d.Name)
	desc, err := handle.Describe(ctx)
	if err != nil {
		return ScheduleDetail{}, translateSDKError("schedule-describe", err)
	}
	sum := ScheduleSummary{
		ID:           d.Name,
		WorkflowType: d.Name,
	}
	if desc.Schedule.State != nil {
		sum.Paused = desc.Schedule.State.Paused
		sum.Note = desc.Schedule.State.Note
	}
	det := ScheduleDetail{
		ScheduleSummary: sum,
	}
	if d.Schedule != nil {
		det.Spec = *d.Schedule
	}
	if len(desc.Info.NextActionTimes) > 0 {
		nt := desc.Info.NextActionTimes[0]
		det.NextRunTime = &nt
	}
	if len(desc.Info.RecentActions) > 0 {
		last := desc.Info.RecentActions[len(desc.Info.RecentActions)-1]
		lt := last.ActualTime
		det.LastRunTime = &lt
	}
	// RecentRuns left empty for v1; populating requires extra queries.
	return det, nil
}
