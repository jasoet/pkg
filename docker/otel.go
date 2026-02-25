package docker

import (
	"context"

	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// otelInstrumentation holds OpenTelemetry instrumentation components.
type otelInstrumentation struct {
	tracer  trace.Tracer
	meter   metric.Meter
	config  *otel.Config
	enabled bool

	// Metrics
	containersStarted    metric.Int64Counter
	containersStopped    metric.Int64Counter
	containersTerminated metric.Int64Counter
	containersRestarted  metric.Int64Counter
	containerErrors      metric.Int64Counter
}

// newOTelInstrumentation creates OpenTelemetry instrumentation.
func newOTelInstrumentation(cfg *otel.Config) *otelInstrumentation {
	if cfg == nil {
		return &otelInstrumentation{enabled: false}
	}

	inst := &otelInstrumentation{
		config:  cfg,
		enabled: true,
	}

	// Get tracer
	if cfg.TracerProvider != nil {
		inst.tracer = cfg.TracerProvider.Tracer(
			"github.com/jasoet/pkg/v2/docker",
			trace.WithInstrumentationVersion("v2.0.0"),
		)
	}

	// Get meter and create metrics
	if cfg.MeterProvider != nil {
		inst.meter = cfg.MeterProvider.Meter(
			"github.com/jasoet/pkg/v2/docker",
			metric.WithInstrumentationVersion("v2.0.0"),
		)

		// Create counters (errors intentionally ignored - metrics are optional)
		inst.containersStarted, _ = inst.meter.Int64Counter( //nolint:errcheck
			"docker.containers.started",
			metric.WithDescription("Number of containers started"),
			metric.WithUnit("{container}"),
		)

		inst.containersStopped, _ = inst.meter.Int64Counter( //nolint:errcheck
			"docker.containers.stopped",
			metric.WithDescription("Number of containers stopped"),
			metric.WithUnit("{container}"),
		)

		inst.containersTerminated, _ = inst.meter.Int64Counter( //nolint:errcheck
			"docker.containers.terminated",
			metric.WithDescription("Number of containers terminated"),
			metric.WithUnit("{container}"),
		)

		inst.containersRestarted, _ = inst.meter.Int64Counter( //nolint:errcheck
			"docker.containers.restarted",
			metric.WithDescription("Number of containers restarted"),
			metric.WithUnit("{container}"),
		)

		inst.containerErrors, _ = inst.meter.Int64Counter( //nolint:errcheck
			"docker.container.errors",
			metric.WithDescription("Number of container operation errors"),
			metric.WithUnit("{error}"),
		)
	}

	return inst
}

// startSpan starts a new trace span.
func (i *otelInstrumentation) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !i.enabled || i.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return i.tracer.Start(ctx, name)
}

// recordError records an error metric and adds it to the span.
func (i *otelInstrumentation) recordError(ctx context.Context, errorType string, err error) {
	if !i.enabled {
		return
	}

	// Add error to span
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", errorType),
			attribute.String("error.message", err.Error()),
		)
	}

	// Increment error counter
	if i.containerErrors != nil {
		i.containerErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("error.type", errorType),
			),
		)
	}
}

// incrementCounter increments a counter metric.
func (i *otelInstrumentation) incrementCounter(ctx context.Context, counterName string, value int64) {
	if !i.enabled {
		return
	}

	var counter metric.Int64Counter
	switch counterName {
	case "containers_started":
		counter = i.containersStarted
	case "containers_stopped":
		counter = i.containersStopped
	case "containers_terminated":
		counter = i.containersTerminated
	case "containers_restarted":
		counter = i.containersRestarted
	}

	if counter != nil {
		counter.Add(ctx, value)
	}
}

// addSpanAttributes adds common attributes to the current span.
func (i *otelInstrumentation) addSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if !i.enabled {
		return
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// setSpanStatus sets the span status.
func (i *otelInstrumentation) setSpanStatus(ctx context.Context, code int, description string) {
	if !i.enabled {
		return
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		if code != 0 {
			span.SetStatus(codes.Error, description)
		} else {
			span.SetStatus(codes.Ok, description)
		}
	}
}
