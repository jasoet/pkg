package grpc

import (
	"context"
	"errors"
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
	otellog "go.opentelemetry.io/otel/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server and gateway
type Server struct {
	config        *config
	grpcServer    *grpc.Server
	echo          *echo.Echo
	httpServer    *http.Server // Used only for H2C mode
	gatewayMux    *runtime.ServeMux
	healthManager *HealthManager
	shutdownOnce  *sync.Once
	metricsOnce   sync.Once // guards registerServerMetrics so restarts don't duplicate observable gauges
	running       bool
	starting      bool // true while Start is in flight, before all handles are published
	startCond     *sync.Cond
	mu            sync.RWMutex
}

// New creates a new server instance with the given options
func New(opts ...Option) (*Server, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	server := &Server{
		config:        cfg,
		healthManager: NewHealthManager(),
		shutdownOnce:  &sync.Once{},
	}
	server.startCond = sync.NewCond(&server.mu)

	// Setup gRPC server
	server.setupGRPCServer()

	// Register default health checks
	for name, checker := range DefaultHealthCheckers() {
		server.healthManager.RegisterCheck(name, checker)
	}

	return server, nil
}

// logInfo emits an info-level message using the OTel logger when OTelConfig is
// set, otherwise falls back to the standard log package.
func (s *Server) logInfo(msg string) {
	if s.config.otelConfig != nil && s.config.otelConfig.IsLoggingEnabled() {
		logger := s.config.otelConfig.GetLogger("grpc.server")
		var rec otellog.Record
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetBody(otellog.StringValue(msg))
		logger.Emit(context.Background(), rec)
		return
	}
	log.Printf("%s", msg)
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
		// Chain unary interceptors: tracing -> logging -> metrics -> handler.
		// Tracing MUST run first (outermost) so it establishes the span in the
		// context before logging runs; otherwise the access log emitted by the
		// logging interceptor carries no trace_id/span_id (broken correlation).
		unaryInterceptors := []grpc.UnaryServerInterceptor{
			createGRPCTracingInterceptor(s.config.otelConfig),
			createGRPCLoggingInterceptor(s.config.otelConfig),
			createGRPCMetricsInterceptor(s.config.otelConfig),
		}
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))

		// Chain stream interceptors: tracing -> logging -> metrics -> handler.
		// Same ordering rationale as the unary chain.
		streamInterceptors := []grpc.StreamServerInterceptor{
			createGRPCStreamTracingInterceptor(s.config.otelConfig),
			createGRPCStreamLoggingInterceptor(s.config.otelConfig),
			createGRPCStreamMetricsInterceptor(s.config.otelConfig),
		}
		opts = append(opts, grpc.ChainStreamInterceptor(streamInterceptors...))

		// Register server uptime/start_time observable gauges exactly once per
		// Server: setupGRPCServer runs again on every restart, and re-registering
		// the same observable gauges on a shared meter provider would leave
		// duplicate callbacks producing conflicting values.
		s.metricsOnce.Do(func() {
			registerServerMetrics(s.config.otelConfig)
		})
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

	// Add OpenTelemetry middleware if configured.
	// Order matters: tracing is registered BEFORE logging so it runs first
	// (outermost) and installs the span into the request context; the logging
	// middleware then re-reads that context after next() so access logs carry
	// trace_id/span_id. Registering logging first would leave access logs
	// uncorrelated.
	if s.config.otelConfig != nil {
		// Add OTel middleware: tracing -> logging -> metrics
		if s.config.otelConfig.IsTracingEnabled() {
			e.Use(createHTTPGatewayTracingMiddleware(s.config.otelConfig))
		}
		if s.config.otelConfig.IsLoggingEnabled() {
			e.Use(createHTTPGatewayLoggingMiddleware(s.config.otelConfig))
		}
		if s.config.otelConfig.IsMetricsEnabled() {
			e.Use(createHTTPGatewayMetricsMiddleware(s.config.otelConfig))
		}
	}

	e.Use(middleware.Recover())

	// Add health checks
	if s.config.enableHealthCheck {
		s.healthManager.RegisterEchoHealthChecks(e, s.config.healthPath)
	}

	// Add optional CORS middleware
	if s.config.enableCORS {
		if s.config.corsConfig != nil {
			e.Use(middleware.CORSWithConfig(*s.config.corsConfig))
		} else {
			e.Use(middleware.CORS())
		}
	}

	// Add optional rate limiting middleware
	if s.config.enableRateLimit {
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(s.config.rateLimit))))
	}

	// Add custom middleware
	for _, mw := range s.config.middleware {
		e.Use(mw)
	}

	// Setup gateway integration if a service or gateway registrar is provided.
	// NOTE: the gateway is mounted as a wildcard catch-all (basePath + "/*"),
	// so Echo's route priority — static routes win over wildcards — lets
	// user-supplied routes from echoConfigurer take precedence over the
	// auto-generated gateway routes regardless of registration order.
	if s.config.serviceRegistrar != nil || s.config.gatewayRegistrar != nil {
		if err := s.setupGatewayIntegration(e); err != nil {
			return fmt.Errorf("failed to setup gateway integration: %w", err)
		}
	}

	// Apply custom Echo configuration (routes registered here override gateway routes).
	if s.config.echoConfigurer != nil {
		s.config.echoConfigurer(e)
	}

	// Store Echo instance under the lock: the H2C mixed handler reads s.echo
	// concurrently (see startH2CMode), and a restart rewrites it.
	s.mu.Lock()
	s.echo = e
	s.mu.Unlock()

	return nil
}

// setupGatewayIntegration configures gRPC gateway integration with Echo
func (s *Server) setupGatewayIntegration(e *echo.Echo) error {
	// Create gateway mux with standard configuration
	gatewayMux := CreateGatewayMux()

	// Let the consumer register generated gateway handlers before mounting;
	// without this the mounted gateway serves nothing.
	if s.config.gatewayRegistrar != nil {
		s.config.gatewayRegistrar(gatewayMux)
	}

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
	// Mark running inside the same critical section as the check so two
	// concurrent Start calls cannot both pass the guard; every error path
	// below rolls this back to false. Also mark starting so a concurrent
	// Stop blocks (instead of tearing down a half-published server) until
	// startup either completes or is rolled back.
	s.running = true
	s.starting = true
	// On restart after a completed Stop the previous gRPC server is spent
	// (Serve returns grpc.ErrServerStopped after GracefulStop) and
	// shutdownOnce has been consumed; rebuild both so Start/Stop cycles work.
	if s.grpcServer == nil {
		s.setupGRPCServer()
		// Re-arm shutdown with a fresh Once. Use a new pointer rather than
		// resetting the existing value: a slow Stop from the previous cycle may
		// still be unwinding its shutdownOnce.Do call, and mutating that Once
		// concurrently would be a data race. Stop captures the pointer under the
		// lock before calling Do, so it always operates on a stable Once.
		s.shutdownOnce = &sync.Once{}
	}
	s.mu.Unlock()

	if err := s.setupEchoServer(); err != nil {
		s.mu.Lock()
		s.running = false
		s.starting = false
		s.mu.Unlock()
		s.startCond.Broadcast()
		return fmt.Errorf("failed to setup Echo server: %w", err)
	}

	var err error
	switch s.config.mode {
	case SeparateMode:
		err = s.startSeparateMode()
	case H2CMode:
		err = s.startH2CMode()
	default:
		err = fmt.Errorf("unsupported server mode: %s", s.config.mode)
	}

	if err != nil {
		// Roll back: the server is not running, and the gRPC server may have
		// been started (SeparateMode) or be in an unknown state — stop it and
		// mark it spent so a subsequent Start rebuilds it.
		s.mu.Lock()
		s.running = false
		s.starting = false
		if s.grpcServer != nil {
			s.grpcServer.Stop()
			s.grpcServer = nil
		}
		s.mu.Unlock()
		s.startCond.Broadcast()
		return err
	}

	return nil
}

// endStartup closes the startup window and wakes any Stop callers blocked
// waiting for startup to settle. It must be called after every handle Stop
// needs (s.echo, s.httpServer) has been published and before Start blocks in
// the serve loop. The lock round-trip publishes those handles to the woken
// Stop.
func (s *Server) endStartup() {
	s.mu.Lock()
	s.starting = false
	s.mu.Unlock()
	s.startCond.Broadcast()
}

// startSeparateMode starts gRPC and HTTP servers on separate ports
func (s *Server) startSeparateMode() error {
	// Start gRPC server
	grpcListener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", s.config.getGRPCAddress())
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", s.config.grpcPort, err)
	}

	// Ownership of grpcListener transfers to grpcServer.Serve below, which
	// closes it when it exits (on GracefulStop/Stop). We deliberately do NOT
	// defer Close() here: this function only returns after Echo's serve loop
	// ends, by which point Stop has already closed the listener via Serve, and
	// a second Close would race that path and log a spurious "use of closed
	// network connection". On the rollback path (busy HTTP port) Start's error
	// handling calls grpcServer.Stop(), which closes the listener.

	// Capture the current gRPC server into a local before launching the
	// goroutine: Stop/rollback may nil the field concurrently, and reading it
	// unsynchronized inside the goroutine could panic on a nil dereference.
	s.mu.RLock()
	grpcServer := s.grpcServer
	s.mu.RUnlock()

	// Start gRPC server in goroutine; it now owns the listener.
	go func() {
		s.logInfo(fmt.Sprintf("gRPC server starting on port %s", s.config.grpcPort))
		if s.config.enableReflection {
			s.logInfo("gRPC reflection enabled")
		}
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// All handles a concurrent Stop needs are published (s.echo during
	// setupEchoServer, the gRPC server above); close the startup window
	// before blocking in Echo's serve loop.
	s.endStartup()

	// Start Echo HTTP server
	s.logInfo(fmt.Sprintf("Echo HTTP server starting on port %s", s.config.httpPort))
	if s.config.enableHealthCheck {
		s.logInfo(fmt.Sprintf("Health checks available at http://localhost:%s%s", s.config.httpPort, s.config.healthPath))
	}
	if s.config.serviceRegistrar != nil || s.config.gatewayRegistrar != nil {
		s.logInfo(fmt.Sprintf("gRPC Gateway available at http://localhost:%s%s", s.config.httpPort, s.config.gatewayBasePath))
	}

	if err := s.echo.Start(s.config.getHTTPAddress()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// startH2CMode starts a mixed gRPC/HTTP server on a single port
func (s *Server) startH2CMode() error {
	// Create mixed handler for H2C that routes between gRPC and Echo
	mixedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			// Read the current field per request under the lock: Stop nils it
			// and restart rebuilds it, so the handler must follow the field,
			// not a value captured when the handler was created.
			s.mu.RLock()
			grpcServer := s.grpcServer
			s.mu.RUnlock()
			if grpcServer == nil {
				// Server is stopped; reject rather than panic on nil.
				http.Error(w, "gRPC server is not running", http.StatusServiceUnavailable)
				return
			}
			grpcServer.ServeHTTP(w, r)
		} else {
			// Read s.echo under the lock: a restart rewrites it and stale
			// hijacked-connection closures may still invoke this handler.
			s.mu.RLock()
			e := s.echo
			s.mu.RUnlock()
			e.ServeHTTP(w, r) // Echo implements http.Handler
		}
	})

	// Create HTTP server with H2C support.
	//
	// CRITICAL: ReadTimeout and WriteTimeout are left at zero here. Under h2c
	// the *http.Server's Read/WriteTimeout become per-stream HTTP/2 deadlines
	// (via http2.Server's BaseConfig), and grpc-go's serverHandlerTransport
	// never clears them. A non-zero WriteTimeout would abort any unary RPC that
	// takes longer than it and would kill client/bidi streams that send after
	// the deadline (RST_STREAM, surfaced to the client as Internal). Because
	// gRPC and plain HTTP share this port in H2C mode, we cannot safely apply a
	// connection-level write deadline; enforce HTTP read/write timeouts with
	// SeparateMode or an upstream proxy instead. ReadHeaderTimeout and
	// IdleTimeout remain safe and are kept for slowloris/idle protection.
	s.httpServer = &http.Server{
		Addr:              s.config.getGRPCAddress(),
		Handler:           h2c.NewHandler(mixedHandler, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       s.config.idleTimeout,
	}

	// s.echo and s.httpServer are now published; close the startup window
	// before blocking in ListenAndServe so a concurrent Stop can proceed.
	s.endStartup()

	s.logInfo(fmt.Sprintf("Mixed gRPC+Echo server starting on port %s (H2C mode)", s.config.grpcPort))
	s.logInfo(fmt.Sprintf("gRPC endpoints available on port %s", s.config.grpcPort))
	if s.config.enableReflection {
		s.logInfo("gRPC reflection enabled")
	}
	if s.config.enableHealthCheck {
		s.logInfo(fmt.Sprintf("Health checks available at http://localhost:%s%s", s.config.grpcPort, s.config.healthPath))
	}
	if s.config.serviceRegistrar != nil || s.config.gatewayRegistrar != nil {
		s.logInfo(fmt.Sprintf("gRPC Gateway available at http://localhost:%s%s", s.config.grpcPort, s.config.gatewayBasePath))
	}

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop gracefully stops the server. If a Start call is currently in flight,
// Stop blocks until startup has completed (or been rolled back) before
// proceeding, so it never tears down a half-published server and never
// leaves an unstoppable zombie behind.
func (s *Server) Stop() error {
	s.mu.Lock()
	for s.starting {
		// Start is between the running check and publishing all handles
		// (s.echo, s.httpServer); wait for endStartup to close that window.
		// The lock round-trip in endStartup also publishes those handles.
		s.startCond.Wait()
	}
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	// Capture the current Once under the lock so a concurrent restart, which
	// swaps in a fresh Once, cannot race the Do call below.
	once := s.shutdownOnce
	s.mu.Unlock()

	var stopErr error
	once.Do(func() {
		log.Println("Stopping server gracefully...")

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
			// H2C mode uses httpServer. Note: h2c hijacks the underlying
			// net.Conn to serve HTTP/2, so http.Server.Shutdown does not track
			// or drain those connections (it returns without waiting on them).
			// The graceful drain of in-flight gRPC calls and the GOAWAY to gRPC
			// clients are handled by grpcServer.GracefulStop below instead.
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
		s.mu.RLock()
		grpcServer := s.grpcServer
		s.mu.RUnlock()
		if grpcServer != nil {
			done := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
				log.Println("gRPC server stopped gracefully")
			case <-ctx.Done():
				log.Println("gRPC server shutdown timeout, forcing stop")
				grpcServer.Stop()
				// Graceful shutdown did not complete within shutdownTimeout and
				// connections were force-closed. Surface this as an error so
				// callers do not mistake a forced kill for a clean shutdown.
				stopErr = fmt.Errorf("graceful shutdown timed out after %s, forced stop: %w",
					s.config.shutdownTimeout, ctx.Err())
			}
		}

		s.mu.Lock()
		s.running = false
		// The gRPC server cannot be reused after GracefulStop/Stop; clear it
		// so a subsequent Start rebuilds it via setupGRPCServer.
		s.grpcServer = nil
		s.mu.Unlock()

		log.Println("Server stopped")
	})

	return stopErr
}

// GetHealthManager returns the health manager
func (s *Server) GetHealthManager() *HealthManager {
	return s.healthManager
}

// GetGRPCServer returns the underlying gRPC server. It returns nil after Stop
// until the next Start rebuilds the server.
func (s *Server) GetGRPCServer() *grpc.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.grpcServer
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
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

	return startWithSignalHandling(server)
}

// startWithSignalHandling installs SIGINT/SIGTERM handling for graceful
// shutdown, then blocks in server.Start until it returns. It cleans up after
// itself: signal.Stop unregisters the handler and the done channel terminates
// the watcher goroutine, so no goroutine or signal registration leaks per call
// (which would otherwise swallow a subsequent SIGTERM, leaving the process
// killable only via SIGKILL).
func startWithSignalHandling(server *Server) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case sig := <-sigChan:
			log.Printf("Received signal: %v", sig)
			if err := server.Stop(); err != nil {
				log.Printf("Error stopping server: %v", err)
			}
		case <-done:
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

	return startWithSignalHandling(server)
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

	return startWithSignalHandling(server)
}
