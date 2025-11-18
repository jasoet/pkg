package otel_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
)

// Example_withoutOTelConfig demonstrates span creation without OTel configuration.
// Useful for simple apps that only need basic tracing without logging.
func Example_withoutOTelConfig() {
	ctx := context.Background()

	// No config in context - spans work, but logger is nil
	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	// lc.Logger is nil - only spans are created
	if lc.Logger == nil {
		fmt.Println("Logger is nil without config")
	}

	// Span tracking still works
	lc.Span.AddAttribute("status", "processing")

	// Error handling works (recorded in span)
	err := errors.New("validation failed")
	if err != nil {
		_ = lc.Error(err, "User creation failed")
		return
	}

	_ = lc.Success("User created successfully")

	fmt.Println("Spans work without OTel config")
}

// Example_withOTelConfig demonstrates full OTel integration with tracing and structured logging.
func Example_withOTelConfig() {
	// Create OTel config with default logger (zerolog-based with OTel integration)
	cfg := otel.NewConfig("my-service")

	// Add TracerProvider here if you have one
	// cfg = cfg.WithTracerProvider(tracerProvider)

	// Store config in context for automatic propagation
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	// Logs will include trace_id and span_id automatically when spans are active
	if lc.Logger != nil {
		lc.Logger.Info("Creating user with OTel", otel.F("email", "user@example.com"))
	}

	_ = lc.Success("User created successfully")

	fmt.Println("OTel integration active")
}

// Example_layerPropagation demonstrates context propagation through layers.
// Config stored in context once is automatically available to all nested layers.
func Example_layerPropagation() {
	// Single config instance stored in context once
	cfg := otel.NewConfig("my-service")
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	// Handler layer - receives HTTP request
	handler := otel.Layers.StartHandler(ctx, "user", "CreateUser",
		otel.F("http.method", "POST"),
		otel.F("http.path", "/users"))
	defer handler.End()

	if handler.Logger != nil {
		handler.Logger.Info("HTTP request received")
	}

	// Operations layer - config automatically available from context
	ops := otel.Layers.StartOperations(handler.Context(), "user", "CreateUserOperation")
	defer ops.End()

	if ops.Logger != nil {
		ops.Logger.Info("Validating request")
	}

	// Service layer - config still available
	service := otel.Layers.StartService(ops.Context(), "user", "CreateUser",
		otel.F("user.email", "user@example.com"))
	defer service.End()

	if service.Logger != nil {
		service.Logger.Info("Creating user entity")
	}

	// Repository layer - config still available
	repo := otel.Layers.StartRepository(service.Context(), "user", "Insert",
		otel.F("db.operation", "insert"),
		otel.F("db.table", "users"))
	defer repo.End()

	if repo.Logger != nil {
		repo.Logger.Info("Inserting into database")
	}
	_ = repo.Success("User inserted")

	// All logs will be correlated with trace_id and span_id
	fmt.Println("Request completed with full trace")
}

// Example_gradualOTelAdoption shows how to add OTel config to an existing app.
func Example_gradualOTelAdoption() {
	ctx := context.Background()

	// Phase 1: Start without config (spans only, no logging)
	lc1 := otel.Layers.StartService(ctx, "user", "CreateUser")
	defer lc1.End()
	// lc1.Logger is nil

	// Phase 2: Add OTel config via context
	cfg := otel.NewConfig("my-service")
	ctx = otel.ContextWithConfig(ctx, cfg)
	// Later: cfg = cfg.WithTracerProvider(tp)

	lc2 := otel.Layers.StartService(ctx, "user", "CreateUser")
	defer lc2.End()
	if lc2.Logger != nil {
		lc2.Logger.Info("Phase 2: OTel integration added")
	}

	fmt.Println("Gradual OTel adoption completed")
}

// Example_configOptionalButRecommended demonstrates that config is optional but recommended.
func Example_configOptionalButRecommended() {
	ctx := context.Background()

	// Option 1: Without config (spans only, no logging)
	simpleApp := otel.Layers.StartService(ctx, "simple", "DoWork")
	defer simpleApp.End()
	// simpleApp.Logger is nil

	// Option 2: With config (full observability)
	cfg := otel.NewConfig("production-service")
	ctx = otel.ContextWithConfig(ctx, cfg)
	productionApp := otel.Layers.StartService(ctx, "user", "ProcessOrder")
	defer productionApp.End()
	if productionApp.Logger != nil {
		productionApp.Logger.Info("Production app - full observability")
	}

	// Recommendation: Always pass config for production to enable:
	// - Automatic trace correlation
	// - Service name in logs
	// - Easy tracing integration later
	// - Consistent log formatting

	fmt.Println("Both patterns work, config recommended for production")
}
