//go:build example

package main

import (
	"context"
	"github.com/jasoet/pkg/rest"
	"time"

	"github.com/rs/zerolog/log"
)

// requestStartTimeKey is a custom type for the context key to avoid collisions
type requestStartTimeKey string

// Define a constant for the request start time key value
const requestStartTimeKeyValue requestStartTimeKey = "rest.request_start_time"

// Example middleware implementations for demonstration purposes

type NoOpMiddleware struct{}
type DatabaseLoggingMiddleware struct{}

func NewNoOpMiddleware() *NoOpMiddleware {
	return &NoOpMiddleware{}
}

func NewDatabaseLoggingMiddleware() *DatabaseLoggingMiddleware {
	return &DatabaseLoggingMiddleware{}
}

func (m *NoOpMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return ctx
}

func (m *NoOpMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
}

func (m *DatabaseLoggingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return context.WithValue(ctx, requestStartTimeKeyValue, time.Now())
}

func (m *DatabaseLoggingMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
	logger := log.With().Ctx(ctx).
		Str("method", info.Method).
		Str("url", info.URL).
		Int("status_code", info.StatusCode).
		Dur("duration", info.Duration).
		Logger()

	logger.Info().Msg("Would log request to database")
}
