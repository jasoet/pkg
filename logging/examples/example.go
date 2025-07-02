//go:build example

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jasoet/pkg/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Example data structures
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type APIResponse struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type DatabaseOperation struct {
	Table     string
	Operation string
	Duration  time.Duration
	Success   bool
}

func main() {
	fmt.Println("Logging Package Examples")
	fmt.Println("========================")

	// Example 1: Basic Logging Setup
	fmt.Println("\n1. Basic Logging Setup")
	basicLoggingExample()

	// Example 2: Context-Aware Logging
	fmt.Println("\n2. Context-Aware Logging")
	contextAwareLoggingExample()

	// Example 3: Structured Logging
	fmt.Println("\n3. Structured Logging")
	structuredLoggingExample()

	// Example 4: Different Log Levels
	fmt.Println("\n4. Different Log Levels")
	logLevelsExample()

	// Example 5: Error Logging
	fmt.Println("\n5. Error Logging")
	errorLoggingExample()

	// Example 6: Performance Monitoring
	fmt.Println("\n6. Performance Monitoring")
	performanceMonitoringExample()

	// Example 7: HTTP Request Logging
	fmt.Println("\n7. HTTP Request Logging")
	httpRequestLoggingExample()

	// Example 8: Database Operation Logging
	fmt.Println("\n8. Database Operation Logging")
	databaseOperationLoggingExample()

	// Example 9: Integration Examples
	fmt.Println("\n9. Integration with Other Packages")
	integrationExamples()

	// Example 10: Advanced Patterns
	fmt.Println("\n10. Advanced Logging Patterns")
	advancedPatternsExample()
}

func basicLoggingExample() {
	fmt.Println("Setting up basic logging configuration...")

	// Initialize logging with service name and debug mode
	logging.Initialize("logging-examples", true)

	// Use the global logger directly
	log.Info().Msg("Application started")
	log.Debug().Str("version", "1.0.0").Msg("Debug information")
	log.Info().Str("environment", "development").Msg("Environment configured")

	fmt.Println("✓ Basic logging setup completed")
	fmt.Println("  Check above for log output with timestamps, service name, and caller info")
}

func contextAwareLoggingExample() {
	ctx := context.Background()

	// Create different component loggers
	userLogger := logging.ContextLogger(ctx, "user-service")
	authLogger := logging.ContextLogger(ctx, "auth-service")
	dbLogger := logging.ContextLogger(ctx, "database")

	userLogger.Info().Msg("User service started")
	userLogger.Debug().Int("user_id", 123).Msg("Processing user")

	authLogger.Info().Msg("Authentication service initialized")
	authLogger.Debug().Str("method", "JWT").Msg("Using JWT authentication")

	dbLogger.Info().Msg("Database connection established")
	dbLogger.Debug().Str("driver", "postgresql").Msg("Using PostgreSQL driver")

	fmt.Println("✓ Context-aware logging demonstrated")
	fmt.Println("  Notice how each log entry includes the component name")
}

func structuredLoggingExample() {
	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "api-server")

	// HTTP request logging with structured data
	logger.Info().
		Str("method", "POST").
		Str("path", "/api/users").
		Str("remote_addr", "192.168.1.100").
		Int("status", 201).
		Dur("duration", 45*time.Millisecond).
		Int64("response_size", 1024).
		Msg("Request completed")

	// User operation logging
	logger.Info().
		Int("user_id", 12345).
		Str("action", "profile_update").
		Str("fields", "name,email").
		Bool("success", true).
		Msg("User profile updated")

	// System metrics logging
	logger.Info().
		Float64("cpu_usage", 75.5).
		Int64("memory_used", 1073741824). // 1GB in bytes
		Int("active_connections", 150).
		Dur("uptime", 2*time.Hour+30*time.Minute).
		Msg("System metrics")

	fmt.Println("✓ Structured logging demonstrated")
	fmt.Println("  Notice the various field types and structured data")
}

func logLevelsExample() {
	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "log-levels")

	// Different log levels for different scenarios
	logger.Debug().
		Str("function", "processData").
		Interface("input", map[string]interface{}{"key": "value"}).
		Msg("Detailed debugging information")

	logger.Info().
		Str("event", "user_login").
		Str("user_id", "12345").
		Msg("User logged in successfully")

	logger.Warn().
		Str("resource", "memory").
		Float64("usage_percent", 85.0).
		Msg("Resource usage is high")

	logger.Error().
		Str("operation", "database_query").
		Str("error_type", "timeout").
		Msg("Database operation failed")

	// Note: Fatal would exit the application, so we'll just demonstrate the pattern
	fmt.Println("Fatal log example (not executed):")
	fmt.Println("  logger.Fatal().Msg(\"Critical system failure\")")

	fmt.Println("✓ Different log levels demonstrated")
	fmt.Println("  Debug, Info, Warn, Error levels shown")
}

func errorLoggingExample() {
	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "error-handling")

	// Simulate different types of errors
	errorCases := []struct {
		err       error
		operation string
		context   map[string]interface{}
	}{
		{
			err:       errors.New("connection timeout"),
			operation: "database_connection",
			context:   map[string]interface{}{"host": "db.example.com", "timeout": "30s"},
		},
		{
			err:       errors.New("invalid JSON payload"),
			operation: "api_request_parsing",
			context:   map[string]interface{}{"content_type": "application/json", "size": 1024},
		},
		{
			err:       errors.New("user not found"),
			operation: "user_lookup",
			context:   map[string]interface{}{"user_id": "12345", "source": "database"},
		},
	}

	for _, errInfo := range errorCases {
		logger.Error().
			Err(errInfo.err).
			Str("operation", errInfo.operation).
			Interface("context", errInfo.context).
			Msg("Operation failed")
	}

	// Error with retry information
	retryErr := errors.New("service unavailable")
	logger.Error().
		Err(retryErr).
		Str("service", "payment-processor").
		Int("retry_count", 3).
		Dur("backoff", 5*time.Second).
		Bool("will_retry", false).
		Msg("Service call failed after retries")

	// Wrapped error logging
	originalErr := errors.New("disk full")
	wrappedErr := fmt.Errorf("failed to write file: %w", originalErr)

	logger.Error().
		Err(wrappedErr).
		Str("file_path", "/var/log/app.log").
		Int64("attempted_size", 2048).
		Msg("File write operation failed")

	fmt.Println("✓ Error logging patterns demonstrated")
	fmt.Println("  Various error scenarios with context information")
}

func performanceMonitoringExample() {
	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "performance")

	// Simulate various operations with timing
	operations := []struct {
		name     string
		duration time.Duration
		success  bool
		metrics  map[string]interface{}
	}{
		{
			name:     "database_query",
			duration: 150 * time.Millisecond,
			success:  true,
			metrics:  map[string]interface{}{"rows_returned": 25, "cache_hit": false},
		},
		{
			name:     "cache_lookup",
			duration: 5 * time.Millisecond,
			success:  true,
			metrics:  map[string]interface{}{"cache_hit": true, "ttl": "300s"},
		},
		{
			name:     "external_api_call",
			duration: 750 * time.Millisecond,
			success:  true,
			metrics:  map[string]interface{}{"endpoint": "/users", "response_size": 4096},
		},
		{
			name:     "file_processing",
			duration: 2 * time.Second,
			success:  false,
			metrics:  map[string]interface{}{"file_size": 1048576, "processed_bytes": 524288},
		},
	}

	for _, op := range operations {
		event := logger.Info()
		if !op.success {
			event = logger.Warn()
		}

		event.
			Str("operation", op.name).
			Dur("duration", op.duration).
			Bool("success", op.success).
			Interface("metrics", op.metrics).
			Msg("Operation completed")
	}

	// Performance threshold alerts
	slowQuery := 500 * time.Millisecond
	if slowQuery > 100*time.Millisecond {
		logger.Warn().
			Dur("duration", slowQuery).
			Dur("threshold", 100*time.Millisecond).
			Str("query", "SELECT * FROM users WHERE active = TRUE").
			Msg("Slow query detected")
	}

	fmt.Println("✓ Performance monitoring demonstrated")
	fmt.Println("  Operation timing and performance metrics logged")
}

func httpRequestLoggingExample() {
	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "http-server")

	// Simulate HTTP requests
	requests := []struct {
		method       string
		path         string
		status       int
		duration     time.Duration
		userAgent    string
		remoteAddr   string
		requestSize  int64
		responseSize int64
	}{
		{
			method: "GET", path: "/api/users", status: 200,
			duration: 45 * time.Millisecond, userAgent: "curl/7.68.0",
			remoteAddr: "192.168.1.100", requestSize: 0, responseSize: 2048,
		},
		{
			method: "POST", path: "/api/users", status: 201,
			duration: 120 * time.Millisecond, userAgent: "Mozilla/5.0",
			remoteAddr: "192.168.1.101", requestSize: 512, responseSize: 256,
		},
		{
			method: "DELETE", path: "/api/users/123", status: 404,
			duration: 25 * time.Millisecond, userAgent: "PostmanRuntime/7.26.8",
			remoteAddr: "192.168.1.102", requestSize: 0, responseSize: 128,
		},
		{
			method: "PUT", path: "/api/users/456", status: 500,
			duration: 200 * time.Millisecond, userAgent: "axios/0.21.1",
			remoteAddr: "192.168.1.103", requestSize: 1024, responseSize: 64,
		},
	}

	for _, req := range requests {
		// Determine log level based on status code
		var event *zerolog.Event
		switch {
		case req.status >= 500:
			event = logger.Error()
		case req.status >= 400:
			event = logger.Warn()
		default:
			event = logger.Info()
		}

		event.
			Str("method", req.method).
			Str("path", req.path).
			Int("status", req.status).
			Dur("duration", req.duration).
			Str("user_agent", req.userAgent).
			Str("remote_addr", req.remoteAddr).
			Int64("request_size", req.requestSize).
			Int64("response_size", req.responseSize).
			Msg("HTTP request")
	}

	// Request middleware example
	logHTTPRequest := func(r *http.Request, status int, duration time.Duration) {
		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Int("status", status).
			Dur("duration", duration).
			Msg("HTTP request")
	}

	// Simulate middleware usage
	fmt.Println("\nMiddleware usage example:")
	req, _ := http.NewRequest("GET", "/api/health?detailed=true", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "health-checker/1.0")

	logHTTPRequest(req, 200, 10*time.Millisecond)

	fmt.Println("✓ HTTP request logging demonstrated")
	fmt.Println("  Different status codes and comprehensive request information")
}

func databaseOperationLoggingExample() {
	ctx := context.Background()

	// Simulate database operations with different loggers
	operations := []DatabaseOperation{
		{Table: "users", Operation: "SELECT", Duration: 25 * time.Millisecond, Success: true},
		{Table: "orders", Operation: "INSERT", Duration: 50 * time.Millisecond, Success: true},
		{Table: "products", Operation: "UPDATE", Duration: 35 * time.Millisecond, Success: true},
		{Table: "users", Operation: "DELETE", Duration: 200 * time.Millisecond, Success: false},
	}

	for _, op := range operations {
		logDatabaseOperation(ctx, op.Operation, op.Table, op.Duration, op.Success, nil)
	}

	// Simulate error scenarios
	connectionErr := errors.New("connection pool exhausted")
	logDatabaseOperation(ctx, "SELECT", "users", 5*time.Second, false, connectionErr)

	timeoutErr := errors.New("query timeout")
	logDatabaseOperation(ctx, "UPDATE", "large_table", 30*time.Second, false, timeoutErr)

	// Migration logging
	logMigrationOperation(ctx, "001_create_users_table", "up", 1*time.Second, true)
	logMigrationOperation(ctx, "002_add_indexes", "up", 500*time.Millisecond, true)

	fmt.Println("✓ Database operation logging demonstrated")
	fmt.Println("  CRUD operations, errors, and migrations logged")
}

func logDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, success bool, err error) {
	logger := logging.ContextLogger(ctx, "database")

	event := logger.Info()
	if !success && err != nil {
		event = logger.Error().Err(err)
	} else if !success {
		event = logger.Warn()
	}

	event.
		Str("operation", operation).
		Str("table", table).
		Dur("duration", duration).
		Bool("success", success).
		Msg("Database operation")
}

func logMigrationOperation(ctx context.Context, migration, direction string, duration time.Duration, success bool) {
	logger := logging.ContextLogger(ctx, "db-migration")

	event := logger.Info()
	if !success {
		event = logger.Error()
	}

	event.
		Str("migration", migration).
		Str("direction", direction).
		Dur("duration", duration).
		Bool("success", success).
		Msg("Database migration")
}

func integrationExamples() {
	ctx := context.Background()

	// Example: Logging in a service that uses multiple packages
	userServiceExample(ctx)

	// Example: API client with logging
	apiClientExample(ctx)

	// Example: Background worker with logging
	backgroundWorkerExample(ctx)

	fmt.Println("✓ Integration examples demonstrated")
	fmt.Println("  Logging patterns for services using multiple packages")
}

func userServiceExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "user-service")

	logger.Info().Msg("User service starting up")

	// Simulate service operations
	user := User{ID: 123, Name: "John Doe", Email: "john@example.com"}

	logger.Info().
		Int("user_id", user.ID).
		Str("operation", "create_user").
		Msg("Creating new user")

	// Simulate database operation
	start := time.Now()
	// db.Create(&user) - simulated
	dbDuration := 45 * time.Millisecond

	logger.Info().
		Int("user_id", user.ID).
		Dur("db_duration", dbDuration).
		Str("table", "users").
		Msg("User created in database")

	// Simulate cache operation
	cacheStart := time.Now()
	// cache.Set(userKey, user) - simulated
	time.Sleep(5 * time.Millisecond) // Simulate cache operation
	cacheDuration := time.Since(cacheStart)

	logger.Debug().
		Int("user_id", user.ID).
		Dur("cache_duration", cacheDuration).
		Str("cache_key", fmt.Sprintf("user:%d", user.ID)).
		Msg("User cached")

	totalDuration := time.Since(start)
	logger.Info().
		Int("user_id", user.ID).
		Dur("total_duration", totalDuration).
		Msg("User creation completed")
}

func apiClientExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "api-client")

	apiURL := "https://api.example.com/users"

	logger.Info().
		Str("url", apiURL).
		Str("method", "GET").
		Msg("Making API request")

	start := time.Now()

	// Simulate API call
	time.Sleep(250 * time.Millisecond) // Simulate API call duration
	response := APIResponse{
		Status:    "success",
		Data:      []User{{ID: 1, Name: "API User", Email: "api@example.com"}},
		Timestamp: time.Now(),
	}
	duration := time.Since(start)

	logger.Info().
		Str("url", apiURL).
		Str("status", response.Status).
		Dur("duration", duration).
		Int("response_size", 512).
		Msg("API request completed")

	// Simulate retry scenario
	retryLogger := logging.ContextLogger(ctx, "api-client-retry")
	for attempt := 1; attempt <= 3; attempt++ {
		retryLogger.Info().
			Str("url", apiURL).
			Int("attempt", attempt).
			Msg("API request attempt")

		if attempt == 3 {
			retryLogger.Info().
				Str("url", apiURL).
				Int("attempt", attempt).
				Msg("API request succeeded")
			break
		} else {
			retryLogger.Warn().
				Str("url", apiURL).
				Int("attempt", attempt).
				Dur("retry_after", time.Duration(attempt)*time.Second).
				Msg("API request failed, retrying")
		}
	}
}

func backgroundWorkerExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "background-worker")

	logger.Info().
		Str("worker_type", "email_sender").
		Int("queue_size", 150).
		Msg("Background worker started")

	// Simulate processing jobs
	for i := 1; i <= 5; i++ {
		jobLogger := logging.ContextLogger(ctx, "email-job")

		jobStart := time.Now()

		jobLogger.Info().
			Int("job_id", i).
			Str("type", "welcome_email").
			Str("recipient", fmt.Sprintf("user%d@example.com", i)).
			Msg("Processing email job")

		// Simulate job processing
		processingTime := time.Duration(i*50) * time.Millisecond
		time.Sleep(processingTime)

		if i == 4 {
			// Simulate failure
			jobLogger.Error().
				Int("job_id", i).
				Dur("duration", time.Since(jobStart)).
				Str("error", "SMTP server unavailable").
				Msg("Email job failed")
		} else {
			jobLogger.Info().
				Int("job_id", i).
				Dur("duration", time.Since(jobStart)).
				Msg("Email job completed")
		}
	}

	logger.Info().
		Str("worker_type", "email_sender").
		Int("processed", 4).
		Int("failed", 1).
		Msg("Background worker batch completed")
}

func advancedPatternsExample() {
	ctx := context.Background()

	// Pattern 1: Request ID tracking
	requestTrackingExample(ctx)

	// Pattern 2: Conditional debug logging
	conditionalLoggingExample(ctx)

	// Pattern 3: Log sampling for high-volume events
	logSamplingExample(ctx)

	// Pattern 4: Service boundaries
	serviceBoundariesExample(ctx)

	fmt.Println("✓ Advanced logging patterns demonstrated")
	fmt.Println("  Request tracking, conditional logging, sampling, and boundaries")
}

func requestTrackingExample(ctx context.Context) {
	// Simulate request ID in context
	requestID := "req_" + fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	ctx = context.WithValue(ctx, "request_id", requestID)

	logger := logging.ContextLogger(ctx, "api-handler")

	logger.Info().
		Str("request_id", requestID).
		Str("endpoint", "/api/users/123").
		Msg("Request started")

	// Simulate calling multiple services
	services := []string{"auth-service", "user-service", "notification-service"}

	for _, service := range services {
		serviceLogger := logging.ContextLogger(ctx, service)
		serviceLogger.Info().
			Str("request_id", requestID).
			Str("action", "process_request").
			Msg("Processing request")
	}

	logger.Info().
		Str("request_id", requestID).
		Dur("total_duration", 150*time.Millisecond).
		Msg("Request completed")
}

func conditionalLoggingExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "performance-critical")

	// Only generate expensive debug data if debug logging is enabled
	if logger.Debug().Enabled() {
		expensiveDebugData := generateDebugData()
		logger.Debug().
			Interface("debug_data", expensiveDebugData).
			Msg("Expensive debug information")
	}

	// Always log important information
	logger.Info().
		Str("operation", "data_processing").
		Int("items_processed", 1000).
		Msg("Batch processing completed")
}

func generateDebugData() map[string]interface{} {
	// Simulate expensive debug data generation
	return map[string]interface{}{
		"memory_usage": "125MB",
		"cpu_time":     "1.5s",
		"cache_stats":  map[string]int{"hits": 850, "misses": 150},
		"query_plan":   "SELECT * FROM users WHERE active = TRUE ORDER BY created_at",
	}
}

func logSamplingExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "high-volume-service")

	// Simulate high-volume events with sampling
	for i := 1; i <= 100; i++ {
		// Only log every 10th event
		if i%10 == 0 {
			logger.Info().
				Int("event_number", i).
				Str("event_type", "user_action").
				Msg("High-volume event (sampled)")
		}

		// Always log errors
		if i%25 == 0 {
			logger.Error().
				Int("event_number", i).
				Str("error", "validation_failed").
				Msg("Error occurred")
		}
	}

	logger.Info().
		Int("total_events", 100).
		Int("sampled_events", 10).
		Int("errors", 4).
		Msg("High-volume processing completed")
}

func serviceBoundariesExample(ctx context.Context) {
	// Service A calling Service B
	serviceALogger := logging.ContextLogger(ctx, "service-a")

	serviceALogger.Info().
		Str("target_service", "service-b").
		Str("operation", "get_user_data").
		Msg("Calling external service")

	start := time.Now()

	// Simulate service call
	success := callServiceB(ctx)
	duration := time.Since(start)

	if success {
		serviceALogger.Info().
			Str("target_service", "service-b").
			Dur("duration", duration).
			Msg("External service call successful")
	} else {
		serviceALogger.Error().
			Str("target_service", "service-b").
			Dur("duration", duration).
			Msg("External service call failed")
	}
}

func callServiceB(ctx context.Context) bool {
	serviceBLogger := logging.ContextLogger(ctx, "service-b")

	serviceBLogger.Info().
		Str("operation", "get_user_data").
		Msg("Processing request from service-a")

	// Simulate processing
	time.Sleep(50 * time.Millisecond)

	serviceBLogger.Info().
		Str("operation", "get_user_data").
		Int("records_returned", 1).
		Msg("Request processed successfully")

	return true
}
