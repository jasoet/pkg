//go:build example

package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/v2/temporal"
	"github.com/jasoet/pkg/v2/temporal/examples/activities"
	"github.com/jasoet/pkg/v2/temporal/examples/workflows"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/worker"
)

const (
	// TaskQueue is the task queue name used for the examples
	TaskQueue = "example-task-queue"
)

// BasicWorker demonstrates a simple worker setup.
// It shows how to create a worker, register workflows and activities,
// and handle proper shutdown.
func RunBasicWorker() error {
	logger := log.With().Str("component", "BasicWorker").Logger()
	logger.Info().Msg("Starting basic worker example")

	// Step 1: Create a Temporal client
	logger.Info().Msg("Creating Temporal client")
	config := temporal.DefaultConfig()
	workerManager, err := temporal.NewWorkerManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create worker manager")
		return err
	}
	defer workerManager.Close()

	// Step 2: Register a worker with the task queue
	logger.Info().Str("taskQueue", TaskQueue).Msg("Registering worker")
	w := workerManager.Register(TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Step 3: Register workflows
	logger.Info().Msg("Registering workflows")
	w.RegisterWorkflow(workflows.SimpleWorkflow)
	w.RegisterWorkflow(workflows.SimpleWorkflowWithParams)
	w.RegisterWorkflow(workflows.ActivityWorkflow)
	w.RegisterWorkflow(workflows.ActivityWorkflowWithChildWorkflow)
	w.RegisterWorkflow(workflows.ErrorHandlingWorkflow)
	w.RegisterWorkflow(workflows.TimerWorkflow)
	w.RegisterWorkflow(workflows.ScheduledWorkflow)

	// Step 4: Register activities
	logger.Info().Msg("Registering activities")
	w.RegisterActivity(activities.Greeting)
	w.RegisterActivity(activities.ProcessData)
	w.RegisterActivity(activities.FetchExternalData)

	// Step 5: Start the worker
	logger.Info().Msg("Starting worker")
	err = workerManager.Start(context.Background(), w)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to start worker")
		return err
	}

	// Step 6: Set up graceful shutdown
	logger.Info().Msg("Worker started, press Ctrl+C to exit")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info().Msg("Shutdown signal received, stopping worker")
	return nil
}

// MultiTaskQueueWorker demonstrates a worker that handles multiple task queues.
// It shows how to create multiple workers for different task queues.
func RunMultiTaskQueueWorker() error {
	logger := log.With().Str("component", "MultiTaskQueueWorker").Logger()
	logger.Info().Msg("Starting multi-task queue worker example")

	// Step 1: Create a Temporal client
	logger.Info().Msg("Creating Temporal client")
	config := temporal.DefaultConfig()
	workerManager, err := temporal.NewWorkerManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create worker manager")
		return err
	}
	defer workerManager.Close()

	// Step 2: Register workers for different task queues
	// Worker 1: For simple workflows
	logger.Info().Str("taskQueue", "simple-workflows").Msg("Registering worker for simple workflows")
	simpleWorker := workerManager.Register("simple-workflows", worker.Options{
		MaxConcurrentActivityExecutionSize:     5,
		MaxConcurrentWorkflowTaskExecutionSize: 5,
	})

	// Register simple workflows
	simpleWorker.RegisterWorkflow(workflows.SimpleWorkflow)
	simpleWorker.RegisterWorkflow(workflows.SimpleWorkflowWithParams)

	// Worker 2: For activity workflows
	logger.Info().Str("taskQueue", "activity-workflows").Msg("Registering worker for activity workflows")
	activityWorker := workerManager.Register("activity-workflows", worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 5,
	})

	// Register activity workflows
	activityWorker.RegisterWorkflow(workflows.ActivityWorkflow)
	activityWorker.RegisterWorkflow(workflows.ActivityWorkflowWithChildWorkflow)

	// Register activities for both workers
	simpleWorker.RegisterActivity(activities.Greeting)
	activityWorker.RegisterActivity(activities.Greeting)
	activityWorker.RegisterActivity(activities.ProcessData)
	activityWorker.RegisterActivity(activities.FetchExternalData)

	// Step 3: Start all workers
	logger.Info().Msg("Starting all workers")
	err = workerManager.StartAll(context.Background())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to start workers")
		return err
	}

	// Step 4: Set up graceful shutdown
	logger.Info().Msg("Workers started, press Ctrl+C to exit")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info().Msg("Shutdown signal received, stopping workers")
	return nil
}

// GracefulShutdownWorker demonstrates a worker with proper shutdown handling.
// It shows how to implement a graceful shutdown with a timeout.
func RunGracefulShutdownWorker() error {
	logger := log.With().Str("component", "GracefulShutdownWorker").Logger()
	logger.Info().Msg("Starting graceful shutdown worker example")

	// Step 1: Create a Temporal client
	logger.Info().Msg("Creating Temporal client")
	config := temporal.DefaultConfig()
	workerManager, err := temporal.NewWorkerManager(config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create worker manager")
		return err
	}
	// Don't defer Close() here, we'll handle it manually

	// Step 2: Register a worker with the task queue
	logger.Info().Str("taskQueue", TaskQueue).Msg("Registering worker")
	w := workerManager.Register(TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Step 3: Register workflows and activities
	logger.Info().Msg("Registering workflows and activities")
	w.RegisterWorkflow(workflows.SimpleWorkflow)
	w.RegisterWorkflow(workflows.ActivityWorkflow)
	w.RegisterActivity(activities.Greeting)
	w.RegisterActivity(activities.ProcessData)
	w.RegisterActivity(activities.FetchExternalData)

	// Step 4: Start the worker
	logger.Info().Msg("Starting worker")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = workerManager.Start(ctx, w)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to start worker")
		return err
	}

	// Step 5: Set up graceful shutdown
	logger.Info().Msg("Worker started, press Ctrl+C to exit")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info().Msg("Shutdown signal received, initiating graceful shutdown")

	// Create a context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Create a channel to signal when shutdown is complete
	shutdownComplete := make(chan struct{})

	go func() {
		// Cancel the worker context to stop accepting new tasks
		cancel()

		// Wait a moment for in-flight tasks to complete
		logger.Info().Msg("Waiting for in-flight tasks to complete...")
		time.Sleep(5 * time.Second)

		// Close the worker manager
		logger.Info().Msg("Closing worker manager")
		workerManager.Close()

		// Signal that shutdown is complete
		close(shutdownComplete)
	}()

	// Wait for either shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		logger.Info().Msg("Graceful shutdown completed successfully")
	case <-shutdownCtx.Done():
		logger.Warn().Msg("Graceful shutdown timed out, forcing exit")
	}

	return nil
}

// To run these examples, you can use the following code:
//
// ```go
// package main
//
// import (
//     "github.com/amanata-dev/twc-report-backend/pkg/temporal/examples/worker"
//     "github.com/rs/zerolog/log"
// )
//
// func main() {
//     // Choose which worker example to run
//     if err := worker.RunBasicWorker(); err != nil {
//         log.Fatal().Err(err).Msg("Worker failed")
//     }
//
//     // Or run the multi-task queue worker
//     // if err := worker.RunMultiTaskQueueWorker(); err != nil {
//     //     log.Fatal().Err(err).Msg("Worker failed")
//     // }
//
//     // Or run the graceful shutdown worker
//     // if err := worker.RunGracefulShutdownWorker(); err != nil {
//     //     log.Fatal().Err(err).Msg("Worker failed")
//     // }
// }
// ```
