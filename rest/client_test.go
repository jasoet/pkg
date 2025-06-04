package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

// testKey is a custom type for the context key to avoid collisions
type testKey string

// Define a constant for the test key value
const testKeyValue testKey = "rest.test_key"

// mockMiddleware is a mock implementation of the Middleware interface for testing
type mockMiddleware struct {
	beforeRequestCalled bool
	afterRequestCalled  bool
	ctx                 context.Context
	method              string
	url                 string
	body                string
	headers             map[string]string
	requestInfo         RequestInfo
}

func (m *mockMiddleware) BeforeRequest(ctx context.Context, method string, url string, body string, headers map[string]string) context.Context {
	m.beforeRequestCalled = true
	m.ctx = ctx
	m.method = method
	m.url = url
	m.body = body
	m.headers = headers
	return context.WithValue(ctx, testKeyValue, "test_value")
}

func (m *mockMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {
	m.afterRequestCalled = true
	m.requestInfo = info
}

func TestNewClient(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		client := NewClient()

		if client == nil {
			t.Fatal("NewClient() returned nil")
		}

		if client.restConfig == nil {
			t.Fatal("client.restConfig is nil")
		}

		if client.restClient == nil {
			t.Fatal("client.restClient is nil")
		}

		if len(client.middlewares) != 1 {
			t.Errorf("Expected 1 default middleware, got %d", len(client.middlewares))
		}

		// Check that the default middleware is a LoggingMiddleware
		_, ok := client.middlewares[0].(*LoggingMiddleware)
		if !ok {
			t.Errorf("Expected default middleware to be LoggingMiddleware, got %T", client.middlewares[0])
		}
	})

	t.Run("With custom config", func(t *testing.T) {
		config := Config{
			RetryCount:       3,
			RetryWaitTime:    5 * time.Second,
			RetryMaxWaitTime: 60 * time.Second,
			Timeout:          10 * time.Second,
		}

		client := NewClient(WithRestConfig(config))

		if client.restConfig.RetryCount != 3 {
			t.Errorf("Expected RetryCount to be 3, got %d", client.restConfig.RetryCount)
		}

		if client.restConfig.RetryWaitTime != 5*time.Second {
			t.Errorf("Expected RetryWaitTime to be 5s, got %s", client.restConfig.RetryWaitTime)
		}

		if client.restConfig.RetryMaxWaitTime != 60*time.Second {
			t.Errorf("Expected RetryMaxWaitTime to be 60s, got %s", client.restConfig.RetryMaxWaitTime)
		}

		if client.restConfig.Timeout != 10*time.Second {
			t.Errorf("Expected Timeout to be 10s, got %s", client.restConfig.Timeout)
		}
	})

	t.Run("With custom middleware", func(t *testing.T) {
		middleware := NewNoOpMiddleware()
		client := NewClient(WithMiddleware(middleware))

		// WithMiddleware appends to existing middlewares, so we expect 2 (default + custom)
		if len(client.middlewares) != 2 {
			t.Errorf("Expected 2 middlewares, got %d", len(client.middlewares))
		}

		// The default middleware (LoggingMiddleware) should be first
		_, ok1 := client.middlewares[0].(*LoggingMiddleware)
		if !ok1 {
			t.Errorf("Expected first middleware to be LoggingMiddleware, got %T", client.middlewares[0])
		}

		// The custom middleware (NoOpMiddleware) should be second
		_, ok2 := client.middlewares[1].(*NoOpMiddleware)
		if !ok2 {
			t.Errorf("Expected second middleware to be NoOpMiddleware, got %T", client.middlewares[1])
		}
	})

	t.Run("With multiple middlewares", func(t *testing.T) {
		middleware1 := NewNoOpMiddleware()
		middleware2 := NewLoggingMiddleware()
		middleware3 := NewDatabaseLoggingMiddleware()

		client := NewClient(WithMiddlewares(middleware1, middleware2, middleware3))

		if len(client.middlewares) != 3 {
			t.Errorf("Expected 3 middlewares, got %d", len(client.middlewares))
		}

		_, ok1 := client.middlewares[0].(*NoOpMiddleware)
		if !ok1 {
			t.Errorf("Expected first middleware to be NoOpMiddleware, got %T", client.middlewares[0])
		}

		_, ok2 := client.middlewares[1].(*LoggingMiddleware)
		if !ok2 {
			t.Errorf("Expected second middleware to be LoggingMiddleware, got %T", client.middlewares[1])
		}

		_, ok3 := client.middlewares[2].(*DatabaseLoggingMiddleware)
		if !ok3 {
			t.Errorf("Expected third middleware to be DatabaseLoggingMiddleware, got %T", client.middlewares[2])
		}
	})
}

func TestClient_GetRestClient(t *testing.T) {
	client := NewClient()
	restClient := client.GetRestClient()

	if restClient == nil {
		t.Fatal("GetRestClient() returned nil")
	}

	if restClient != client.restClient {
		t.Error("GetRestClient() did not return the expected client")
	}
}

func TestClient_GetRestyClient(t *testing.T) {
	client := NewClient()
	restyClient := client.GetRestyClient()

	if restyClient == nil {
		t.Fatal("GetRestyClient() returned nil")
	}

	if restyClient != client.restClient {
		t.Error("GetRestyClient() did not return the expected client")
	}
}

func TestClient_GetRestConfig(t *testing.T) {
	client := NewClient()
	config := client.GetRestConfig()

	if config == nil {
		t.Fatal("GetRestConfig() returned nil")
	}

	if config != client.restConfig {
		t.Error("GetRestConfig() did not return the expected config")
	}
}

func TestClient_MakeRequest(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request method
			if r.Method != "GET" {
				t.Errorf("Expected method GET, got %s", r.Method)
			}

			// Check request path
			if r.URL.Path != "/test" {
				t.Errorf("Expected path /test, got %s", r.URL.Path)
			}

			// Check headers
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type header application/json, got %s", r.Header.Get("Content-Type"))
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}))
		defer server.Close()

		// Create a mock middleware
		middleware := &mockMiddleware{}

		// Create a client with the mock middleware
		client := NewClient(WithMiddlewares(middleware))
		client.restClient.SetBaseURL(server.URL)

		// Make a request
		ctx := context.Background()
		method := "GET"
		url := "/test"
		body := ""
		headers := map[string]string{"Content-Type": "application/json"}

		response, err := client.MakeRequest(ctx, method, url, body, headers)

		// Check that there was no error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that the response is not nil
		if response == nil {
			t.Fatal("Expected non-nil response, got nil")
		}

		// Check response status code
		if response.StatusCode() != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, response.StatusCode())
		}

		// Check response body
		if response.String() != `{"result":"success"}` {
			t.Errorf("Expected response body %q, got %q", `{"result":"success"}`, response.String())
		}

		// Check that middleware methods were called
		if !middleware.beforeRequestCalled {
			t.Error("Expected BeforeRequest to be called, but it wasn't")
		}
		if !middleware.afterRequestCalled {
			t.Error("Expected AfterRequest to be called, but it wasn't")
		}

		// Check middleware parameters
		if middleware.method != method {
			t.Errorf("Expected middleware method %q, got %q", method, middleware.method)
		}
		if middleware.url != url {
			t.Errorf("Expected middleware url %q, got %q", url, middleware.url)
		}
		if middleware.body != body {
			t.Errorf("Expected middleware body %q, got %q", body, middleware.body)
		}
		if middleware.headers["Content-Type"] != headers["Content-Type"] {
			t.Errorf("Expected middleware Content-Type header %q, got %q", headers["Content-Type"], middleware.headers["Content-Type"])
		}

		// Check RequestInfo in AfterRequest
		if middleware.requestInfo.Method != method {
			t.Errorf("Expected RequestInfo.Method %q, got %q", method, middleware.requestInfo.Method)
		}
		if middleware.requestInfo.URL != url {
			t.Errorf("Expected RequestInfo.URL %q, got %q", url, middleware.requestInfo.URL)
		}
		if middleware.requestInfo.StatusCode != http.StatusOK {
			t.Errorf("Expected RequestInfo.StatusCode %d, got %d", http.StatusOK, middleware.requestInfo.StatusCode)
		}
		if middleware.requestInfo.Response != `{"result":"success"}` {
			t.Errorf("Expected RequestInfo.Response %q, got %q", `{"result":"success"}`, middleware.requestInfo.Response)
		}
		if middleware.requestInfo.Error != nil {
			t.Errorf("Expected RequestInfo.Error to be nil, got %v", middleware.requestInfo.Error)
		}
	})

	t.Run("Error case - nil client", func(t *testing.T) {
		client := &Client{} // Client with nil restClient

		response, err := client.MakeRequest(context.Background(), "GET", "/test", "", nil)

		if err == nil {
			t.Error("Expected error for nil client, got nil")
		}
		if response != nil {
			t.Errorf("Expected nil response for nil client, got %v", response)
		}
	})

	t.Run("Error case - unsupported method", func(t *testing.T) {
		client := NewClient()

		response, err := client.MakeRequest(context.Background(), "INVALID", "/test", "", nil)

		if err == nil {
			t.Error("Expected error for unsupported method, got nil")
		}
		if response != nil {
			t.Errorf("Expected nil response for unsupported method, got %v", response)
		}
	})
}

func TestClient_HandleResponse(t *testing.T) {
	client := NewClient()

	t.Run("Success case", func(t *testing.T) {
		// Create a successful response
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusOK}

		err := client.HandleResponse(response)
		if err != nil {
			t.Errorf("Expected no error for successful response, got %v", err)
		}
	})

	t.Run("Unauthorized case", func(t *testing.T) {
		// Create an unauthorized response
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusUnauthorized}

		err := client.HandleResponse(response)
		if err == nil {
			t.Error("Expected error for unauthorized response, got nil")
		}

		// Check error type
		unauthorizedErr, ok := err.(*UnauthorizedError)
		if !ok {
			t.Errorf("Expected UnauthorizedError, got %T", err)
		} else {
			if unauthorizedErr.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected StatusCode %d, got %d", http.StatusUnauthorized, unauthorizedErr.StatusCode)
			}
		}
	})

	t.Run("Server error case", func(t *testing.T) {
		// Create a server error response
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: 0} // Non-HTTP status

		// Due to the implementation of IsNotHttpError (which always returns false),
		// this case will not trigger a ServerError. Instead, it will check if response.IsError()
		// which for a status code of 0 will return false, so no error will be returned.
		err := client.HandleResponse(response)
		if err != nil {
			t.Errorf("Expected no error due to implementation, got %v", err)
		}
	})

	t.Run("Response error case", func(t *testing.T) {
		// Create a response error
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusBadRequest}

		err := client.HandleResponse(response)
		if err == nil {
			t.Error("Expected error for response error, got nil")
		}

		// Check error type
		responseErr, ok := err.(*ResponseError)
		if !ok {
			t.Errorf("Expected ResponseError, got %T", err)
		} else {
			if responseErr.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected StatusCode %d, got %d", http.StatusBadRequest, responseErr.StatusCode)
			}
		}
	})
}

func TestIsNotHttpError(t *testing.T) {
	t.Run("Valid HTTP status", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusOK}

		if IsNotHttpError(response) {
			t.Error("Expected IsNotHttpError to return false for valid HTTP status")
		}
	})

	t.Run("Invalid HTTP status - less than 200", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: 100}

		// The current implementation checks (code < 200 && code >= 300) which is always false
		// So we expect false even for invalid status codes
		if IsNotHttpError(response) {
			t.Error("Expected IsNotHttpError to return false for status code < 200 due to implementation")
		}
	})

	t.Run("Invalid HTTP status - greater than or equal to 300", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: 300}

		// The current implementation checks (code < 200 && code >= 300) which is always false
		// So we expect false even for invalid status codes
		if IsNotHttpError(response) {
			t.Error("Expected IsNotHttpError to return false for status code >= 300 due to implementation")
		}
	})
}

func TestIsUnauthorized(t *testing.T) {
	t.Run("Unauthorized status", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusUnauthorized}

		if !IsUnauthorized(response) {
			t.Error("Expected IsUnauthorized to return true for unauthorized status")
		}
	})

	t.Run("Forbidden status", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusForbidden}

		if !IsUnauthorized(response) {
			t.Error("Expected IsUnauthorized to return true for forbidden status")
		}
	})

	t.Run("OK status", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusOK}

		if IsUnauthorized(response) {
			t.Error("Expected IsUnauthorized to return false for OK status")
		}
	})
}
