package otel

import (
	"context"
	"errors"
	"testing"
)

// TestLayerContext_WithNilConfig verifies that LayerContext works with nil config (zerolog fallback)
func TestLayerContext_WithNilConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("StartService with nil config uses zerolog", func(t *testing.T) {
		lc := Layers.StartService(ctx, nil, "user", "CreateUser",
			"user.id", "123")
		defer lc.End()

		// Should not panic
		lc.Logger.Info("Creating user", F("email", "test@example.com"))
		lc.Logger.Debug("Debug message")
		lc.Logger.Warn("Warning message")

		err := errors.New("test error")
		_ = lc.Error(err, "Test error", F("code", 500))

		// Should not panic
		_ = lc.Success("Success message", F("result", "ok"))
	})

	t.Run("StartRepository with nil config uses zerolog", func(t *testing.T) {
		lc := Layers.StartRepository(ctx, nil, "user", "FindByID",
			"user.id", "123")
		defer lc.End()

		lc.Logger.Debug("Querying database")
		_ = lc.Success("Found user")
	})

	t.Run("StartHandler with nil config uses zerolog", func(t *testing.T) {
		lc := Layers.StartHandler(ctx, nil, "user", "GetUser",
			"http.method", "GET")
		defer lc.End()

		lc.Logger.Info("Handling request")
		_ = lc.Success("Request handled")
	})

	t.Run("StartOperations with nil config uses zerolog", func(t *testing.T) {
		lc := Layers.StartOperations(ctx, nil, "user", "ProcessQueue",
			"queue.name", "user-events")
		defer lc.End()

		lc.Logger.Info("Processing queue")
		_ = lc.Success("Queue processed")
	})
}

// TestLayerContext_WithConfig verifies LayerContext works with proper OTel config
func TestLayerContext_WithConfig(t *testing.T) {
	ctx := context.Background()
	cfg := NewConfig("test-service")

	t.Run("StartService with config uses OTel logging", func(t *testing.T) {
		lc := Layers.StartService(ctx, cfg, "user", "CreateUser",
			"user.id", "123")
		defer lc.End()

		// Should not panic
		lc.Logger.Info("Creating user", F("email", "test@example.com"))

		if lc.Span == nil {
			t.Error("Expected Span to be set")
		}

		if lc.Logger == nil {
			t.Error("Expected Logger to be set")
		}
	})

	t.Run("Context returns span context", func(t *testing.T) {
		lc := Layers.StartService(ctx, cfg, "user", "CreateUser")
		defer lc.End()

		spanCtx := lc.Context()
		if spanCtx == nil {
			t.Error("Expected context to be returned")
		}
	})

	t.Run("Error records to both span and log", func(t *testing.T) {
		lc := Layers.StartService(ctx, cfg, "user", "CreateUser")
		defer lc.End()

		err := errors.New("test error")
		returnedErr := lc.Error(err, "Failed to create user", F("user.id", "123"))

		if returnedErr != err {
			t.Errorf("Expected Error to return the same error, got %v", returnedErr)
		}
	})

	t.Run("Success returns nil", func(t *testing.T) {
		lc := Layers.StartService(ctx, cfg, "user", "CreateUser")
		defer lc.End()

		err := lc.Success("User created", F("user.id", "123"))
		if err != nil {
			t.Errorf("Expected Success to return nil, got %v", err)
		}
	})
}

// TestLayerContext_NestedCalls verifies context propagation through layers
func TestLayerContext_NestedCalls(t *testing.T) {
	ctx := context.Background()
	cfg := NewConfig("test-service")

	// Handler layer
	handlerCtx := Layers.StartHandler(ctx, cfg, "user", "GetUser")
	defer handlerCtx.End()

	handlerCtx.Logger.Info("Handler started")

	// Operations layer (uses handler context)
	opsCtx := Layers.StartOperations(handlerCtx.Context(), cfg, "user", "ProcessRequest")
	defer opsCtx.End()

	opsCtx.Logger.Info("Operations started")

	// Service layer (uses operations context)
	serviceCtx := Layers.StartService(opsCtx.Context(), cfg, "user", "GetUser")
	defer serviceCtx.End()

	serviceCtx.Logger.Info("Service started")

	// Repository layer (uses service context)
	repoCtx := Layers.StartRepository(serviceCtx.Context(), cfg, "user", "FindByID")
	defer repoCtx.End()

	repoCtx.Logger.Info("Repository query")
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
		{"Handler", Layers.StartHandler(ctx, nil, "test", "Operation")},
		{"Operations", Layers.StartOperations(ctx, nil, "test", "Operation")},
		{"Service", Layers.StartService(ctx, nil, "test", "Operation")},
		{"Repository", Layers.StartRepository(ctx, nil, "test", "Operation")},
	}

	for _, layer := range layers {
		t.Run(layer.name+" works without config", func(t *testing.T) {
			defer layer.lc.End()

			// Should not panic
			layer.lc.Logger.Info("Test message", F("key", "value"))

			err := errors.New("test error")
			_ = layer.lc.Error(err, "Test error")

			_ = layer.lc.Success("Test success")

			if layer.lc.Span == nil {
				t.Errorf("%s: Expected Span to be set", layer.name)
			}

			if layer.lc.Logger == nil {
				t.Errorf("%s: Expected Logger to be set", layer.name)
			}
		})
	}
}
