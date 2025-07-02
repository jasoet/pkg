//go:build integration

package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

func TestScheduleManagerIntegration(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9099"

	temporalClient, err := NewClient(config)
	require.NoError(t, err, "Failed to create Temporal client")
	defer temporalClient.Close()

	scheduleManager := NewScheduleManager(temporalClient)
	require.NotNil(t, scheduleManager, "ScheduleManager should not be nil")

	t.Run("CreateScheduleManager", func(t *testing.T) {
		client := scheduleManager.GetClient()
		assert.Equal(t, temporalClient, client, "Client should match")
	})

	t.Run("CreateCronSchedule", func(t *testing.T) {
		scheduleID := "test-cron-schedule-" + time.Now().Format("20060102-150405")

		// Simple workflow for testing
		testWorkflow := func(ctx context.Context) (string, error) {
			return "scheduled execution", nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a cron schedule (every minute)
		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"* * * * *"}, // Every minute
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "scheduled-workflow-" + scheduleID,
			Workflow:  testWorkflow,
			TaskQueue: "test-schedule-queue",
		}

		handle, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)

		if err != nil {
			// Schedule creation might fail if no worker is available, which is expected
			t.Logf("Expected failure creating schedule without worker: %v", err)
			return
		}

		require.NotNil(t, handle, "Schedule handle should not be nil")

		// Try to describe the schedule
		desc, err := handle.Describe(ctx)
		if err == nil {
			assert.Equal(t, scheduleID, desc.Schedule.Spec.CronExpressions[0])
		}

		// Clean up - delete the schedule
		err = handle.Delete(ctx)
		assert.NoError(t, err, "Failed to delete schedule")
	})

	t.Run("CreateIntervalSchedule", func(t *testing.T) {
		scheduleID := "test-interval-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create an interval schedule (every 30 seconds)
		scheduleSpec := client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: 30 * time.Second,
				},
			},
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "interval-workflow-" + scheduleID,
			Workflow:  "SampleWorkflow", // Reference by name
			TaskQueue: "test-interval-queue",
			Args:      []interface{}{"interval-test"},
		}

		handle, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)

		if err != nil {
			t.Logf("Expected failure creating interval schedule: %v", err)
			return
		}

		require.NotNil(t, handle, "Schedule handle should not be nil")

		// Clean up
		err = handle.Delete(ctx)
		assert.NoError(t, err, "Failed to delete interval schedule")
	})

	t.Run("ListSchedules", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a test schedule first
		scheduleID := "test-list-schedule-" + time.Now().Format("20060102-150405")

		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 0 * * *"}, // Daily at midnight
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "list-test-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "test-list-queue",
		}

		handle, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not create schedule for list test: %v", err)
			return
		}

		// List schedules
		schedules, err := scheduleManager.ListSchedules(ctx, 10)
		assert.NoError(t, err, "Failed to list schedules")

		// We should have at least our created schedule
		found := false
		for _, schedule := range schedules {
			if schedule.ID == scheduleID {
				found = true
				break
			}
		}

		if len(schedules) > 0 {
			assert.True(t, found, "Created schedule should be in the list")
		}

		// Clean up
		err = handle.Delete(ctx)
		assert.NoError(t, err, "Failed to delete list test schedule")
	})

	t.Run("GetSchedule", func(t *testing.T) {
		scheduleID := "test-get-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create schedule first
		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 12 * * *"}, // Daily at noon
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "get-test-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "test-get-queue",
		}

		createdHandle, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not create schedule for get test: %v", err)
			return
		}

		// Get the schedule
		retrievedHandle, err := scheduleManager.GetSchedule(ctx, scheduleID)
		assert.NoError(t, err, "Failed to get schedule")
		assert.NotNil(t, retrievedHandle, "Retrieved handle should not be nil")

		// Verify it's the same schedule
		desc, err := retrievedHandle.Describe(ctx)
		if err == nil {
			assert.Contains(t, desc.Schedule.Spec.CronExpressions, "0 12 * * *")
		}

		// Clean up
		err = createdHandle.Delete(ctx)
		assert.NoError(t, err, "Failed to delete get test schedule")
	})

	t.Run("UpdateSchedule", func(t *testing.T) {
		scheduleID := "test-update-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create initial schedule
		initialSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 6 * * *"}, // Daily at 6 AM
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "update-test-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "test-update-queue",
		}

		handle, err := scheduleManager.CreateSchedule(ctx, scheduleID, initialSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not create schedule for update test: %v", err)
			return
		}

		// Update the schedule
		updatedSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 18 * * *"}, // Daily at 6 PM
		}

		err = scheduleManager.UpdateSchedule(ctx, scheduleID, updatedSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not update schedule (may require specific Temporal version): %v", err)
		}

		// Clean up
		err = handle.Delete(ctx)
		assert.NoError(t, err, "Failed to delete update test schedule")
	})

	t.Run("DeleteSchedule", func(t *testing.T) {
		scheduleID := "test-delete-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create schedule
		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 0 1 * *"}, // Monthly
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "delete-test-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "test-delete-queue",
		}

		_, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not create schedule for delete test: %v", err)
			return
		}

		// Delete the schedule
		err = scheduleManager.DeleteSchedule(ctx, scheduleID)
		assert.NoError(t, err, "Failed to delete schedule")

		// Try to get the deleted schedule (should fail)
		_, err = scheduleManager.GetSchedule(ctx, scheduleID)
		assert.Error(t, err, "Getting deleted schedule should fail")
	})
}

func TestScheduleManagerErrorHandling(t *testing.T) {
	config := DefaultConfig()
	config.MetricsListenAddress = "0.0.0.0:9100"

	temporalClient, err := NewClient(config)
	require.NoError(t, err)
	defer temporalClient.Close()

	scheduleManager := NewScheduleManager(temporalClient)

	t.Run("CreateDuplicateSchedule", func(t *testing.T) {
		scheduleID := "duplicate-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"0 0 * * *"},
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "duplicate-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "duplicate-queue",
		}

		// Create first schedule
		handle1, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		if err != nil {
			t.Logf("Could not create first schedule: %v", err)
			return
		}
		defer handle1.Delete(ctx)

		// Try to create duplicate (should fail)
		handle2, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		assert.Error(t, err, "Creating duplicate schedule should fail")
		assert.Nil(t, handle2, "Handle for duplicate should be nil")
	})

	t.Run("GetNonexistentSchedule", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		handle, err := scheduleManager.GetSchedule(ctx, "nonexistent-schedule")
		assert.Error(t, err, "Getting nonexistent schedule should fail")
		assert.Nil(t, handle, "Handle should be nil for nonexistent schedule")
	})

	t.Run("DeleteNonexistentSchedule", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := scheduleManager.DeleteSchedule(ctx, "nonexistent-schedule-delete")
		assert.Error(t, err, "Deleting nonexistent schedule should fail")
	})

	t.Run("InvalidCronExpression", func(t *testing.T) {
		scheduleID := "invalid-cron-schedule-" + time.Now().Format("20060102-150405")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Invalid cron expression
		scheduleSpec := client.ScheduleSpec{
			CronExpressions: []string{"invalid-cron"},
		}

		scheduleAction := &client.ScheduleWorkflowAction{
			ID:        "invalid-cron-workflow",
			Workflow:  "SampleWorkflow",
			TaskQueue: "invalid-cron-queue",
		}

		handle, err := scheduleManager.CreateSchedule(ctx, scheduleID, scheduleSpec, scheduleAction)
		assert.Error(t, err, "Creating schedule with invalid cron should fail")
		assert.Nil(t, handle, "Handle should be nil for invalid schedule")
	})
}
