package temporal

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

type WorkflowScheduleOptions struct {
	WorkflowID string
	Workflow   any
	TaskQueue  string
	Interval   time.Duration
	Args       []any
}

type ScheduleManager struct {
	client           client.Client
	scheduleHandlers map[string]client.ScheduleHandle
}

func NewScheduleManager(clientOrConfig interface{}) *ScheduleManager {
	logger := log.With().Str("function", "temporal.NewScheduleManager").Logger()

	var temporalClient client.Client

	switch v := clientOrConfig.(type) {
	case client.Client:
		// If passed a client directly, use it
		temporalClient = v
		logger.Debug().Msg("Using provided Temporal client for Schedule Manager")
	case *Config:
		// If passed a config, create a new client
		logger.Debug().
			Str("hostPort", v.HostPort).
			Str("namespace", v.Namespace).
			Msg("Creating new Schedule Manager with config")

		var err error
		temporalClient, err = NewClientWithMetrics(v, false)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create Temporal client for Schedule Manager")
			return nil
		}
	default:
		logger.Error().Msg("Invalid argument type for NewScheduleManager")
		return nil
	}

	logger.Debug().Msg("Schedule Manager created successfully")
	return &ScheduleManager{
		client:           temporalClient,
		scheduleHandlers: make(map[string]client.ScheduleHandle),
	}
}

func (sm *ScheduleManager) Close() {
	logger := log.With().Str("function", "ScheduleManager.Close").Logger()
	logger.Debug().Msg("Closing Schedule Manager")

	if sm.client != nil {
		logger.Debug().Msg("Closing Temporal client")
		sm.client.Close()
	}

	logger.Debug().Msg("Schedule Manager closed")
}

func (sm *ScheduleManager) CreateSchedule(ctx context.Context, scheduleID string, spec client.ScheduleSpec, action *client.ScheduleWorkflowAction) (client.ScheduleHandle, error) {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.CreateSchedule").Logger()
	logger.Debug().
		Str("scheduleID", scheduleID).
		Msg("Creating schedule")

	options := client.ScheduleOptions{
		ID:     scheduleID,
		Spec:   spec,
		Action: action,
	}

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error().Err(err).
			Str("scheduleID", scheduleID).
			Msg("Failed to create schedule")
		return nil, err
	}

	logger.Debug().
		Str("scheduleID", scheduleID).
		Msg("Adding schedule to handlers map")
	sm.scheduleHandlers[scheduleID] = sh

	logger.Debug().
		Str("scheduleID", scheduleID).
		Msg("Schedule created successfully")
	return sh, nil
}

func (sm *ScheduleManager) CreateScheduleWithOptions(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.CreateSchedule").Logger()
	logger.Debug().
		Str("scheduleName", options.ID).
		Msg("Creating schedule")

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error().Err(err).
			Str("scheduleName", options.ID).
			Msg("Failed to create schedule")
		return nil, err
	}

	logger.Debug().
		Str("scheduleName", options.ID).
		Msg("Adding schedule to handlers map")
	sm.scheduleHandlers[options.ID] = sh

	logger.Debug().
		Str("scheduleName", options.ID).
		Msg("Schedule created successfully")
	return sh, nil
}

func (sm *ScheduleManager) CreateWorkflowSchedule(ctx context.Context, scheduleName string, options WorkflowScheduleOptions) (client.ScheduleHandle, error) {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.CreateWorkflowSchedule").Logger()
	logger.Debug().
		Str("scheduleName", scheduleName).
		Str("workflowID", options.WorkflowID).
		Str("taskQueue", options.TaskQueue).
		Dur("interval", options.Interval).
		Msg("Creating workflow schedule")

	scheduleOptions := client.ScheduleOptions{
		ID: scheduleName,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: options.Interval,
				},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        options.WorkflowID,
			Workflow:  options.Workflow,
			TaskQueue: options.TaskQueue,
			Args:      options.Args,
		},
	}

	handle, err := sm.CreateScheduleWithOptions(ctx, scheduleOptions)
	if err != nil {
		logger.Error().Err(err).
			Str("scheduleName", scheduleName).
			Str("workflowID", options.WorkflowID).
			Msg("Failed to create workflow schedule")
		return nil, err
	}

	logger.Debug().
		Str("scheduleName", scheduleName).
		Str("workflowID", options.WorkflowID).
		Msg("Workflow schedule created successfully")
	return handle, nil
}

func (sm *ScheduleManager) DeleteSchedules(ctx context.Context) error {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.DeleteSchedules").Logger()

	scheduleCount := len(sm.scheduleHandlers)
	logger.Debug().Int("scheduleCount", scheduleCount).Msg("Deleting all Temporal schedules")

	if scheduleCount == 0 {
		logger.Debug().Msg("No schedules to delete")
		return nil
	}

	for name, handle := range sm.scheduleHandlers {
		logger.Debug().Str("scheduleName", name).Msg("Deleting schedule")
		err := handle.Delete(ctx)
		if err != nil {
			logger.Error().Err(err).Str("scheduleName", name).Msg("Failed to delete schedule")
			return err
		}
		logger.Debug().Str("scheduleName", name).Msg("Schedule deleted successfully")
	}

	logger.Debug().Msg("Clearing schedule handlers map")
	sm.scheduleHandlers = make(map[string]client.ScheduleHandle)

	logger.Debug().Int("deletedCount", scheduleCount).Msg("All schedules deleted successfully")
	return nil
}

func (sm *ScheduleManager) GetClient() client.Client {
	return sm.client
}

func (sm *ScheduleManager) GetScheduleHandlers() map[string]client.ScheduleHandle {
	return sm.scheduleHandlers
}

// GetSchedule retrieves a schedule handle by ID
func (sm *ScheduleManager) GetSchedule(ctx context.Context, scheduleID string) (client.ScheduleHandle, error) {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.GetSchedule").Logger()
	logger.Debug().Str("scheduleID", scheduleID).Msg("Getting schedule")

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)

	// Test if the schedule exists by trying to describe it
	_, err := handle.Describe(ctx)
	if err != nil {
		logger.Error().Err(err).Str("scheduleID", scheduleID).Msg("Failed to get schedule")
		return nil, err
	}

	logger.Debug().Str("scheduleID", scheduleID).Msg("Schedule retrieved successfully")
	return handle, nil
}

// ListSchedules lists all schedules with a limit
func (sm *ScheduleManager) ListSchedules(ctx context.Context, limit int) ([]*client.ScheduleListEntry, error) {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.ListSchedules").Logger()
	logger.Debug().Int("limit", limit).Msg("Listing schedules")

	scheduleClient := sm.client.ScheduleClient()

	var schedules []*client.ScheduleListEntry
	iter, err := scheduleClient.List(ctx, client.ScheduleListOptions{
		PageSize: limit,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create schedule list iterator")
		return nil, err
	}

	for iter.HasNext() {
		schedule, err := iter.Next()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get next schedule from iterator")
			return nil, err
		}
		schedules = append(schedules, schedule)

		if len(schedules) >= limit {
			break
		}
	}

	logger.Debug().Int("count", len(schedules)).Msg("Schedules listed successfully")
	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (sm *ScheduleManager) UpdateSchedule(ctx context.Context, scheduleID string, spec client.ScheduleSpec, action *client.ScheduleWorkflowAction) error {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.UpdateSchedule").Logger()
	logger.Debug().Str("scheduleID", scheduleID).Msg("Updating schedule")

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)

	err := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			// Get the current schedule from input and modify it
			schedule := input.Description.Schedule

			// Update the spec
			schedule.Spec = &spec

			// Update the action if provided
			if action != nil {
				schedule.Action = action
			}

			return &client.ScheduleUpdate{
				Schedule: &schedule,
			}, nil
		},
	})

	if err != nil {
		logger.Error().Err(err).Str("scheduleID", scheduleID).Msg("Failed to update schedule")
		return err
	}

	logger.Debug().Str("scheduleID", scheduleID).Msg("Schedule updated successfully")
	return nil
}

// DeleteSchedule deletes a specific schedule by ID
func (sm *ScheduleManager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	logger := log.With().Ctx(ctx).Str("function", "ScheduleManager.DeleteSchedule").Logger()
	logger.Debug().Str("scheduleID", scheduleID).Msg("Deleting schedule")

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		logger.Error().Err(err).Str("scheduleID", scheduleID).Msg("Failed to delete schedule")
		return err
	}

	// Remove from local handlers map if it exists
	delete(sm.scheduleHandlers, scheduleID)

	logger.Debug().Str("scheduleID", scheduleID).Msg("Schedule deleted successfully")
	return nil
}
