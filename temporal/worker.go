package temporal

import (
	"context"
	"sync"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/jasoet/pkg/v2/otel"
)

type WorkerManager struct {
	client  client.Client
	mu      sync.RWMutex
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

	wm.mu.RLock()
	workerCount := len(wm.workers)
	wm.mu.RUnlock()

	logger.Debug("Closing Worker Manager", otel.F("workerCount", workerCount))

	if workerCount > 0 {
		logger.Debug("Stopping all workers")
		wm.mu.RLock()
		for i, w := range wm.workers {
			logger.Debug("Stopping worker", otel.F("workerIndex", i))
			w.Stop()
		}
		wm.mu.RUnlock()
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

	wm.mu.Lock()
	wm.workers = append(wm.workers, w)
	totalWorkers := len(wm.workers)
	wm.mu.Unlock()

	logger.Debug("Worker registered successfully",
		otel.F("taskQueue", taskQueue),
		otel.F("totalWorkers", totalWorkers))
	return w
}

// Start starts the given worker. The ctx parameter is used for logging only;
// the worker's internal lifecycle is managed by the Temporal SDK.
func (wm *WorkerManager) Start(ctx context.Context, w worker.Worker) error {
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "WorkerManager.Start")

	// Try to get the worker index from the registered list for logging purposes.
	workerIndex := -1
	wm.mu.RLock()
	for i, registeredWorker := range wm.workers {
		if registeredWorker == w {
			workerIndex = i
			break
		}
	}
	wm.mu.RUnlock()

	if workerIndex >= 0 {
		logger.Debug("Starting Temporal worker", otel.F("workerIndex", workerIndex))
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

	wm.mu.RLock()
	workerCount := len(wm.workers)
	wm.mu.RUnlock()

	logger.Debug("Starting all Temporal workers", otel.F("workerCount", workerCount))

	if workerCount == 0 {
		logger.Warn("No workers to start")
		return nil
	}

	wm.mu.RLock()
	for i, w := range wm.workers {
		logger.Debug("Starting worker", otel.F("workerIndex", i))
		err := w.Start()
		if err != nil {
			wm.mu.RUnlock()
			logger.Error(err, "Failed to start worker", otel.F("workerIndex", i))
			return err
		}
		logger.Debug("Worker started successfully", otel.F("workerIndex", i))
	}
	wm.mu.RUnlock()

	logger.Debug("All Temporal workers started successfully", otel.F("workerCount", workerCount))
	return nil
}

// GetClient returns the internal Temporal client. Callers must not close this
// client independently; use Close() on the manager instead.
func (wm *WorkerManager) GetClient() client.Client {
	return wm.client
}

func (wm *WorkerManager) GetWorkers() []worker.Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	// Return a copy to prevent concurrent slice access
	workers := make([]worker.Worker, len(wm.workers))
	copy(workers, wm.workers)
	return workers
}
