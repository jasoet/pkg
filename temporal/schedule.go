package temporal

import (
	"context"
	"time"

	"github.com/jasoet/pkg/v2/otel"
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
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.NewScheduleManager")

	var temporalClient client.Client

	switch v := clientOrConfig.(type) {
	case client.Client:
		// If passed a client directly, use it
		temporalClient = v
		logger.Debug("Using provided Temporal client for Schedule Manager")
	case *Config:
		// If passed a config, create a new client
		logger.Debug("Creating new Schedule Manager with config",
			otel.F("hostPort", v.HostPort),
			otel.F("namespace", v.Namespace))

		var err error
		temporalClient, err = NewClientWithMetrics(v, false)
		if err != nil {
			logger.Error(err, "Failed to create Temporal client for Schedule Manager")
			return nil
		}
	default:
		logger.Error(nil, "Invalid argument type for NewScheduleManager")
		return nil
	}

	logger.Debug("Schedule Manager created successfully")
	return &ScheduleManager{
		client:           temporalClient,
		scheduleHandlers: make(map[string]client.ScheduleHandle),
	}
}

func (sm *ScheduleManager) Close() {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.Close")

	logger.Debug("Closing Schedule Manager")

	if sm.client != nil {
		logger.Debug("Closing Temporal client")
		sm.client.Close()
	}

	logger.Debug("Schedule Manager closed")
}

func (sm *ScheduleManager) CreateSchedule(ctx context.Context, scheduleID string, spec client.ScheduleSpec, action *client.ScheduleWorkflowAction) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.CreateSchedule")

	logger.Debug("Creating schedule", otel.F("scheduleID", scheduleID))

	options := client.ScheduleOptions{
		ID:     scheduleID,
		Spec:   spec,
		Action: action,
	}

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error(err, "Failed to create schedule", otel.F("scheduleID", scheduleID))
		return nil, err
	}

	logger.Debug("Adding schedule to handlers map", otel.F("scheduleID", scheduleID))
	sm.scheduleHandlers[scheduleID] = sh

	logger.Debug("Schedule created successfully", otel.F("scheduleID", scheduleID))
	return sh, nil
}

func (sm *ScheduleManager) CreateScheduleWithOptions(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.CreateScheduleWithOptions")

	logger.Debug("Creating schedule", otel.F("scheduleName", options.ID))

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error(err, "Failed to create schedule", otel.F("scheduleName", options.ID))
		return nil, err
	}

	logger.Debug("Adding schedule to handlers map", otel.F("scheduleName", options.ID))
	sm.scheduleHandlers[options.ID] = sh

	logger.Debug("Schedule created successfully", otel.F("scheduleName", options.ID))
	return sh, nil
}

func (sm *ScheduleManager) CreateWorkflowSchedule(ctx context.Context, scheduleName string, options WorkflowScheduleOptions) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.CreateWorkflowSchedule")

	logger.Debug("Creating workflow schedule",
		otel.F("scheduleName", scheduleName),
		otel.F("workflowID", options.WorkflowID),
		otel.F("taskQueue", options.TaskQueue),
		otel.F("interval", options.Interval))

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
		logger.Error(err, "Failed to create workflow schedule",
			otel.F("scheduleName", scheduleName),
			otel.F("workflowID", options.WorkflowID))
		return nil, err
	}

	logger.Debug("Workflow schedule created successfully",
		otel.F("scheduleName", scheduleName),
		otel.F("workflowID", options.WorkflowID))
	return handle, nil
}

func (sm *ScheduleManager) DeleteSchedules(ctx context.Context) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.DeleteSchedules")

	scheduleCount := len(sm.scheduleHandlers)
	logger.Debug("Deleting all Temporal schedules", otel.F("scheduleCount", scheduleCount))

	if scheduleCount == 0 {
		logger.Debug("No schedules to delete")
		return nil
	}

	for name, handle := range sm.scheduleHandlers {
		logger.Debug("Deleting schedule", otel.F("scheduleName", name))
		err := handle.Delete(ctx)
		if err != nil {
			logger.Error(err, "Failed to delete schedule", otel.F("scheduleName", name))
			return err
		}
		logger.Debug("Schedule deleted successfully", otel.F("scheduleName", name))
	}

	logger.Debug("Clearing schedule handlers map")
	sm.scheduleHandlers = make(map[string]client.ScheduleHandle)

	logger.Debug("All schedules deleted successfully", otel.F("deletedCount", scheduleCount))
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
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.GetSchedule")

	logger.Debug("Getting schedule", otel.F("scheduleID", scheduleID))

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)

	// Test if the schedule exists by trying to describe it
	_, err := handle.Describe(ctx)
	if err != nil {
		logger.Error(err, "Failed to get schedule", otel.F("scheduleID", scheduleID))
		return nil, err
	}

	logger.Debug("Schedule retrieved successfully", otel.F("scheduleID", scheduleID))
	return handle, nil
}

// ListSchedules lists all schedules with a limit
func (sm *ScheduleManager) ListSchedules(ctx context.Context, limit int) ([]*client.ScheduleListEntry, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.ListSchedules")

	logger.Debug("Listing schedules", otel.F("limit", limit))

	scheduleClient := sm.client.ScheduleClient()

	var schedules []*client.ScheduleListEntry
	iter, err := scheduleClient.List(ctx, client.ScheduleListOptions{
		PageSize: limit,
	})
	if err != nil {
		logger.Error(err, "Failed to create schedule list iterator")
		return nil, err
	}

	for iter.HasNext() {
		schedule, err := iter.Next()
		if err != nil {
			logger.Error(err, "Failed to get next schedule from iterator")
			return nil, err
		}
		schedules = append(schedules, schedule)

		if len(schedules) >= limit {
			break
		}
	}

	logger.Debug("Schedules listed successfully", otel.F("count", len(schedules)))
	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (sm *ScheduleManager) UpdateSchedule(ctx context.Context, scheduleID string, spec client.ScheduleSpec, action *client.ScheduleWorkflowAction) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.UpdateSchedule")

	logger.Debug("Updating schedule", otel.F("scheduleID", scheduleID))

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
		logger.Error(err, "Failed to update schedule", otel.F("scheduleID", scheduleID))
		return err
	}

	logger.Debug("Schedule updated successfully", otel.F("scheduleID", scheduleID))
	return nil
}

// DeleteSchedule deletes a specific schedule by ID
func (sm *ScheduleManager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "ScheduleManager.DeleteSchedule")

	logger.Debug("Deleting schedule", otel.F("scheduleID", scheduleID))

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		logger.Error(err, "Failed to delete schedule", otel.F("scheduleID", scheduleID))
		return err
	}

	// Remove from local handlers map if it exists
	delete(sm.scheduleHandlers, scheduleID)

	logger.Debug("Schedule deleted successfully", otel.F("scheduleID", scheduleID))
	return nil
}
