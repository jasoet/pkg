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
	// Fields passed here are automatically included in all log calls
	fmt.Println("=== Example 1: LayerContext ===")
	lc := otel.Layers.StartService(ctx, "user", "CreateUser",
		otel.F("user.id", "12345"))
	defer lc.End()

	// Logger is always available, fields auto-included
	lc.Logger.Info("Creating user", otel.F("email", "user@example.com"))
	// Simulate success - user.id="12345" automatically included
	_ = lc.Success("User created successfully")

	// Example 2: SpanHelper with Logger() method
	fmt.Println("\n=== Example 2: SpanHelper.Logger() ===")
	span := otel.StartSpan(ctx, "service.order", "ProcessOrder",
		otel.WithAttribute("order.id", "ORD-123"))
	defer span.End()

	// Logger is always available
	logger := span.Logger("service.order")
	logger.Info("Processing order", otel.F("items", 3))

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

	// Example 5: All five layers (config propagates automatically via context)
	// Fields are automatically included in all log calls for each layer
	fmt.Println("\n=== Example 5: All Layers ===")

	// Middleware layer (auth, CORS, rate limiting, etc.)
	middlewareCtx := otel.Layers.StartMiddleware(ctx, "auth", "ValidateToken",
		otel.F("http.path", "/api/users"),
		otel.F("http.method", "GET"))
	defer middlewareCtx.End()
	middlewareCtx.Logger.Info("Validating authentication token")

	// Handler layer (config available from middleware.Context())
	handlerCtx := otel.Layers.StartHandler(middlewareCtx.Context(), "user", "GetUser",
		otel.F("http.method", "GET"))
	defer handlerCtx.End()
	handlerCtx.Logger.Info("Handling request")

	// Operations layer (config available from handler.Context())
	opsCtx := otel.Layers.StartOperations(handlerCtx.Context(), "user", "ProcessUserRequest")
	defer opsCtx.End()
	opsCtx.Logger.Info("Orchestrating user request")

	// Service layer
	serviceCtx := otel.Layers.StartService(opsCtx.Context(), "user", "GetUser",
		otel.F("user.id", "123"))
	defer serviceCtx.End()
	serviceCtx.Logger.Info("Fetching user data")

	// Repository layer
	repoCtx := otel.Layers.StartRepository(serviceCtx.Context(), "user", "FindByID",
		otel.F("user.id", "123"),
		otel.F("db.operation", "select"))
	defer repoCtx.End()
	repoCtx.Logger.Debug("Querying database")
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

// Example showing middleware layer instrumentation
func Example_middlewareLayer() {
	cfg := otel.NewConfig("api-service")
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	fmt.Println("=== Middleware Layer Examples ===")

	// Example 1: Authentication middleware
	// Fields automatically included in all auth logs
	fmt.Println("\n--- Authentication Middleware ---")
	authCtx := otel.Layers.StartMiddleware(ctx, "auth", "ValidateToken",
		otel.F("http.path", "/api/users"),
		otel.F("http.method", "GET"))
	defer authCtx.End()

	authCtx.Logger.Info("Checking authorization header")
	// Simulate successful auth - http.path and http.method auto-included
	_ = authCtx.Success("Token validated", otel.F("user.id", "user-123"))

	// Example 2: CORS middleware
	fmt.Println("\n--- CORS Middleware ---")
	corsCtx := otel.Layers.StartMiddleware(ctx, "cors", "SetHeaders",
		otel.F("origin", "https://example.com"))
	defer corsCtx.End()

	corsCtx.Logger.Info("Setting CORS headers")
	_ = corsCtx.Success("CORS headers configured")

	// Example 3: Rate limiting middleware with error
	fmt.Println("\n--- Rate Limiting Middleware ---")
	rateLimitCtx := otel.Layers.StartMiddleware(ctx, "ratelimit", "CheckLimit",
		otel.F("client.ip", "192.168.1.100"),
		otel.F("endpoint", "/api/data"))
	defer rateLimitCtx.End()

	rateLimitCtx.Logger.Warn("Rate limit exceeded", otel.F("limit", 100))
	err := errors.New("rate limit exceeded")
	_ = rateLimitCtx.Error(err, "request throttled", otel.F("retry_after", "60s"))

	// Example 4: Middleware chain with context propagation
	fmt.Println("\n--- Middleware Chain ---")
	mw1Ctx := otel.Layers.StartMiddleware(ctx, "logging", "RequestLogger")
	defer mw1Ctx.End()
	mw1Ctx.Logger.Info("Incoming request", otel.F("request.id", "req-456"))

	// Next middleware gets context from previous one
	mw2Ctx := otel.Layers.StartMiddleware(mw1Ctx.Context(), "validation", "ValidateInput")
	defer mw2Ctx.End()
	mw2Ctx.Logger.Info("Validating request body")
	_ = mw2Ctx.Success("Validation passed")
	_ = mw1Ctx.Success("Request logged")

	fmt.Println("\nAll middleware examples completed")

	// Output:
	// === Middleware Layer Examples ===
	//
	// --- Authentication Middleware ---
	//
	// --- CORS Middleware ---
	//
	// --- Rate Limiting Middleware ---
	//
	// --- Middleware Chain ---
	//
	// All middleware examples completed
}
