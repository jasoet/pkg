package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type Client struct {
	restClient  *resty.Client
	restConfig  *Config
	middlewares []Middleware
}

type ClientOption func(*Client)

func WithRestConfig(restConfig Config) ClientOption {
	return func(client *Client) {
		client.restConfig = &restConfig
	}
}

func WithMiddleware(middleware Middleware) ClientOption {
	return func(client *Client) {
		client.middlewares = append(client.middlewares, middleware)
	}
}

func WithMiddlewares(middlewares ...Middleware) ClientOption {
	return func(client *Client) {
		client.middlewares = middlewares
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

func (c *Client) GetRestyClient() *resty.Client {
	return c.restClient
}

func (c *Client) GetRestConfig() *Config {
	return c.restConfig
}

func (c *Client) MakeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) (*resty.Response, error) {
	_log := log.With().Ctx(ctx).Str("function", "MakeRequest").Str("url", url).Logger()

	if c.restClient == nil {
		return nil, errors.New("rest client is nil")
	}

	// Apply BeforeRequest middleware
	startTime := time.Now()
	for _, middleware := range c.middlewares {
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

	// Create RequestInfo for middleware
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
		requestInfo.TraceInfo = response.Request.TraceInfo()
	}

	// Apply AfterRequest middleware
	for _, middleware := range c.middlewares {
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
