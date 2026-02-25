package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server and gateway
type Server struct {
	config         *config
	grpcServer     *grpc.Server
	echo           *echo.Echo
	httpServer     *http.Server // Used only for H2C mode
	gatewayMux     *runtime.ServeMux
	healthManager  *HealthManager
	metricsManager *MetricsManager
	shutdownOnce   sync.Once
	running        bool
	mu             sync.RWMutex
	stopUptime     chan struct{} // signals trackUptime goroutine to exit
}

// New creates a new server instance with the given options
func New(opts ...Option) (*Server, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	server := &Server{
		config:         cfg,
		healthManager:  NewHealthManager(),
		metricsManager: NewMetricsManager("grpc_server"),
	}

	// Setup gRPC server
	server.setupGRPCServer()

	// Register default health checks
	for name, checker := range DefaultHealthCheckers() {
		server.healthManager.RegisterCheck(name, checker)
	}

	return server, nil
}

// setupGRPCServer configures the gRPC server with options
func (s *Server) setupGRPCServer() {
	var opts []grpc.ServerOption

	// Add connection timeout options
	if s.config.maxConnectionIdle > 0 {
		opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     s.config.maxConnectionIdle,
			MaxConnectionAge:      s.config.maxConnectionAge,
			MaxConnectionAgeGrace: s.config.maxConnectionAgeGrace,
		}))
	}

	// Add OpenTelemetry interceptors if configured
	if s.config.otelConfig != nil {
		// Chain interceptors: logging -> tracing -> metrics -> handler
		interceptors := []grpc.UnaryServerInterceptor{
			createGRPCLoggingInterceptor(s.config.otelConfig),
			createGRPCTracingInterceptor(s.config.otelConfig),
			createGRPCMetricsInterceptor(s.config.otelConfig),
		}

		// Create chain interceptor
		opts = append(opts, grpc.ChainUnaryInterceptor(interceptors...))
	}

	// Create gRPC server
	s.grpcServer = grpc.NewServer(opts...)

	// Enable reflection if configured
	if s.config.enableReflection {
		reflection.Register(s.grpcServer)
	}

	// Apply custom gRPC configuration
	if s.config.grpcConfigurer != nil {
		s.config.grpcConfigurer(s.grpcServer)
	}

	// Register services
	if s.config.serviceRegistrar != nil {
		s.config.serviceRegistrar(s.grpcServer)
	}
}

// setupEchoServer configures the Echo HTTP server
func (s *Server) setupEchoServer() error {
	e := echo.New()

	// Configure Echo basics
	e.HideBanner = true
	e.HidePort = true

	// Add OpenTelemetry middleware if configured
	if s.config.otelConfig != nil {
		// Add OTel middleware: logging -> tracing -> metrics
		if s.config.otelConfig.IsLoggingEnabled() {
			e.Use(createHTTPGatewayLoggingMiddleware(s.config.otelConfig))
		}
		if s.config.otelConfig.IsTracingEnabled() {
			e.Use(createHTTPGatewayTracingMiddleware(s.config.otelConfig))
		}
		if s.config.otelConfig.IsMetricsEnabled() {
			e.Use(createHTTPGatewayMetricsMiddleware(s.config.otelConfig))
		}
	} else {
		// Fallback to traditional logging and metrics (backwards compatibility)
		if s.config.enableLogging {
			e.Use(middleware.Logger())
		}
		if s.config.enableMetrics {
			e.Use(s.metricsManager.EchoMetricsMiddleware())
			s.metricsManager.RegisterEchoMetrics(e, s.config.metricsPath)
		}
	}

	e.Use(middleware.Recover())

	// Add health checks
	if s.config.enableHealthCheck {
		s.healthManager.RegisterEchoHealthChecks(e, s.config.healthPath)
	}

	// Add optional CORS middleware
	if s.config.enableCORS {
		e.Use(middleware.CORS())
	}

	// Add optional rate limiting middleware
	if s.config.enableRateLimit {
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(s.config.rateLimit))))
	}

	// Add custom middleware
	for _, mw := range s.config.middleware {
		e.Use(mw)
	}

	// Setup gateway integration if service registrar is provided
	if s.config.serviceRegistrar != nil {
		if err := s.setupGatewayIntegration(e); err != nil {
			return fmt.Errorf("failed to setup gateway integration: %w", err)
		}
	}

	// Apply custom Echo configuration
	if s.config.echoConfigurer != nil {
		s.config.echoConfigurer(e)
	}

	// Store Echo instance
	s.echo = e

	return nil
}

// setupGatewayIntegration configures gRPC gateway integration with Echo
func (s *Server) setupGatewayIntegration(e *echo.Echo) error {
	// Create gateway mux with standard configuration
	gatewayMux := CreateGatewayMux()

	// Mount gateway on Echo at the configured base path
	MountGatewayOnEcho(e, gatewayMux, s.config.gatewayBasePath)

	// Store gateway mux for service registration
	s.gatewayMux = gatewayMux

	return nil
}

// Start starts the server with the configured mode
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	if err := s.setupEchoServer(); err != nil {
		return fmt.Errorf("failed to setup Echo server: %w", err)
	}

	// Start metrics uptime tracking
	if s.config.enableMetrics {
		s.stopUptime = make(chan struct{})
		go s.trackUptime()
	}

	switch s.config.mode {
	case SeparateMode:
		return s.startSeparateMode()
	case H2CMode:
		return s.startH2CMode()
	default:
		return fmt.Errorf("unsupported server mode: %s", s.config.mode)
	}
}

// startSeparateMode starts gRPC and HTTP servers on separate ports
func (s *Server) startSeparateMode() error {
	// Start gRPC server
	grpcListener, err := net.Listen("tcp", s.config.getGRPCAddress())
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", s.config.grpcPort, err)
	}

	// Start gRPC server in goroutine
	go func() {
		log.Printf("gRPC server starting on port %s", s.config.grpcPort)
		if s.config.enableReflection {
			log.Printf("gRPC reflection enabled")
		}
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start Echo HTTP server
	log.Printf("Echo HTTP server starting on port %s", s.config.httpPort)
	if s.config.enableHealthCheck {
		log.Printf("Health checks available at http://localhost:%s%s", s.config.httpPort, s.config.healthPath)
	}
	if s.config.enableMetrics {
		log.Printf("Metrics available at http://localhost:%s%s", s.config.httpPort, s.config.metricsPath)
	}
	if s.config.serviceRegistrar != nil {
		log.Printf("gRPC Gateway available at http://localhost:%s%s", s.config.httpPort, s.config.gatewayBasePath)
	}

	return s.echo.Start(s.config.getHTTPAddress())
}

// startH2CMode starts a mixed gRPC/HTTP server on a single port
func (s *Server) startH2CMode() error {
	// Create mixed handler for H2C that routes between gRPC and Echo
	mixedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
		} else {
			s.echo.ServeHTTP(w, r) // Echo implements http.Handler
		}
	})

	// Create HTTP server with H2C support
	s.httpServer = &http.Server{
		Addr:         s.config.getGRPCAddress(),
		Handler:      h2c.NewHandler(mixedHandler, &http2.Server{}),
		ReadTimeout:  s.config.readTimeout,
		WriteTimeout: s.config.writeTimeout,
		IdleTimeout:  s.config.idleTimeout,
	}

	log.Printf("Mixed gRPC+Echo server starting on port %s (H2C mode)", s.config.grpcPort)
	log.Printf("gRPC endpoints available on port %s", s.config.grpcPort)
	if s.config.enableReflection {
		log.Printf("gRPC reflection enabled")
	}
	if s.config.enableHealthCheck {
		log.Printf("Health checks available at http://localhost:%s%s", s.config.grpcPort, s.config.healthPath)
	}
	if s.config.enableMetrics {
		log.Printf("Metrics available at http://localhost:%s%s", s.config.grpcPort, s.config.metricsPath)
	}
	if s.config.serviceRegistrar != nil {
		log.Printf("gRPC Gateway available at http://localhost:%s%s", s.config.grpcPort, s.config.gatewayBasePath)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	var stopErr error
	s.shutdownOnce.Do(func() {
		log.Println("Stopping server gracefully...")

		// Stop uptime goroutine
		if s.stopUptime != nil {
			close(s.stopUptime)
		}

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), s.config.shutdownTimeout)
		defer cancel()

		// Run custom shutdown handler
		if s.config.shutdown != nil {
			if err := s.config.shutdown(); err != nil {
				log.Printf("Custom shutdown handler error: %v", err)
			}
		}

		// Stop HTTP/Echo server based on mode
		if s.config.mode == H2CMode && s.httpServer != nil {
			// H2C mode uses httpServer
			if err := s.httpServer.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
				stopErr = err
			}
		} else if s.config.mode == SeparateMode && s.echo != nil {
			// Separate mode uses Echo
			if err := s.echo.Shutdown(ctx); err != nil {
				log.Printf("Echo server shutdown error: %v", err)
				stopErr = err
			}
		}

		// Stop gRPC server
		if s.grpcServer != nil {
			done := make(chan struct{})
			go func() {
				s.grpcServer.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
				log.Println("gRPC server stopped gracefully")
			case <-ctx.Done():
				log.Println("gRPC server shutdown timeout, forcing stop")
				s.grpcServer.Stop()
			}
		}

		s.mu.Lock()
		s.running = false
		s.mu.Unlock()

		log.Println("Server stopped")
	})

	return stopErr
}

// GetHealthManager returns the health manager
func (s *Server) GetHealthManager() *HealthManager {
	return s.healthManager
}

// GetMetricsManager returns the metrics manager
func (s *Server) GetMetricsManager() *MetricsManager {
	return s.metricsManager
}

// GetGRPCServer returns the underlying gRPC server
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// trackUptime periodically updates the uptime metric until stopUptime is closed.
func (s *Server) trackUptime() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.metricsManager.UpdateUptime()
		case <-s.stopUptime:
			return
		}
	}
}

// Start creates and starts a server with the given options
func Start(port string, serviceRegistrar func(*grpc.Server), opts ...Option) error {
	// Prepend required options
	allOpts := append([]Option{
		WithGRPCPort(port),
		WithServiceRegistrar(serviceRegistrar),
	}, opts...)

	server, err := New(allOpts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	return server.Start()
}

// StartH2C creates and starts a server in H2C mode with custom service registrar.
func StartH2C(port string, serviceRegistrar func(*grpc.Server), opts ...Option) error {
	// Prepend required options; we create the server directly to avoid
	// Start() prepending the same port/registrar options a second time.
	allOpts := append([]Option{
		WithH2CMode(),
		WithGRPCPort(port),
		WithServiceRegistrar(serviceRegistrar),
	}, opts...)

	server, err := New(allOpts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	return server.Start()
}

// StartSeparate creates and starts a server in separate mode with custom service registrar
func StartSeparate(grpcPort, httpPort string, serviceRegistrar func(*grpc.Server), opts ...Option) error {
	// Prepend required options
	allOpts := append([]Option{
		WithSeparateMode(grpcPort, httpPort),
		WithServiceRegistrar(serviceRegistrar),
	}, opts...)

	server, err := New(allOpts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	return server.Start()
}
