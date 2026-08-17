package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/jasoet/pkg/v3/otel"
)

// sumRetryCounter collects the http.client.retry.count metric and returns its total.
func sumRetryCounter(t *testing.T, reader sdkmetric.Reader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	var total int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.client.retry.count" {
				continue
			}
			found = true
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "expected Sum[int64] data for retry counter, got %T", m.Data)
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
		}
	}
	require.True(t, found, "http.client.retry.count metric not found")
	return total
}

// TestRetryMetricWiring verifies that the http.client.retry.count counter is
// actually incremented when resty retries a failed request. The server fails
// with 500 twice, then succeeds; with RetryCount=2 the counter must be 2.
func TestRetryMetricWiring(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() { _ = mp.Shutdown(context.Background()) }()

	otelCfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))

	restConfig := DefaultRestConfig()
	restConfig.RetryCount = 2
	restConfig.RetryWaitTime = time.Millisecond
	restConfig.RetryMaxWaitTime = 5 * time.Millisecond
	restConfig.OTelConfig = otelCfg

	client := NewClient(WithRestConfig(*restConfig))

	resp, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(3), calls.Load(), "expected 3 server calls (1 initial + 2 retries)")

	assert.Equal(t, int64(2), sumRetryCounter(t, reader))
}

// TestRetryMetricAllAttemptsFail verifies that when every attempt fails (the
// server always returns 500), the retry counter records only the retries
// actually performed, not resty's extra hook fire after the final attempt.
// With RetryCount=2 the counter must be 2, not 3.
func TestRetryMetricAllAttemptsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() { _ = mp.Shutdown(context.Background()) }()

	otelCfg := otel.NewConfig("test-service", otel.WithMeterProvider(mp))

	restConfig := DefaultRestConfig()
	restConfig.RetryCount = 2
	restConfig.RetryWaitTime = time.Millisecond
	restConfig.RetryMaxWaitTime = 5 * time.Millisecond
	restConfig.OTelConfig = otelCfg

	client := NewClient(WithRestConfig(*restConfig))

	resp, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.Error(t, err, "expected a typed error for the persistent 500 response")
	require.NotNil(t, resp)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	assert.Equal(t, int64(2), sumRetryCounter(t, reader), "only performed retries must be counted")
}
