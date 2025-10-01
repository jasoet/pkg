//go:build example

package rest

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// Example middleware implementations for demonstration purposes

type (
	NoOpMiddleware            struct{}
	DatabaseLoggingMiddleware struct{}
)

func NewNoOpMiddleware() *NoOpMiddleware {
	return &NoOpMiddleware{}
}

func NewDatabaseLoggingMiddleware() *DatabaseLoggingMiddleware {
	return &DatabaseLoggingMiddleware{}
}

func (m *NoOpMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return ctx
}

func (m *NoOpMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
}

func (m *DatabaseLoggingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return context.WithValue(ctx, requestStartTimeKeyValue, time.Now())
}

func (m *DatabaseLoggingMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	logger := log.With().Ctx(ctx).
		Str("method", info.Method).
		Str("url", info.URL).
		Int("status_code", info.StatusCode).
		Dur("duration", info.Duration).
		Logger()

	logger.Info().Msg("Would log request to database")
}
