package builder

import (
	"context"
	"errors"
	"testing"

	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	noopt "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewOTelInstrumentation(t *testing.T) {
	t.Run("creates disabled instrumentation with nil config", func(t *testing.T) {
		inst := newOTelInstrumentation(nil)
		require.NotNil(t, inst)
		assert.False(t, inst.enabled)
		assert.Nil(t, inst.tracer)
		assert.Nil(t, inst.meter)
	})

	t.Run("creates enabled instrumentation with config", func(t *testing.T) {
		tracerProvider := noopt.NewTracerProvider()
		meterProvider := sdkmetric.NewMeterProvider()

		cfg := otel.NewConfig("test-service").
			WithTracerProvider(tracerProvider).
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		require.NotNil(t, inst)
		assert.True(t, inst.enabled)
		assert.NotNil(t, inst.tracer)
		assert.NotNil(t, inst.meter)
		assert.NotNil(t, inst.workflowsBuilt)
		assert.NotNil(t, inst.templatesAdded)
		assert.NotNil(t, inst.sourcesAdded)
		assert.NotNil(t, inst.buildDuration)
	})

	t.Run("creates instrumentation with tracer only", func(t *testing.T) {
		tracerProvider := noopt.NewTracerProvider()
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(tracerProvider)

		inst := newOTelInstrumentation(cfg)
		require.NotNil(t, inst)
		assert.True(t, inst.enabled)
		assert.NotNil(t, inst.tracer)
		assert.Nil(t, inst.meter)
	})

	t.Run("creates instrumentation with meter only", func(t *testing.T) {
		meterProvider := sdkmetric.NewMeterProvider()
		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		require.NotNil(t, inst)
		assert.True(t, inst.enabled)
		assert.Nil(t, inst.tracer)
		assert.NotNil(t, inst.meter)
	})
}

func TestStartSpan(t *testing.T) {
	t.Run("returns no-op span when disabled", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: false}
		ctx := context.Background()

		newCtx, span := inst.startSpan(ctx, "test-span")
		assert.Equal(t, ctx, newCtx)
		assert.NotNil(t, span)
		assert.False(t, span.IsRecording())
	})

	t.Run("returns no-op span when tracer is nil", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: true, tracer: nil}
		ctx := context.Background()

		newCtx, span := inst.startSpan(ctx, "test-span")
		assert.Equal(t, ctx, newCtx)
		assert.NotNil(t, span)
	})

	t.Run("creates span when enabled with tracer", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tracerProvider := noopt.NewTracerProvider(
			noopt.WithSyncer(exporter),
		)

		cfg := otel.NewConfig("test-service").
			WithTracerProvider(tracerProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		newCtx, span := inst.startSpan(ctx, "test-operation")
		require.NotNil(t, span)
		assert.True(t, span.IsRecording())

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-operation", spans[0].Name)
		assert.NotEqual(t, ctx, newCtx)
	})
}

func TestRecordError(t *testing.T) {
	t.Run("does nothing when disabled", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: false}
		ctx := context.Background()
		testErr := errors.New("test error")

		// Should not panic
		inst.recordError(ctx, "test_error", testErr)
	})

	t.Run("records error with span and metrics", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tracerProvider := noopt.NewTracerProvider(
			noopt.WithSyncer(exporter),
		)

		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithTracerProvider(tracerProvider).
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		// Start a span
		ctx, span := inst.startSpan(ctx, "test-op")
		testErr := errors.New("test error occurred")

		inst.recordError(ctx, "validation_error", testErr)
		span.End()

		// Check span recorded error
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.True(t, len(spans[0].Events) > 0)

		// Check metrics
		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})
}

func TestIncrementCounter(t *testing.T) {
	t.Run("does nothing when disabled", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: false}
		ctx := context.Background()

		// Should not panic
		inst.incrementCounter(ctx, "workflows_built", 1)
	})

	t.Run("increments workflows_built counter", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.incrementCounter(ctx, "workflows_built", 1)
		inst.incrementCounter(ctx, "workflows_built", 2)

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("increments templates_added counter", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.incrementCounter(ctx, "templates_added", 5)

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("increments sources_added counter", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.incrementCounter(ctx, "sources_added", 3)

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("increments counter with attributes", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.incrementCounter(ctx, "workflows_built", 1,
			attribute.String("namespace", "argo"),
			attribute.String("type", "deployment"))

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("handles unknown counter name", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		// Should not panic with unknown counter
		inst.incrementCounter(ctx, "unknown_counter", 1)
	})
}

func TestRecordDuration(t *testing.T) {
	t.Run("does nothing when disabled", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: false}
		ctx := context.Background()

		// Should not panic
		inst.recordDuration(ctx, "build_duration", 123.45)
	})

	t.Run("records build_duration histogram", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.recordDuration(ctx, "build_duration", 123.45)
		inst.recordDuration(ctx, "build_duration", 234.56)

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("records duration with attributes", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		inst.recordDuration(ctx, "build_duration", 100.0,
			attribute.String("workflow", "test-workflow"),
			attribute.Int("templates", 5))

		var rm metricdata.ResourceMetrics
		err := reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("handles unknown metric name", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
		)

		cfg := otel.NewConfig("test-service").
			WithMeterProvider(meterProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		// Should not panic with unknown metric
		inst.recordDuration(ctx, "unknown_metric", 100.0)
	})
}

func TestAddSpanAttributes(t *testing.T) {
	t.Run("does nothing when disabled", func(t *testing.T) {
		inst := &otelInstrumentation{enabled: false}
		ctx := context.Background()

		// Should not panic
		inst.addSpanAttributes(ctx, attribute.String("key", "value"))
	})

	t.Run("adds attributes to active span", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tracerProvider := noopt.NewTracerProvider(
			noopt.WithSyncer(exporter),
		)

		cfg := otel.NewConfig("test-service").
			WithTracerProvider(tracerProvider)

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		ctx, span := inst.startSpan(ctx, "test-op")
		inst.addSpanAttributes(ctx,
			attribute.String("workflow.name", "test-workflow"),
			attribute.Int("workflow.templates", 3),
			attribute.Bool("workflow.has_exit_handler", true))
		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		attrs := spans[0].Attributes

		assert.Contains(t, attrs, attribute.String("workflow.name", "test-workflow"))
		assert.Contains(t, attrs, attribute.Int("workflow.templates", 3))
		assert.Contains(t, attrs, attribute.Bool("workflow.has_exit_handler", true))
	})

	t.Run("does nothing when no active span", func(t *testing.T) {
		cfg := otel.NewConfig("test-service").
			WithTracerProvider(noopt.NewTracerProvider())

		inst := newOTelInstrumentation(cfg)
		ctx := context.Background()

		// Should not panic when no active span
		inst.addSpanAttributes(ctx, attribute.String("key", "value"))
	})
}
