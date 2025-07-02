package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jasoet/pkg/config"
	"github.com/jasoet/pkg/db"
	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/rest"
	"github.com/jasoet/pkg/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AppConfig defines the e-commerce application configuration
type AppConfig struct {
	Environment string              `yaml:"environment" mapstructure:"environment" validate:"required,oneof=development staging production"`
	Debug       bool                `yaml:"debug" mapstructure:"debug"`
	Server      server.Config       `yaml:"server" mapstructure:"server" validate:"required"`
	Database    db.ConnectionConfig `yaml:"database" mapstructure:"database" validate:"required"`
	JWT         JWTConfig           `yaml:"jwt" mapstructure:"jwt" validate:"required"`
	Payment     PaymentConfig       `yaml:"payment" mapstructure:"payment"`
	Upload      UploadConfig        `yaml:"upload" mapstructure:"upload"`
}

type JWTConfig struct {
	Secret     string        `yaml:"secret" mapstructure:"secret" validate:"required,min=32"`
	Expiration time.Duration `yaml:"expiration" mapstructure:"expiration" validate:"required"`
}

type PaymentConfig struct {
	Provider string `yaml:"provider" mapstructure:"provider" validate:"required"`
	APIKey   string `yaml:"apiKey" mapstructure:"apiKey" validate:"required"`
	BaseURL  string `yaml:"baseUrl" mapstructure:"baseUrl" validate:"required,url"`
}

type UploadConfig struct {
	MaxFileSize   int64  `yaml:"maxFileSize" mapstructure:"maxFileSize" validate:"min=1"`
	AllowedTypes  []string `yaml:"allowedTypes" mapstructure:"allowedTypes"`
	UploadDir     string `yaml:"uploadDir" mapstructure:"uploadDir" validate:"required"`
}

// Services contains all application dependencies
type Services struct {
	DB            *gorm.DB
	PaymentClient *rest.Client
	Config        *AppConfig
	Logger        zerolog.Logger
}

// Database Models
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	FirstName string    `json:"firstName" gorm:"not null"`
	LastName  string    `json:"lastName" gorm:"not null"`
	Role      string    `json:"role" gorm:"default:customer"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	
	// Relationships
	Orders []Order `json:"orders,omitempty" gorm:"foreignKey:UserID"`
}

type Category struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"unique;not null"`
	Description string    `json:"description"`
	ImageURL    string    `json:"imageUrl"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	
	// Relationships
	Products []Product `json:"products,omitempty" gorm:"foreignKey:CategoryID"`
}

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Price       float64   `json:"price" gorm:"not null"`
	Stock       int       `json:"stock" gorm:"default:0"`
	SKU         string    `json:"sku" gorm:"unique;not null"`
	ImageURL    string    `json:"imageUrl"`
	CategoryID  uint      `json:"categoryId" gorm:"not null"`
	IsActive    bool      `json:"isActive" gorm:"default:true"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	
	// Relationships
	Category   Category    `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	OrderItems []OrderItem `json:"orderItems,omitempty" gorm:"foreignKey:ProductID"`
}

type Order struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"userId" gorm:"not null"`
	Status      string    `json:"status" gorm:"default:pending"`
	TotalAmount float64   `json:"totalAmount" gorm:"not null"`
	PaymentID   string    `json:"paymentId"`
	ShippingAddress string `json:"shippingAddress" gorm:"not null"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	
	// Relationships
	User       User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	OrderItems []OrderItem `json:"orderItems,omitempty" gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	OrderID   uint    `json:"orderId" gorm:"not null"`
	ProductID uint    `json:"productId" gorm:"not null"`
	Quantity  int     `json:"quantity" gorm:"not null"`
	Price     float64 `json:"price" gorm:"not null"`
	
	// Relationships
	Order   Order   `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	Product Product `json:"product,omitempty" gorm:"foreignKey:ProductID"`
}

func main() {
	// 1. Initialize logging first (CRITICAL)
	logging.Initialize("ecommerce-api", os.Getenv("DEBUG") == "true")

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

	// 4. Setup external services
	paymentClient := rest.NewClient(rest.WithRestConfig(rest.Config{
		Timeout:    30 * time.Second,
		RetryCount: 3,
	}))

	// 5. Create services container
	services := &Services{
		DB:            database,
		PaymentClient: paymentClient,
		Config:        appConfig,
		Logger:        logger,
	}

	// 6. Setup server with routes
	appConfig.Server.EchoConfigurer = setupRoutes(services)
	srv := server.New(&appConfig.Server)

	// 7. Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// 8. Start server
	logger.Info().
		Str("environment", appConfig.Environment).
		Int("port", appConfig.Server.Port).
		Msg("Starting e-commerce API server")

	if err := srv.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Server failed to start")
	}

	logger.Info().Msg("E-commerce API server shutdown completed")
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
  dbName: ecommerce_db
  timeout: 30s
  maxIdleConns: 10
  maxOpenConns: 50
jwt:
  secret: your-super-secret-jwt-key-at-least-32-characters-long
  expiration: 24h
payment:
  provider: stripe
  apiKey: sk_test_your_stripe_secret_key
  baseUrl: https://api.stripe.com
upload:
  maxFileSize: 5242880  # 5MB
  allowedTypes:
    - image/jpeg
    - image/png
    - image/gif
  uploadDir: ./uploads
`

	// Load configuration with environment variable overrides
	appConfig, err := config.LoadString[AppConfig](defaultConfig, "ECOMMERCE")
	if err != nil {
		return nil, err
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(appConfig.Upload.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return appConfig, nil
}

func runMigrations(database *gorm.DB) error {
	// Auto-migrate all models
	return database.AutoMigrate(
		&User{},
		&Category{},
		&Product{},
		&Order{},
		&OrderItem{},
	)
}

func setupRoutes(services *Services) func(*echo.Echo) {
	return func(e *echo.Echo) {
		// Middleware setup
		e.Use(middleware.RequestID())
		e.Use(middleware.Recover())
		e.Use(middleware.CORS())
		e.Use(loggingMiddleware(services))

		// Static file serving for uploads
		e.Static("/uploads", services.Config.Upload.UploadDir)

		// API routes
		api := e.Group("/api/v1")

		// Public routes
		auth := api.Group("/auth")
		auth.POST("/register", registerHandler(services))
		auth.POST("/login", loginHandler(services))

		// Product catalog (public)
		api.GET("/categories", listCategoriesHandler(services))
		api.GET("/categories/:id/products", listProductsByCategoryHandler(services))
		api.GET("/products", listProductsHandler(services))
		api.GET("/products/:id", getProductHandler(services))

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(authMiddleware(services))

		// User management
		users := protected.Group("/users")
		users.GET("/profile", getUserProfileHandler(services))
		users.PUT("/profile", updateUserProfileHandler(services))

		// Shopping cart and orders
		orders := protected.Group("/orders")
		orders.GET("", listUserOrdersHandler(services))
		orders.POST("", createOrderHandler(services))
		orders.GET("/:id", getOrderHandler(services))

		// Admin routes (require admin role)
		admin := protected.Group("/admin")
		admin.Use(adminMiddleware(services))

		// Category management
		adminCategories := admin.Group("/categories")
		adminCategories.GET("", listCategoriesHandler(services))
		adminCategories.POST("", createCategoryHandler(services))
		adminCategories.PUT("/:id", updateCategoryHandler(services))
		adminCategories.DELETE("/:id", deleteCategoryHandler(services))

		// Product management
		adminProducts := admin.Group("/products")
		adminProducts.GET("", listProductsHandler(services))
		adminProducts.POST("", createProductHandler(services))
		adminProducts.PUT("/:id", updateProductHandler(services))
		adminProducts.DELETE("/:id", deleteProductHandler(services))
		adminProducts.POST("/:id/upload", uploadProductImageHandler(services))

		// Order management
		adminOrders := admin.Group("/orders")
		adminOrders.GET("", listAllOrdersHandler(services))
		adminOrders.PUT("/:id/status", updateOrderStatusHandler(services))

		// Health check
		api.GET("/health", healthHandler(services))

		// Metrics endpoint
		api.GET("/metrics", metricsHandler(services))
	}
}

// Middleware functions
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
				Str("request_id", c.Response().Header().Get(echo.HeaderXRequestID)).
				Msg("HTTP request")

			return err
		}
	}
}

func authMiddleware(services *Services) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := logging.ContextLogger(c.Request().Context(), "auth-middleware")

			// Extract JWT token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn().Str("path", c.Request().URL.Path).Msg("Missing authorization header")
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header required")
			}

			// Parse Bearer token
			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				logger.Warn().Str("header", authHeader).Msg("Invalid authorization header format")
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			token := authHeader[7:]

			// TODO: Implement JWT token validation
			// For demo purposes, we'll use a simple validation
			userID, err := validateJWTToken(token, services.Config.JWT.Secret)
			if err != nil {
				logger.Error().Err(err).Str("token", token).Msg("Token validation failed")
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Store user ID in context
			c.Set("userID", userID)
			return next(c)
		}
	}
}

func adminMiddleware(services *Services) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := logging.ContextLogger(c.Request().Context(), "admin-middleware")

			userID := c.Get("userID").(uint)

			// Check if user has admin role
			var user User
			if err := services.DB.First(&user, userID).Error; err != nil {
				logger.Error().Err(err).Uint("user_id", userID).Msg("Failed to fetch user")
				return echo.NewHTTPError(http.StatusInternalServerError, "User lookup failed")
			}

			if user.Role != "admin" {
				logger.Warn().Uint("user_id", userID).Str("role", user.Role).Msg("Access denied: admin role required")
				return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
			}

			return next(c)
		}
	}
}

// Handler functions (placeholder implementations)
func registerHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement user registration with password hashing
		return c.JSON(http.StatusCreated, map[string]string{"message": "User registered successfully"})
	}
}

func loginHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement user authentication and JWT generation
		return c.JSON(http.StatusOK, map[string]string{"token": "jwt-token-here"})
	}
}

func listCategoriesHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		var categories []Category
		if err := services.DB.Find(&categories).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch categories"})
		}
		return c.JSON(http.StatusOK, categories)
	}
}

func listProductsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		var products []Product
		query := services.DB.Preload("Category")

		// Add pagination
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"products": products,
			"page":     page,
			"limit":    limit,
		})
	}
}

func getProductHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		}

		var product Product
		if err := services.DB.Preload("Category").First(&product, uint(id)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product"})
		}

		return c.JSON(http.StatusOK, product)
	}
}

func listProductsByCategoryHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		categoryID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid category ID"})
		}

		var products []Product
		if err := services.DB.Where("category_id = ?", uint(categoryID)).Preload("Category").Find(&products).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products"})
		}

		return c.JSON(http.StatusOK, products)
	}
}

func getUserProfileHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("userID").(uint)

		var user User
		if err := services.DB.First(&user, userID).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user profile"})
		}

		return c.JSON(http.StatusOK, user)
	}
}

func updateUserProfileHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement user profile update
		return c.JSON(http.StatusOK, map[string]string{"message": "Profile updated successfully"})
	}
}

func listUserOrdersHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("userID").(uint)

		var orders []Order
		if err := services.DB.Where("user_id = ?", userID).Preload("OrderItems.Product").Find(&orders).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch orders"})
		}

		return c.JSON(http.StatusOK, orders)
	}
}

func createOrderHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement order creation with payment processing
		return c.JSON(http.StatusCreated, map[string]string{"message": "Order created successfully"})
	}
}

func getOrderHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement order retrieval
		return c.JSON(http.StatusOK, map[string]string{"message": "Order details"})
	}
}

func createCategoryHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement category creation
		return c.JSON(http.StatusCreated, map[string]string{"message": "Category created successfully"})
	}
}

func updateCategoryHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement category update
		return c.JSON(http.StatusOK, map[string]string{"message": "Category updated successfully"})
	}
}

func deleteCategoryHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement category deletion
		return c.JSON(http.StatusOK, map[string]string{"message": "Category deleted successfully"})
	}
}

func createProductHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement product creation
		return c.JSON(http.StatusCreated, map[string]string{"message": "Product created successfully"})
	}
}

func updateProductHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement product update
		return c.JSON(http.StatusOK, map[string]string{"message": "Product updated successfully"})
	}
}

func deleteProductHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement product deletion
		return c.JSON(http.StatusOK, map[string]string{"message": "Product deleted successfully"})
	}
}

func uploadProductImageHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement file upload
		return c.JSON(http.StatusOK, map[string]string{"message": "Image uploaded successfully"})
	}
}

func listAllOrdersHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement admin order listing
		return c.JSON(http.StatusOK, []Order{})
	}
}

func updateOrderStatusHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement order status update
		return c.JSON(http.StatusOK, map[string]string{"message": "Order status updated successfully"})
	}
}

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
			"services": map[string]string{
				"database": "healthy",
				"payment":  "healthy",
			},
		})
	}
}

func metricsHandler(services *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: Implement metrics collection
		return c.JSON(http.StatusOK, map[string]interface{}{
			"uptime":         time.Since(startTime),
			"total_users":    getTotalUsers(services.DB),
			"total_products": getTotalProducts(services.DB),
			"total_orders":   getTotalOrders(services.DB),
		})
	}
}

// Helper functions
var startTime = time.Now()

func validateJWTToken(token, secret string) (uint, error) {
	// TODO: Implement proper JWT validation
	// For demo purposes, return a dummy user ID
	if token == "demo-token" {
		return 1, nil
	}
	return 0, fmt.Errorf("invalid token")
}

func getTotalUsers(db *gorm.DB) int64 {
	var count int64
	db.Model(&User{}).Count(&count)
	return count
}

func getTotalProducts(db *gorm.DB) int64 {
	var count int64
	db.Model(&Product{}).Count(&count)
	return count
}

func getTotalOrders(db *gorm.DB) int64 {
	var count int64
	db.Model(&Order{}).Count(&count)
	return count
}