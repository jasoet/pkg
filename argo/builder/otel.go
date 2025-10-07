package builder

import (
	"context"

	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// otelInstrumentation holds OpenTelemetry instrumentation components for the workflow builder.
// It provides tracing, metrics, and logging capabilities for workflow build operations.
type otelInstrumentation struct {
	tracer  trace.Tracer
	meter   metric.Meter
	config  *otel.Config
	enabled bool

	// Metrics counters
	workflowsBuilt       metric.Int64Counter
	workflowsBuildErrors metric.Int64Counter
	templatesAdded       metric.Int64Counter
	sourcesAdded         metric.Int64Counter

	// Metrics histograms
	buildDuration metric.Float64Histogram
}

// newOTelInstrumentation creates OpenTelemetry instrumentation for the workflow builder.
// Returns a disabled instrumentation if cfg is nil.
func newOTelInstrumentation(cfg *otel.Config) *otelInstrumentation {
	if cfg == nil {
		return &otelInstrumentation{enabled: false}
	}

	inst := &otelInstrumentation{
		config:  cfg,
		enabled: true,
	}

	// Get tracer for distributed tracing
	if cfg.TracerProvider != nil {
		inst.tracer = cfg.TracerProvider.Tracer(
			"github.com/jasoet/pkg/v2/argo/builder",
			trace.WithInstrumentationVersion("v2.0.0"),
		)
	}

	// Get meter and create metrics
	if cfg.MeterProvider != nil {
		inst.meter = cfg.MeterProvider.Meter(
			"github.com/jasoet/pkg/v2/argo/builder",
			metric.WithInstrumentationVersion("v2.0.0"),
		)

		// Create counter metrics (errors intentionally ignored - metrics are optional)
		inst.workflowsBuilt, _ = inst.meter.Int64Counter( //nolint:errcheck
			"argo.workflows.built",
			metric.WithDescription("Number of workflows successfully built"),
			metric.WithUnit("{workflow}"),
		)

		inst.workflowsBuildErrors, _ = inst.meter.Int64Counter( //nolint:errcheck
			"argo.workflows.build_errors",
			metric.WithDescription("Number of workflow build errors"),
			metric.WithUnit("{error}"),
		)

		inst.templatesAdded, _ = inst.meter.Int64Counter( //nolint:errcheck
			"argo.workflows.templates_added",
			metric.WithDescription("Number of templates added to workflows"),
			metric.WithUnit("{template}"),
		)

		inst.sourcesAdded, _ = inst.meter.Int64Counter( //nolint:errcheck
			"argo.workflows.sources_added",
			metric.WithDescription("Number of workflow sources added"),
			metric.WithUnit("{source}"),
		)

		// Create histogram metrics
		inst.buildDuration, _ = inst.meter.Float64Histogram( //nolint:errcheck
			"argo.workflows.build_duration",
			metric.WithDescription("Workflow build duration in milliseconds"),
			metric.WithUnit("ms"),
		)
	}

	return inst
}

// startSpan starts a new trace span if tracing is enabled.
// Returns the original context and a no-op span if tracing is disabled.
func (i *otelInstrumentation) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !i.enabled || i.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return i.tracer.Start(ctx, name)
}

// recordError records an error metric and adds it to the active span.
func (i *otelInstrumentation) recordError(ctx context.Context, errorType string, err error) {
	if !i.enabled {
		return
	}

	// Add error to active span
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", errorType),
			attribute.String("error.message", err.Error()),
		)
	}

	// Increment error counter
	if i.workflowsBuildErrors != nil {
		i.workflowsBuildErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("error.type", errorType),
			),
		)
	}
}

// incrementCounter increments a named counter metric.
func (i *otelInstrumentation) incrementCounter(ctx context.Context, counterName string, value int64, attrs ...attribute.KeyValue) {
	if !i.enabled {
		return
	}

	var counter metric.Int64Counter
	switch counterName {
	case "workflows_built":
		counter = i.workflowsBuilt
	case "workflows_build_errors":
		counter = i.workflowsBuildErrors
	case "templates_added":
		counter = i.templatesAdded
	case "sources_added":
		counter = i.sourcesAdded
	}

	if counter != nil {
		if len(attrs) > 0 {
			counter.Add(ctx, value, metric.WithAttributes(attrs...))
		} else {
			counter.Add(ctx, value)
		}
	}
}

// recordDuration records a duration histogram metric.
func (i *otelInstrumentation) recordDuration(ctx context.Context, metricName string, durationMs float64, attrs ...attribute.KeyValue) {
	if !i.enabled {
		return
	}

	var histogram metric.Float64Histogram
	switch metricName {
	case "build_duration":
		histogram = i.buildDuration
	}

	if histogram != nil {
		if len(attrs) > 0 {
			histogram.Record(ctx, durationMs, metric.WithAttributes(attrs...))
		} else {
			histogram.Record(ctx, durationMs)
		}
	}
}

// addSpanAttributes adds attributes to the active span.
func (i *otelInstrumentation) addSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if !i.enabled {
		return
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}
