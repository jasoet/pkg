package otel_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
)

// Example_withoutOTelConfig demonstrates span creation without OTel configuration.
// Logger is always available with zerolog fallback when no config is in context.
func Example_withoutOTelConfig() {
	ctx := context.Background()

	// No config in context - spans work, logger uses zerolog fallback
	// Fields passed here are automatically included in all log calls
	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	// Logger is always available (zerolog fallback without OTel config)
	// All logs automatically include: layer="service", user.id="12345"
	lc.Logger.Info("Creating user")

	// Span tracking still works
	lc.Span.AddAttribute("status", "processing")

	// Error handling works (recorded in span and log)
	err := errors.New("validation failed")
	if err != nil {
		_ = lc.Error(err, "User creation failed")
		return
	}

	lc.Success("User created successfully")

	fmt.Println("Spans and logging work without OTel config")
}

// Example_withOTelConfig demonstrates full OTel integration with tracing and structured logging.
func Example_withOTelConfig() {
	// Create OTel config with default logger (zerolog-based with OTel integration)
	cfg := otel.NewConfig("my-service")

	// Add TracerProvider here if you have one
	// cfg = cfg.WithTracerProvider(tracerProvider)

	// Store config in context for automatic propagation
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	// Fields passed here are automatically included in all log calls
	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	// Logs include trace_id/span_id automatically when spans are active
	// Fields "layer=service" and "user.id=12345" are automatically included
	lc.Logger.Info("Creating user with OTel", otel.F("email", "user@example.com"))

	lc.Success("User created successfully")

	fmt.Println("OTel integration active")
}

// Example_layerPropagation demonstrates context propagation through layers.
// Config stored in context once is automatically available to all nested layers.
// Fields passed to each layer are automatically included in all log calls.
func Example_layerPropagation() {
	// Single config instance stored in context once
	cfg := otel.NewConfig("my-service")
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	// Handler layer - receives HTTP request
	// Fields automatically included in all handler logs
	handler := otel.Layers.StartHandler(ctx, "user", "CreateUser",
		otel.F("http.method", "POST"),
		otel.F("http.path", "/users"))
	defer handler.End()

	// Logs include: layer="handler", http.method="POST", http.path="/users"
	handler.Logger.Info("HTTP request received")

	// Operations layer - config automatically available from context
	ops := otel.Layers.StartOperations(handler.Context(), "user", "CreateUserOperation")
	defer ops.End()

	// Logs include: layer="operations"
	ops.Logger.Info("Validating request")

	// Service layer - config still available
	service := otel.Layers.StartService(ops.Context(), "user", "CreateUser",
		otel.F("user.email", "user@example.com"))
	defer service.End()

	// Logs include: layer="service", user.email="user@example.com"
	service.Logger.Info("Creating user entity")

	// Repository layer - config still available
	repo := otel.Layers.StartRepository(service.Context(), "user", "Insert",
		otel.F("db.operation", "insert"),
		otel.F("db.table", "users"))
	defer repo.End()

	// Logs include: layer="repository", db.operation="insert", db.table="users"
	repo.Logger.Info("Inserting into database")
	repo.Success("User inserted")

	// All logs will be correlated with trace_id and span_id
	fmt.Println("Request completed with full trace")
}

// Example_gradualOTelAdoption shows how to add OTel config to an existing app.
func Example_gradualOTelAdoption() {
	ctx := context.Background()

	// Phase 1: Start without config (uses zerolog fallback)
	lc1 := otel.Layers.StartService(ctx, "user", "CreateUser")
	defer lc1.End()
	// Logger uses zerolog fallback
	lc1.Logger.Info("Phase 1: Basic logging")

	// Phase 2: Add OTel config via context
	cfg := otel.NewConfig("my-service")
	ctx = otel.ContextWithConfig(ctx, cfg)
	// Later: cfg = cfg.WithTracerProvider(tp)

	lc2 := otel.Layers.StartService(ctx, "user", "CreateUser")
	defer lc2.End()
	// Logger now uses OTel with trace correlation
	lc2.Logger.Info("Phase 2: OTel integration added")

	fmt.Println("Gradual OTel adoption completed")
}

// Example_configOptionalButRecommended demonstrates that config is optional but recommended.
func Example_configOptionalButRecommended() {
	ctx := context.Background()

	// Option 1: Without config (uses zerolog fallback)
	simpleApp := otel.Layers.StartService(ctx, "simple", "DoWork")
	defer simpleApp.End()
	// Logger available with zerolog fallback
	simpleApp.Logger.Info("Simple app - basic logging")

	// Option 2: With config (full observability)
	cfg := otel.NewConfig("production-service")
	ctx = otel.ContextWithConfig(ctx, cfg)
	productionApp := otel.Layers.StartService(ctx, "user", "ProcessOrder")
	defer productionApp.End()
	// Logger with OTel integration for production
	productionApp.Logger.Info("Production app - full observability")

	// Recommendation: Always pass config for production to enable:
	// - Automatic trace correlation
	// - Service name in logs
	// - Easy tracing integration later
	// - Consistent log formatting

	fmt.Println("Both patterns work, config recommended for production")
}
