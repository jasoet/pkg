package rest

import (
	"errors"
	"testing"
)

func TestUnauthorizedError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 401
		msg := "Unauthorized access"
		respBody := `{"error":"invalid_token"}`

		err := newUnauthorizedError(statusCode, msg, respBody)

		if err == nil {
			t.Fatal("newUnauthorizedError() returned nil")
		}

		if err.StatusCode != statusCode {
			t.Errorf("Expected StatusCode %d, got %d", statusCode, err.StatusCode)
		}

		if err.Msg != msg {
			t.Errorf("Expected Msg %q, got %q", msg, err.Msg)
		}

		if err.RespBody != respBody {
			t.Errorf("Expected RespBody %q, got %q", respBody, err.RespBody)
		}
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Unauthorized access"
		err := &UnauthorizedError{
			StatusCode: 401,
			Msg:        msg,
			RespBody:   `{"error":"invalid_token"}`,
		}

		expected := "unauthorized (HTTP 401): Unauthorized access"
		if err.Error() != expected {
			t.Errorf("Expected Error() to return %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newUnauthorizedError(401, "test", "body")
		if !errors.Is(err, ErrUnauthorized) {
			t.Error("Expected errors.Is(err, ErrUnauthorized) to be true")
		}
	})
}

func TestExecutionError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		msg := "Failed to execute request"
		cause := errors.New("network error")

		err := newExecutionError(msg, cause)

		if err == nil {
			t.Fatal("newExecutionError() returned nil")
		}

		if err.Msg != msg {
			t.Errorf("Expected Msg %q, got %q", msg, err.Msg)
		}

		if err.Err != cause {
			t.Errorf("Expected Err %v, got %v", cause, err.Err)
		}
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Failed to execute request"
		err := &ExecutionError{
			Msg: msg,
			Err: errors.New("network error"),
		}

		if err.Error() != msg {
			t.Errorf("Expected Error() to return %q, got %q", msg, err.Error())
		}
	})

	t.Run("Unwrap method", func(t *testing.T) {
		cause := errors.New("network error")
		err := &ExecutionError{
			Msg: "Failed to execute request",
			Err: cause,
		}

		unwrapped := err.Unwrap()
		if unwrapped != cause {
			t.Errorf("Expected Unwrap() to return %v, got %v", cause, unwrapped)
		}

		// Test with errors.Is
		if !errors.Is(err, cause) {
			t.Errorf("Expected errors.Is(err, cause) to be true")
		}
	})
}

func TestServerError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 500
		msg := "Internal server error"
		respBody := `{"error":"server_error"}`

		err := newServerError(statusCode, msg, respBody)

		if err == nil {
			t.Fatal("newServerError() returned nil")
		}

		if err.StatusCode != statusCode {
			t.Errorf("Expected StatusCode %d, got %d", statusCode, err.StatusCode)
		}

		if err.Msg != msg {
			t.Errorf("Expected Msg %q, got %q", msg, err.Msg)
		}

		if err.RespBody != respBody {
			t.Errorf("Expected RespBody %q, got %q", respBody, err.RespBody)
		}
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Internal server error"
		respBody := `{"error":"server_error"}`
		err := &ServerError{
			StatusCode: 500,
			Msg:        msg,
			RespBody:   respBody,
		}

		expected := msg + ": " + respBody
		if err.Error() != expected {
			t.Errorf("Expected Error() to return %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newServerError(500, "test", "body")
		if !errors.Is(err, ErrServer) {
			t.Error("Expected errors.Is(err, ErrServer) to be true")
		}
	})
}

func TestResponseError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 400
		msg := "Bad request"
		respBody := `{"error":"invalid_request"}`

		err := newResponseError(statusCode, msg, respBody)

		if err == nil {
			t.Fatal("newResponseError() returned nil")
		}

		if err.StatusCode != statusCode {
			t.Errorf("Expected StatusCode %d, got %d", statusCode, err.StatusCode)
		}

		if err.Msg != msg {
			t.Errorf("Expected Msg %q, got %q", msg, err.Msg)
		}

		if err.RespBody != respBody {
			t.Errorf("Expected RespBody %q, got %q", respBody, err.RespBody)
		}
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Bad request"
		respBody := `{"error":"invalid_request"}`
		err := &ResponseError{
			StatusCode: 400,
			Msg:        msg,
			RespBody:   respBody,
		}

		expected := msg + ": " + respBody
		if err.Error() != expected {
			t.Errorf("Expected Error() to return %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newResponseError(400, "test", "body")
		if !errors.Is(err, ErrResponse) {
			t.Error("Expected errors.Is(err, ErrResponse) to be true")
		}
	})
}

func TestResourceNotFoundError(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		statusCode := 404
		msg := "Resource not found"
		respBody := `{"error":"not_found"}`

		err := newResourceNotFoundError(statusCode, msg, respBody)

		if err == nil {
			t.Fatal("newResourceNotFoundError() returned nil")
		}

		if err.StatusCode != statusCode {
			t.Errorf("Expected StatusCode %d, got %d", statusCode, err.StatusCode)
		}

		if err.Msg != msg {
			t.Errorf("Expected Msg %q, got %q", msg, err.Msg)
		}

		if err.RespBody != respBody {
			t.Errorf("Expected RespBody %q, got %q", respBody, err.RespBody)
		}
	})

	t.Run("Error method", func(t *testing.T) {
		msg := "Resource not found"
		respBody := `{"error":"not_found"}`
		err := &ResourceNotFoundError{
			StatusCode: 404,
			Msg:        msg,
			RespBody:   respBody,
		}

		expected := msg + ": " + respBody
		if err.Error() != expected {
			t.Errorf("Expected Error() to return %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns sentinel", func(t *testing.T) {
		err := newResourceNotFoundError(404, "test", "body")
		if !errors.Is(err, ErrResourceNotFound) {
			t.Error("Expected errors.Is(err, ErrResourceNotFound) to be true")
		}
	})
}
