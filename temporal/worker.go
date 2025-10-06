package temporal

import (
	"context"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type WorkerManager struct {
	client  client.Client
	workers []worker.Worker
}

func NewWorkerManager(config *Config) (*WorkerManager, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.NewWorkerManager")

	logger.Debug("Creating new Worker Manager",
		otel.F("hostPort", config.HostPort),
		otel.F("namespace", config.Namespace))

	temporalClient, err := NewClient(config)
	if err != nil {
		logger.Error(err, "Failed to create Temporal client for Worker Manager")
		return nil, err
	}

	logger.Debug("Worker Manager created successfully")
	return &WorkerManager{
		client:  temporalClient,
		workers: make([]worker.Worker, 0),
	}, nil
}

func (wm *WorkerManager) Close() {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkerManager.Close")

	workerCount := len(wm.workers)
	logger.Debug("Closing Worker Manager", otel.F("workerCount", workerCount))

	if workerCount > 0 {
		logger.Debug("Stopping all workers")
		for i, w := range wm.workers {
			logger.Debug("Stopping worker", otel.F("workerIndex", i))
			w.Stop()
		}
		logger.Debug("All workers stopped")
	} else {
		logger.Debug("No workers to stop")
	}

	if wm.client != nil {
		logger.Debug("Closing Temporal client")
		wm.client.Close()
	}

	logger.Debug("Worker Manager closed")
}

func (wm *WorkerManager) Register(taskQueue string, options worker.Options) worker.Worker {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkerManager.Register")

	logger.Debug("Registering new Temporal worker", otel.F("taskQueue", taskQueue))

	logger.Debug("Creating worker instance")
	w := worker.New(wm.client, taskQueue, options)

	logger.Debug("Adding worker to manager's workers list")
	wm.workers = append(wm.workers, w)

	logger.Debug("Worker registered successfully",
		otel.F("taskQueue", taskQueue),
		otel.F("totalWorkers", len(wm.workers)))
	return w
}

func (wm *WorkerManager) Start(ctx context.Context, w worker.Worker) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkerManager.Start")

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
		logger.Debug("Starting Temporal worker", otel.F("taskQueue", taskQueue))
	} else {
		logger.Debug("Starting Temporal worker")
	}

	err := w.Start()
	if err != nil {
		logger.Error(err, "Failed to start Temporal worker")
		return err
	}

	logger.Debug("Temporal worker started successfully")
	return nil
}

func (wm *WorkerManager) StartAll(ctx context.Context) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkerManager.StartAll")

	workerCount := len(wm.workers)
	logger.Debug("Starting all Temporal workers", otel.F("workerCount", workerCount))

	if workerCount == 0 {
		logger.Warn("No workers to start")
		return nil
	}

	for i, w := range wm.workers {
		logger.Debug("Starting worker", otel.F("workerIndex", i))
		err := w.Start()
		if err != nil {
			logger.Error(err, "Failed to start worker", otel.F("workerIndex", i))
			return err
		}
		logger.Debug("Worker started successfully", otel.F("workerIndex", i))
	}

	logger.Debug("All Temporal workers started successfully", otel.F("workerCount", workerCount))
	return nil
}

func (wm *WorkerManager) GetClient() client.Client {
	return wm.client
}

func (wm *WorkerManager) GetWorkers() []worker.Worker {
	return wm.workers
}
