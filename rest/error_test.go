package rest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnauthorizedError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 401
		msg := "Unauthorized access"
		respBody := `{"error":"invalid_token"}`

		err := newUnauthorizedError(statusCode, msg, respBody)

		require.NotNil(t, err)
		assert.Equal(t, statusCode, err.StatusCode)
		assert.Equal(t, msg, err.Msg)
		assert.Equal(t, respBody, err.RespBody)
	})

	t.Run("Error method includes response body", func(t *testing.T) {
		err := &UnauthorizedError{
			StatusCode: 401,
			Msg:        "Unauthorized access",
			RespBody:   `{"error":"invalid_token"}`,
		}

		expected := `unauthorized (HTTP 401): Unauthorized access: {"error":"invalid_token"}`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newUnauthorizedError(401, "test", "body")
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
}

func TestExecutionError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		msg := "Failed to execute request"
		cause := errors.New("network error")

		err := newExecutionError(msg, cause)

		require.NotNil(t, err)
		assert.Equal(t, msg, err.Msg)
		assert.Equal(t, cause, err.Err)
	})

	t.Run("Error method includes cause", func(t *testing.T) {
		err := &ExecutionError{
			Msg: "Failed to execute request",
			Err: errors.New("network error"),
		}

		assert.Equal(t, "Failed to execute request: network error", err.Error())
	})

	t.Run("Error method without cause", func(t *testing.T) {
		err := &ExecutionError{Msg: "Failed to execute request"}
		assert.Equal(t, "Failed to execute request", err.Error())
	})

	t.Run("Unwrap method", func(t *testing.T) {
		cause := errors.New("network error")
		err := &ExecutionError{
			Msg: "Failed to execute request",
			Err: cause,
		}

		assert.Equal(t, cause, err.Unwrap())
		assert.ErrorIs(t, err, cause)
	})
}

func TestServerError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 500
		msg := "Internal server error"
		respBody := `{"error":"server_error"}`

		err := newServerError(statusCode, msg, respBody)

		require.NotNil(t, err)
		assert.Equal(t, statusCode, err.StatusCode)
		assert.Equal(t, msg, err.Msg)
		assert.Equal(t, respBody, err.RespBody)
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Internal server error"
		respBody := `{"error":"server_error"}`
		err := &ServerError{StatusCode: 500, Msg: msg, RespBody: respBody}

		assert.Equal(t, msg+": "+respBody, err.Error())
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newServerError(500, "test", "body")
		assert.ErrorIs(t, err, ErrServer)
	})
}

func TestResponseError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 400
		msg := "Bad request"
		respBody := `{"error":"invalid_request"}`

		err := newResponseError(statusCode, msg, respBody)

		require.NotNil(t, err)
		assert.Equal(t, statusCode, err.StatusCode)
		assert.Equal(t, msg, err.Msg)
		assert.Equal(t, respBody, err.RespBody)
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Bad request"
		respBody := `{"error":"invalid_request"}`
		err := &ResponseError{StatusCode: 400, Msg: msg, RespBody: respBody}

		assert.Equal(t, msg+": "+respBody, err.Error())
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newResponseError(400, "test", "body")
		assert.ErrorIs(t, err, ErrResponse)
	})
}

func TestResourceNotFoundError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 404
		msg := "Resource not found"
		respBody := `{"error":"not_found"}`

		err := newResourceNotFoundError(statusCode, msg, respBody)

		require.NotNil(t, err)
		assert.Equal(t, statusCode, err.StatusCode)
		assert.Equal(t, msg, err.Msg)
		assert.Equal(t, respBody, err.RespBody)
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Resource not found"
		respBody := `{"error":"not_found"}`
		err := &ResourceNotFoundError{StatusCode: 404, Msg: msg, RespBody: respBody}

		assert.Equal(t, msg+": "+respBody, err.Error())
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newResourceNotFoundError(404, "test", "body")
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}
