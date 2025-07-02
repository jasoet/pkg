//go:build example

package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jasoet/pkg/concurrent"
	"github.com/jasoet/pkg/config"
	"github.com/jasoet/pkg/db"
	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AppConfig defines the analytics dashboard configuration
type AppConfig struct {
	Environment string              `yaml:"environment" mapstructure:"environment" validate:"required"`
	Debug       bool                `yaml:"debug" mapstructure:"debug"`
	Server      server.Config       `yaml:"server" mapstructure:"server" validate:"required"`
	Database    db.ConnectionConfig `yaml:"database" mapstructure:"database" validate:"required"`
	Analytics   AnalyticsConfig     `yaml:"analytics" mapstructure:"analytics"`
	DataSources []DataSourceConfig  `yaml:"dataSources" mapstructure:"dataSources"`
}

type AnalyticsConfig struct {
	RetentionDays   int           `yaml:"retentionDays" mapstructure:"retentionDays" validate:"min=1"`
	ProcessInterval time.Duration `yaml:"processInterval" mapstructure:"processInterval" validate:"min=1s"`
	BatchSize       int           `yaml:"batchSize" mapstructure:"batchSize" validate:"min=1"`
	EnableRealTime  bool          `yaml:"enableRealTime" mapstructure:"enableRealTime"`
}

type DataSourceConfig struct {
	Name     string        `yaml:"name" mapstructure:"name" validate:"required"`
	Type     string        `yaml:"type" mapstructure:"type" validate:"required"`
	URL      string        `yaml:"url" mapstructure:"url"`
	Interval time.Duration `yaml:"interval" mapstructure:"interval"`
}

// Services contains all application dependencies
type Services struct {
	DB     *gorm.DB
	Config *AppConfig
	Logger zerolog.Logger
}

// Analytics Data Models
type Event struct {
	ID         uint                   `json:"id" gorm:"primaryKey"`
	EventType  string                 `json:"eventType" gorm:"not null;index"`
	UserID     string                 `json:"userId" gorm:"index"`
	SessionID  string                 `json:"sessionId" gorm:"index"`
	Properties map[string]interface{} `json:"properties" gorm:"type:jsonb"`
	Timestamp  time.Time              `json:"timestamp" gorm:"not null;index"`
	CreatedAt  time.Time              `json:"createdAt" gorm:"autoCreateTime"`
}

type DailyMetric struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Date       time.Time `json:"date" gorm:"not null;uniqueIndex:idx_daily_metric"`
	MetricName string    `json:"metricName" gorm:"not null;uniqueIndex:idx_daily_metric"`
	Value      float64   `json:"value" gorm:"not null"`
	Dimension  string    `json:"dimension" gorm:"index"`
	CreatedAt  time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

type UserSession struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	SessionID string     `json:"sessionId" gorm:"unique;not null"`
	UserID    string     `json:"userId" gorm:"index"`
	StartTime time.Time  `json:"startTime" gorm:"not null;index"`
	EndTime   *time.Time `json:"endTime" gorm:"index"`
	PageViews int        `json:"pageViews" gorm:"default:0"`
	Duration  int        `json:"duration" gorm:"default:0"` // seconds
	UserAgent string     `json:"userAgent"`
	IPAddress string     `json:"ipAddress"`
	Referrer  string     `json:"referrer"`
	CreatedAt time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
}

// Analytics Response Types
type DashboardResponse struct {
	Overview     OverviewMetrics `json:"overview"`
	TrafficData  []TrafficPoint  `json:"trafficData"`
	TopPages     []PageMetric    `json:"topPages"`
	UserMetrics  UserMetrics     `json:"userMetrics"`
	RealtimeData RealtimeData    `json:"realtimeData"`
	Timestamp    time.Time       `json:"timestamp"`
}

type OverviewMetrics struct {
	TotalUsers     int64   `json:"totalUsers"`
	TotalPageViews int64   `json:"totalPageViews"`
	TotalSessions  int64   `json:"totalSessions"`
	AvgSessionTime float64 `json:"avgSessionTime"`
	BounceRate     float64 `json:"bounceRate"`
}

type TrafficPoint struct {
	Date      time.Time `json:"date"`
	PageViews int64     `json:"pageViews"`
	Users     int64     `json:"users"`
	Sessions  int64     `json:"sessions"`
}

type PageMetric struct {
	Page        string  `json:"page"`
	Views       int64   `json:"views"`
	UniqueUsers int64   `json:"uniqueUsers"`
	AvgTime     float64 `json:"avgTime"`
}

type UserMetrics struct {
	NewUsers           int64   `json:"newUsers"`
	ReturningUsers     int64   `json:"returningUsers"`
	AvgSessionsPerUser float64 `json:"avgSessionsPerUser"`
}

type RealtimeData struct {
	CurrentUsers      int64        `json:"currentUsers"`
	PageViewsLast5Min int64        `json:"pageViewsLast5Min"`
	TopPagesRealtime  []PageMetric `json:"topPagesRealtime"`
	LastUpdated       time.Time    `json:"lastUpdated"`
}

func main() {
	// 1. Initialize logging first
	logging.Initialize("analytics-dashboard", os.Getenv("DEBUG") == "true")

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

	// Run database migrations
	if err := runMigrations(database); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// 4. Create services container
	services := &Services{
		DB:     database,
		Config: appConfig,
		Logger: logger,
	}

	// 5. Start background data processing
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start data processors
	go startEventProcessor(ctx, services)
	go startMetricsAggregator(ctx, services)
	if appConfig.Analytics.EnableRealTime {
		go startRealtimeProcessor(ctx, services)
	}

	// 6. Setup server with routes
	appConfig.Server.EchoConfigurer = setupRoutes(services)

	// 7. Start server (this will handle graceful shutdown automatically)
	logger.Info().
		Str("environment", appConfig.Environment).
		Int("port", appConfig.Server.Port).
		Bool("realtime", appConfig.Analytics.EnableRealTime).
		Msg("Starting analytics dashboard")

	server.StartWithConfig(appConfig.Server)

	logger.Info().Msg("Analytics dashboard shutdown completed")
}

func loadConfiguration() (*AppConfig, error) {
	defaultConfig := `
environment: development
debug: true
server:
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s
  shutdownTimeout: 10s
  enableHealthChecks: true
  enableMetrics: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: analytics_db
  timeout: 30s
  maxIdleConns: 10
  maxOpenConns: 50
analytics:
  retentionDays: 90
  processInterval: 1m
  batchSize: 1000
  enableRealTime: true
dataSources:
  - name: web_analytics
    type: webhook
    url: /api/v1/events
    interval: 0s
  - name: app_analytics
    type: api
    url: https://api.example.com/analytics
    interval: 5m
`

	appConfig, err := config.LoadString[AppConfig](defaultConfig, "ANALYTICS")
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

func runMigrations(database *gorm.DB) error {
	return database.AutoMigrate(
		&Event{},
		&DailyMetric{},
		&UserSession{},
	)
}

func setupRoutes(services *Services) func(*echo.Echo) {
	return func(e *echo.Echo) {
		// Middleware
		e.Use(middleware.RequestID())
		e.Use(middleware.Recover())
		e.Use(middleware.CORS())
		e.Use(loggingMiddleware(services))

		// Static files for dashboard
		e.Static("/", "static")

		// API routes
		api := e.Group("/api/v1")

		// Event ingestion
		api.POST("/events", ingestEventHandler(services))
		api.POST("/events/batch", ingestEventBatchHandler(services))

		// Dashboard data
		api.GET("/dashboard", getDashboardDataHandler(services))
		api.GET("/dashboard/overview", getOverviewHandler(services))
		api.GET("/dashboard/traffic", getTrafficDataHandler(services))
		api.GET("/dashboard/pages", getTopPagesHandler(services))
		api.GET("/dashboard/users", getUserMetricsHandler(services))
		api.GET("/dashboard/realtime", getRealtimeDataHandler(services))

		// Analytics queries
		api.GET("/analytics/events", queryEventsHandler(services))
		api.GET("/analytics/sessions", querySessionsHandler(services))
		api.GET("/analytics/metrics", queryMetricsHandler(services))

		// Custom reports
		api.POST("/reports/custom", generateCustomReportHandler(services))
		api.GET("/reports/:id", getReportHandler(services))

		// Health and metrics
		api.GET("/health", healthHandler(services))
		api.GET("/metrics", metricsHandler(services))
	}
}

func loggingMiddleware(services *Services) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			req := c.Request()
			res := c.Response()

			logger := logging.ContextLogger(req.Context(), "http-request")
			logger.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("status", res.Status).
				Dur("duration", time.Since(start)).
				Str("remote_ip", c.RealIP()).
				Msg("HTTP request")

			return err
		}
	}
}

// Event ingestion handlers
func ingestEventHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := logging.ContextLogger(c.Request().Context(), "ingest-event")

		var event Event
		if err := c.Bind(&event); err != nil {
			logger.Warn().Err(err).Msg("Invalid event data")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid event data"})
		}

		// Set timestamp if not provided
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		// Store event
		if err := services.DB.Create(&event).Error; err != nil {
			logger.Error().Err(err).Msg("Failed to store event")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store event"})
		}

		logger.Info().
			Str("event_type", event.EventType).
			Str("user_id", event.UserID).
			Msg("Event ingested")

		return c.JSON(http.StatusCreated, map[string]string{"status": "success"})
	}
}

func ingestEventBatchHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := logging.ContextLogger(c.Request().Context(), "ingest-batch")

		var events []Event
		if err := c.Bind(&events); err != nil {
			logger.Warn().Err(err).Msg("Invalid batch data")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid batch data"})
		}

		// Process events concurrently
		batchFunctions := make(map[string]concurrent.Func[bool])
		for i, event := range events {
			eventCopy := event
			key := fmt.Sprintf("event_%d", i)
			batchFunctions[key] = func(ctx context.Context) (bool, error) {
				if eventCopy.Timestamp.IsZero() {
					eventCopy.Timestamp = time.Now()
				}
				return true, services.DB.Create(&eventCopy).Error
			}
		}

		results, err := concurrent.ExecuteConcurrently(c.Request().Context(), batchFunctions)
		if err != nil {
			logger.Error().Err(err).Msg("Batch processing failed")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Batch processing failed"})
		}

		successCount := 0
		for _, result := range results {
			if result {
				successCount++
			}
		}

		logger.Info().
			Int("total", len(events)).
			Int("successful", successCount).
			Msg("Event batch processed")

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"status":    "success",
			"processed": successCount,
			"total":     len(events),
		})
	}
}

// Dashboard data handlers
func getDashboardDataHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := logging.ContextLogger(c.Request().Context(), "dashboard-data")

		// Get date range from query params
		days, _ := strconv.Atoi(c.QueryParam("days"))
		if days == 0 {
			days = 7
		}

		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days)

		// Fetch all dashboard data concurrently
		dataFunctions := map[string]concurrent.Func[interface{}]{
			"overview": func(ctx context.Context) (interface{}, error) {
				return getOverviewMetrics(services.DB, startDate, endDate)
			},
			"traffic": func(ctx context.Context) (interface{}, error) {
				return getTrafficData(services.DB, startDate, endDate)
			},
			"pages": func(ctx context.Context) (interface{}, error) {
				return getTopPages(services.DB, startDate, endDate)
			},
			"users": func(ctx context.Context) (interface{}, error) {
				return getUserMetrics(services.DB, startDate, endDate)
			},
			"realtime": func(ctx context.Context) (interface{}, error) {
				if services.Config.Analytics.EnableRealTime {
					return getRealtimeData(services.DB)
				}
				return RealtimeData{}, nil
			},
		}

		results, err := concurrent.ExecuteConcurrently(c.Request().Context(), dataFunctions)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to fetch dashboard data")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch data"})
		}

		response := DashboardResponse{
			Overview:     results["overview"].(OverviewMetrics),
			TrafficData:  results["traffic"].([]TrafficPoint),
			TopPages:     results["pages"].([]PageMetric),
			UserMetrics:  results["users"].(UserMetrics),
			RealtimeData: results["realtime"].(RealtimeData),
			Timestamp:    time.Now(),
		}

		logger.Info().Dur("range_days", time.Duration(days)*24*time.Hour).Msg("Dashboard data fetched")
		return c.JSON(http.StatusOK, response)
	}
}

func getOverviewHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.QueryParam("days"))
		if days == 0 {
			days = 7
		}

		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days)

		overview, err := getOverviewMetrics(services.DB, startDate, endDate)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch overview"})
		}

		return c.JSON(http.StatusOK, overview)
	}
}

func getTrafficDataHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.QueryParam("days"))
		if days == 0 {
			days = 7
		}

		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days)

		traffic, err := getTrafficData(services.DB, startDate, endDate)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch traffic data"})
		}

		return c.JSON(http.StatusOK, traffic)
	}
}

func getTopPagesHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.QueryParam("days"))
		if days == 0 {
			days = 7
		}

		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days)

		pages, err := getTopPages(services.DB, startDate, endDate)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch page data"})
		}

		return c.JSON(http.StatusOK, pages)
	}
}

func getUserMetricsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.QueryParam("days"))
		if days == 0 {
			days = 7
		}

		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days)

		userMetrics, err := getUserMetrics(services.DB, startDate, endDate)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user metrics"})
		}

		return c.JSON(http.StatusOK, userMetrics)
	}
}

func getRealtimeDataHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !services.Config.Analytics.EnableRealTime {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Realtime data not enabled"})
		}

		realtimeData, err := getRealtimeData(services.DB)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch realtime data"})
		}

		return c.JSON(http.StatusOK, realtimeData)
	}
}

func queryEventsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement event querying with filters
		return c.JSON(http.StatusOK, []Event{})
	}
}

func querySessionsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement session querying
		return c.JSON(http.StatusOK, []UserSession{})
	}
}

func queryMetricsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement metrics querying
		return c.JSON(http.StatusOK, []DailyMetric{})
	}
}

func generateCustomReportHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement custom report generation
		return c.JSON(http.StatusCreated, map[string]string{"reportId": "report_123"})
	}
}

func getReportHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement report retrieval
		return c.JSON(http.StatusOK, map[string]string{"status": "completed"})
	}
}

func healthHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		sqlDB, err := services.DB.DB()
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
		}

		if err := sqlDB.Ping(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "database ping failed",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
			"features": map[string]bool{
				"realtime": services.Config.Analytics.EnableRealTime,
			},
		})
	}
}

func metricsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		var totalEvents, totalSessions int64
		services.DB.Model(&Event{}).Count(&totalEvents)
		services.DB.Model(&UserSession{}).Count(&totalSessions)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"uptime":          time.Since(startTime),
			"total_events":    totalEvents,
			"total_sessions":  totalSessions,
			"database_status": "healthy",
		})
	}
}

// Background processors
func startEventProcessor(ctx context.Context, services *Services) {
	logger := logging.ContextLogger(ctx, "event-processor")
	logger.Info().Msg("Starting event processor")

	ticker := time.NewTicker(services.Config.Analytics.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Event processor shutting down")
			return
		case <-ticker.C:
			if err := processEvents(ctx, services); err != nil {
				logger.Error().Err(err).Msg("Event processing failed")
			}
		}
	}
}

func startMetricsAggregator(ctx context.Context, services *Services) {
	logger := logging.ContextLogger(ctx, "metrics-aggregator")
	logger.Info().Msg("Starting metrics aggregator")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Metrics aggregator shutting down")
			return
		case <-ticker.C:
			if err := aggregateMetrics(ctx, services); err != nil {
				logger.Error().Err(err).Msg("Metrics aggregation failed")
			}
		}
	}
}

func startRealtimeProcessor(ctx context.Context, services *Services) {
	logger := logging.ContextLogger(ctx, "realtime-processor")
	logger.Info().Msg("Starting realtime processor")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Realtime processor shutting down")
			return
		case <-ticker.C:
			if err := updateRealtimeMetrics(ctx, services); err != nil {
				logger.Error().Err(err).Msg("Realtime update failed")
			}
		}
	}
}

// Helper functions for data processing
func processEvents(ctx context.Context, services *Services) error {
	// TODO: Implement event processing logic
	return nil
}

func aggregateMetrics(ctx context.Context, services *Services) error {
	// TODO: Implement metrics aggregation
	return nil
}

func updateRealtimeMetrics(ctx context.Context, services *Services) error {
	// TODO: Implement realtime metrics update
	return nil
}

// Data fetching functions
func getOverviewMetrics(db *gorm.DB, startDate, endDate time.Time) (OverviewMetrics, error) {
	// Mock implementation - replace with actual queries
	return OverviewMetrics{
		TotalUsers:     rand.Int63n(10000),
		TotalPageViews: rand.Int63n(50000),
		TotalSessions:  rand.Int63n(20000),
		AvgSessionTime: 180.5 + rand.Float64()*120,
		BounceRate:     0.3 + rand.Float64()*0.4,
	}, nil
}

func getTrafficData(db *gorm.DB, startDate, endDate time.Time) ([]TrafficPoint, error) {
	// Mock implementation - replace with actual queries
	var traffic []TrafficPoint
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		traffic = append(traffic, TrafficPoint{
			Date:      d,
			PageViews: rand.Int63n(5000) + 1000,
			Users:     rand.Int63n(2000) + 500,
			Sessions:  rand.Int63n(3000) + 800,
		})
	}
	return traffic, nil
}

func getTopPages(db *gorm.DB, startDate, endDate time.Time) ([]PageMetric, error) {
	// Mock implementation
	pages := []string{"/", "/products", "/about", "/contact", "/blog", "/login"}
	var metrics []PageMetric

	for _, page := range pages {
		metrics = append(metrics, PageMetric{
			Page:        page,
			Views:       rand.Int63n(10000) + 500,
			UniqueUsers: rand.Int63n(5000) + 200,
			AvgTime:     60 + rand.Float64()*300,
		})
	}
	return metrics, nil
}

func getUserMetrics(db *gorm.DB, startDate, endDate time.Time) (UserMetrics, error) {
	// Mock implementation
	return UserMetrics{
		NewUsers:           rand.Int63n(5000) + 1000,
		ReturningUsers:     rand.Int63n(8000) + 2000,
		AvgSessionsPerUser: 2.5 + rand.Float64()*2,
	}, nil
}

func getRealtimeData(db *gorm.DB) (RealtimeData, error) {
	// Mock implementation
	pages, _ := getTopPages(db, time.Now().Add(-time.Hour), time.Now())
	if len(pages) > 3 {
		pages = pages[:3]
	}

	return RealtimeData{
		CurrentUsers:      rand.Int63n(500) + 50,
		PageViewsLast5Min: rand.Int63n(200) + 20,
		TopPagesRealtime:  pages,
		LastUpdated:       time.Now(),
	}, nil
}

var startTime = time.Now()
