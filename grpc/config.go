package grpc

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
)

// ServerMode defines the server operation mode
type ServerMode string

const (
	// SeparateMode runs gRPC and HTTP servers on separate ports
	SeparateMode ServerMode = "separate"
	// H2CMode runs both gRPC and HTTP on a single port using HTTP/2 cleartext
	H2CMode ServerMode = "h2c"
)

// Config represents the configuration for the gRPC server and gateway
type Config struct {
	// Server Configuration
	GRPCPort string     // Port for gRPC server
	HTTPPort string     // Port for HTTP gateway (only used in SeparateMode)
	Mode     ServerMode // Server operation mode

	// Timeouts and Limits
	ShutdownTimeout       time.Duration // Maximum time to wait for graceful shutdown
	ReadTimeout           time.Duration // HTTP server read timeout
	WriteTimeout          time.Duration // HTTP server write timeout
	IdleTimeout           time.Duration // HTTP server idle timeout
	MaxConnectionIdle     time.Duration // gRPC server max connection idle time
	MaxConnectionAge      time.Duration // gRPC server max connection age
	MaxConnectionAgeGrace time.Duration // gRPC server max connection age grace

	// Production Features
	EnableMetrics     bool   // Enable Prometheus metrics endpoint
	MetricsPath       string // Path for metrics endpoint
	EnableHealthCheck bool   // Enable health check endpoints
	HealthPath        string // Base path for health check endpoints
	EnableLogging     bool   // Enable request/response logging
	EnableReflection  bool   // Enable gRPC server reflection

	// Customization Hooks
	GRPCConfigurer   func(*grpc.Server) // Configure gRPC server
	EchoConfigurer   func(*echo.Echo)   // Configure Echo HTTP server
	ServiceRegistrar func(*grpc.Server) // Register gRPC services
	Shutdown         func() error       // Custom shutdown handler

	// Gateway Configuration
	GatewayBasePath string // Base path for gRPC gateway routes (default: "/api/v1")

	// Echo-specific Features
	EnableCORS      bool                  // Enable CORS middleware
	EnableRateLimit bool                  // Enable rate limiting middleware
	RateLimit       float64               // Requests per second for rate limiting
	Middleware      []echo.MiddlewareFunc // Custom Echo middleware

	// TLS Configuration (for future use)
	EnableTLS bool   // Enable TLS
	CertFile  string // Path to certificate file
	KeyFile   string // Path to private key file
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		// Server Configuration
		GRPCPort: "8080",
		HTTPPort: "8081",
		Mode:     H2CMode, // Default to H2C for simpler development

		// Timeouts and Limits
		ShutdownTimeout:       30 * time.Second,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           60 * time.Second,
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Second,

		// Production Features
		EnableMetrics:     true,
		MetricsPath:       "/metrics",
		EnableHealthCheck: true,
		HealthPath:        "/health",
		EnableLogging:     true,
		EnableReflection:  true,

		// Gateway Configuration
		GatewayBasePath: "/api/v1",

		// Echo-specific Features
		EnableCORS:      false, // Disabled by default, enable as needed
		EnableRateLimit: false, // Disabled by default, enable as needed
		RateLimit:       100.0, // 100 requests per second default
		Middleware:      []echo.MiddlewareFunc{},

		// TLS Configuration
		EnableTLS: false,
	}
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	defaultConfig := DefaultConfig()

	if c.GRPCPort == "" {
		c.GRPCPort = defaultConfig.GRPCPort
	}
	if c.HTTPPort == "" && c.Mode == SeparateMode {
		c.HTTPPort = defaultConfig.HTTPPort
	}
	if c.Mode == "" {
		c.Mode = defaultConfig.Mode
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = defaultConfig.ShutdownTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaultConfig.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaultConfig.WriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = defaultConfig.IdleTimeout
	}
	if c.MaxConnectionIdle == 0 {
		c.MaxConnectionIdle = defaultConfig.MaxConnectionIdle
	}
	if c.MaxConnectionAge == 0 {
		c.MaxConnectionAge = defaultConfig.MaxConnectionAge
	}
	if c.MaxConnectionAgeGrace == 0 {
		c.MaxConnectionAgeGrace = defaultConfig.MaxConnectionAgeGrace
	}
	if c.MetricsPath == "" {
		c.MetricsPath = defaultConfig.MetricsPath
	}
	if c.HealthPath == "" {
		c.HealthPath = defaultConfig.HealthPath
	}
	if c.GatewayBasePath == "" {
		c.GatewayBasePath = defaultConfig.GatewayBasePath
	}
	if c.RateLimit == 0 {
		c.RateLimit = defaultConfig.RateLimit
	}
	if c.Middleware == nil {
		c.Middleware = []echo.MiddlewareFunc{}
	}

	// Set boolean defaults only if not explicitly set
	// Note: This is a simplification - in practice you might want to use pointers
	// to distinguish between false and unset
	if !c.EnableMetrics && !c.EnableHealthCheck && !c.EnableLogging {
		c.EnableMetrics = defaultConfig.EnableMetrics
		c.EnableHealthCheck = defaultConfig.EnableHealthCheck
		c.EnableLogging = defaultConfig.EnableLogging
	}
}

// Validate ensures the configuration is valid
func (c *Config) Validate() error {
	if c.GRPCPort == "" {
		return fmt.Errorf("GRPCPort cannot be empty")
	}

	if c.Mode == SeparateMode && c.HTTPPort == "" {
		return fmt.Errorf("HTTPPort cannot be empty when using SeparateMode")
	}

	if c.Mode != H2CMode && c.Mode != SeparateMode {
		return fmt.Errorf("invalid server mode: %s", c.Mode)
	}

	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("ShutdownTimeout cannot be negative")
	}

	if c.ReadTimeout < 0 {
		return fmt.Errorf("ReadTimeout cannot be negative")
	}

	if c.WriteTimeout < 0 {
		return fmt.Errorf("WriteTimeout cannot be negative")
	}

	if c.IdleTimeout < 0 {
		return fmt.Errorf("IdleTimeout cannot be negative")
	}

	return nil
}

// GetGRPCAddress returns the full address for the gRPC server
func (c *Config) GetGRPCAddress() string {
	return ":" + c.GRPCPort
}

// GetHTTPAddress returns the full address for the HTTP server
func (c *Config) GetHTTPAddress() string {
	if c.Mode == H2CMode {
		return c.GetGRPCAddress() // Use same port for H2C
	}
	return ":" + c.HTTPPort
}

// IsH2CMode returns true if server is running in H2C mode
func (c *Config) IsH2CMode() bool {
	return c.Mode == H2CMode
}

// IsSeparateMode returns true if server is running in separate mode
func (c *Config) IsSeparateMode() bool {
	return c.Mode == SeparateMode
}
