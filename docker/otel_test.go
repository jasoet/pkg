package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestOTelInstrumentation_AddSpanAttributes(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		attributes []attribute.KeyValue
		wantAttrs  bool
	}{
		{
			name:    "enabled instrumentation adds attributes",
			enabled: true,
			attributes: []attribute.KeyValue{
				attribute.String("container.id", "abc123"),
				attribute.String("container.image", "nginx:latest"),
			},
			wantAttrs: true,
		},
		{
			name:       "disabled instrumentation ignores attributes",
			enabled:    false,
			attributes: []attribute.KeyValue{attribute.String("test", "value")},
			wantAttrs:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inst *otelInstrumentation
			var exporter *tracetest.InMemoryExporter
			var ctx context.Context

			if tt.enabled {
				// Create in-memory exporter for testing
				exporter = tracetest.NewInMemoryExporter()
				tp := sdktrace.NewTracerProvider(
					sdktrace.WithSyncer(exporter),
				)

				cfg := &otel.Config{
					TracerProvider: tp,
				}
				inst = newOTelInstrumentation(cfg)

				// Start a span so we have something to add attributes to
				var span trace.Span
				ctx, span = inst.startSpan(context.Background(), "test.operation")

				// Add attributes
				inst.addSpanAttributes(ctx, tt.attributes...)

				// End span to flush to exporter
				span.End()
			} else {
				inst = &otelInstrumentation{enabled: false}
				ctx = context.Background()

				// Add attributes (should be no-op)
				inst.addSpanAttributes(ctx, tt.attributes...)
			}

			if tt.wantAttrs && exporter != nil {
				// Check attributes
				spans := exporter.GetSpans()
				if len(spans) == 0 {
					t.Fatal("expected span to be created")
				}

				span := spans[0]
				attrMap := make(map[attribute.Key]attribute.Value)
				for _, attr := range span.Attributes {
					attrMap[attr.Key] = attr.Value
				}

				for _, expectedAttr := range tt.attributes {
					if val, ok := attrMap[expectedAttr.Key]; !ok {
						t.Errorf("expected attribute %s not found", expectedAttr.Key)
					} else if val != expectedAttr.Value {
						t.Errorf("attribute %s = %v, want %v", expectedAttr.Key, val, expectedAttr.Value)
					}
				}
			}
		})
	}
}

func TestOTelInstrumentation_SetSpanStatus(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		code        int
		description string
		wantError   bool
	}{
		{
			name:        "enabled instrumentation sets error status",
			enabled:     true,
			code:        1,
			description: "operation failed",
			wantError:   true,
		},
		{
			name:        "enabled instrumentation sets ok status",
			enabled:     true,
			code:        0,
			description: "operation succeeded",
			wantError:   false,
		},
		{
			name:        "disabled instrumentation ignores status",
			enabled:     false,
			code:        1,
			description: "ignored",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inst *otelInstrumentation
			var exporter *tracetest.InMemoryExporter
			var ctx context.Context

			if tt.enabled {
				// Create in-memory exporter for testing
				exporter = tracetest.NewInMemoryExporter()
				tp := sdktrace.NewTracerProvider(
					sdktrace.WithSyncer(exporter),
				)

				cfg := &otel.Config{
					TracerProvider: tp,
				}
				inst = newOTelInstrumentation(cfg)

				// Start a span
				var span trace.Span
				ctx, span = inst.startSpan(context.Background(), "test.operation")

				// Set span status
				inst.setSpanStatus(ctx, tt.code, tt.description)

				// End span to flush to exporter
				span.End()
			} else {
				inst = &otelInstrumentation{enabled: false}
				ctx = context.Background()

				// Set span status (should be no-op)
				inst.setSpanStatus(ctx, tt.code, tt.description)
			}

			if tt.enabled && exporter != nil {
				spans := exporter.GetSpans()
				if len(spans) == 0 {
					t.Fatal("expected span to be created")
				}

				span := spans[0]

				// Check if status.description attribute is set when code != 0
				if tt.code != 0 {
					found := false
					for _, attr := range span.Attributes {
						if attr.Key == "status.description" {
							found = true
							if attr.Value.AsString() != tt.description {
								t.Errorf("status.description = %v, want %v", attr.Value.AsString(), tt.description)
							}
						}
					}
					if !found && tt.wantError {
						t.Error("expected status.description attribute not found")
					}
				}
			}
		})
	}
}

func TestOTelInstrumentation_RecordError(t *testing.T) {
	// Create in-memory exporter and meter reader
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	cfg := &otel.Config{
		TracerProvider: tp,
		MeterProvider:  mp,
	}
	inst := newOTelInstrumentation(cfg)

	ctx, span := inst.startSpan(context.Background(), "test.operation")

	// Record an error
	testErr := errors.New("test error")
	inst.recordError(ctx, "test_error", testErr)

	// End span to flush to exporter
	span.End()

	// Check span attributes
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected span to be created")
	}

	spanStub := spans[0]
	foundErrorType := false
	foundErrorMsg := false

	for _, attr := range spanStub.Attributes {
		if attr.Key == "error.type" && attr.Value.AsString() == "test_error" {
			foundErrorType = true
		}
		if attr.Key == "error.message" && attr.Value.AsString() == testErr.Error() {
			foundErrorMsg = true
		}
	}

	if !foundErrorType {
		t.Error("expected error.type attribute not found")
	}
	if !foundErrorMsg {
		t.Error("expected error.message attribute not found")
	}

	// Check metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Verify error counter was incremented
	foundCounter := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "docker.container.errors" {
				foundCounter = true
			}
		}
	}

	if !foundCounter {
		t.Error("expected error counter metric not found")
	}
}

func TestOTelInstrumentation_IncrementCounter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	cfg := &otel.Config{
		MeterProvider: mp,
	}
	inst := newOTelInstrumentation(cfg)

	ctx := context.Background()

	// Test all counter types
	counters := []string{
		"containers_started",
		"containers_stopped",
		"containers_terminated",
		"containers_restarted",
	}

	for _, counter := range counters {
		inst.incrementCounter(ctx, counter, 1)
	}

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Verify all counters exist
	expectedMetrics := map[string]bool{
		"docker.containers.started":    false,
		"docker.containers.stopped":    false,
		"docker.containers.terminated": false,
		"docker.containers.restarted":  false,
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if _, ok := expectedMetrics[m.Name]; ok {
				expectedMetrics[m.Name] = true
			}
		}
	}

	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("expected metric %s not found", name)
		}
	}
}

func TestOTelInstrumentation_Disabled(t *testing.T) {
	// Test with nil config
	inst := newOTelInstrumentation(nil)
	if inst.enabled {
		t.Error("expected instrumentation to be disabled with nil config")
	}

	ctx := context.Background()

	// These should all be no-ops and not panic
	_, _ = inst.startSpan(ctx, "test")
	inst.recordError(ctx, "test", errors.New("test error"))
	inst.incrementCounter(ctx, "containers_started", 1)
	inst.addSpanAttributes(ctx, attribute.String("test", "value"))
	inst.setSpanStatus(ctx, 1, "test")
}
