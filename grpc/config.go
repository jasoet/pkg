package grpc

import (
	"fmt"
	"time"

	"github.com/jasoet/pkg/v2/otel"
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

// Option is a functional option for configuring the server
type Option func(*config)

// config represents the internal configuration for the gRPC server and gateway
type config struct {
	// Server Configuration
	grpcPort string     // Port for gRPC server
	httpPort string     // Port for HTTP gateway (only used in SeparateMode)
	mode     ServerMode // Server operation mode

	// Timeouts and Limits
	shutdownTimeout       time.Duration // Maximum time to wait for graceful shutdown
	readTimeout           time.Duration // HTTP server read timeout
	writeTimeout          time.Duration // HTTP server write timeout
	idleTimeout           time.Duration // HTTP server idle timeout
	maxConnectionIdle     time.Duration // gRPC server max connection idle time
	maxConnectionAge      time.Duration // gRPC server max connection age
	maxConnectionAgeGrace time.Duration // gRPC server max connection age grace

	// Production Features
	enableMetrics     bool   // Enable Prometheus metrics endpoint
	metricsPath       string // Path for metrics endpoint
	enableHealthCheck bool   // Enable health check endpoints
	healthPath        string // Base path for health check endpoints
	enableLogging     bool   // Enable request/response logging
	enableReflection  bool   // Enable gRPC server reflection

	// Customization Hooks
	grpcConfigurer   func(*grpc.Server) // Configure gRPC server
	echoConfigurer   func(*echo.Echo)   // Configure Echo HTTP server
	serviceRegistrar func(*grpc.Server) // Register gRPC services
	shutdown         func() error       // Custom shutdown handler

	// Gateway Configuration
	gatewayBasePath string // Base path for gRPC gateway routes (default: "/api/v1")

	// Echo-specific Features
	enableCORS      bool                  // Enable CORS middleware
	enableRateLimit bool                  // Enable rate limiting middleware
	rateLimit       float64               // Requests per second for rate limiting
	middleware      []echo.MiddlewareFunc // Custom Echo middleware

	// TLS Configuration (for future use)
	enableTLS bool   // Enable TLS
	certFile  string // Path to certificate file
	keyFile   string // Path to private key file

	// OpenTelemetry Configuration (optional - nil disables telemetry)
	otelConfig *otel.Config // OpenTelemetry configuration for traces, metrics, and logs
}

// newConfig creates a new config with defaults and applies the provided options
func newConfig(opts ...Option) (*config, error) {
	// Start with sensible defaults
	cfg := &config{
		// Server Configuration
		grpcPort: "8080",
		httpPort: "8081",
		mode:     H2CMode, // Default to H2C for simpler development

		// Timeouts and Limits
		shutdownTimeout:       30 * time.Second,
		readTimeout:           5 * time.Second,
		writeTimeout:          10 * time.Second,
		idleTimeout:           60 * time.Second,
		maxConnectionIdle:     15 * time.Minute,
		maxConnectionAge:      30 * time.Minute,
		maxConnectionAgeGrace: 5 * time.Second,

		// Production Features
		enableMetrics:     true,
		metricsPath:       "/metrics",
		enableHealthCheck: true,
		healthPath:        "/health",
		enableLogging:     true,
		enableReflection:  true,

		// Gateway Configuration
		gatewayBasePath: "/api/v1",

		// Echo-specific Features
		enableCORS:      false, // Disabled by default, enable as needed
		enableRateLimit: false, // Disabled by default, enable as needed
		rateLimit:       100.0, // 100 requests per second default
		middleware:      []echo.MiddlewareFunc{},

		// TLS Configuration
		enableTLS: false,
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate ensures the configuration is valid
func (c *config) validate() error {
	if c.grpcPort == "" {
		return fmt.Errorf("gRPC port cannot be empty")
	}

	if c.mode == SeparateMode && c.httpPort == "" {
		return fmt.Errorf("HTTP port cannot be empty when using SeparateMode")
	}

	if c.mode != H2CMode && c.mode != SeparateMode {
		return fmt.Errorf("invalid server mode: %s", c.mode)
	}

	if c.shutdownTimeout < 0 {
		return fmt.Errorf("shutdown timeout cannot be negative")
	}

	if c.readTimeout < 0 {
		return fmt.Errorf("read timeout cannot be negative")
	}

	if c.writeTimeout < 0 {
		return fmt.Errorf("write timeout cannot be negative")
	}

	if c.idleTimeout < 0 {
		return fmt.Errorf("idle timeout cannot be negative")
	}

	return nil
}

// getGRPCAddress returns the full address for the gRPC server
func (c *config) getGRPCAddress() string {
	return ":" + c.grpcPort
}

// getHTTPAddress returns the full address for the HTTP server
func (c *config) getHTTPAddress() string {
	if c.mode == H2CMode {
		return c.getGRPCAddress() // Use same port for H2C
	}
	return ":" + c.httpPort
}

// isH2CMode returns true if server is running in H2C mode
func (c *config) isH2CMode() bool {
	return c.mode == H2CMode
}

// isSeparateMode returns true if server is running in separate mode
func (c *config) isSeparateMode() bool {
	return c.mode == SeparateMode
}

// ============================================================================
// Server Mode & Port Options
// ============================================================================

// WithH2CMode sets the server to H2C mode (gRPC and HTTP on same port)
func WithH2CMode() Option {
	return func(c *config) {
		c.mode = H2CMode
	}
}

// WithSeparateMode sets the server to separate mode with different ports for gRPC and HTTP
func WithSeparateMode(grpcPort, httpPort string) Option {
	return func(c *config) {
		c.mode = SeparateMode
		c.grpcPort = grpcPort
		c.httpPort = httpPort
	}
}

// WithGRPCPort sets the gRPC server port
func WithGRPCPort(port string) Option {
	return func(c *config) {
		c.grpcPort = port
	}
}

// WithHTTPPort sets the HTTP gateway port (only used in SeparateMode)
func WithHTTPPort(port string) Option {
	return func(c *config) {
		c.httpPort = port
	}
}

// ============================================================================
// Timeout Options
// ============================================================================

// WithShutdownTimeout sets the graceful shutdown timeout
func WithShutdownTimeout(d time.Duration) Option {
	return func(c *config) {
		c.shutdownTimeout = d
	}
}

// WithReadTimeout sets the HTTP server read timeout
func WithReadTimeout(d time.Duration) Option {
	return func(c *config) {
		c.readTimeout = d
	}
}

// WithWriteTimeout sets the HTTP server write timeout
func WithWriteTimeout(d time.Duration) Option {
	return func(c *config) {
		c.writeTimeout = d
	}
}

// WithIdleTimeout sets the HTTP server idle timeout
func WithIdleTimeout(d time.Duration) Option {
	return func(c *config) {
		c.idleTimeout = d
	}
}

// WithConnectionTimeouts sets all gRPC connection timeout values
func WithConnectionTimeouts(idle, age, grace time.Duration) Option {
	return func(c *config) {
		c.maxConnectionIdle = idle
		c.maxConnectionAge = age
		c.maxConnectionAgeGrace = grace
	}
}

// WithMaxConnectionIdle sets the maximum connection idle time for gRPC
func WithMaxConnectionIdle(d time.Duration) Option {
	return func(c *config) {
		c.maxConnectionIdle = d
	}
}

// WithMaxConnectionAge sets the maximum connection age for gRPC
func WithMaxConnectionAge(d time.Duration) Option {
	return func(c *config) {
		c.maxConnectionAge = d
	}
}

// WithMaxConnectionAgeGrace sets the connection age grace period for gRPC
func WithMaxConnectionAgeGrace(d time.Duration) Option {
	return func(c *config) {
		c.maxConnectionAgeGrace = d
	}
}

// ============================================================================
// Feature Toggle Options
// ============================================================================

// WithMetrics enables Prometheus metrics endpoint
func WithMetrics() Option {
	return func(c *config) {
		c.enableMetrics = true
	}
}

// WithoutMetrics disables Prometheus metrics endpoint
func WithoutMetrics() Option {
	return func(c *config) {
		c.enableMetrics = false
	}
}

// WithHealthCheck enables health check endpoints
func WithHealthCheck() Option {
	return func(c *config) {
		c.enableHealthCheck = true
	}
}

// WithoutHealthCheck disables health check endpoints
func WithoutHealthCheck() Option {
	return func(c *config) {
		c.enableHealthCheck = false
	}
}

// WithLogging enables request/response logging
func WithLogging() Option {
	return func(c *config) {
		c.enableLogging = true
	}
}

// WithoutLogging disables request/response logging
func WithoutLogging() Option {
	return func(c *config) {
		c.enableLogging = false
	}
}

// WithReflection enables gRPC server reflection
func WithReflection() Option {
	return func(c *config) {
		c.enableReflection = true
	}
}

// WithoutReflection disables gRPC server reflection
func WithoutReflection() Option {
	return func(c *config) {
		c.enableReflection = false
	}
}

// WithCORS enables CORS middleware
func WithCORS() Option {
	return func(c *config) {
		c.enableCORS = true
	}
}

// WithRateLimit enables rate limiting with the specified requests per second
func WithRateLimit(rps float64) Option {
	return func(c *config) {
		c.enableRateLimit = true
		c.rateLimit = rps
	}
}

// WithTLS enables TLS with the specified certificate and key files
func WithTLS(certFile, keyFile string) Option {
	return func(c *config) {
		c.enableTLS = true
		c.certFile = certFile
		c.keyFile = keyFile
	}
}

// ============================================================================
// Path Configuration Options
// ============================================================================

// WithMetricsPath sets the metrics endpoint path
func WithMetricsPath(path string) Option {
	return func(c *config) {
		c.metricsPath = path
	}
}

// WithHealthPath sets the health check base path
func WithHealthPath(path string) Option {
	return func(c *config) {
		c.healthPath = path
	}
}

// WithGatewayBasePath sets the base path for gRPC gateway routes
func WithGatewayBasePath(path string) Option {
	return func(c *config) {
		c.gatewayBasePath = path
	}
}

// ============================================================================
// Hook/Callback Options
// ============================================================================

// WithServiceRegistrar sets the function to register gRPC services
func WithServiceRegistrar(fn func(*grpc.Server)) Option {
	return func(c *config) {
		c.serviceRegistrar = fn
	}
}

// WithGRPCConfigurer sets the function to configure the gRPC server
func WithGRPCConfigurer(fn func(*grpc.Server)) Option {
	return func(c *config) {
		c.grpcConfigurer = fn
	}
}

// WithEchoConfigurer sets the function to configure the Echo HTTP server
func WithEchoConfigurer(fn func(*echo.Echo)) Option {
	return func(c *config) {
		c.echoConfigurer = fn
	}
}

// WithShutdownHandler sets a custom shutdown handler
func WithShutdownHandler(fn func() error) Option {
	return func(c *config) {
		c.shutdown = fn
	}
}

// ============================================================================
// Middleware Options
// ============================================================================

// WithMiddleware adds custom Echo middleware
func WithMiddleware(mw ...echo.MiddlewareFunc) Option {
	return func(c *config) {
		c.middleware = append(c.middleware, mw...)
	}
}

// ============================================================================
// OpenTelemetry Options
// ============================================================================

// WithOTelConfig sets the OpenTelemetry configuration for traces, metrics, and logs
// When set, the gRPC server will instrument with OTel instead of Prometheus
func WithOTelConfig(cfg *otel.Config) Option {
	return func(c *config) {
		c.otelConfig = cfg
	}
}
