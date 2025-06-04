package temporal

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"time"
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

func NewScheduleManager(config *Config) (*ScheduleManager, error) {
	logger := log.With().Str("function", "temporal.NewScheduleManager").Logger()
	logger.Debug().
		Str("hostPort", config.HostPort).
		Str("namespace", config.Namespace).
		Msg("Creating new Schedule Manager")

	temporalClient, err := NewClientWithMetrics(config, false)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Temporal client for Schedule Manager")
		return nil, err
	}

	logger.Debug().Msg("Schedule Manager created successfully")
	return &ScheduleManager{
		client:           temporalClient,
		scheduleHandlers: make(map[string]client.ScheduleHandle),
	}, nil
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

func (sm *ScheduleManager) CreateSchedule(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
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

	handle, err := sm.CreateSchedule(ctx, scheduleOptions)
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
