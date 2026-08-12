// Package rest provides an HTTP client with middleware support, retry logic,
// and optional OpenTelemetry instrumentation.
package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/jasoet/pkg/v3/otel"
)

// Client wraps a resty HTTP client with middleware and OTel support.
type Client struct {
	restClient  *resty.Client
	restConfig  *Config
	middlewares []Middleware
	mu          sync.RWMutex

	// otelConfig and retryNonIdempotent hold values supplied by WithOTelConfig
	// and WithRetryNonIdempotent. They are merged into restConfig after all
	// options run, so those options are order-independent with WithRestConfig
	// (which replaces restConfig wholesale).
	otelConfig         *otel.Config
	retryNonIdempotent *bool
}

// ClientOption configures a Client during construction.
type ClientOption func(*Client)

// WithRestConfig sets the REST client configuration.
//
// A previously configured OTel config (via WithOTelConfig) or retry-idempotency
// override (via WithRetryNonIdempotent) is preserved regardless of option order:
// those values are merged into the configuration after all options run.
func WithRestConfig(restConfig Config) ClientOption {
	return func(client *Client) {
		client.restConfig = &restConfig
	}
}

// WithMiddleware appends a single middleware to the existing middleware chain.
// The lock is not held here because option functions run only during NewClient construction.
func WithMiddleware(middleware Middleware) ClientOption {
	return func(client *Client) {
		client.middlewares = append(client.middlewares, middleware)
	}
}

// WithMiddlewares replaces the entire middleware chain with the provided middlewares.
// Use WithMiddleware to append instead.
// The lock is not held here because option functions run only during NewClient construction.
func WithMiddlewares(middlewares ...Middleware) ClientOption {
	return func(client *Client) {
		client.middlewares = middlewares
	}
}

// WithOTelConfig sets the OpenTelemetry configuration for the REST client.
// When set, adds OTel tracing, metrics, and logging middleware automatically.
//
// The config is stored on the Client and merged into the REST configuration
// after all options run, so this option is order-independent with respect to
// WithRestConfig. Passing nil is a no-op (it does not clear a config supplied
// via WithRestConfig).
func WithOTelConfig(cfg *otel.Config) ClientOption {
	return func(client *Client) {
		if cfg != nil {
			client.otelConfig = cfg
		}
	}
}

// WithRetryNonIdempotent opts in to retrying non-idempotent HTTP methods
// (POST, PATCH, and any custom method). By default only idempotent methods
// (GET, HEAD, PUT, DELETE, OPTIONS) are retried, because retrying a
// non-idempotent request can duplicate side effects (e.g. a double charge).
//
// Like WithOTelConfig, this is order-independent with WithRestConfig.
func WithRetryNonIdempotent() ClientOption {
	return func(client *Client) {
		v := true
		client.retryNonIdempotent = &v
	}
}

// truncateBody limits the body string to maxLen bytes, appending "...(truncated)" if truncated.
// If maxLen is 0 or negative, the full body is returned unchanged.
func truncateBody(body string, maxLen int) string {
	if maxLen > 0 && len(body) > maxLen {
		return body[:maxLen] + "...(truncated)"
	}
	return body
}

// isIdempotentMethod reports whether an HTTP method is safe to retry per
// RFC 7231: GET, HEAD, PUT, DELETE, and OPTIONS are idempotent.
func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

// isRetryableError classifies a transport-level error as transient (retryable)
// or permanent. Permanent failures — malformed URL or unsupported scheme,
// x509/TLS certificate problems, and context cancellation or deadline — will
// not succeed on retry, so they are excluded to avoid wasting the backoff budget.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation / deadline (including the client Timeout) is permanent.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// TLS/x509 certificate verification failures will not be fixed by retrying.
	var certVerifyErr *tls.CertificateVerificationError
	var x509UnknownAuthority x509.UnknownAuthorityError
	var x509Hostname x509.HostnameError
	var x509Invalid x509.CertificateInvalidError
	if errors.As(err, &certVerifyErr) ||
		errors.As(err, &x509UnknownAuthority) ||
		errors.As(err, &x509Hostname) ||
		errors.As(err, &x509Invalid) {
		return false
	}

	// net/http wraps every client-side failure in *url.Error. A url.Error whose
	// cause is neither a network operation error nor a net.Error is a permanent
	// client-side problem (unsupported scheme, malformed URL) and must not retry.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		var netErr net.Error
		var opErr *net.OpError
		if !errors.As(urlErr.Err, &netErr) && !errors.As(urlErr.Err, &opErr) {
			return false
		}
	}

	return true
}

// parseRetryAfter parses a Retry-After header value, supporting both the
// delta-seconds and HTTP-date forms. It returns 0 when the header is absent or
// unparseable, signaling the caller to fall back to the default backoff.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// NewClient creates a new REST client with the given options.
// For custom TLS configuration, use GetRestClient() to access the underlying resty client
// and call SetTLSClientConfig().
func NewClient(options ...ClientOption) *Client {
	client := &Client{
		restConfig:  DefaultRestConfig(),
		middlewares: []Middleware{NewLoggingMiddleware()}, // Default middleware
	}

	for _, option := range options {
		option(client)
	}

	// Merge order-independent options into the (possibly replaced) restConfig.
	// This runs after every option so WithOTelConfig/WithRetryNonIdempotent are
	// not silently discarded by a later WithRestConfig.
	if client.restConfig == nil {
		client.restConfig = DefaultRestConfig()
	}
	if client.otelConfig != nil {
		client.restConfig.OTelConfig = client.otelConfig
	}
	if client.retryNonIdempotent != nil {
		client.restConfig.RetryNonIdempotent = *client.retryNonIdempotent
	}

	// Add OTel middleware if configured (prepend to user middleware)
	var metricsMW *OTelMetricsMiddleware
	if client.restConfig.OTelConfig != nil {
		// Save user-provided middlewares
		userMiddlewares := make([]Middleware, len(client.middlewares))
		copy(userMiddlewares, client.middlewares)

		// Reset and add OTel middleware first
		client.middlewares = []Middleware{}

		// Add OTel middleware in order: tracing -> metrics -> logging
		if tracingMW := NewOTelTracingMiddleware(client.restConfig.OTelConfig); tracingMW != nil {
			client.middlewares = append(client.middlewares, tracingMW)
		}
		if m := NewOTelMetricsMiddleware(client.restConfig.OTelConfig); m != nil {
			client.middlewares = append(client.middlewares, m)
			metricsMW = m
		}
		if loggingMW := NewOTelLoggingMiddleware(client.restConfig.OTelConfig); loggingMW != nil {
			client.middlewares = append(client.middlewares, loggingMW)
		}

		// Append user-provided middlewares (excluding default LoggingMiddleware)
		for _, mw := range userMiddlewares {
			// Skip default LoggingMiddleware as OTel provides logging
			if _, isLogging := mw.(*LoggingMiddleware); !isLogging {
				client.middlewares = append(client.middlewares, mw)
			}
		}
	}

	httpClient := resty.New()
	httpClient.
		SetRetryCount(client.restConfig.RetryCount).
		SetRetryWaitTime(client.restConfig.RetryWaitTime).
		SetRetryMaxWaitTime(client.restConfig.RetryMaxWaitTime).
		SetTimeout(client.restConfig.Timeout).
		// Honor a caller-supplied body on GET/HEAD/OPTIONS so it is actually
		// transmitted rather than silently dropped while metrics still record
		// its size.
		SetAllowGetMethodPayload(true)

	retryNonIdempotent := client.restConfig.RetryNonIdempotent
	httpClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		// Only retry idempotent methods unless the caller opted in. Retrying a
		// non-idempotent request (POST/PATCH) risks duplicating side effects.
		method := ""
		if r != nil && r.Request != nil {
			method = r.Request.Method
		}
		if !retryNonIdempotent && method != "" && !isIdempotentMethod(method) {
			return false
		}

		// Transport-level failure: retry only transient errors. Permanent
		// failures (bad URL/scheme, x509/TLS, context canceled/deadline) will
		// never succeed on retry and would only waste the backoff budget.
		if err != nil {
			return isRetryableError(err)
		}

		// Status-based retry: 5xx server errors and 429 Too Many Requests.
		if r == nil {
			return false
		}
		status := r.StatusCode()
		return status == http.StatusTooManyRequests || status >= 500
	})

	// Honor a Retry-After header (delta-seconds or HTTP-date) when present.
	// Returning 0 lets resty fall back to its jittered exponential backoff.
	httpClient.SetRetryAfter(func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
		if resp == nil {
			return 0, nil
		}
		return parseRetryAfter(resp.Header().Get("Retry-After")), nil
	})

	// Wire the retry counter into resty's retry hook so it actually increments.
	// The hook fires on both transport errors and status-based retries; resp is
	// nil for transport errors before a response was received. resty fires the
	// hook once more after the final attempt fails (with Attempt > RetryCount);
	// skip that extra fire so the counter only counts retries actually performed.
	// Nil-resp transport-error fires cannot be filtered this way and are counted.
	if metricsMW != nil {
		httpClient.AddRetryHook(func(resp *resty.Response, _ error) {
			if resp != nil && resp.Request != nil && resp.Request.Attempt > client.restConfig.RetryCount {
				return
			}
			ctx := context.Background()
			method := "UNKNOWN"
			attempt := 0
			if resp != nil && resp.Request != nil {
				method = resp.Request.Method
				attempt = resp.Request.Attempt
				if reqCtx := resp.Request.Context(); reqCtx != nil {
					ctx = reqCtx
				}
			}
			metricsMW.recordRetry(ctx, method, attempt)
		})
	}

	client.restClient = httpClient

	return client
}

// GetRestClient returns the underlying resty client.
// Mutations to this client after NewClient returns are not thread-safe for
// concurrent use with doRequest.
func (c *Client) GetRestClient() *resty.Client {
	return c.restClient
}

// GetRestConfig returns a copy of the current REST configuration.
func (c *Client) GetRestConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	configCopy := *c.restConfig
	return &configCopy
}

// AddMiddleware appends a middleware to the chain.
func (c *Client) AddMiddleware(middleware Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware)
}

// SetMiddlewares replaces the entire middleware chain.
//
// Warning: this replaces every middleware, including the OTel tracing, metrics,
// and logging middlewares that NewClient installs automatically when an OTel
// config is provided. After calling SetMiddlewares those are gone; use
// AddMiddleware to append without disturbing the existing chain, or re-add the
// OTel middlewares explicitly if you need them.
func (c *Client) SetMiddlewares(middlewares ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = middlewares
}

// GetMiddlewares returns a copy of the current middleware chain.
func (c *Client) GetMiddlewares() []Middleware {
	c.mu.RLock()
	defer c.mu.RUnlock()
	middlewaresCopy := make([]Middleware, len(c.middlewares))
	copy(middlewaresCopy, c.middlewares)
	return middlewaresCopy
}

// MakeRequestWithTrace executes an HTTP request with resty trace enabled.
//
// The body parameter is a string; for binary payloads, use GetRestClient()
// and build the request directly with resty's SetBody(interface{}).
func (c *Client) MakeRequestWithTrace(ctx context.Context, method string, url string, body string, headers map[string]string) (*Response, error) {
	return c.doRequest(ctx, method, url, body, headers, true)
}

// MakeRequest executes an HTTP request without resty trace.
//
// The body parameter is a string; for binary payloads, use GetRestClient()
// and build the request directly with resty's SetBody(interface{}).
func (c *Client) MakeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) (*Response, error) {
	return c.doRequest(ctx, method, url, body, headers, false)
}

// doRequest is the shared implementation for MakeRequest and MakeRequestWithTrace.
//
// Note: The url parameter is passed directly to resty with no validation. Callers
// accepting URLs from external input must validate scheme, host, and port before calling.
//
// restConfig is treated as immutable after NewClient returns, so it is read here
// without holding the mutex.
//
// The full response body is buffered in memory intentionally so that middleware in
// AfterRequest can inspect the response content.
func (c *Client) doRequest(ctx context.Context, method string, url string, body string, headers map[string]string, enableTrace bool) (*Response, error) {
	if c.restClient == nil {
		return nil, errors.New("rest client is nil")
	}

	// Normalize headers to a non-nil private copy before running the middleware
	// chain. Middleware (notably OTel trace-context injection) writes into this
	// map; using a copy keeps the caller's map untouched (avoiding a data race on
	// a shared map) and makes nil headers safe rather than a nil-map-write panic.
	reqHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		reqHeaders[k] = v
	}

	startTime := time.Now()
	c.mu.RLock()
	middlewaresCopy := make([]Middleware, len(c.middlewares))
	copy(middlewaresCopy, c.middlewares)
	c.mu.RUnlock()

	for _, middleware := range middlewaresCopy {
		ctx = middleware.BeforeRequest(ctx, method, url, body, reqHeaders)
	}

	request := c.restClient.R().
		SetHeaders(reqHeaders).
		SetContext(ctx)

	if enableTrace {
		request.EnableTrace()
	}

	if body != "" {
		request.SetBody(body)
	}

	var response *resty.Response
	var err error

	switch method {
	case http.MethodGet:
		response, err = request.Get(url)
	case http.MethodPost:
		response, err = request.Post(url)
	case http.MethodPut:
		response, err = request.Put(url)
	case http.MethodDelete:
		response, err = request.Delete(url)
	case http.MethodPatch:
		response, err = request.Patch(url)
	case http.MethodHead:
		response, err = request.Head(url)
	case http.MethodOptions:
		response, err = request.Options(url)
	default:
		response, err = request.Execute(method, url)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	requestInfo := RequestInfo{
		Method:    method,
		URL:       url,
		Headers:   reqHeaders,
		Body:      body,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Error:     err,
	}

	if response != nil {
		requestInfo.StatusCode = response.StatusCode()
		maxLog := 0
		if c.restConfig != nil {
			maxLog = c.restConfig.MaxResponseBodyLog
		}
		requestInfo.Response = truncateBody(response.String(), maxLog)
		// ResponseSize carries the true body size from resty so downstream
		// metrics/traces report the real size even when Response is truncated.
		requestInfo.ResponseSize = response.Size()
		if enableTrace && response.Request != nil {
			requestInfo.TraceInfo = traceInfoFromResty(response.Request.TraceInfo())
		}
	}

	for _, middleware := range middlewaresCopy {
		middleware.AfterRequest(ctx, requestInfo)
	}

	result := fromResty(response)

	if err != nil {
		// Construct the logger only on the error path so the common success
		// path does not allocate a LogHelper (and its console writer) per request.
		var otelConfig *otel.Config
		if c.restConfig != nil {
			otelConfig = c.restConfig.OTelConfig
		}
		logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v3/rest", "rest.MakeRequest")
		logger.Error(err, "Failed to make request")
		return result, newExecutionError("Failed to make request", err)
	}

	if result == nil {
		return nil, nil
	}

	err = c.handleResponse(result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// handleResponse checks the HTTP status code and returns a typed error for
// non-success responses. Checks are ordered from most specific to least:
// 401/403 -> 404 -> 5xx -> other 4xx.
func (c *Client) handleResponse(response *Response) error {
	maxLog := 0
	if c.restConfig != nil {
		maxLog = c.restConfig.MaxResponseBodyLog
	}
	body := truncateBody(response.Body, maxLog)

	if response.IsAuthError() {
		return newUnauthorizedError(response.StatusCode, "Unauthorized access", body)
	}

	if response.IsNotFound() {
		return newResourceNotFoundError(response.StatusCode, "Resource not found", body)
	}

	if response.IsServerError() {
		return newServerError(response.StatusCode, "Server error", body)
	}

	if response.IsClientError() {
		return newResponseError(response.StatusCode, "Client error", body)
	}

	return nil
}
