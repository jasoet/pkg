package temporal

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type WorkerManager struct {
	client  client.Client
	workers []worker.Worker
}

func NewWorkerManager(config *Config) (*WorkerManager, error) {
	logger := log.With().Str("function", "temporal.NewWorkerManager").Logger()
	logger.Debug().
		Str("hostPort", config.HostPort).
		Str("namespace", config.Namespace).
		Msg("Creating new Worker Manager")

	temporalClient, err := NewClient(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Temporal client for Worker Manager")
		return nil, err
	}

	logger.Debug().Msg("Worker Manager created successfully")
	return &WorkerManager{
		client:  temporalClient,
		workers: make([]worker.Worker, 0),
	}, nil
}

func (wm *WorkerManager) Close() {
	logger := log.With().Str("function", "WorkerManager.Close").Logger()
	workerCount := len(wm.workers)
	logger.Debug().Int("workerCount", workerCount).Msg("Closing Worker Manager")

	if workerCount > 0 {
		logger.Debug().Msg("Stopping all workers")
		for i, w := range wm.workers {
			logger.Debug().Int("workerIndex", i).Msg("Stopping worker")
			w.Stop()
		}
		logger.Debug().Msg("All workers stopped")
	} else {
		logger.Debug().Msg("No workers to stop")
	}

	if wm.client != nil {
		logger.Debug().Msg("Closing Temporal client")
		wm.client.Close()
	}

	logger.Debug().Msg("Worker Manager closed")
}

func (wm *WorkerManager) Register(taskQueue string, options worker.Options) worker.Worker {
	logger := log.With().Str("function", "WorkerManager.Register").Logger()
	logger.Debug().
		Str("taskQueue", taskQueue).
		Msg("Registering new Temporal worker")

	logger.Debug().Msg("Creating worker instance")
	w := worker.New(wm.client, taskQueue, options)

	logger.Debug().Msg("Adding worker to manager's workers list")
	wm.workers = append(wm.workers, w)

	logger.Debug().
		Str("taskQueue", taskQueue).
		Int("totalWorkers", len(wm.workers)).
		Msg("Worker registered successfully")
	return w
}

func (wm *WorkerManager) Start(ctx context.Context, w worker.Worker) error {
	logger := log.With().Ctx(ctx).Str("function", "WorkerManager.Start").Logger()

	// Try to get the task queue from the worker if possible
	// This is a bit of a hack since the worker doesn't expose its task queue directly
	var taskQueue string
	for i, registeredWorker := range wm.workers {
		if registeredWorker == w {
			taskQueue = fmt.Sprintf("worker-%d", i)
			break
		}
	}

	if taskQueue != "" {
		logger.Debug().Str("taskQueue", taskQueue).Msg("Starting Temporal worker")
	} else {
		logger.Debug().Msg("Starting Temporal worker")
	}

	err := w.Start()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to start Temporal worker")
		return err
	}

	logger.Debug().Msg("Temporal worker started successfully")
	return nil
}

func (wm *WorkerManager) StartAll(ctx context.Context) error {
	logger := log.With().Ctx(ctx).Str("function", "WorkerManager.StartAll").Logger()
	workerCount := len(wm.workers)

	logger.Debug().Int("workerCount", workerCount).Msg("Starting all Temporal workers")

	if workerCount == 0 {
		logger.Warn().Msg("No workers to start")
		return nil
	}

	for i, w := range wm.workers {
		logger.Debug().Int("workerIndex", i).Msg("Starting worker")
		err := w.Start()
		if err != nil {
			logger.Error().Err(err).Int("workerIndex", i).Msg("Failed to start worker")
			return err
		}
		logger.Debug().Int("workerIndex", i).Msg("Worker started successfully")
	}

	logger.Debug().Int("workerCount", workerCount).Msg("All Temporal workers started successfully")
	return nil
}

func (wm *WorkerManager) GetClient() client.Client {
	return wm.client
}

func (wm *WorkerManager) GetWorkers() []worker.Worker {
	return wm.workers
}
