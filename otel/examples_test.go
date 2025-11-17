package otel_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
)

// Example_withoutOTelConfig demonstrates that logging still works even without OTel configuration.
// This is useful for simple applications that don't need distributed tracing but still want structured logging.
func Example_withoutOTelConfig() {
	ctx := context.Background()

	// Use nil config - logging falls back to zerolog
	lc := otel.Layers.StartService(ctx, nil, "user", "CreateUser",
		"user.id", "12345")
	defer lc.End()

	// Logs will use zerolog (JSON format to stderr)
	lc.Logger.Info("Creating user without OTel", otel.F("email", "user@example.com"))

	// Error handling still works
	err := errors.New("validation failed")
	if err != nil {
		_ = lc.Error(err, "User creation failed", otel.F("reason", "invalid email"))
		return
	}

	_ = lc.Success("User created successfully")

	fmt.Println("Logging works without OTel config")
}

// Example_withOTelConfig demonstrates full OTel integration with tracing and structured logging.
func Example_withOTelConfig() {
	ctx := context.Background()

	// Create OTel config with default logger (zerolog-based with OTel integration)
	cfg := otel.NewConfig("my-service")

	// Add TracerProvider here if you have one
	// cfg = cfg.WithTracerProvider(tracerProvider)

	lc := otel.Layers.StartService(ctx, cfg, "user", "CreateUser",
		"user.id", "12345")
	defer lc.End()

	// Logs will include trace_id and span_id automatically when spans are active
	lc.Logger.Info("Creating user with OTel", otel.F("email", "user@example.com"))

	_ = lc.Success("User created successfully")

	fmt.Println("OTel integration active")
}

// Example_layerPropagation demonstrates how to pass config through all layers
// to ensure consistent logging and tracing throughout the request lifecycle.
func Example_layerPropagation() {
	ctx := context.Background()

	// Single config instance shared across all layers
	cfg := otel.NewConfig("my-service")

	// Handler layer - receives HTTP request
	handler := otel.Layers.StartHandler(ctx, cfg, "user", "CreateUser",
		"http.method", "POST",
		"http.path", "/users")
	defer handler.End()

	handler.Logger.Info("HTTP request received")

	// Operations layer - orchestrates the operation
	ops := otel.Layers.StartOperations(handler.Context(), cfg, "user", "CreateUserOperation")
	defer ops.End()

	ops.Logger.Info("Validating request")

	// Service layer - business logic
	service := otel.Layers.StartService(ops.Context(), cfg, "user", "CreateUser",
		"user.email", "user@example.com")
	defer service.End()

	service.Logger.Info("Creating user entity")

	// Repository layer - database access
	repo := otel.Layers.StartRepository(service.Context(), cfg, "user", "Insert",
		"db.operation", "insert",
		"db.table", "users")
	defer repo.End()

	repo.Logger.Info("Inserting into database")
	_ = repo.Success("User inserted")

	// All logs will be correlated with trace_id and span_id
	fmt.Println("Request completed with full trace")
}

// Example_gradualOTelAdoption shows how to start without OTel and add it later.
func Example_gradualOTelAdoption() {
	ctx := context.Background()

	// Phase 1: Start with nil config (just logging)
	phase1Config := (*otel.Config)(nil)

	lc1 := otel.Layers.StartService(ctx, phase1Config, "user", "CreateUser")
	defer lc1.End()
	lc1.Logger.Info("Phase 1: Basic logging only")

	// Phase 2: Add OTel config later (no code changes needed!)
	phase2Config := otel.NewConfig("my-service")
	// Later: phase2Config = phase2Config.WithTracerProvider(tp)

	lc2 := otel.Layers.StartService(ctx, phase2Config, "user", "CreateUser")
	defer lc2.End()
	lc2.Logger.Info("Phase 2: OTel integration added")

	fmt.Println("Gradual OTel adoption completed")
}

// Example_configOptionalButRecommended demonstrates that config is optional but recommended.
func Example_configOptionalButRecommended() {
	ctx := context.Background()

	// Option 1: Without config (simple apps, local dev)
	simpleApp := otel.Layers.StartService(ctx, nil, "simple", "DoWork")
	defer simpleApp.End()
	simpleApp.Logger.Info("Simple app - just logging")

	// Option 2: With config (production apps)
	cfg := otel.NewConfig("production-service")
	productionApp := otel.Layers.StartService(ctx, cfg, "user", "ProcessOrder")
	defer productionApp.End()
	productionApp.Logger.Info("Production app - full observability")

	// Recommendation: Always pass config for production to enable:
	// - Automatic trace correlation
	// - Service name in logs
	// - Easy tracing integration later
	// - Consistent log formatting

	fmt.Println("Both patterns work, config recommended for production")
}
