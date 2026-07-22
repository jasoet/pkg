package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newSpanRecorder returns an in-memory exporter and a context carrying a Config
// whose TracerProvider syncs ended spans to that exporter.
func newSpanRecorder(t *testing.T) (*tracetest.InMemoryExporter, context.Context) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := NewConfig("test-service", WithTracerProvider(tp))
	return exporter, ContextWithConfig(context.Background(), cfg)
}

// requireSingleSpan asserts exactly one ended span and returns it.
func requireSingleSpan(t *testing.T, exporter *tracetest.InMemoryExporter) tracetest.SpanStub {
	t.Helper()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one ended span")
	return spans[0]
}

// spanAttribute returns the value of the named attribute on a span stub.
func spanAttribute(span tracetest.SpanStub, key string) (attribute.Value, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

// TestSpanHelper_SpanCreationAndEnd verifies StartSpan creates a span with the
// given operation name and End() exports exactly one ended span.
func TestSpanHelper_SpanCreationAndEnd(t *testing.T) {
	exporter, ctx := newSpanRecorder(t)

	span := StartSpan(ctx, "service.user", "UserService.CreateUser")

	// Not ended yet: exporter must be empty before End().
	assert.Empty(t, exporter.GetSpans(), "span should not be exported before End()")

	span.End()

	stub := requireSingleSpan(t, exporter)
	assert.Equal(t, "UserService.CreateUser", stub.Name)
	assert.Equal(t, "service.user", stub.InstrumentationScope.Name)
}

// TestSpanHelper_KindAndAttributes verifies WithSpanKind and WithAttributes
// are applied to the started span.
func TestSpanHelper_KindAndAttributes(t *testing.T) {
	exporter, ctx := newSpanRecorder(t)

	span := StartSpan(ctx, "repository.user", "UserRepository.FindByID",
		WithSpanKind(trace.SpanKindClient),
		WithAttribute("db.operation", "select"),
		WithAttributes(
			F("user.id", "123"),
			F("db.rows", 1),
		),
	)
	span.End()

	stub := requireSingleSpan(t, exporter)
	assert.Equal(t, trace.SpanKindClient, stub.SpanKind)

	for key, want := range map[string]string{
		"db.operation": "select",
		"user.id":      "123",
	} {
		got, ok := spanAttribute(stub, key)
		require.True(t, ok, "expected attribute %q on span", key)
		assert.Equal(t, want, got.AsString())
	}

	rows, ok := spanAttribute(stub, "db.rows")
	require.True(t, ok, "expected attribute %q on span", "db.rows")
	assert.Equal(t, int64(1), rows.AsInt64())
}

// TestSpanHelper_Error verifies Error records an exception event on the span,
// sets error status, and returns the same error for propagation.
func TestSpanHelper_Error(t *testing.T) {
	exporter, ctx := newSpanRecorder(t)

	span := StartSpan(ctx, "service.user", "UserService.CreateUser")
	sentinel := errors.New("database unavailable")
	returned := span.Error(sentinel, "failed to create user")
	span.End()

	assert.ErrorIs(t, returned, sentinel, "Error must return the passed error unchanged")

	stub := requireSingleSpan(t, exporter)
	assert.Equal(t, codes.Error, stub.Status.Code)
	assert.Equal(t, "failed to create user", stub.Status.Description)

	require.Len(t, stub.Events, 1, "expected one recorded error event")
	assert.Equal(t, "exception", stub.Events[0].Name)
}

// TestSpanHelper_AddAttribute verifies attributes added after span start
// mutate the live span and are visible on the ended span.
func TestSpanHelper_AddAttribute(t *testing.T) {
	exporter, ctx := newSpanRecorder(t)

	span := StartSpan(ctx, "service.user", "UserService.CreateUser",
		WithAttribute("initial", "present"))
	span.AddAttribute("user.id", "456")
	span.AddAttributes(F("retry.count", 2), F("cache.hit", true))
	span.End()

	stub := requireSingleSpan(t, exporter)

	initial, ok := spanAttribute(stub, "initial")
	require.True(t, ok)
	assert.Equal(t, "present", initial.AsString())

	userID, ok := spanAttribute(stub, "user.id")
	require.True(t, ok, "AddAttribute after start must be visible on ended span")
	assert.Equal(t, "456", userID.AsString())

	retries, ok := spanAttribute(stub, "retry.count")
	require.True(t, ok)
	assert.Equal(t, int64(2), retries.AsInt64())

	cacheHit, ok := spanAttribute(stub, "cache.hit")
	require.True(t, ok)
	assert.True(t, cacheHit.AsBool())
}

// TestLayers_SpanNamesAndScopes documents the span name and instrumentation
// scope naming of all five Layers starters.
func TestLayers_SpanNamesAndScopes(t *testing.T) {
	cases := []struct {
		name      string
		start     func(ctx context.Context, component, operation string, fields ...Field) *LayerContext
		wantScope string
		wantSpan  string
		wantKind  trace.SpanKind
		wantLayer string
	}{
		{"StartService", Layers.StartService, "service.user", "user.CreateUser", trace.SpanKindInternal, "service"},
		{"StartHandler", Layers.StartHandler, "handler.user", "user.CreateUser", trace.SpanKindServer, "handler"},
		{"StartRepository", Layers.StartRepository, "repository.user", "user.CreateUser", trace.SpanKindClient, "repository"},
		{"StartOperations", Layers.StartOperations, "operations.user", "user.CreateUser", trace.SpanKindInternal, "operations"},
		{"StartMiddleware", Layers.StartMiddleware, "middleware.user", "user.CreateUser", trace.SpanKindServer, "middleware"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exporter, ctx := newSpanRecorder(t)

			lc := tc.start(ctx, "user", "CreateUser")
			lc.End()

			stub := requireSingleSpan(t, exporter)
			assert.Equal(t, tc.wantSpan, stub.Name, "span name must be {component}.{operation}")
			assert.Equal(t, tc.wantScope, stub.InstrumentationScope.Name, "scope name must be {layer}.{component}")
			assert.Equal(t, tc.wantKind, stub.SpanKind)

			layer, ok := spanAttribute(stub, "layer")
			require.True(t, ok, "starter must set the layer attribute")
			assert.Equal(t, tc.wantLayer, layer.AsString())
		})
	}
}

// TestLayerContext_ErrorSuccessEnd verifies LayerContext error/success/end
// behavior against an in-memory exporter.
func TestLayerContext_ErrorSuccessEnd(t *testing.T) {
	t.Run("Error returns the passed error and marks the span", func(t *testing.T) {
		exporter, ctx := newSpanRecorder(t)

		lc := Layers.StartService(ctx, "user", "CreateUser")
		sentinel := errors.New("unique constraint violation")
		returned := lc.Error(sentinel, "failed to create user", F("user.id", "789"))
		lc.End()

		assert.ErrorIs(t, returned, sentinel)

		stub := requireSingleSpan(t, exporter)
		assert.Equal(t, codes.Error, stub.Status.Code)

		// Fields passed to Error are added as span attributes.
		userID, ok := spanAttribute(stub, "user.id")
		require.True(t, ok)
		assert.Equal(t, "789", userID.AsString())
	})

	t.Run("Success marks the span ok without panic", func(t *testing.T) {
		exporter, ctx := newSpanRecorder(t)

		lc := Layers.StartService(ctx, "user", "CreateUser")
		lc.Success("user created", F("user.id", "123"))
		lc.End()

		stub := requireSingleSpan(t, exporter)
		assert.Equal(t, codes.Ok, stub.Status.Code)
		// Note: the OTel SDK drops the status description for codes.Ok per spec,
		// so only the code is asserted here.

		userID, ok := spanAttribute(stub, "user.id")
		require.True(t, ok)
		assert.Equal(t, "123", userID.AsString())
	})

	t.Run("End exports the span exactly once", func(t *testing.T) {
		exporter, ctx := newSpanRecorder(t)

		lc := Layers.StartRepository(ctx, "user", "FindByID")
		assert.Empty(t, exporter.GetSpans())
		lc.End()

		requireSingleSpan(t, exporter)
	})
}
