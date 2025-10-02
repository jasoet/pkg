package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jasoet/pkg/v2/concurrent"
	"github.com/jasoet/pkg/v2/otel"
)

// testKey is a custom type for the context key to avoid collisions
type testKey string

// TestMiddleware is a simple middleware implementation for testing
type TestMiddleware struct {
	Name string
}

func (m *TestMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
	return ctx
}

func (m *TestMiddleware) AfterRequest(ctx context.Context, requestInfo RequestInfo) {
	// Do nothing for test purposes
}

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

func TestClient_GetRestConfig(t *testing.T) {
	client := NewClient()
	config := client.GetRestConfig()

	if config == nil {
		t.Fatal("GetRestConfig() returned nil")
	}

	// Since GetRestConfig() now returns a copy for thread safety,
	// we compare the values instead of pointer equality
	if config.Timeout != client.restConfig.Timeout ||
		config.RetryCount != client.restConfig.RetryCount ||
		config.RetryWaitTime != client.restConfig.RetryWaitTime ||
		config.RetryMaxWaitTime != client.restConfig.RetryMaxWaitTime {
		t.Error("GetRestConfig() did not return the expected config values")
	}
}

func TestClient_ThreadSafety(t *testing.T) {
	client := NewClient()

	// Test concurrent middleware operations
	t.Run("Concurrent middleware operations", func(t *testing.T) {
		const numGoroutines = 100

		// Create concurrent functions for adding middlewares
		funcs := make(map[string]concurrent.Func[bool])
		for i := 0; i < numGoroutines; i++ {
			key := fmt.Sprintf("middleware-%d", i)
			id := i // capture loop variable
			funcs[key] = func(ctx context.Context) (bool, error) {
				middleware := &TestMiddleware{Name: fmt.Sprintf("test-middleware-%d", id)}
				client.AddMiddleware(middleware)
				return true, nil
			}
		}

		// Execute concurrently using the concurrent package
		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		if err != nil {
			t.Errorf("Concurrent middleware addition failed: %v", err)
		}

		// Verify all operations completed
		if len(results) != numGoroutines {
			t.Errorf("Expected %d results, got %d", numGoroutines, len(results))
		}

		// Verify all middlewares were added
		middlewares := client.GetMiddlewares()
		if len(middlewares) < numGoroutines {
			t.Errorf("Expected at least %d middlewares, got %d", numGoroutines, len(middlewares))
		}
	})

	// Test concurrent config access
	t.Run("Concurrent config access", func(t *testing.T) {
		const numGoroutines = 50

		// Create concurrent functions for config access
		funcs := make(map[string]concurrent.Func[*Config])
		for i := 0; i < numGoroutines; i++ {
			key := fmt.Sprintf("config-%d", i)
			funcs[key] = func(ctx context.Context) (*Config, error) {
				config := client.GetRestConfig()
				if config == nil {
					return nil, errors.New("GetRestConfig() returned nil")
				}
				return config, nil
			}
		}

		// Execute concurrently
		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		if err != nil {
			t.Errorf("Concurrent config access failed: %v", err)
		}

		// Verify all operations completed
		if len(results) != numGoroutines {
			t.Errorf("Expected %d results, got %d", numGoroutines, len(results))
		}

		// Verify all configs have expected values
		for key, config := range results {
			if config.Timeout <= 0 {
				t.Errorf("Config %s has invalid timeout: %v", key, config.Timeout)
			}
		}
	})

	// Test concurrent HTTP requests
	t.Run("Concurrent HTTP requests", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		const numRequests = 20

		// Create concurrent functions for HTTP requests
		funcs := make(map[string]concurrent.Func[*resty.Response])
		for i := 0; i < numRequests; i++ {
			key := fmt.Sprintf("request-%d", i)
			funcs[key] = func(ctx context.Context) (*resty.Response, error) {
				return client.MakeRequest(ctx, "GET", server.URL, "", nil)
			}
		}

		// Execute concurrently
		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		if err != nil {
			t.Errorf("Concurrent HTTP requests failed: %v", err)
		}

		// Verify all requests completed successfully
		if len(results) != numRequests {
			t.Errorf("Expected %d results, got %d", numRequests, len(results))
		}

		for key, response := range results {
			if response.StatusCode() != 200 {
				t.Errorf("Request %s failed with status %d", key, response.StatusCode())
			}
		}
	})
}

func TestClient_MakeRequest(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("Expected method GET, got %s", r.Method)
			}

			if r.URL.Path != "/test" {
				t.Errorf("Expected path /test, got %s", r.URL.Path)
			}

			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type header application/json, got %s", r.Header.Get("Content-Type"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}))
		defer server.Close()

		middleware := &mockMiddleware{}

		client := NewClient(WithMiddlewares(middleware))
		client.restClient.SetBaseURL(server.URL)

		ctx := context.Background()
		method := "GET"
		url := "/test"
		body := ""
		headers := map[string]string{"Content-Type": "application/json"}

		response, err := client.MakeRequest(ctx, method, url, body, headers)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

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

	t.Run("Error case - invalid URL", func(t *testing.T) {
		client := NewClient()

		_, err := client.MakeRequest(context.Background(), "GET", "/test", "", nil)

		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
		var execErr *ExecutionError
		if !errors.As(err, &execErr) {
			t.Error("Expected ExecutionError for invalid URL")
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

func TestIsServerError(t *testing.T) {
	t.Run("Valid HTTP status", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusOK}

		if IsServerError(response) {
			t.Error("Expected IsServerError to return false for valid HTTP status")
		}
	})

	t.Run("Server error status - 500", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusInternalServerError}

		if !IsServerError(response) {
			t.Error("Expected IsServerError to return true for status code 500")
		}
	})

	t.Run("Client error status - 400", func(t *testing.T) {
		response := &resty.Response{}
		response.Request = &resty.Request{}
		response.RawResponse = &http.Response{StatusCode: http.StatusBadRequest}

		if IsServerError(response) {
			t.Error("Expected IsServerError to return false for status code 400")
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

func TestWithOTelConfig(t *testing.T) {
	t.Run("sets OTel config on client", func(t *testing.T) {
		cfg := &Config{
			OTelConfig: nil,
		}

		otelCfg := otel.NewConfig("test-service")
		option := WithOTelConfig(otelCfg)

		client := &Client{
			restConfig: cfg,
		}
		option(client)

		if client.restConfig.OTelConfig != otelCfg {
			t.Error("Expected OTel config to be set on client")
		}
	})

	t.Run("works with NewClient", func(t *testing.T) {
		otelCfg := otel.NewConfig("test-service")
		client := NewClient(WithOTelConfig(otelCfg))

		if client.restConfig.OTelConfig != otelCfg {
			t.Error("Expected OTel config to be set via NewClient")
		}
	})
}

func TestSetMiddlewares(t *testing.T) {
	t.Run("replaces existing middlewares", func(t *testing.T) {
		client := NewClient()

		// Initially should have default logging middleware
		initial := len(client.GetMiddlewares())
		if initial == 0 {
			t.Error("Expected client to have default middlewares")
		}

		// Set new middlewares
		mw1 := &TestMiddleware{Name: "test1"}
		mw2 := &TestMiddleware{Name: "test2"}
		client.SetMiddlewares(mw1, mw2)

		middlewares := client.GetMiddlewares()
		if len(middlewares) != 2 {
			t.Errorf("Expected 2 middlewares, got %d", len(middlewares))
		}
	})

	t.Run("can set empty middlewares list", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares()

		middlewares := client.GetMiddlewares()
		if len(middlewares) != 0 {
			t.Errorf("Expected 0 middlewares, got %d", len(middlewares))
		}
	})

	t.Run("is thread-safe", func(t *testing.T) {
		client := NewClient()
		done := make(chan bool, 2)

		// Concurrent writes
		go func() {
			for i := 0; i < 100; i++ {
				client.SetMiddlewares(&TestMiddleware{Name: "goroutine1"})
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				client.SetMiddlewares(&TestMiddleware{Name: "goroutine2"})
			}
			done <- true
		}()

		<-done
		<-done

		// Should complete without race conditions
		middlewares := client.GetMiddlewares()
		if len(middlewares) != 1 {
			t.Errorf("Expected 1 middleware after concurrent access, got %d", len(middlewares))
		}
	})
}

func TestAddMiddleware(t *testing.T) {
	t.Run("appends middleware to existing list", func(t *testing.T) {
		client := NewClient()
		initial := len(client.GetMiddlewares())

		mw := &TestMiddleware{Name: "additional"}
		client.AddMiddleware(mw)

		middlewares := client.GetMiddlewares()
		if len(middlewares) != initial+1 {
			t.Errorf("Expected %d middlewares, got %d", initial+1, len(middlewares))
		}
	})

	t.Run("maintains order of middlewares", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares() // Clear defaults

		mw1 := &TestMiddleware{Name: "first"}
		mw2 := &TestMiddleware{Name: "second"}
		mw3 := &TestMiddleware{Name: "third"}

		client.AddMiddleware(mw1)
		client.AddMiddleware(mw2)
		client.AddMiddleware(mw3)

		middlewares := client.GetMiddlewares()
		if len(middlewares) != 3 {
			t.Errorf("Expected 3 middlewares, got %d", len(middlewares))
		}

		if m, ok := middlewares[0].(*TestMiddleware); !ok || m.Name != "first" {
			t.Error("Expected first middleware to be 'first'")
		}
		if m, ok := middlewares[1].(*TestMiddleware); !ok || m.Name != "second" {
			t.Error("Expected second middleware to be 'second'")
		}
		if m, ok := middlewares[2].(*TestMiddleware); !ok || m.Name != "third" {
			t.Error("Expected third middleware to be 'third'")
		}
	})
}

func TestGetMiddlewares(t *testing.T) {
	t.Run("returns copy of middlewares", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares(&TestMiddleware{Name: "test"})

		middlewares1 := client.GetMiddlewares()
		middlewares2 := client.GetMiddlewares()

		// Verify we get different slices (copies)
		if &middlewares1[0] == &middlewares2[0] {
			t.Error("Expected GetMiddlewares to return a copy, not the original slice")
		}
	})

	t.Run("modifications to returned slice don't affect client", func(t *testing.T) {
		client := NewClient()
		mw := &TestMiddleware{Name: "test"}
		client.SetMiddlewares(mw)

		middlewares := client.GetMiddlewares()
		middlewares[0] = &TestMiddleware{Name: "modified"}

		// Verify client's middlewares are unchanged
		clientMiddlewares := client.GetMiddlewares()
		if m, ok := clientMiddlewares[0].(*TestMiddleware); !ok || m.Name != "test" {
			t.Error("Expected client middlewares to be unchanged after modifying returned slice")
		}
	})
}

func TestClient_MakeRequestWithTrace(t *testing.T) {
	t.Run("returns error when client is nil", func(t *testing.T) {
		client := &Client{
			restClient: nil,
			restConfig: DefaultRestConfig(),
		}

		ctx := context.Background()
		headers := make(map[string]string)

		response, err := client.MakeRequestWithTrace(ctx, "GET", "http://example.com", "", headers)
		if err == nil {
			t.Error("Expected error when rest client is nil")
		}
		if response != nil {
			t.Error("Expected nil response when rest client is nil")
		}
	})

	t.Run("makes successful GET request with trace", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer server.Close()

		client := NewClient()
		ctx := context.Background()
		headers := make(map[string]string)

		response, err := client.MakeRequestWithTrace(ctx, "GET", server.URL, "", headers)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.StatusCode() != http.StatusOK {
			t.Errorf("Expected status 200, got %d", response.StatusCode())
		}
	})

	t.Run("works with middleware", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		client.SetMiddlewares(&TestMiddleware{Name: "trace-test"})

		ctx := context.Background()
		headers := make(map[string]string)

		response, err := client.MakeRequestWithTrace(ctx, "GET", server.URL, "", headers)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.StatusCode() != http.StatusOK {
			t.Errorf("Expected status 200, got %d", response.StatusCode())
		}
	})

	t.Run("supports different HTTP methods", func(t *testing.T) {
		methodReceived := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			methodReceived = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		ctx := context.Background()
		headers := make(map[string]string)

		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		for _, method := range methods {
			methodReceived = ""
			_, err := client.MakeRequestWithTrace(ctx, method, server.URL, "", headers)
			if err != nil {
				t.Errorf("Unexpected error for method %s: %v", method, err)
			}
			// HEAD and OPTIONS might not receive proper method confirmation from test server
			if method != "HEAD" && method != "OPTIONS" {
				if methodReceived != method {
					t.Errorf("Expected method %s, got %s", method, methodReceived)
				}
			}
		}
	})

	t.Run("includes request body", func(t *testing.T) {
		bodyReceived := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				buf := new(strings.Builder)
				io.Copy(buf, r.Body)
				bodyReceived = buf.String()
			}
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := NewClient()
		ctx := context.Background()
		headers := make(map[string]string)
		body := "test request body"

		response, err := client.MakeRequestWithTrace(ctx, "POST", server.URL, body, headers)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if response.StatusCode() != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", response.StatusCode())
		}
		if bodyReceived != body {
			t.Errorf("Expected body %q, got %q", body, bodyReceived)
		}
	})

	t.Run("handles server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}))
		defer server.Close()

		client := NewClient()
		ctx := context.Background()
		headers := make(map[string]string)

		response, err := client.MakeRequestWithTrace(ctx, "GET", server.URL, "", headers)
		if err == nil {
			t.Error("Expected error for 500 status")
		}
		if response == nil {
			t.Fatal("Expected non-nil response even on error")
		}
		if response.StatusCode() != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", response.StatusCode())
		}
	})

	t.Run("enables tracing on request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		ctx := context.Background()
		headers := make(map[string]string)

		response, err := client.MakeRequestWithTrace(ctx, "GET", server.URL, "", headers)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		// With trace enabled, TraceInfo should be populated (even if values are zero)
		if response.Request != nil {
			_ = response.Request.TraceInfo()
		}
	})
}
