package rest

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jasoet/pkg/otel"
	"github.com/rs/zerolog/log"
)

type Client struct {
	restClient  *resty.Client
	restConfig  *Config
	middlewares []Middleware
	mu          sync.RWMutex
}

type ClientOption func(*Client)

func WithRestConfig(restConfig Config) ClientOption {
	return func(client *Client) {
		client.restConfig = &restConfig
	}
}

func WithMiddleware(middleware Middleware) ClientOption {
	return func(client *Client) {
		client.mu.Lock()
		defer client.mu.Unlock()
		client.middlewares = append(client.middlewares, middleware)
	}
}

func WithMiddlewares(middlewares ...Middleware) ClientOption {
	return func(client *Client) {
		client.mu.Lock()
		defer client.mu.Unlock()
		client.middlewares = middlewares
	}
}

// WithOTelConfig sets the OpenTelemetry configuration for the REST client
// When set, adds OTel tracing, metrics, and logging middleware automatically
func WithOTelConfig(cfg *otel.Config) ClientOption {
	return func(client *Client) {
		client.restConfig.OTelConfig = cfg
	}
}

func NewClient(options ...ClientOption) *Client {
	client := &Client{
		restConfig:  DefaultRestConfig(),
		middlewares: []Middleware{NewLoggingMiddleware()}, // Default middleware
	}

	for _, option := range options {
		option(client)
	}

	// Add OTel middleware if configured (prepend to user middleware)
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
		if metricsMW := NewOTelMetricsMiddleware(client.restConfig.OTelConfig); metricsMW != nil {
			client.middlewares = append(client.middlewares, metricsMW)
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
		SetTimeout(client.restConfig.Timeout)

	client.restClient = httpClient

	return client
}

func (c *Client) GetRestClient() *resty.Client {
	return c.restClient
}

func (c *Client) GetRestConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	configCopy := *c.restConfig
	return &configCopy
}

func (c *Client) AddMiddleware(middleware Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware)
}

func (c *Client) SetMiddlewares(middlewares ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = middlewares
}

func (c *Client) GetMiddlewares() []Middleware {
	c.mu.RLock()
	defer c.mu.RUnlock()
	middlewaresCopy := make([]Middleware, len(c.middlewares))
	copy(middlewaresCopy, c.middlewares)
	return middlewaresCopy
}

func (c *Client) MakeRequestWithTrace(ctx context.Context, method string, url string, body string, headers map[string]string) (*resty.Response, error) {
	_log := log.With().Ctx(ctx).Str("function", "MakeRequestWithTrace").Str("url", url).Logger()

	if c.restClient == nil {
		return nil, errors.New("rest client is nil")
	}

	startTime := time.Now()
	c.mu.RLock()
	middlewaresCopy := make([]Middleware, len(c.middlewares))
	copy(middlewaresCopy, c.middlewares)
	c.mu.RUnlock()

	for _, middleware := range middlewaresCopy {
		ctx = middleware.BeforeRequest(ctx, method, url, body, headers)
	}

	request := c.restClient.R().
		SetHeaders(headers).
		SetContext(ctx).
		EnableTrace()

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
		Headers:   headers,
		Body:      body,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Error:     err,
	}

	if response != nil {
		requestInfo.StatusCode = response.StatusCode()
		requestInfo.Response = response.String()
		if response.Request != nil {
			requestInfo.TraceInfo = response.Request.TraceInfo()
		}
	}

	for _, middleware := range middlewaresCopy {
		middleware.AfterRequest(ctx, requestInfo)
	}

	if err != nil {
		_log.Error().Err(err).Msg("Failed to make request")
		return response, NewExecutionError("Failed to make request", err)
	}

	err = c.HandleResponse(response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (c *Client) MakeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) (*resty.Response, error) {
	_log := log.With().Ctx(ctx).Str("function", "MakeRequest").Str("url", url).Logger()

	if c.restClient == nil {
		return nil, errors.New("rest client is nil")
	}

	startTime := time.Now()
	c.mu.RLock()
	middlewaresCopy := make([]Middleware, len(c.middlewares))
	copy(middlewaresCopy, c.middlewares)
	c.mu.RUnlock()

	for _, middleware := range middlewaresCopy {
		ctx = middleware.BeforeRequest(ctx, method, url, body, headers)
	}

	request := c.restClient.R().
		SetHeaders(headers).
		SetContext(ctx)

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
		Headers:   headers,
		Body:      body,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Error:     err,
	}

	if response != nil {
		requestInfo.StatusCode = response.StatusCode()
		requestInfo.Response = response.String()
	}

	for _, middleware := range middlewaresCopy {
		middleware.AfterRequest(ctx, requestInfo)
	}

	if err != nil {
		_log.Error().Err(err).Msg("Failed to make request")
		return response, NewExecutionError("Failed to make request", err)
	}

	err = c.HandleResponse(response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (c *Client) HandleResponse(response *resty.Response) error {
	if IsUnauthorized(response) {
		return NewUnauthorizedError(response.StatusCode(), "Unauthorized access", response.String())
	}

	if IsNotFound(response) {
		return NewResourceNotFoundError(response.StatusCode(), "Resource not found", response.String())
	}

	if IsServerError(response) {
		return NewServerError(response.StatusCode(), "Server error", response.String())
	}

	if IsClientError(response) {
		return NewResponseError(response.StatusCode(), "Client error", response.String())
	}

	return nil
}

func IsServerError(response *resty.Response) bool {
	return response.StatusCode() >= 500
}

func IsUnauthorized(response *resty.Response) bool {
	return response.StatusCode() == http.StatusUnauthorized || response.StatusCode() == http.StatusForbidden
}

func IsNotFound(response *resty.Response) bool {
	return response.StatusCode() == http.StatusNotFound
}

func IsClientError(response *resty.Response) bool {
	return response.StatusCode() >= 400 && response.StatusCode() < 500
}
