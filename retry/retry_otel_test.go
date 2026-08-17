package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// newRetrySpanRecorder returns an in-memory span exporter and an OTel Config
// whose TracerProvider syncs ended spans to that exporter. Logging is disabled
// to keep test output quiet while still exercising the OTel tracing branches.
func newRetrySpanRecorder(t *testing.T) (*tracetest.InMemoryExporter, *pkgotel.Config) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("retry-test",
		pkgotel.WithTracerProvider(tp),
		pkgotel.WithoutLogging(),
	)
	return exporter, cfg
}

func requireSingleSpan(t *testing.T, exporter *tracetest.InMemoryExporter) tracetest.SpanStub {
	t.Helper()
	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one ended span")
	return spans[0]
}

func TestDo_OTel_SuccessRecordsSpan(t *testing.T) {
	exporter, otelCfg := newRetrySpanRecorder(t)

	cfg := New(
		WithName("otel.success"),
		WithMaxRetries(3),
		WithInitialInterval(5*time.Millisecond),
		WithOTelConfig(otelCfg),
	)

	attempts := 0
	err := Do(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("transient")
		}
		return nil
	})
	assert.NoError(t, err)

	span := requireSingleSpan(t, exporter)
	assert.Equal(t, "otel.success", span.Name)
	assert.Equal(t, codes.Ok, span.Status.Code)

	var attemptsAttr int64 = -1
	for _, kv := range span.Attributes {
		if string(kv.Key) == "retry.attempts" {
			attemptsAttr = kv.Value.AsInt64()
		}
	}
	assert.Equal(t, int64(2), attemptsAttr, "span should record the number of attempts")
}

func TestDo_OTel_FailureRecordsErrorSpan(t *testing.T) {
	exporter, otelCfg := newRetrySpanRecorder(t)

	cfg := New(
		WithName("otel.failure"),
		WithMaxRetries(2),
		WithInitialInterval(5*time.Millisecond),
		WithOTelConfig(otelCfg),
	)

	opErr := errors.New("persistent failure")
	err := Do(context.Background(), cfg, func(ctx context.Context) error {
		return opErr
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, opErr)

	span := requireSingleSpan(t, exporter)
	assert.Equal(t, "otel.failure", span.Name)
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Equal(t, "Operation failed after all retries", span.Status.Description)
	require.NotEmpty(t, span.Events, "the failing error should be recorded on the span")
}

func TestDo_OTel_CancellationRecordsCanceledSpan(t *testing.T) {
	exporter, otelCfg := newRetrySpanRecorder(t)

	ctx, cancel := context.WithCancel(context.Background())
	cfg := New(
		WithName("otel.cancel"),
		WithMaxRetries(5),
		WithInitialInterval(50*time.Millisecond),
		WithOTelConfig(otelCfg),
	)

	attempts := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return errors.New("error")
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	span := requireSingleSpan(t, exporter)
	assert.Equal(t, "otel.cancel", span.Name)
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Equal(t, "Operation canceled", span.Status.Description)
}
