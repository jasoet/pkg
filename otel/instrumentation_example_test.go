package otel_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasoet/pkg/v2/otel"
)

// Example demonstrates the new integrated span-logging features
func Example_layerContextIntegration() {
	// Setup OTel config
	cfg := otel.NewConfig("example-service")

	// Store config in context for automatic propagation
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	// Example 1: Using LayerContext for simplified span + logging
	fmt.Println("=== Example 1: LayerContext ===")
	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	if lc.Logger != nil {
		lc.Logger.Info("Creating user", otel.F("email", "user@example.com"))
	}
	// Simulate success
	_ = lc.Success("User created successfully", otel.F("user.id", "12345"))

	// Example 2: SpanHelper with Logger() method
	fmt.Println("\n=== Example 2: SpanHelper.Logger() ===")
	span := otel.StartSpan(ctx, "service.order", "ProcessOrder",
		otel.WithAttribute("order.id", "ORD-123"))
	defer span.End()

	logger := span.Logger("service.order")
	if logger != nil {
		logger.Info("Processing order", otel.F("items", 3))
	}

	// Example 3: LogEvent for dual span events + logs
	fmt.Println("\n=== Example 3: LogEvent ===")
	span2 := otel.StartSpan(ctx, "service.cache", "GetFromCache",
		otel.WithAttribute("cache.key", "user:123"))
	defer span2.End()

	logger2 := span2.Logger("service.cache")
	span2.LogEvent(logger2, "cache.miss",
		otel.F("key", "user:123"),
		otel.F("reason", "expired"))

	// Example 4: Error handling with LayerContext
	fmt.Println("\n=== Example 4: Error Handling ===")
	lc2 := otel.Layers.StartRepository(ctx, "user", "FindByID",
		otel.F("user.id", "999"))
	defer lc2.End()

	// Simulate error
	err := errors.New("user not found")
	_ = lc2.Error(err, "failed to find user", otel.F("user.id", "999"))

	// Example 5: All four layers (config propagates automatically via context)
	fmt.Println("\n=== Example 5: All Layers ===")

	// Handler layer
	handlerCtx := otel.Layers.StartHandler(ctx, "user", "GetUser",
		otel.F("http.method", "GET"))
	defer handlerCtx.End()
	if handlerCtx.Logger != nil {
		handlerCtx.Logger.Info("Handling request")
	}

	// Operations layer (config available from handler.Context())
	opsCtx := otel.Layers.StartOperations(handlerCtx.Context(), "user", "ProcessUserRequest")
	defer opsCtx.End()
	if opsCtx.Logger != nil {
		opsCtx.Logger.Info("Orchestrating user request")
	}

	// Service layer
	serviceCtx := otel.Layers.StartService(opsCtx.Context(), "user", "GetUser",
		otel.F("user.id", "123"))
	defer serviceCtx.End()
	if serviceCtx.Logger != nil {
		serviceCtx.Logger.Info("Fetching user data")
	}

	// Repository layer
	repoCtx := otel.Layers.StartRepository(serviceCtx.Context(), "user", "FindByID",
		otel.F("user.id", "123"),
		otel.F("db.operation", "select"))
	defer repoCtx.End()
	if repoCtx.Logger != nil {
		repoCtx.Logger.Debug("Querying database")
	}
	_ = repoCtx.Success("User found")

	fmt.Println("\nAll examples completed successfully")

	// Output:
	// === Example 1: LayerContext ===
	//
	// === Example 2: SpanHelper.Logger() ===
	//
	// === Example 3: LogEvent ===
	//
	// === Example 4: Error Handling ===
	//
	// === Example 5: All Layers ===
	//
	// All examples completed successfully
}

// Example showing optional function parameter
func Example_optionalFunctionParameter() {
	cfg := otel.NewConfig("test-service")
	ctx := context.Background()

	// With function name
	logger1 := otel.NewLogHelper(ctx, cfg, "mypackage", "MyFunction")
	logger1.Info("Message with function", otel.F("key", "value"))

	// Without function name (when used with spans)
	logger2 := otel.NewLogHelper(ctx, cfg, "mypackage", "")
	logger2.Info("Message without function", otel.F("key", "value"))

	// Output:
}

// Example showing LogHelper.Span() accessor
func Example_logHelperSpanAccessor() {
	cfg := otel.NewConfig("test-service")
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	span := otel.StartSpan(ctx, "service.test", "Operation")
	defer span.End()

	logger := span.Logger("service.test")

	// Access span from logger
	if logger != nil && logger.Span().IsRecording() {
		logger.Info("Span is active")
	}

	// Output:
}
