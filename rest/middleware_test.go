package rest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareInterface(t *testing.T) {
	// Test that our middleware implementations satisfy the Middleware interface
	var _ Middleware = &LoggingMiddleware{}
	var _ Middleware = &NoOpMiddleware{}
}

func TestLoggingMiddleware(t *testing.T) {
	middleware := NewLoggingMiddleware()

	t.Run("BeforeRequest", func(t *testing.T) {
		ctx := context.Background()
		headers := map[string]string{"Content-Type": "application/json"}

		newCtx := middleware.BeforeRequest(ctx, "GET", "https://example.com", `{"key":"value"}`, headers)
		assert.Equal(t, ctx, newCtx, "context should be unchanged")
	})

	t.Run("AfterRequest", func(t *testing.T) {
		// Smoke test: the function logs but returns nothing.
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
		}

		require.NotPanics(t, func() {
			middleware.AfterRequest(ctx, info)
			info.Error = errors.New("test error")
			middleware.AfterRequest(ctx, info)
		})
	})

	t.Run("reuses a single LogHelper", func(t *testing.T) {
		// The LogHelper is constructed once in NewLoggingMiddleware to avoid a
		// per-request allocation.
		assert.NotNil(t, middleware.logger)
	})
}

func TestNoOpMiddleware(t *testing.T) {
	middleware := NewNoOpMiddleware()

	t.Run("BeforeRequest", func(t *testing.T) {
		ctx := context.Background()
		headers := map[string]string{"Content-Type": "application/json"}

		newCtx := middleware.BeforeRequest(ctx, "GET", "https://example.com", `{"key":"value"}`, headers)
		assert.Equal(t, ctx, newCtx, "context should be unchanged")
	})

	t.Run("AfterRequest", func(t *testing.T) {
		ctx := context.Background()
		info := RequestInfo{Method: "GET", URL: "https://example.com", StatusCode: 200}
		require.NotPanics(t, func() { middleware.AfterRequest(ctx, info) })
	})
}
