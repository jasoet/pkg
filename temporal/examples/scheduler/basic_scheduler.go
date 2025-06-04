//go:build example

package scheduler

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/temporal"
	"github.com/jasoet/pkg/temporal/examples/workflows"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

const (
	// TaskQueue is the task queue name used for the examples
	TaskQueue = "example-task-queue"
)

// IntervalScheduler demonstrates a scheduler that uses interval-based scheduling.
// It shows how to create a schedule that runs a workflow at regular intervals.
func RunIntervalScheduler() error {
	logger := log.With().Str("component", "IntervalScheduler").Logger()
	logger.Info().Msg("Starting interval scheduler example")

	// Step 1: Create a Temporal client and schedule manager
	logger.Info().Msg("Creating schedule manager")
	config := temporal.DefaultConfig()
	scheduleManager, err := temporal.NewScheduleManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule manager")
		return err
	}
	defer scheduleManager.Close()

	// Step 2: Create an interval-based schedule
	ctx := context.Background()
	scheduleName := "interval-schedule-example"

	// Set up workflow schedule options with an interval
	options := temporal.WorkflowScheduleOptions{
		WorkflowID: "scheduled-workflow-interval",
		Workflow:   workflows.ScheduledWorkflow,
		TaskQueue:  TaskQueue,
		Interval:   1 * time.Minute, // Run every minute
	}

	// Create the schedule
	logger.Info().
		Str("scheduleName", scheduleName).
		Dur("interval", options.Interval).
		Msg("Creating interval-based schedule")

	scheduleHandle, err := scheduleManager.CreateWorkflowSchedule(ctx, scheduleName, options)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule")
		return err
	}

	logger.Info().
		Str("scheduleName", scheduleName).
		Msg("Schedule created successfully")

	// Step 3: Wait for user to terminate
	logger.Info().Msg("Schedule is running, press Ctrl+C to exit and delete the schedule")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Step 4: Clean up the schedule
	logger.Info().Msg("Deleting schedule")
	err = scheduleHandle.Delete(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete schedule")
		return err
	}

	logger.Info().Msg("Schedule deleted successfully")
	return nil
}

// CronScheduler demonstrates a scheduler that uses cron-based scheduling.
// It shows how to create a schedule that runs a workflow based on a cron expression.
func RunCronScheduler() error {
	logger := log.With().Str("component", "CronScheduler").Logger()
	logger.Info().Msg("Starting cron scheduler example")

	// Step 1: Create a Temporal client and schedule manager
	logger.Info().Msg("Creating schedule manager")
	config := temporal.DefaultConfig()
	scheduleManager, err := temporal.NewScheduleManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule manager")
		return err
	}
	defer scheduleManager.Close()

	// Step 2: Create a cron-based schedule
	ctx := context.Background()
	scheduleName := "cron-schedule-example"

	// Create schedule options with a cron expression
	// This example runs every 5 minutes
	scheduleOptions := client.ScheduleOptions{
		ID: scheduleName,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"*/5 * * * *"}, // Every 5 minutes
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "scheduled-workflow-cron",
			Workflow:  workflows.ScheduledWorkflow,
			TaskQueue: TaskQueue,
		},
	}

	// Create the schedule
	logger.Info().
		Str("scheduleName", scheduleName).
		Str("cronExpression", "*/5 * * * *").
		Msg("Creating cron-based schedule")

	scheduleHandle, err := scheduleManager.CreateSchedule(ctx, scheduleOptions)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule")
		return err
	}

	logger.Info().
		Str("scheduleName", scheduleName).
		Msg("Schedule created successfully")

	// Step 3: Wait for user to terminate
	logger.Info().Msg("Schedule is running, press Ctrl+C to exit and delete the schedule")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Step 4: Clean up the schedule
	logger.Info().Msg("Deleting schedule")
	err = scheduleHandle.Delete(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete schedule")
		return err
	}

	logger.Info().Msg("Schedule deleted successfully")
	return nil
}

// OneTimeScheduler demonstrates a scheduler that creates a one-time schedule.
// It shows how to create a schedule that runs a workflow once at a specific time.
func RunOneTimeScheduler() error {
	logger := log.With().Str("component", "OneTimeScheduler").Logger()
	logger.Info().Msg("Starting one-time scheduler example")

	// Step 1: Create a Temporal client and schedule manager
	logger.Info().Msg("Creating schedule manager")
	config := temporal.DefaultConfig()
	scheduleManager, err := temporal.NewScheduleManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule manager")
		return err
	}
	defer scheduleManager.Close()

	// Step 2: Create a one-time schedule
	ctx := context.Background()
	scheduleName := "one-time-schedule-example"

	// Schedule the workflow to run 1 minute from now
	startTime := time.Now().Add(1 * time.Minute)

	// Create schedule options with a specific start time
	scheduleOptions := client.ScheduleOptions{
		ID: scheduleName,
		Spec: client.ScheduleSpec{
			StartAt: startTime,
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "scheduled-workflow-one-time",
			Workflow:  workflows.ScheduledWorkflow,
			TaskQueue: TaskQueue,
		},
	}

	// Create the schedule
	logger.Info().
		Str("scheduleName", scheduleName).
		Time("startTime", startTime).
		Msg("Creating one-time schedule")

	scheduleHandle, err := scheduleManager.CreateSchedule(ctx, scheduleOptions)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule")
		return err
	}

	logger.Info().
		Str("scheduleName", scheduleName).
		Msg("Schedule created successfully")

	// Step 3: Wait for user to terminate
	logger.Info().Msg("Schedule is running, press Ctrl+C to exit and delete the schedule")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Step 4: Clean up the schedule
	logger.Info().Msg("Deleting schedule")
	err = scheduleHandle.Delete(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete schedule")
		return err
	}

	logger.Info().Msg("Schedule deleted successfully")
	return nil
}

// MultiScheduleManager demonstrates managing multiple schedules.
// It shows how to create, list, and delete multiple schedules.
func RunMultiScheduleManager() error {
	logger := log.With().Str("component", "MultiScheduleManager").Logger()
	logger.Info().Msg("Starting multi-schedule manager example")

	// Step 1: Create a Temporal client and schedule manager
	logger.Info().Msg("Creating schedule manager")
	config := temporal.DefaultConfig()
	scheduleManager, err := temporal.NewScheduleManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule manager")
		return err
	}
	defer scheduleManager.Close()

	// Step 2: Create multiple schedules
	ctx := context.Background()

	// Create an interval-based schedule
	intervalScheduleName := "multi-interval-schedule"
	intervalOptions := temporal.WorkflowScheduleOptions{
		WorkflowID: "multi-scheduled-workflow-interval",
		Workflow:   workflows.ScheduledWorkflow,
		TaskQueue:  TaskQueue,
		Interval:   2 * time.Minute, // Run every 2 minutes
	}

	logger.Info().
		Str("scheduleName", intervalScheduleName).
		Dur("interval", intervalOptions.Interval).
		Msg("Creating interval-based schedule")

	_, err = scheduleManager.CreateWorkflowSchedule(ctx, intervalScheduleName, intervalOptions)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create interval schedule")
		return err
	}

	// Create a cron-based schedule
	cronScheduleName := "multi-cron-schedule"

	cronOptions := client.ScheduleOptions{
		ID: cronScheduleName,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"*/10 * * * *"}, // Every 10 minutes
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "multi-scheduled-workflow-cron",
			Workflow:  workflows.ScheduledWorkflow,
			TaskQueue: TaskQueue,
		},
	}

	logger.Info().
		Str("scheduleName", cronScheduleName).
		Str("cronExpression", "*/10 * * * *").
		Msg("Creating cron-based schedule")

	_, err = scheduleManager.CreateSchedule(ctx, cronOptions)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create cron schedule")
		return err
	}

	// Step 3: List all schedules
	logger.Info().Msg("Listing all schedules")
	scheduleHandlers := scheduleManager.GetScheduleHandlers()
	for name := range scheduleHandlers {
		logger.Info().Str("scheduleName", name).Msg("Found schedule")
	}

	// Step 4: Wait for user to terminate
	logger.Info().Msg("Schedules are running, press Ctrl+C to exit and delete all schedules")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Step 5: Clean up all schedules
	logger.Info().Msg("Deleting all schedules")
	err = scheduleManager.DeleteSchedules(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete schedules")
		return err
	}

	logger.Info().Msg("All schedules deleted successfully")
	return nil
}

// To run these examples, you can use the following code:
//
// ```go
// package main
//
// import (
//     "github.com/amanata-dev/twc-report-backend/pkg/temporal/examples/scheduler"
//     "github.com/rs/zerolog/log"
// )
//
// func main() {
//     // Choose which scheduler example to run
//     if err := scheduler.RunIntervalScheduler(); err != nil {
//         log.Fatal().Err(err).Msg("Scheduler failed")
//     }
//
//     // Or run the cron scheduler
//     // if err := scheduler.RunCronScheduler(); err != nil {
//     //     log.Fatal().Err(err).Msg("Scheduler failed")
//     // }
//
//     // Or run the one-time scheduler
//     // if err := scheduler.RunOneTimeScheduler(); err != nil {
//     //     log.Fatal().Err(err).Msg("Scheduler failed")
//     // }
//
//     // Or run the multi-schedule manager
//     // if err := scheduler.RunMultiScheduleManager(); err != nil {
//     //     log.Fatal().Err(err).Msg("Scheduler failed")
//     // }
// }
// ```
//
// Note: Before running these examples, make sure you have a worker running
// that can process the scheduled workflows. You can use the worker examples
// from the worker package.
