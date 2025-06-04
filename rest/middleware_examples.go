package rest

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// requestStartTimeKey is a custom type for the context key to avoid collisions
type requestStartTimeKey string

// Define a constant for the request start time key value
const requestStartTimeKeyValue requestStartTimeKey = "rest.request_start_time"

type LoggingMiddleware struct{}
type NoOpMiddleware struct{}
type DatabaseLoggingMiddleware struct{}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{}
}

func NewNoOpMiddleware() *NoOpMiddleware {
	return &NoOpMiddleware{}
}

func NewDatabaseLoggingMiddleware() *DatabaseLoggingMiddleware {
	return &DatabaseLoggingMiddleware{}
}

func (m *LoggingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return context.WithValue(ctx, requestStartTimeKeyValue, time.Now())
}

func (m *LoggingMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	logger := log.With().Ctx(ctx).
		Str("method", info.Method).
		Str("url", info.URL).
		Int("status_code", info.StatusCode).
		Dur("duration", info.Duration).
		Logger()

	if info.Error != nil {
		logger.Error().Err(info.Error).Msg("Request failed")
	} else {
		logger.Info().Msg("Request completed")
	}
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
