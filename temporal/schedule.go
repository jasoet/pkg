package temporal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/jasoet/pkg/v3/otel"
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
	mu               sync.RWMutex
	scheduleHandlers map[string]client.ScheduleHandle
}

// NewScheduleManager creates a ScheduleManager using the provided client.
// The caller retains ownership of the client and is responsible for closing
// it; Close does not close the client.
func NewScheduleManager(temporalClient client.Client) (*ScheduleManager, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "temporal.NewScheduleManager")

	if temporalClient == nil {
		return nil, fmt.Errorf("temporal client must not be nil")
	}

	logger.Debug("Schedule Manager created successfully")
	return &ScheduleManager{
		client:           temporalClient,
		scheduleHandlers: make(map[string]client.ScheduleHandle),
	}, nil
}

// Close closes the ScheduleManager. It does not close the Temporal client;
// the caller owns the client and must close it. The ctx parameter is used for
// logging only.
func (sm *ScheduleManager) Close(ctx context.Context) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.Close")

	logger.Debug("Closing Schedule Manager")
	logger.Debug("Schedule Manager closed")
}

func (sm *ScheduleManager) CreateSchedule(ctx context.Context, scheduleID string, spec client.ScheduleSpec, action *client.ScheduleWorkflowAction) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.CreateSchedule")

	logger.Debug("Creating schedule", otel.F("scheduleID", scheduleID))

	options := client.ScheduleOptions{
		ID:     scheduleID,
		Spec:   spec,
		Action: action,
	}

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error(err, "Failed to create schedule", otel.F("scheduleID", scheduleID))
		return nil, fmt.Errorf("create schedule %q: %w", scheduleID, err)
	}

	sm.mu.Lock()
	sm.scheduleHandlers[scheduleID] = sh
	sm.mu.Unlock()

	logger.Debug("Schedule created successfully", otel.F("scheduleID", scheduleID))
	return sh, nil
}

func (sm *ScheduleManager) CreateScheduleWithOptions(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.CreateScheduleWithOptions")

	logger.Debug("Creating schedule", otel.F("scheduleName", options.ID))

	sh, err := sm.client.ScheduleClient().Create(ctx, options)
	if err != nil {
		logger.Error(err, "Failed to create schedule", otel.F("scheduleName", options.ID))
		return nil, fmt.Errorf("create schedule %q: %w", options.ID, err)
	}

	sm.mu.Lock()
	sm.scheduleHandlers[options.ID] = sh
	sm.mu.Unlock()

	logger.Debug("Schedule created successfully", otel.F("scheduleName", options.ID))
	return sh, nil
}

func (sm *ScheduleManager) CreateWorkflowSchedule(ctx context.Context, scheduleName string, options WorkflowScheduleOptions) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.CreateWorkflowSchedule")

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
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.DeleteSchedules")

	sm.mu.RLock()
	scheduleCount := len(sm.scheduleHandlers)
	sm.mu.RUnlock()

	logger.Debug("Deleting all Temporal schedules", otel.F("scheduleCount", scheduleCount))

	if scheduleCount == 0 {
		logger.Debug("No schedules to delete")
		return nil
	}

	sm.mu.RLock()
	for name, handle := range sm.scheduleHandlers {
		logger.Debug("Deleting schedule", otel.F("scheduleName", name))
		err := handle.Delete(ctx)
		if err != nil {
			sm.mu.RUnlock()
			logger.Error(err, "Failed to delete schedule", otel.F("scheduleName", name))
			return fmt.Errorf("delete schedule %q: %w", name, err)
		}
		logger.Debug("Schedule deleted successfully", otel.F("scheduleName", name))
	}
	sm.mu.RUnlock()

	sm.mu.Lock()
	sm.scheduleHandlers = make(map[string]client.ScheduleHandle)
	sm.mu.Unlock()

	logger.Debug("All schedules deleted successfully", otel.F("deletedCount", scheduleCount))
	return nil
}

func (sm *ScheduleManager) GetClient() client.Client {
	return sm.client
}

func (sm *ScheduleManager) GetScheduleHandlers() map[string]client.ScheduleHandle {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	// Return a copy to prevent concurrent map access
	handlers := make(map[string]client.ScheduleHandle, len(sm.scheduleHandlers))
	for k, v := range sm.scheduleHandlers {
		handlers[k] = v
	}
	return handlers
}

// GetSchedule retrieves a schedule handle by ID
func (sm *ScheduleManager) GetSchedule(ctx context.Context, scheduleID string) (client.ScheduleHandle, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.GetSchedule")

	logger.Debug("Getting schedule", otel.F("scheduleID", scheduleID))

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)

	// Test if the schedule exists by trying to describe it
	_, err := handle.Describe(ctx)
	if err != nil {
		logger.Error(err, "Failed to get schedule", otel.F("scheduleID", scheduleID))
		return nil, fmt.Errorf("get schedule %q: %w", scheduleID, err)
	}

	logger.Debug("Schedule retrieved successfully", otel.F("scheduleID", scheduleID))
	return handle, nil
}

// ListSchedules lists all schedules with a limit
func (sm *ScheduleManager) ListSchedules(ctx context.Context, limit int) ([]*client.ScheduleListEntry, error) {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.ListSchedules")

	logger.Debug("Listing schedules", otel.F("limit", limit))

	scheduleClient := sm.client.ScheduleClient()

	var schedules []*client.ScheduleListEntry
	iter, err := scheduleClient.List(ctx, client.ScheduleListOptions{
		PageSize: limit,
	})
	if err != nil {
		logger.Error(err, "Failed to create schedule list iterator")
		return nil, fmt.Errorf("list schedules: %w", err)
	}

	for iter.HasNext() {
		schedule, err := iter.Next()
		if err != nil {
			logger.Error(err, "Failed to get next schedule from iterator")
			return nil, fmt.Errorf("iterate schedules: %w", err)
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
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.UpdateSchedule")

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
		return fmt.Errorf("update schedule %q: %w", scheduleID, err)
	}

	logger.Debug("Schedule updated successfully", otel.F("scheduleID", scheduleID))
	return nil
}

// DeleteSchedule deletes a specific schedule by ID
func (sm *ScheduleManager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "ScheduleManager.DeleteSchedule")

	logger.Debug("Deleting schedule", otel.F("scheduleID", scheduleID))

	handle := sm.client.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		logger.Error(err, "Failed to delete schedule", otel.F("scheduleID", scheduleID))
		return fmt.Errorf("delete schedule %q: %w", scheduleID, err)
	}

	// Remove from local handlers map if it exists
	sm.mu.Lock()
	delete(sm.scheduleHandlers, scheduleID)
	sm.mu.Unlock()

	logger.Debug("Schedule deleted successfully", otel.F("scheduleID", scheduleID))
	return nil
}
