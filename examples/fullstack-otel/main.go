//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jasoet/fullstack-otel-example/proto"
	"github.com/jasoet/pkg/v2/db"
	grpcserver "github.com/jasoet/pkg/v2/grpc"
	"github.com/jasoet/pkg/v2/logging"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/jasoet/pkg/v2/rest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// User model for database
type User struct {
	ID        string    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// UserServiceImpl implements the User gRPC service
type UserServiceImpl struct {
	user.UnimplementedUserServiceServer
	db         *gorm.DB
	restClient *rest.Client
	otelCfg    *otel.Config
}

// GetUser retrieves a user by ID with full OTel instrumentation
func (s *UserServiceImpl) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	// Create custom span for business logic
	tracer := s.otelCfg.GetTracer("user-service")
	ctx, span := tracer.Start(ctx, "GetUser.BusinessLogic")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.Id),
	)

	// Database query (automatically traced)
	var dbUser User
	if err := s.db.WithContext(ctx).First(&dbUser, "id = ?", req.Id).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Example: Make REST call to external service (trace propagation)
	// This demonstrates how trace context is propagated to downstream services
	if s.restClient != nil {
		_, _ = s.restClient.MakeRequestWithTrace(
			ctx,
			"GET",
			fmt.Sprintf("https://api.example.com/user-details/%s", req.Id),
			"",
			nil,
		)
		// Note: This will fail in real scenarios but demonstrates trace propagation
	}

	return &user.GetUserResponse{
		User: &user.User{
			Id:        dbUser.ID,
			Name:      dbUser.Name,
			Email:     dbUser.Email,
			CreatedAt: dbUser.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// ListUsers lists all users with pagination
func (s *UserServiceImpl) ListUsers(ctx context.Context, req *user.ListUsersRequest) (*user.ListUsersResponse, error) {
	tracer := s.otelCfg.GetTracer("user-service")
	ctx, span := tracer.Start(ctx, "ListUsers.BusinessLogic")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)

	// Database query with pagination
	var users []User
	var total int64

	offset := int(req.Page * req.PageSize)
	limit := int(req.PageSize)

	s.db.WithContext(ctx).Model(&User{}).Count(&total)
	s.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users)

	protoUsers := make([]*user.User, len(users))
	for i, u := range users {
		protoUsers[i] = &user.User{
			Id:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}

	return &user.ListUsersResponse{
		Users: protoUsers,
		Total: int32(total),
	}, nil
}

func main() {
	ctx := context.Background()

	// =========================================================================
	// Step 1: Setup OpenTelemetry Providers
	// =========================================================================

	// Resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("fullstack-example"),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("deployment.environment", "development"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// TracerProvider with OTLP exporter (optional - comment out if no collector)
	// Uncomment when you have an OTel collector or Jaeger running
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Printf("Warning: OTLP exporter failed (no collector running?): %v", err)
		log.Println("Continuing with NoOp tracer provider...")
	}

	var tracerProvider *trace.TracerProvider
	if traceExporter != nil {
		tracerProvider = trace.NewTracerProvider(
			trace.WithBatcher(traceExporter),
			trace.WithResource(res),
			trace.WithSampler(trace.AlwaysSample()),
		)
	}

	// MeterProvider for metrics
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
	)

	// LoggerProvider with zerolog backend (automatic trace correlation)
	loggerProvider := logging.NewLoggerProvider("fullstack-example", true)

	// Create OTel config
	otelCfg := &otel.Config{
		ServiceName:    "fullstack-example",
		ServiceVersion: "1.0.0",
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		LoggerProvider: loggerProvider,
	}

	log.Println("✓ OpenTelemetry providers configured")

	// =========================================================================
	// Step 2: Setup Database with OTel
	// =========================================================================

	// Database configuration using PostgreSQL from docker-compose
	// Use SQLite in-memory if PostgreSQL is not available
	dbConfig := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         "localhost",
		Port:         5432,
		Username:     "user",
		Password:     "password",
		DbName:       "testdb",
		Timeout:      30 * time.Second,
		MaxIdleConns: 2,
		MaxOpenConns: 5,
		OTelConfig:   otelCfg,
	}

	database, err := dbConfig.Pool()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := database.AutoMigrate(&User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed some data
	users := []User{
		{ID: "1", Name: "Alice Johnson", Email: "alice@example.com"},
		{ID: "2", Name: "Bob Smith", Email: "bob@example.com"},
		{ID: "3", Name: "Charlie Brown", Email: "charlie@example.com"},
	}
	for _, u := range users {
		database.Create(&u)
	}

	log.Println("✓ Database connected with OTel instrumentation")
	log.Println("✓ Seeded 3 sample users")

	// =========================================================================
	// Step 3: Setup REST Client with OTel
	// =========================================================================

	restConfig := rest.Config{
		RetryCount:       2,
		RetryWaitTime:    5 * time.Second,
		RetryMaxWaitTime: 15 * time.Second,
		Timeout:          30 * time.Second,
	}

	restClient := rest.NewClient(
		rest.WithRestConfig(restConfig),
		rest.WithOTelConfig(otelCfg),
	)
	log.Println("✓ REST client configured with OTel instrumentation")

	// =========================================================================
	// Step 4: Setup gRPC Server with OTel
	// =========================================================================

	server, err := grpcserver.New(
		grpcserver.WithGRPCPort("50051"),
		grpcserver.WithOTelConfig(otelCfg),
		grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
			// Register User service
			userService := &UserServiceImpl{
				db:         database,
				restClient: restClient,
				otelCfg:    otelCfg,
			}
			user.RegisterUserServiceServer(s, userService)
			log.Println("✓ UserService registered")
		}),
		grpcserver.WithShutdownHandler(func() error {
			log.Println("Shutting down OTel providers...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if tracerProvider != nil {
				if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
					log.Printf("Error shutting down tracer provider: %v", err)
				}
			}
			if err := meterProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down meter provider: %v", err)
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("\n" + strings.Repeat("=", 80))
	log.Println("Full-Stack OpenTelemetry Example Server")
	log.Println(strings.Repeat("=", 80))
	log.Println("\n📡 Server Endpoints:")
	log.Println("  gRPC:          localhost:50051")
	log.Println("  HTTP Gateway:  http://localhost:50051")
	log.Println("\n🔍 Observability:")
	if traceExporter != nil {
		log.Println("  Jaeger UI:     http://localhost:16686 (if running)")
		log.Println("  OTLP Endpoint: localhost:4318")
	} else {
		log.Println("  No OTLP collector configured (traces to stdout)")
	}
	log.Println("\n✨ Instrumented Components:")
	log.Println("  ✓ Distributed tracing (gRPC, HTTP, DB, REST client)")
	log.Println("  ✓ Metrics collection (request count, duration, DB pool)")
	log.Println("  ✓ Structured logs with trace correlation")
	log.Println("\n📝 Try these commands:")
	log.Println("  # Get user via HTTP")
	log.Println("  curl http://localhost:50051/api/v1/users/1")
	log.Println("\n  # List users via HTTP")
	log.Println("  curl 'http://localhost:50051/api/v1/users?page=0&page_size=10'")
	log.Println("\n  # Get user via gRPC")
	log.Println("  grpcurl -plaintext -d '{\"id\":\"1\"}' localhost:50051 user.UserService/GetUser")
	log.Println("\n" + strings.Repeat("=", 80) + "\n")

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
