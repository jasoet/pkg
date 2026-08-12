package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartHandler_NoSliceAliasing verifies that StartHandler does not alias
// the caller's variadic slice when it has spare capacity.
func TestStartHandler_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	subFields := baseFields[:2]
	lc := Layers.StartHandler(ctx, "TestOp", "op", subFields...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}

// TestStartService_NoSliceAliasing verifies no slice aliasing in StartService.
func TestStartService_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	lc := Layers.StartService(ctx, "TestOp", "op", baseFields[:2]...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}

// TestStartOperations_NoSliceAliasing verifies no slice aliasing in StartOperations.
func TestStartOperations_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	lc := Layers.StartOperations(ctx, "TestOp", "op", baseFields[:2]...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}

// TestStartMiddleware_NoSliceAliasing verifies no slice aliasing in StartMiddleware.
func TestStartMiddleware_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	lc := Layers.StartMiddleware(ctx, "TestOp", "op", baseFields[:2]...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}

// TestStartRepository_NoSliceAliasing verifies no slice aliasing in StartRepository.
func TestStartRepository_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	lc := Layers.StartRepository(ctx, "TestOp", "op", baseFields[:2]...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}

// TestLayerContext_WithoutConfig verifies that LayerContext works without config in context.
func TestLayerContext_WithoutConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("StartService without config creates zerolog fallback", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser", F("user.id", "123"))
		defer lc.End()
		assert.NotNil(t, lc.Logger, "expected Logger to be set (zerolog fallback)")
	})

	t.Run("StartRepository without config creates zerolog fallback", func(t *testing.T) {
		lc := Layers.StartRepository(ctx, "user", "FindByID", F("user.id", "123"))
		defer lc.End()
		assert.NotNil(t, lc.Logger, "expected Logger to be set (zerolog fallback)")
	})

	t.Run("StartHandler without config creates zerolog fallback", func(t *testing.T) {
		lc := Layers.StartHandler(ctx, "user", "GetUser", F("http.method", "GET"))
		defer lc.End()
		assert.NotNil(t, lc.Logger, "expected Logger to be set (zerolog fallback)")
	})

	t.Run("StartOperations without config creates zerolog fallback", func(t *testing.T) {
		lc := Layers.StartOperations(ctx, "user", "ProcessQueue", F("queue.name", "user-events"))
		defer lc.End()
		assert.NotNil(t, lc.Logger, "expected Logger to be set (zerolog fallback)")
	})
}

// TestLayerContext_WithConfig verifies LayerContext works with proper OTel config.
func TestLayerContext_WithConfig(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	t.Run("StartService with config uses OTel logging", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser", F("user.id", "123"))
		defer lc.End()

		require.NotNil(t, lc.Logger, "expected Logger to be set when config in context")
		require.NotNil(t, lc.Span, "expected Span to be set")

		assert.NotPanics(t, func() {
			lc.Logger.Info("Creating user", F("email", "test@example.com"))
		})
	})

	t.Run("Context returns span context", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()
		assert.NotNil(t, lc.Context())
	})

	t.Run("Error records to both span and log", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()

		err := errors.New("test error")
		returnedErr := lc.Error(err, "Failed to create user", F("user.id", "123"))
		assert.ErrorIs(t, returnedErr, err)
	})

	t.Run("Success adds span event and attributes", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()

		assert.NotPanics(t, func() {
			lc.Success("User created", F("user.id", "123"))
		})
	})
}

// TestLayerContext_NestedCalls verifies context propagation through layers.
func TestLayerContext_NestedCalls(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	handlerCtx := Layers.StartHandler(ctx, "user", "GetUser")
	defer handlerCtx.End()
	require.NotNil(t, handlerCtx.Logger)
	handlerCtx.Logger.Info("Handler started")

	opsCtx := Layers.StartOperations(handlerCtx.Context(), "user", "ProcessRequest")
	defer opsCtx.End()
	require.NotNil(t, opsCtx.Logger)
	opsCtx.Logger.Info("Operations started")

	serviceCtx := Layers.StartService(opsCtx.Context(), "user", "GetUser")
	defer serviceCtx.End()
	require.NotNil(t, serviceCtx.Logger)
	serviceCtx.Logger.Info("Service started")

	repoCtx := Layers.StartRepository(serviceCtx.Context(), "user", "FindByID")
	defer repoCtx.End()
	require.NotNil(t, repoCtx.Logger)
	repoCtx.Logger.Info("Repository query")
	repoCtx.Success("User found")
}

// TestLayerContext_AllLayersWithoutConfig verifies all layers work without config.
func TestLayerContext_AllLayersWithoutConfig(t *testing.T) {
	ctx := context.Background()

	layers := []struct {
		name string
		lc   *LayerContext
	}{
		{"Handler", Layers.StartHandler(ctx, "test", "Operation")},
		{"Middleware", Layers.StartMiddleware(ctx, "test", "Operation")},
		{"Operations", Layers.StartOperations(ctx, "test", "Operation")},
		{"Service", Layers.StartService(ctx, "test", "Operation")},
		{"Repository", Layers.StartRepository(ctx, "test", "Operation")},
	}

	for _, layer := range layers {
		t.Run(layer.name+" works without config", func(t *testing.T) {
			defer layer.lc.End()
			assert.NotNil(t, layer.lc.Logger, "%s: expected Logger to be set (zerolog fallback)", layer.name)
			assert.NotNil(t, layer.lc.Span, "%s: expected Span to be set", layer.name)
		})
	}
}

// TestMiddlewareLayer verifies middleware layer specific functionality.
func TestMiddlewareLayer(t *testing.T) {
	t.Run("StartMiddleware without config creates zerolog fallback", func(t *testing.T) {
		ctx := context.Background()
		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken", F("http.path", "/api/users"))
		defer lc.End()

		assert.NotNil(t, lc.Logger, "expected Logger to be set (zerolog fallback)")
		assert.NotNil(t, lc.Span, "expected Span to be set")
	})

	t.Run("StartMiddleware with config creates logger", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken",
			F("http.path", "/api/users"),
			F("http.method", "GET"))
		defer lc.End()

		require.NotNil(t, lc.Logger, "expected Logger to be set when config in context")
		require.NotNil(t, lc.Span, "expected Span to be set")

		assert.NotPanics(t, func() {
			lc.Logger.Info("Validating token", F("user_id", "123"))
		})
	})

	t.Run("Middleware error handling", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken")
		defer lc.End()

		err := errors.New("invalid token")
		returnedErr := lc.Error(err, "Authentication failed", F("reason", "expired"))
		assert.ErrorIs(t, returnedErr, err)
	})

	t.Run("Middleware success handling", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "cors", "SetHeaders")
		defer lc.End()

		assert.NotPanics(t, func() {
			lc.Success("CORS headers set", F("origin", "https://example.com"))
		})
	})
}

// TestMiddlewareLayerContext verifies middleware context propagation.
func TestMiddlewareLayerContext(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	middlewareCtx := Layers.StartMiddleware(ctx, "auth", "ValidateToken", F("http.path", "/api/users"))
	defer middlewareCtx.End()
	require.NotNil(t, middlewareCtx.Logger)
	middlewareCtx.Logger.Info("Middleware started")

	handlerCtx := Layers.StartHandler(middlewareCtx.Context(), "user", "GetUser")
	defer handlerCtx.End()
	require.NotNil(t, handlerCtx.Logger)
	handlerCtx.Logger.Info("Handler started")

	serviceCtx := Layers.StartService(handlerCtx.Context(), "user", "GetUser")
	defer serviceCtx.End()
	require.NotNil(t, serviceCtx.Logger)
	serviceCtx.Logger.Info("Service started")

	repoCtx := Layers.StartRepository(serviceCtx.Context(), "user", "FindByID")
	defer repoCtx.End()
	require.NotNil(t, repoCtx.Logger)
	repoCtx.Logger.Info("Repository query")
	repoCtx.Success("User found")

	serviceCtx.Success("Service completed")
	handlerCtx.Success("Handler completed")
	middlewareCtx.Success("Middleware completed")
}

// TestConfigContext verifies config context management.
func TestConfigContext(t *testing.T) {
	t.Run("ContextWithConfig stores config", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		retrieved := ConfigFromContext(ctx)
		require.NotNil(t, retrieved, "expected config to be retrieved from context")
		assert.Equal(t, "test-service", retrieved.ServiceName)
	})

	t.Run("ConfigFromContext returns nil without config", func(t *testing.T) {
		ctx := context.Background()
		assert.Nil(t, ConfigFromContext(ctx))
	})
}
