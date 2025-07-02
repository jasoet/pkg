package rest

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type RequestInfo struct {
	Method     string
	URL        string
	Headers    map[string]string
	Body       string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	StatusCode int
	Response   string
	Error      error
	TraceInfo  resty.TraceInfo
}

type Middleware interface {
	BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context
	AfterRequest(ctx context.Context, info RequestInfo)
}

// requestStartTimeKey is a custom type for the context key to avoid collisions
type requestStartTimeKey string

// Define a constant for the request start time key value
const requestStartTimeKeyValue requestStartTimeKey = "rest.request_start_time"

// LoggingMiddleware logs HTTP requests and responses
type LoggingMiddleware struct{}

// NewLoggingMiddleware creates a new LoggingMiddleware instance
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{}
}

// BeforeRequest logs the start of the request and stores the start time in context
func (m *LoggingMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	return context.WithValue(ctx, requestStartTimeKeyValue, time.Now())
}

// AfterRequest logs the completion of the request with timing information
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
