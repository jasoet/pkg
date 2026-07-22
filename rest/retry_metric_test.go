package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/jasoet/pkg/v3/otel"
)

// This test covers status-based retries (5xx responses). Note the resty retry
// hook also fires on the final failed attempt, not just on intermediate ones.
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
	if err != nil {
		t.Fatalf("expected request to succeed after retries, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected 3 server calls (1 initial + 2 retries), got %d", got)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	var retryTotal int64
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.client.retry.count" {
				continue
			}
			found = true
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("expected Sum[int64] data for retry counter, got %T", m.Data)
			}
			for _, dp := range sum.DataPoints {
				retryTotal += dp.Value
			}
		}
	}

	if !found {
		t.Fatal("http.client.retry.count metric not found")
	}
	if retryTotal != 2 {
		t.Errorf("expected retry counter == 2, got %d", retryTotal)
	}
}
