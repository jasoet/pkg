package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/config"
	"github.com/jasoet/pkg/db"
	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AppConfig defines the application configuration structure
type AppConfig struct {
	Environment string              `yaml:"environment" mapstructure:"environment" validate:"required,oneof=development staging production"`
	Debug       bool                `yaml:"debug" mapstructure:"debug"`
	Server      server.Config       `yaml:"server" mapstructure:"server" validate:"required"`
	Database    db.ConnectionConfig `yaml:"database" mapstructure:"database" validate:"required"`
}

// Services contains all application dependencies
type Services struct {
	DB     *gorm.DB
	Config *AppConfig
	Logger zerolog.Logger
}

func main() {
	// 1. Initialize logging first (CRITICAL)
	logging.Initialize("web-service", os.Getenv("DEBUG") == "true")

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

	// 4. Create services container
	services := &Services{
		DB:     database,
		Config: appConfig,
		Logger: logger,
	}

	// 5. Setup server with routes
	appConfig.Server.EchoConfigurer = setupRoutes(services)
	srv := server.New(&appConfig.Server)

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

	// 7. Start server
	logger.Info().
		Str("environment", appConfig.Environment).
		Int("port", appConfig.Server.Port).
		Msg("Starting web service")

	if err := server.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Server failed to start")
	}

	logger.Info().Msg("Web service shutdown completed")
}

func loadConfiguration() (*AppConfig, error) {
	// Default configuration for development
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
  dbName: myapp
  timeout: 30s
  maxIdleConns: 10
  maxOpenConns: 100
`

	// Load configuration with environment variable overrides
	appConfig, err := config.LoadString[AppConfig](defaultConfig, "APP")
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

func setupRoutes(services *Services) func(*echo.Echo) {
	return func(e *echo.Echo) {
		// Add middleware
		e.Use(middleware.RequestID())
		e.Use(middleware.Recover())
		e.Use(middleware.CORS())
		e.Use(loggingMiddleware(services))

		// API routes
		api := e.Group("/api/v1")

		// Health check endpoint
		api.GET("/health", healthHandler(services))

		// Example resource endpoints
		setupUserRoutes(api, services)
		setupProductRoutes(api, services)

		// Admin routes (protected)
		admin := api.Group("/admin")
		admin.Use(authMiddleware(services))
		admin.GET("/stats", statsHandler(services))
	}
}

func setupUserRoutes(g *echo.Group, services *Services) {
	users := g.Group("/users")
	handler := NewUserHandler(services)

	users.GET("", handler.ListUsers)
	users.POST("", handler.CreateUser)
	users.GET("/:id", handler.GetUser)
	users.PUT("/:id", handler.UpdateUser)
	users.DELETE("/:id", handler.DeleteUser)
}

func setupProductRoutes(g *echo.Group, services *Services) {
	products := g.Group("/products")
	handler := NewProductHandler(services)

	products.GET("", handler.ListProducts)
	products.POST("", handler.CreateProduct)
	products.GET("/:id", handler.GetProduct)
	products.PUT("/:id", handler.UpdateProduct)
	products.DELETE("/:id", handler.DeleteProduct)
}

// Middleware
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
				Str("query", req.URL.RawQuery).
				Int("status", res.Status).
				Dur("duration", time.Since(start)).
				Str("remote_ip", c.RealIP()).
				Str("user_agent", req.UserAgent()).
				Msg("HTTP request")

			return err
		}
	}
}

func authMiddleware(services *Services) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := logging.ContextLogger(c.Request().Context(), "auth-middleware")

			// Simple authentication example - replace with your auth logic
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn().Str("path", c.Request().URL.Path).Msg("Missing authorization header")
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization required")
			}

			// TODO: Implement actual token validation
			if authHeader != "Bearer valid-token" {
				logger.Error().Str("header", authHeader).Msg("Invalid authorization token")
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			return next(c)
		}
	}
}

// Handlers
func healthHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check database connectivity
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
		})
	}
}

func statsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Example admin statistics
		stats := map[string]interface{}{
			"environment": services.Config.Environment,
			"uptime":      time.Since(startTime),
			"requests":    getRequestCount(), // Implement request counter
		}

		return c.JSON(http.StatusOK, stats)
	}
}

// Example handlers - implement these in separate files
type UserHandler struct {
	services *Services
}

func NewUserHandler(services *Services) *UserHandler {
	return &UserHandler{services: services}
}

func (h *UserHandler) ListUsers(c echo.Context) error {
	// TODO: Implement user listing
	return c.JSON(http.StatusOK, []interface{}{})
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	// TODO: Implement user creation
	return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
}

func (h *UserHandler) GetUser(c echo.Context) error {
	// TODO: Implement user retrieval
	return c.JSON(http.StatusOK, map[string]string{"status": "user found"})
}

func (h *UserHandler) UpdateUser(c echo.Context) error {
	// TODO: Implement user update
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	// TODO: Implement user deletion
	return c.NoContent(http.StatusNoContent)
}

type ProductHandler struct {
	services *Services
}

func NewProductHandler(services *Services) *ProductHandler {
	return &ProductHandler{services: services}
}

func (h *ProductHandler) ListProducts(c echo.Context) error {
	// TODO: Implement product listing
	return c.JSON(http.StatusOK, []interface{}{})
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	// TODO: Implement product creation
	return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	// TODO: Implement product retrieval
	return c.JSON(http.StatusOK, map[string]string{"status": "product found"})
}

func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	// TODO: Implement product update
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	// TODO: Implement product deletion
	return c.NoContent(http.StatusNoContent)
}

// Helper functions
var startTime = time.Now()

func getRequestCount() int {
	// TODO: Implement request counter
	return 0
}
