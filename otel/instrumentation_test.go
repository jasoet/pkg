package otel

import (
	"context"
	"errors"
	"testing"
)

// TestLayerContext_WithoutConfig verifies that LayerContext works without config in context
func TestLayerContext_WithoutConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("StartService without config has nil logger", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser",
			F("user.id", "123"))
		defer lc.End()

		if lc.Logger != nil {
			t.Error("Expected Logger to be nil when no config in context")
		}
	})

	t.Run("StartRepository without config has nil logger", func(t *testing.T) {
		lc := Layers.StartRepository(ctx, "user", "FindByID",
			F("user.id", "123"))
		defer lc.End()

		if lc.Logger != nil {
			t.Error("Expected Logger to be nil when no config in context")
		}
	})

	t.Run("StartHandler without config has nil logger", func(t *testing.T) {
		lc := Layers.StartHandler(ctx, "user", "GetUser",
			F("http.method", "GET"))
		defer lc.End()

		if lc.Logger != nil {
			t.Error("Expected Logger to be nil when no config in context")
		}
	})

	t.Run("StartOperations without config has nil logger", func(t *testing.T) {
		lc := Layers.StartOperations(ctx, "user", "ProcessQueue",
			F("queue.name", "user-events"))
		defer lc.End()

		if lc.Logger != nil {
			t.Error("Expected Logger to be nil when no config in context")
		}
	})
}

// TestLayerContext_WithConfig verifies LayerContext works with proper OTel config
func TestLayerContext_WithConfig(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	t.Run("StartService with config uses OTel logging", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser",
			F("user.id", "123"))
		defer lc.End()

		// Should not panic
		if lc.Logger != nil {
			lc.Logger.Info("Creating user", F("email", "test@example.com"))
		}

		if lc.Span == nil {
			t.Error("Expected Span to be set")
		}

		if lc.Logger == nil {
			t.Error("Expected Logger to be set when config in context")
		}
	})

	t.Run("Context returns span context", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()

		spanCtx := lc.Context()
		if spanCtx == nil {
			t.Error("Expected context to be returned")
		}
	})

	t.Run("Error records to both span and log", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()

		err := errors.New("test error")
		returnedErr := lc.Error(err, "Failed to create user", F("user.id", "123"))

		if returnedErr != err {
			t.Errorf("Expected Error to return the same error, got %v", returnedErr)
		}
	})

	t.Run("Success returns nil", func(t *testing.T) {
		lc := Layers.StartService(ctx, "user", "CreateUser")
		defer lc.End()

		err := lc.Success("User created", F("user.id", "123"))
		if err != nil {
			t.Errorf("Expected Success to return nil, got %v", err)
		}
	})
}

// TestLayerContext_NestedCalls verifies context propagation through layers
func TestLayerContext_NestedCalls(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	// Handler layer
	handlerCtx := Layers.StartHandler(ctx, "user", "GetUser")
	defer handlerCtx.End()

	if handlerCtx.Logger != nil {
		handlerCtx.Logger.Info("Handler started")
	}

	// Operations layer (uses handler context - config is propagated via context)
	opsCtx := Layers.StartOperations(handlerCtx.Context(), "user", "ProcessRequest")
	defer opsCtx.End()

	if opsCtx.Logger != nil {
		opsCtx.Logger.Info("Operations started")
	}

	// Service layer (uses operations context - config is still there)
	serviceCtx := Layers.StartService(opsCtx.Context(), "user", "GetUser")
	defer serviceCtx.End()

	if serviceCtx.Logger != nil {
		serviceCtx.Logger.Info("Service started")
	}

	// Repository layer (uses service context - config is still there)
	repoCtx := Layers.StartRepository(serviceCtx.Context(), "user", "FindByID")
	defer repoCtx.End()

	if repoCtx.Logger != nil {
		repoCtx.Logger.Info("Repository query")
	}
	_ = repoCtx.Success("User found")

	// All layers should complete without panic
}

// TestLayerContext_AllLayersWithoutConfig verifies all layers work without config
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

			// Logger should be nil without config
			if layer.lc.Logger != nil {
				t.Errorf("%s: Expected Logger to be nil without config in context", layer.name)
			}

			if layer.lc.Span == nil {
				t.Errorf("%s: Expected Span to be set", layer.name)
			}
		})
	}
}

// TestMiddlewareLayer verifies middleware layer specific functionality
func TestMiddlewareLayer(t *testing.T) {
	t.Run("Middleware span without config", func(t *testing.T) {
		ctx := context.Background()
		span := Layers.Middleware(ctx, "auth", "ValidateToken",
			F("http.path", "/api/users"),
			F("http.method", "GET"))
		defer span.End()

		if span == nil {
			t.Error("Expected span to be created")
		}
	})

	t.Run("StartMiddleware without config has nil logger", func(t *testing.T) {
		ctx := context.Background()
		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken",
			F("http.path", "/api/users"))
		defer lc.End()

		if lc.Logger != nil {
			t.Error("Expected Logger to be nil when no config in context")
		}

		if lc.Span == nil {
			t.Error("Expected Span to be set")
		}
	})

	t.Run("StartMiddleware with config creates logger", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken",
			F("http.path", "/api/users"),
			F("http.method", "GET"))
		defer lc.End()

		if lc.Logger == nil {
			t.Error("Expected Logger to be set when config in context")
		}

		if lc.Span == nil {
			t.Error("Expected Span to be set")
		}

		// Should not panic
		if lc.Logger != nil {
			lc.Logger.Info("Validating token", F("user_id", "123"))
		}
	})

	t.Run("Middleware error handling", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "auth", "ValidateToken")
		defer lc.End()

		err := errors.New("invalid token")
		returnedErr := lc.Error(err, "Authentication failed", F("reason", "expired"))

		if returnedErr != err {
			t.Errorf("Expected Error to return the same error, got %v", returnedErr)
		}
	})

	t.Run("Middleware success handling", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		lc := Layers.StartMiddleware(ctx, "cors", "SetHeaders")
		defer lc.End()

		err := lc.Success("CORS headers set", F("origin", "https://example.com"))
		if err != nil {
			t.Errorf("Expected Success to return nil, got %v", err)
		}
	})
}

// TestMiddlewareLayerContext verifies middleware context propagation
func TestMiddlewareLayerContext(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	// Middleware layer
	middlewareCtx := Layers.StartMiddleware(ctx, "auth", "ValidateToken",
		F("http.path", "/api/users"))
	defer middlewareCtx.End()

	if middlewareCtx.Logger != nil {
		middlewareCtx.Logger.Info("Middleware started")
	}

	// Handler layer (uses middleware context)
	handlerCtx := Layers.StartHandler(middlewareCtx.Context(), "user", "GetUser")
	defer handlerCtx.End()

	if handlerCtx.Logger != nil {
		handlerCtx.Logger.Info("Handler started")
	}

	// Service layer (uses handler context)
	serviceCtx := Layers.StartService(handlerCtx.Context(), "user", "GetUser")
	defer serviceCtx.End()

	if serviceCtx.Logger != nil {
		serviceCtx.Logger.Info("Service started")
	}

	// Repository layer (uses service context)
	repoCtx := Layers.StartRepository(serviceCtx.Context(), "user", "FindByID")
	defer repoCtx.End()

	if repoCtx.Logger != nil {
		repoCtx.Logger.Info("Repository query")
	}
	_ = repoCtx.Success("User found")

	// All layers should complete without panic
	_ = serviceCtx.Success("Service completed")
	_ = handlerCtx.Success("Handler completed")
	_ = middlewareCtx.Success("Middleware completed")
}

// TestConfigContext verifies config context management
func TestConfigContext(t *testing.T) {
	t.Run("ContextWithConfig stores config", func(t *testing.T) {
		cfg := NewConfig("test-service")
		ctx := ContextWithConfig(context.Background(), cfg)

		retrieved := ConfigFromContext(ctx)
		if retrieved == nil {
			t.Error("Expected config to be retrieved from context")
		}

		if retrieved.ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got '%s'", retrieved.ServiceName)
		}
	})

	t.Run("ConfigFromContext returns nil without config", func(t *testing.T) {
		ctx := context.Background()
		retrieved := ConfigFromContext(ctx)

		if retrieved != nil {
			t.Error("Expected nil when no config in context")
		}
	})
}
