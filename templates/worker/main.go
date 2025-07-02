//go:build template

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/concurrent"
	"github.com/jasoet/pkg/config"
	"github.com/jasoet/pkg/db"
	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/rest"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AppConfig defines the worker configuration structure
type AppConfig struct {
	Environment string              `yaml:"environment" mapstructure:"environment" validate:"required,oneof=development staging production"`
	Debug       bool                `yaml:"debug" mapstructure:"debug"`
	Database    db.ConnectionConfig `yaml:"database" mapstructure:"database" validate:"required"`
	Worker      WorkerConfig        `yaml:"worker" mapstructure:"worker" validate:"required"`
	ExternalAPI ExternalAPIConfig   `yaml:"externalApi" mapstructure:"externalApi"`
}

type WorkerConfig struct {
	PollInterval   time.Duration `yaml:"pollInterval" mapstructure:"pollInterval" validate:"min=1s"`
	BatchSize      int           `yaml:"batchSize" mapstructure:"batchSize" validate:"min=1,max=100"`
	MaxConcurrency int           `yaml:"maxConcurrency" mapstructure:"maxConcurrency" validate:"min=1,max=50"`
	ProcessTimeout time.Duration `yaml:"processTimeout" mapstructure:"processTimeout" validate:"min=1s"`
}

type ExternalAPIConfig struct {
	BaseURL    string        `yaml:"baseUrl" mapstructure:"baseUrl" validate:"required,url"`
	Timeout    time.Duration `yaml:"timeout" mapstructure:"timeout" validate:"min=1s"`
	RetryCount int           `yaml:"retryCount" mapstructure:"retryCount" validate:"min=0,max=10"`
	APIKey     string        `yaml:"apiKey" mapstructure:"apiKey"`
}

// Services contains all worker dependencies
type Services struct {
	DB        *gorm.DB
	APIClient *rest.Client
	Config    *AppConfig
	Logger    zerolog.Logger
}

// Job represents a work item to be processed
type Job struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	Type        string     `json:"type" gorm:"not null"`
	Payload     string     `json:"payload" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:pending"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	ProcessedAt *time.Time `json:"processed_at"`
	RetryCount  int        `json:"retry_count" gorm:"default:0"`
	MaxRetries  int        `json:"max_retries" gorm:"default:3"`
	ErrorMsg    string     `json:"error_msg"`
}

// JobResult represents the result of job processing
type JobResult struct {
	JobID   string
	Success bool
	Error   string
}

func main() {
	// 1. Initialize logging first (CRITICAL)
	logging.Initialize("worker-service", os.Getenv("DEBUG") == "true")

	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "main")

	// 2. Load configuration
	appConfig, err := loadConfiguration()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// 3. Setup database
	database, err := appConfig.Database.Pool()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Auto-migrate job table
	if err := database.AutoMigrate(&Job{}); err != nil {
		logger.Fatal().Err(err).Msg("Failed to migrate database")
	}

	// 4. Setup HTTP client for external API calls
	restConfig := &rest.Config{
		Timeout:    appConfig.ExternalAPI.Timeout,
		RetryCount: appConfig.ExternalAPI.RetryCount,
	}
	apiClient := rest.NewClient(rest.WithRestConfig(*restConfig))

	// 5. Create services container
	services := &Services{
		DB:        database,
		APIClient: apiClient,
		Config:    appConfig,
		Logger:    logger,
	}

	// 6. Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// 7. Start worker
	logger.Info().
		Str("environment", appConfig.Environment).
		Dur("poll_interval", appConfig.Worker.PollInterval).
		Int("batch_size", appConfig.Worker.BatchSize).
		Int("max_concurrency", appConfig.Worker.MaxConcurrency).
		Msg("Starting worker service")

	if err := runWorker(ctx, services); err != nil {
		logger.Error().Err(err).Msg("Worker failed")
	}

	logger.Info().Msg("Worker service shutdown completed")
}

func loadConfiguration() (*AppConfig, error) {
	// Default configuration for development
	defaultConfig := `
environment: development
debug: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: worker_db
  timeout: 30s
  maxIdleConns: 5
  maxOpenConns: 25
worker:
  pollInterval: 30s
  batchSize: 10
  maxConcurrency: 5
  processTimeout: 5m
externalApi:
  baseUrl: https://api.example.com
  timeout: 30s
  retryCount: 3
  apiKey: your-api-key-here
`

	// Load configuration with environment variable overrides
	appConfig, err := config.LoadString[AppConfig](defaultConfig, "WORKER")
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

func runWorker(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "worker")
	ticker := time.NewTicker(services.Config.Worker.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Worker shutdown requested")
			return nil
		case <-ticker.C:
			if err := processBatch(ctx, services); err != nil {
				logger.Error().Err(err).Msg("Batch processing failed")
			}
		}
	}
}

func processBatch(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "batch-processor")

	// Fetch pending jobs
	jobs, err := fetchPendingJobs(services.DB, services.Config.Worker.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		logger.Debug().Msg("No pending jobs found")
		return nil
	}

	logger.Info().Int("job_count", len(jobs)).Msg("Processing job batch")

	// Create processing functions for concurrent execution
	processingFuncs := make(map[string]concurrent.Func[JobResult])
	for _, job := range jobs {
		jobCopy := job // Important: capture loop variable
		processingFuncs[job.ID] = func(ctx context.Context) (JobResult, error) {
			return processJob(ctx, services, &jobCopy)
		}
	}

	// Process jobs concurrently with timeout
	ctx, cancel := context.WithTimeout(ctx, services.Config.Worker.ProcessTimeout)
	defer cancel()

	results, err := concurrent.ExecuteConcurrently(ctx, processingFuncs)
	if err != nil {
		logger.Error().Err(err).Msg("Concurrent processing failed")
		return fmt.Errorf("concurrent processing failed: %w", err)
	}

	// Update job statuses based on results
	successCount := 0
	for jobID, result := range results {
		if result.Success {
			successCount++
			updateJobStatus(services.DB, jobID, "completed", "")
		} else {
			updateJobStatus(services.DB, jobID, "failed", result.Error)
		}
	}

	logger.Info().
		Int("total", len(jobs)).
		Int("successful", successCount).
		Int("failed", len(jobs)-successCount).
		Msg("Batch processing completed")

	return nil
}

func fetchPendingJobs(db *gorm.DB, batchSize int) ([]Job, error) {
	var jobs []Job
	err := db.Where("status = ?", "pending").
		Where("retry_count < max_retries").
		Order("created_at ASC").
		Limit(batchSize).
		Find(&jobs).Error
	return jobs, err
}

func processJob(ctx context.Context, services *Services, job *Job) (JobResult, error) {
	logger := logging.ContextLogger(ctx, "job-processor")
	logger = logger.With().Str("job_id", job.ID).Str("job_type", job.Type).Logger()

	logger.Info().Msg("Starting job processing")

	// Update job status to processing
	updateJobStatus(services.DB, job.ID, "processing", "")

	// Process based on job type
	var err error
	switch job.Type {
	case "send_email":
		err = processSendEmailJob(ctx, services, job)
	case "process_data":
		err = processDataJob(ctx, services, job)
	case "generate_report":
		err = processReportJob(ctx, services, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if err != nil {
		logger.Error().Err(err).Msg("Job processing failed")
		incrementRetryCount(services.DB, job.ID)
		return JobResult{
			JobID:   job.ID,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	logger.Info().Msg("Job processing completed successfully")
	return JobResult{
		JobID:   job.ID,
		Success: true,
	}, nil
}

func processSendEmailJob(ctx context.Context, services *Services, job *Job) error {
	logger := logging.ContextLogger(ctx, "email-processor")

	// TODO: Implement email sending logic
	// This is a placeholder implementation
	logger.Info().Str("payload", job.Payload).Msg("Processing send email job")

	// Simulate external API call
	headers := map[string]string{
		"Authorization": "Bearer " + services.Config.ExternalAPI.APIKey,
		"Content-Type":  "application/json",
	}

	response, err := services.APIClient.MakeRequest(ctx, "POST", "/send-email", job.Payload, headers)
	if err != nil {
		return fmt.Errorf("failed to send email via API: %w", err)
	}

	logger.Info().Str("response", response.String()).Msg("Email sent successfully")
	return nil
}

func processDataJob(ctx context.Context, services *Services, job *Job) error {
	logger := logging.ContextLogger(ctx, "data-processor")

	// TODO: Implement data processing logic
	logger.Info().Str("payload", job.Payload).Msg("Processing data job")

	// Simulate data processing
	time.Sleep(2 * time.Second)

	logger.Info().Msg("Data processing completed")
	return nil
}

func processReportJob(ctx context.Context, services *Services, job *Job) error {
	logger := logging.ContextLogger(ctx, "report-processor")

	// TODO: Implement report generation logic
	logger.Info().Str("payload", job.Payload).Msg("Processing report job")

	// Simulate report generation
	time.Sleep(5 * time.Second)

	logger.Info().Msg("Report generation completed")
	return nil
}

func updateJobStatus(db *gorm.DB, jobID, status, errorMsg string) {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "completed" {
		now := time.Now()
		updates["processed_at"] = &now
	}

	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	db.Model(&Job{}).Where("id = ?", jobID).Updates(updates)
}

func incrementRetryCount(db *gorm.DB, jobID string) {
	db.Model(&Job{}).Where("id = ?", jobID).UpdateColumn("retry_count", gorm.Expr("retry_count + 1"))
}
