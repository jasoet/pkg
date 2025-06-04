package rest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMiddlewareInterface(t *testing.T) {
	// Test that our middleware implementations satisfy the Middleware interface
	var _ Middleware = &LoggingMiddleware{}
	var _ Middleware = &NoOpMiddleware{}
	var _ Middleware = &DatabaseLoggingMiddleware{}
}

func TestLoggingMiddleware(t *testing.T) {
	middleware := NewLoggingMiddleware()

	t.Run("BeforeRequest", func(t *testing.T) {
		ctx := context.Background()
		method := "GET"
		url := "https://example.com"
		body := `{"key":"value"}`
		headers := map[string]string{"Content-Type": "application/json"}

		// Call BeforeRequest
		newCtx := middleware.BeforeRequest(ctx, method, url, body, headers)

		// Verify that a start time was added to the context
		startTime, ok := newCtx.Value(requestStartTimeKeyValue).(time.Time)
		if !ok {
			t.Error("Expected request_start_time in context, but it was not found or not a time.Time")
		}

		// Verify the start time is recent (within the last second)
		now := time.Now()
		if now.Sub(startTime) > time.Second {
			t.Errorf("Expected start time to be recent, but it was %v ago", now.Sub(startTime))
		}
	})

	t.Run("AfterRequest", func(t *testing.T) {
		// This is mostly a smoke test since the function logs but doesn't return anything
		ctx := context.Background()
		info := RequestInfo{
			Method:     "GET",
			URL:        "https://example.com",
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"key":"value"}`,
			StartTime:  time.Now().Add(-100 * time.Millisecond),
			EndTime:    time.Now(),
			Duration:   100 * time.Millisecond,
			StatusCode: 200,
			Response:   `{"result":"success"}`,
			Error:      nil,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)

		// Test with error
		info.Error = errors.New("test error")
		middleware.AfterRequest(ctx, info)
	})
}

func TestNoOpMiddleware(t *testing.T) {
	middleware := NewNoOpMiddleware()

	t.Run("BeforeRequest", func(t *testing.T) {
		ctx := context.Background()
		method := "GET"
		url := "https://example.com"
		body := `{"key":"value"}`
		headers := map[string]string{"Content-Type": "application/json"}

		// Call BeforeRequest
		newCtx := middleware.BeforeRequest(ctx, method, url, body, headers)

		// Verify that the context is unchanged
		if newCtx != ctx {
			t.Error("Expected context to be unchanged, but it was modified")
		}
	})

	t.Run("AfterRequest", func(t *testing.T) {
		// This is a smoke test since the function does nothing
		ctx := context.Background()
		info := RequestInfo{
			Method:     "GET",
			URL:        "https://example.com",
			StatusCode: 200,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)
	})
}

func TestDatabaseLoggingMiddleware(t *testing.T) {
	middleware := NewDatabaseLoggingMiddleware()

	t.Run("BeforeRequest", func(t *testing.T) {
		ctx := context.Background()
		method := "GET"
		url := "https://example.com"
		body := `{"key":"value"}`
		headers := map[string]string{"Content-Type": "application/json"}

		// Call BeforeRequest
		newCtx := middleware.BeforeRequest(ctx, method, url, body, headers)

		// Verify that a start time was added to the context
		startTime, ok := newCtx.Value(requestStartTimeKeyValue).(time.Time)
		if !ok {
			t.Error("Expected request_start_time in context, but it was not found or not a time.Time")
		}

		// Verify the start time is recent (within the last second)
		now := time.Now()
		if now.Sub(startTime) > time.Second {
			t.Errorf("Expected start time to be recent, but it was %v ago", now.Sub(startTime))
		}
	})

	t.Run("AfterRequest", func(t *testing.T) {
		// This is mostly a smoke test since the function logs but doesn't return anything
		ctx := context.Background()
		info := RequestInfo{
			Method:     "GET",
			URL:        "https://example.com",
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"key":"value"}`,
			StartTime:  time.Now().Add(-100 * time.Millisecond),
			EndTime:    time.Now(),
			Duration:   100 * time.Millisecond,
			StatusCode: 200,
			Response:   `{"result":"success"}`,
			Error:      nil,
		}

		// Should not panic
		middleware.AfterRequest(ctx, info)
	})
}
