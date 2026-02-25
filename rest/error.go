package rest

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is.
var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrResourceNotFound = errors.New("resource not found")
	ErrServer           = errors.New("server error")
	ErrResponse         = errors.New("response error")
)

// UnauthorizedError represents an authentication or authorization failure (HTTP 401/403).
type UnauthorizedError struct {
	StatusCode int
	Msg        string
	RespBody   string
}

func (e *UnauthorizedError) Error() string { return e.Msg }
func (e *UnauthorizedError) Unwrap() error { return ErrUnauthorized }

// NewUnauthorizedError creates a new UnauthorizedError
func NewUnauthorizedError(statusCode int, msg string, respBody string) *UnauthorizedError {
	return &UnauthorizedError{
		StatusCode: statusCode,
		Msg:        msg,
		RespBody:   respBody,
	}
}

// ExecutionError represents an error during request execution (e.g. network failure).
type ExecutionError struct {
	Msg string
	Err error
}

func (e *ExecutionError) Error() string { return e.Msg }
func (e *ExecutionError) Unwrap() error { return e.Err }

func NewExecutionError(msg string, err error) *ExecutionError {
	return &ExecutionError{
		Msg: msg,
		Err: err,
	}
}

// ServerError represents a server-side failure (HTTP 5xx).
type ServerError struct {
	StatusCode int
	Msg        string
	RespBody   string
}

func (e *ServerError) Error() string { return fmt.Sprintf("%s: %s", e.Msg, e.RespBody) }
func (e *ServerError) Unwrap() error { return ErrServer }

// NewServerError creates a new ServerError
func NewServerError(statusCode int, msg string, respBody string) *ServerError {
	return &ServerError{
		StatusCode: statusCode,
		Msg:        msg,
		RespBody:   respBody,
	}
}

// ResponseError represents a client-side HTTP error (HTTP 4xx, excluding 401/403/404).
type ResponseError struct {
	StatusCode int
	Msg        string
	RespBody   string
}

func (e *ResponseError) Error() string { return fmt.Sprintf("%s: %s", e.Msg, e.RespBody) }
func (e *ResponseError) Unwrap() error { return ErrResponse }

func NewResponseError(statusCode int, msg string, respBody string) *ResponseError {
	return &ResponseError{
		StatusCode: statusCode,
		Msg:        msg,
		RespBody:   respBody,
	}
}

// ResourceNotFoundError represents a 404 Not Found response.
type ResourceNotFoundError struct {
	StatusCode int
	Msg        string
	RespBody   string
}

func (e *ResourceNotFoundError) Error() string { return fmt.Sprintf("%s: %s", e.Msg, e.RespBody) }
func (e *ResourceNotFoundError) Unwrap() error { return ErrResourceNotFound }

func NewResourceNotFoundError(statusCode int, msg string, respBody string) *ResourceNotFoundError {
	return &ResourceNotFoundError{
		StatusCode: statusCode,
		Msg:        msg,
		RespBody:   respBody,
	}
}
