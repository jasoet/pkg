package rest

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
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
