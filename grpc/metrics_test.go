package grpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsManager(t *testing.T) {
	mm := NewMetricsManager("test_server")

	assert.NotNil(t, mm)
	assert.NotNil(t, mm.registry)
	assert.NotNil(t, mm.grpcRequestsTotal)
	assert.NotNil(t, mm.httpRequestsTotal)
}

func TestMetricsManagerDefaultNamespace(t *testing.T) {
	mm := NewMetricsManager("")

	// Should use default namespace
	assert.NotNil(t, mm)
}

func TestRecordGRPCRequest(t *testing.T) {
	mm := NewMetricsManager("test")

	// Record a gRPC request
	method := "GetTask"
	statusCode := "OK"
	duration := 100 * time.Millisecond
	requestSize := 256
	responseSize := 512

	mm.RecordGRPCRequest(method, statusCode, duration, requestSize, responseSize)

	// Gather metrics to verify
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify metrics were recorded
	found := make(map[string]bool)
	for _, mf := range metricFamilies {
		name := mf.GetName()
		switch name {
		case "test_grpc_requests_total":
			found["requests"] = true
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected gRPC requests metric to have values")
			}
		case "test_grpc_request_duration_seconds":
			found["duration"] = true
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected gRPC duration metric to have values")
			}
		case "test_grpc_request_size_bytes":
			found["request_size"] = true
		case "test_grpc_response_size_bytes":
			found["response_size"] = true
		}
	}

	if !found["requests"] {
		t.Error("Expected gRPC requests metric to be recorded")
	}

	if !found["duration"] {
		t.Error("Expected gRPC duration metric to be recorded")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	mm := NewMetricsManager("test")

	// Record an HTTP request
	method := "GET"
	path := "/api/tasks"
	statusCode := 200
	duration := 50 * time.Millisecond
	requestSize := 128
	responseSize := 1024

	mm.RecordHTTPRequest(method, path, statusCode, duration, requestSize, responseSize)

	// Gather metrics to verify
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify HTTP metrics were recorded
	found := make(map[string]bool)
	for _, mf := range metricFamilies {
		name := mf.GetName()
		switch name {
		case "test_http_requests_total":
			found["requests"] = true
		case "test_http_request_duration_seconds":
			found["duration"] = true
		case "test_http_request_size_bytes":
			found["request_size"] = true
		case "test_http_response_size_bytes":
			found["response_size"] = true
		}
	}

	if !found["requests"] {
		t.Error("Expected HTTP requests metric to be recorded")
	}

	if !found["duration"] {
		t.Error("Expected HTTP duration metric to be recorded")
	}
}

func TestGRPCConnectionTracking(t *testing.T) {
	mm := NewMetricsManager("test")

	// Test increment
	mm.IncrementGRPCConnections()
	mm.IncrementGRPCConnections()

	// Test decrement
	mm.DecrementGRPCConnections()

	// Gather metrics
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the active connections metric
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_grpc_active_connections" {
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected active connections metric to have values")
				return
			}

			metric := mf.GetMetric()[0]
			if metric.GetGauge().GetValue() != 1.0 {
				t.Errorf("Expected active connections to be 1, got %f",
					metric.GetGauge().GetValue())
			}
			return
		}
	}

	t.Error("Active connections metric not found")
}

func TestHTTPRequestTracking(t *testing.T) {
	mm := NewMetricsManager("test")

	// Test increment
	mm.IncrementHTTPRequests()
	mm.IncrementHTTPRequests()

	// Test decrement
	mm.DecrementHTTPRequests()

	// Gather metrics
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the active requests metric
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_http_active_requests" {
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected active requests metric to have values")
				return
			}

			metric := mf.GetMetric()[0]
			if metric.GetGauge().GetValue() != 1.0 {
				t.Errorf("Expected active requests to be 1, got %f",
					metric.GetGauge().GetValue())
			}
			return
		}
	}

	t.Error("Active requests metric not found")
}

func TestUpdateUptime(t *testing.T) {
	mm := NewMetricsManager("test")

	// Wait a short time to ensure uptime is measurable
	time.Sleep(10 * time.Millisecond)

	// Update uptime
	mm.UpdateUptime()

	// Gather metrics
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the uptime metric
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_uptime_seconds" {
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected uptime metric to have values")
				return
			}

			metric := mf.GetMetric()[0]
			uptime := metric.GetGauge().GetValue()
			if uptime <= 0 {
				t.Errorf("Expected uptime to be positive, got %f", uptime)
			}
			return
		}
	}

	t.Error("Uptime metric not found")
}

func TestCreateMetricsHandler(t *testing.T) {
	mm := NewMetricsManager("test")

	// Record some test data
	mm.RecordGRPCRequest("TestMethod", "OK", 100*time.Millisecond, 256, 512)
	mm.RecordHTTPRequest("GET", "/test", 200, 50*time.Millisecond, 128, 1024)

	// Create HTTP handler
	handler := mm.CreateMetricsHandler()

	// Test the handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check that metrics are present in the output
	expectedMetrics := []string{
		"test_grpc_requests_total",
		"test_http_requests_total",
		"test_grpc_request_duration_seconds",
		"test_http_request_duration_seconds",
		"test_uptime_seconds",
		"test_start_time_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected metric %s to be present in output", metric)
		}
	}
}

func TestHTTPMetricsMiddleware(t *testing.T) {
	mm := NewMetricsManager("test")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with metrics middleware
	middleware := mm.HTTPMetricsMiddleware(testHandler)

	// Test the middleware
	req := httptest.NewRequest("GET", "/test/path", strings.NewReader("test request"))
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify that metrics were recorded
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check for HTTP request metrics
	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_http_requests_total" {
			found = true
			if len(mf.GetMetric()) == 0 {
				t.Error("Expected HTTP requests metric to have values")
			}
			break
		}
	}

	if !found {
		t.Error("Expected HTTP requests metric to be recorded by middleware")
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		size:           0,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", rw.statusCode)
	}

	// Test Write
	testData := []byte("test response data")
	n, err := rw.Write(testData)
	if err != nil {
		t.Errorf("Unexpected error writing data: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	if rw.size != len(testData) {
		t.Errorf("Expected size to be %d, got %d", len(testData), rw.size)
	}
}

func TestMetricsRegistry(t *testing.T) {
	mm := NewMetricsManager("test")

	registry := mm.GetRegistry()
	if registry == nil {
		t.Error("Expected registry to be returned")
	}

	if registry != mm.registry {
		t.Error("Expected returned registry to match internal registry")
	}
}

func TestMetricsWithZeroSizes(t *testing.T) {
	mm := NewMetricsManager("test")

	// Record metrics with zero sizes (should be handled gracefully)
	mm.RecordGRPCRequest("TestMethod", "OK", 100*time.Millisecond, 0, 0)
	mm.RecordHTTPRequest("GET", "/test", 200, 50*time.Millisecond, 0, 0)

	// Should not panic or error
	metricFamilies, err := mm.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Error("Expected some metrics to be present")
	}
}
