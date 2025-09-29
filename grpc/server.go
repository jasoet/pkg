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
	config         Config
	grpcServer     *grpc.Server
	echo           *echo.Echo
	httpServer     *http.Server // Used only for H2C mode
	gatewayMux     *runtime.ServeMux
	healthManager  *HealthManager
	metricsManager *MetricsManager
	shutdownOnce   sync.Once
	running        bool
	mu             sync.RWMutex
}

// New creates a new server instance with the given configuration
func New(config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	server := &Server{
		config:         config,
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
	if s.config.MaxConnectionIdle > 0 {
		opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     s.config.MaxConnectionIdle,
			MaxConnectionAge:      s.config.MaxConnectionAge,
			MaxConnectionAgeGrace: s.config.MaxConnectionAgeGrace,
		}))
	}

	// Create gRPC server
	s.grpcServer = grpc.NewServer(opts...)

	// Enable reflection if configured
	if s.config.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	// Apply custom gRPC configuration
	if s.config.GRPCConfigurer != nil {
		s.config.GRPCConfigurer(s.grpcServer)
	}

	// Register services
	if s.config.ServiceRegistrar != nil {
		s.config.ServiceRegistrar(s.grpcServer)
	}
}

// setupEchoServer configures the Echo HTTP server
func (s *Server) setupEchoServer() error {
	e := echo.New()

	// Configure Echo basics
	e.HideBanner = true
	e.HidePort = true

	// Add built-in middleware
	if s.config.EnableLogging {
		e.Use(middleware.Logger())
	}
	e.Use(middleware.Recover())

	// Add metrics middleware
	if s.config.EnableMetrics {
		e.Use(s.metricsManager.EchoMetricsMiddleware())
		s.metricsManager.RegisterEchoMetrics(e, s.config.MetricsPath)
	}

	// Add health checks
	if s.config.EnableHealthCheck {
		s.healthManager.RegisterEchoHealthChecks(e, s.config.HealthPath)
	}

	// Add optional CORS middleware
	if s.config.EnableCORS {
		e.Use(middleware.CORS())
	}

	// Add optional rate limiting middleware
	if s.config.EnableRateLimit {
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(s.config.RateLimit))))
	}

	// Add custom middleware
	for _, mw := range s.config.Middleware {
		e.Use(mw)
	}

	// Setup gateway integration if service registrar is provided
	if s.config.ServiceRegistrar != nil {
		if err := s.setupGatewayIntegration(e); err != nil {
			return fmt.Errorf("failed to setup gateway integration: %w", err)
		}
	}

	// Apply custom Echo configuration
	if s.config.EchoConfigurer != nil {
		s.config.EchoConfigurer(e)
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
	MountGatewayOnEcho(e, gatewayMux, s.config.GatewayBasePath)

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
	if s.config.EnableMetrics {
		go s.trackUptime()
	}

	switch s.config.Mode {
	case SeparateMode:
		return s.startSeparateMode()
	case H2CMode:
		return s.startH2CMode()
	default:
		return fmt.Errorf("unsupported server mode: %s", s.config.Mode)
	}
}

// startSeparateMode starts gRPC and HTTP servers on separate ports
func (s *Server) startSeparateMode() error {
	// Start gRPC server
	grpcListener, err := net.Listen("tcp", s.config.GetGRPCAddress())
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", s.config.GRPCPort, err)
	}

	// Start gRPC server in goroutine
	go func() {
		log.Printf("gRPC server starting on port %s", s.config.GRPCPort)
		if s.config.EnableReflection {
			log.Printf("gRPC reflection enabled")
		}
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start Echo HTTP server
	log.Printf("Echo HTTP server starting on port %s", s.config.HTTPPort)
	if s.config.EnableHealthCheck {
		log.Printf("Health checks available at http://localhost:%s%s", s.config.HTTPPort, s.config.HealthPath)
	}
	if s.config.EnableMetrics {
		log.Printf("Metrics available at http://localhost:%s%s", s.config.HTTPPort, s.config.MetricsPath)
	}
	if s.config.ServiceRegistrar != nil {
		log.Printf("gRPC Gateway available at http://localhost:%s%s", s.config.HTTPPort, s.config.GatewayBasePath)
	}

	return s.echo.Start(s.config.GetHTTPAddress())
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
		Addr:         s.config.GetGRPCAddress(),
		Handler:      h2c.NewHandler(mixedHandler, &http2.Server{}),
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	log.Printf("Mixed gRPC+Echo server starting on port %s (H2C mode)", s.config.GRPCPort)
	log.Printf("gRPC endpoints available on port %s", s.config.GRPCPort)
	if s.config.EnableReflection {
		log.Printf("gRPC reflection enabled")
	}
	if s.config.EnableHealthCheck {
		log.Printf("Health checks available at http://localhost:%s%s", s.config.GRPCPort, s.config.HealthPath)
	}
	if s.config.EnableMetrics {
		log.Printf("Metrics available at http://localhost:%s%s", s.config.GRPCPort, s.config.MetricsPath)
	}
	if s.config.ServiceRegistrar != nil {
		log.Printf("gRPC Gateway available at http://localhost:%s%s", s.config.GRPCPort, s.config.GatewayBasePath)
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

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		// Run custom shutdown handler
		if s.config.Shutdown != nil {
			if err := s.config.Shutdown(); err != nil {
				log.Printf("Custom shutdown handler error: %v", err)
			}
		}

		// Stop HTTP/Echo server based on mode
		if s.config.Mode == H2CMode && s.httpServer != nil {
			// H2C mode uses httpServer
			if err := s.httpServer.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
				stopErr = err
			}
		} else if s.config.Mode == SeparateMode && s.echo != nil {
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

// trackUptime periodically updates the uptime metric
func (s *Server) trackUptime() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.metricsManager.UpdateUptime()
		}
	}
}

// StartWithConfig creates and starts a server with the given configuration
func StartWithConfig(config Config) error {
	server, err := New(config)
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

// Start creates and starts a server with default configuration and custom service registrar
func Start(port string, serviceRegistrar func(*grpc.Server)) error {
	config := DefaultConfig()
	config.GRPCPort = port
	config.ServiceRegistrar = serviceRegistrar

	return StartWithConfig(config)
}

// StartH2C creates and starts a server in H2C mode with custom service registrar
func StartH2C(port string, serviceRegistrar func(*grpc.Server)) error {
	config := DefaultConfig()
	config.GRPCPort = port
	config.Mode = H2CMode
	config.ServiceRegistrar = serviceRegistrar

	return StartWithConfig(config)
}

// StartSeparate creates and starts a server in separate mode with custom service registrar
func StartSeparate(grpcPort, httpPort string, serviceRegistrar func(*grpc.Server)) error {
	config := DefaultConfig()
	config.GRPCPort = grpcPort
	config.HTTPPort = httpPort
	config.Mode = SeparateMode
	config.ServiceRegistrar = serviceRegistrar

	return StartWithConfig(config)
}
