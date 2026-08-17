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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/concurrent"
	"github.com/jasoet/pkg/v3/otel"
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

		require.NotNil(t, client)
		require.NotNil(t, client.restConfig)
		require.NotNil(t, client.restClient)
		require.Len(t, client.middlewares, 1)

		_, ok := client.middlewares[0].(*LoggingMiddleware)
		assert.True(t, ok, "default middleware should be LoggingMiddleware, got %T", client.middlewares[0])
	})

	t.Run("With custom config", func(t *testing.T) {
		config := Config{
			RetryCount:       3,
			RetryWaitTime:    5 * time.Second,
			RetryMaxWaitTime: 60 * time.Second,
			Timeout:          10 * time.Second,
		}

		client := NewClient(WithRestConfig(config))

		assert.Equal(t, 3, client.restConfig.RetryCount)
		assert.Equal(t, 5*time.Second, client.restConfig.RetryWaitTime)
		assert.Equal(t, 60*time.Second, client.restConfig.RetryMaxWaitTime)
		assert.Equal(t, 10*time.Second, client.restConfig.Timeout)
	})

	t.Run("With custom middleware", func(t *testing.T) {
		middleware := NewNoOpMiddleware()
		client := NewClient(WithMiddleware(middleware))

		// WithMiddleware appends to existing middlewares (default + custom).
		require.Len(t, client.middlewares, 2)
		_, ok1 := client.middlewares[0].(*LoggingMiddleware)
		assert.True(t, ok1, "first middleware should be LoggingMiddleware, got %T", client.middlewares[0])
		_, ok2 := client.middlewares[1].(*NoOpMiddleware)
		assert.True(t, ok2, "second middleware should be NoOpMiddleware, got %T", client.middlewares[1])
	})

	t.Run("With multiple middlewares", func(t *testing.T) {
		client := NewClient(WithMiddlewares(NewNoOpMiddleware(), NewLoggingMiddleware(), NewNoOpMiddleware()))

		require.Len(t, client.middlewares, 3)
		_, ok1 := client.middlewares[0].(*NoOpMiddleware)
		assert.True(t, ok1)
		_, ok2 := client.middlewares[1].(*LoggingMiddleware)
		assert.True(t, ok2)
		_, ok3 := client.middlewares[2].(*NoOpMiddleware)
		assert.True(t, ok3)
	})
}

func TestClient_GetRestClient(t *testing.T) {
	client := NewClient()
	restClient := client.GetRestClient()

	require.NotNil(t, restClient)
	assert.Same(t, client.restClient, restClient)
}

func TestClient_GetRestConfig(t *testing.T) {
	client := NewClient()
	config := client.GetRestConfig()

	require.NotNil(t, config)
	// GetRestConfig returns a copy for thread safety; compare values.
	assert.Equal(t, client.restConfig.Timeout, config.Timeout)
	assert.Equal(t, client.restConfig.RetryCount, config.RetryCount)
	assert.Equal(t, client.restConfig.RetryWaitTime, config.RetryWaitTime)
	assert.Equal(t, client.restConfig.RetryMaxWaitTime, config.RetryMaxWaitTime)
}

func TestClient_ThreadSafety(t *testing.T) {
	client := NewClient()

	t.Run("Concurrent middleware operations", func(t *testing.T) {
		const numGoroutines = 100

		funcs := make(map[string]concurrent.Func[bool])
		for i := 0; i < numGoroutines; i++ {
			key := fmt.Sprintf("middleware-%d", i)
			id := i
			funcs[key] = func(ctx context.Context) (bool, error) {
				client.AddMiddleware(&TestMiddleware{Name: fmt.Sprintf("test-middleware-%d", id)})
				return true, nil
			}
		}

		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		require.NoError(t, err)
		assert.Len(t, results, numGoroutines)
		assert.GreaterOrEqual(t, len(client.GetMiddlewares()), numGoroutines)
	})

	t.Run("Concurrent config access", func(t *testing.T) {
		const numGoroutines = 50

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

		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		require.NoError(t, err)
		require.Len(t, results, numGoroutines)
		for key, config := range results {
			assert.Positive(t, config.Timeout, "config %s has invalid timeout", key)
		}
	})

	t.Run("Concurrent HTTP requests", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		const numRequests = 20

		funcs := make(map[string]concurrent.Func[*Response])
		for i := 0; i < numRequests; i++ {
			key := fmt.Sprintf("request-%d", i)
			funcs[key] = func(ctx context.Context) (*Response, error) {
				return client.MakeRequest(ctx, "GET", server.URL, "", nil)
			}
		}

		results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
		require.NoError(t, err)
		require.Len(t, results, numRequests)
		for key, response := range results {
			assert.Equal(t, 200, response.StatusCode, "request %s", key)
		}
	})
}

func TestClient_MakeRequest(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/test", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}))
		defer server.Close()

		middleware := &mockMiddleware{}

		client := NewClient(WithMiddlewares(middleware))
		client.restClient.SetBaseURL(server.URL)

		ctx := context.Background()
		method, url, body := "GET", "/test", ""
		headers := map[string]string{"Content-Type": "application/json"}

		response, err := client.MakeRequest(ctx, method, url, body, headers)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, `{"result":"success"}`, response.Body)

		assert.True(t, middleware.beforeRequestCalled, "BeforeRequest should be called")
		assert.True(t, middleware.afterRequestCalled, "AfterRequest should be called")

		assert.Equal(t, method, middleware.method)
		assert.Equal(t, url, middleware.url)
		assert.Equal(t, body, middleware.body)
		assert.Equal(t, headers["Content-Type"], middleware.headers["Content-Type"])

		assert.Equal(t, method, middleware.requestInfo.Method)
		assert.Equal(t, url, middleware.requestInfo.URL)
		assert.Equal(t, http.StatusOK, middleware.requestInfo.StatusCode)
		assert.Equal(t, `{"result":"success"}`, middleware.requestInfo.Response)
		assert.NoError(t, middleware.requestInfo.Error)
	})

	t.Run("caller headers map is not mutated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// A middleware that injects a header, mimicking auth/trace middleware.
		client := NewClient(WithMiddlewares(&headerInjectingMiddleware{key: "X-Injected", value: "yes"}))

		headers := map[string]string{"X-Original": "1"}
		_, err := client.MakeRequest(context.Background(), "GET", server.URL, "", headers)
		require.NoError(t, err)

		assert.Len(t, headers, 1, "caller map must not gain injected headers")
		_, injected := headers["X-Injected"]
		assert.False(t, injected)
	})

	t.Run("nil headers is safe with header-injecting middleware", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "yes", r.Header.Get("X-Injected"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(WithMiddlewares(&headerInjectingMiddleware{key: "X-Injected", value: "yes"}))

		require.NotPanics(t, func() {
			_, err := client.MakeRequest(context.Background(), "GET", server.URL, "", nil)
			require.NoError(t, err)
		})
	})

	t.Run("Error case - nil client", func(t *testing.T) {
		client := &Client{}
		response, err := client.MakeRequest(context.Background(), "GET", "/test", "", nil)
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Error case - invalid URL", func(t *testing.T) {
		client := NewClient()
		_, err := client.MakeRequest(context.Background(), "GET", "/test", "", nil)
		require.Error(t, err)
		var execErr *ExecutionError
		assert.ErrorAs(t, err, &execErr)
	})
}

// headerInjectingMiddleware writes a header in BeforeRequest, like auth/trace
// middleware would.
type headerInjectingMiddleware struct {
	key   string
	value string
}

func (m *headerInjectingMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
	headers[m.key] = m.value
	return ctx
}

func (m *headerInjectingMiddleware) AfterRequest(ctx context.Context, info RequestInfo) {}

func TestClient_HandleResponse(t *testing.T) {
	client := NewClient()

	t.Run("Success case", func(t *testing.T) {
		err := client.handleResponse(&Response{StatusCode: http.StatusOK})
		assert.NoError(t, err)
	})

	t.Run("Unauthorized case", func(t *testing.T) {
		err := client.handleResponse(&Response{StatusCode: http.StatusUnauthorized})
		require.Error(t, err)
		var unauthorizedErr *UnauthorizedError
		require.ErrorAs(t, err, &unauthorizedErr)
		assert.Equal(t, http.StatusUnauthorized, unauthorizedErr.StatusCode)
	})

	t.Run("Status code 0 is not an error", func(t *testing.T) {
		err := client.handleResponse(&Response{StatusCode: 0})
		assert.NoError(t, err)
	})

	t.Run("Response error case", func(t *testing.T) {
		err := client.handleResponse(&Response{StatusCode: http.StatusBadRequest})
		require.Error(t, err)
		var responseErr *ResponseError
		require.ErrorAs(t, err, &responseErr)
		assert.Equal(t, http.StatusBadRequest, responseErr.StatusCode)
	})
}

func TestResponse_IsServerError(t *testing.T) {
	assert.False(t, (&Response{StatusCode: http.StatusOK}).IsServerError())
	assert.True(t, (&Response{StatusCode: http.StatusInternalServerError}).IsServerError())
	assert.False(t, (&Response{StatusCode: http.StatusBadRequest}).IsServerError())
}

func TestResponse_IsAuthError(t *testing.T) {
	assert.True(t, (&Response{StatusCode: http.StatusUnauthorized}).IsAuthError())
	assert.True(t, (&Response{StatusCode: http.StatusForbidden}).IsAuthError())
	assert.False(t, (&Response{StatusCode: http.StatusOK}).IsAuthError())
}

func TestWithOTelConfig(t *testing.T) {
	t.Run("stores OTel config on client for later merge", func(t *testing.T) {
		otelCfg := otel.NewConfig("test-service")

		client := &Client{restConfig: &Config{}}
		WithOTelConfig(otelCfg)(client)

		assert.Same(t, otelCfg, client.otelConfig)
	})

	t.Run("merged into restConfig via NewClient", func(t *testing.T) {
		otelCfg := otel.NewConfig("test-service")
		client := NewClient(WithOTelConfig(otelCfg))
		assert.Same(t, otelCfg, client.restConfig.OTelConfig)
	})

	t.Run("nil is a no-op and does not clear WithRestConfig OTel", func(t *testing.T) {
		otelCfg := otel.NewConfig("test-service")
		cfg := DefaultRestConfig()
		cfg.OTelConfig = otelCfg
		client := NewClient(WithRestConfig(*cfg), WithOTelConfig(nil))
		assert.Same(t, otelCfg, client.restConfig.OTelConfig)
	})
}

func TestSetMiddlewares(t *testing.T) {
	t.Run("replaces existing middlewares", func(t *testing.T) {
		client := NewClient()
		assert.NotEmpty(t, client.GetMiddlewares())

		client.SetMiddlewares(&TestMiddleware{Name: "test1"}, &TestMiddleware{Name: "test2"})
		assert.Len(t, client.GetMiddlewares(), 2)
	})

	t.Run("can set empty middlewares list", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares()
		assert.Empty(t, client.GetMiddlewares())
	})

	t.Run("is thread-safe", func(t *testing.T) {
		client := NewClient()
		done := make(chan bool, 2)

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
		assert.Len(t, client.GetMiddlewares(), 1)
	})
}

func TestAddMiddleware(t *testing.T) {
	t.Run("appends middleware to existing list", func(t *testing.T) {
		client := NewClient()
		initial := len(client.GetMiddlewares())
		client.AddMiddleware(&TestMiddleware{Name: "additional"})
		assert.Len(t, client.GetMiddlewares(), initial+1)
	})

	t.Run("maintains order of middlewares", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares() // Clear defaults

		client.AddMiddleware(&TestMiddleware{Name: "first"})
		client.AddMiddleware(&TestMiddleware{Name: "second"})
		client.AddMiddleware(&TestMiddleware{Name: "third"})

		middlewares := client.GetMiddlewares()
		require.Len(t, middlewares, 3)

		m0, ok := middlewares[0].(*TestMiddleware)
		require.True(t, ok)
		assert.Equal(t, "first", m0.Name)
		m1, ok := middlewares[1].(*TestMiddleware)
		require.True(t, ok)
		assert.Equal(t, "second", m1.Name)
		m2, ok := middlewares[2].(*TestMiddleware)
		require.True(t, ok)
		assert.Equal(t, "third", m2.Name)
	})
}

func TestGetMiddlewares(t *testing.T) {
	t.Run("returns copy of middlewares", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares(&TestMiddleware{Name: "test"})

		middlewares1 := client.GetMiddlewares()
		middlewares2 := client.GetMiddlewares()
		assert.NotSame(t, &middlewares1[0], &middlewares2[0], "GetMiddlewares should return a copy")
	})

	t.Run("modifications to returned slice don't affect client", func(t *testing.T) {
		client := NewClient()
		client.SetMiddlewares(&TestMiddleware{Name: "test"})

		middlewares := client.GetMiddlewares()
		middlewares[0] = &TestMiddleware{Name: "modified"}

		clientMiddlewares := client.GetMiddlewares()
		m, ok := clientMiddlewares[0].(*TestMiddleware)
		require.True(t, ok)
		assert.Equal(t, "test", m.Name)
	})
}

func TestWithOTelConfig_NilRestConfig(t *testing.T) {
	client := &Client{} // no restConfig
	require.NotPanics(t, func() { WithOTelConfig(nil)(client) })
}

func TestTruncateBody(t *testing.T) {
	assert.Equal(t, "short", truncateBody("short", 1024))

	long := strings.Repeat("x", 2000)
	result := truncateBody(long, 1024)
	assert.Len(t, result, 1024+len("...(truncated)"))
	assert.True(t, strings.HasSuffix(result, "...(truncated)"))
}

func TestClient_MakeRequestWithTrace(t *testing.T) {
	t.Run("returns error when client is nil", func(t *testing.T) {
		client := &Client{restClient: nil, restConfig: DefaultRestConfig()}
		response, err := client.MakeRequestWithTrace(context.Background(), "GET", "http://example.com", "", map[string]string{})
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("makes successful GET request with trace", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		}))
		defer server.Close()

		client := NewClient()
		response, err := client.MakeRequestWithTrace(context.Background(), "GET", server.URL, "", map[string]string{})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("works with middleware", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		client.SetMiddlewares(&TestMiddleware{Name: "trace-test"})

		response, err := client.MakeRequestWithTrace(context.Background(), "GET", server.URL, "", map[string]string{})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("supports different HTTP methods", func(t *testing.T) {
		methodReceived := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			methodReceived = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
			methodReceived = ""
			_, err := client.MakeRequestWithTrace(context.Background(), method, server.URL, "", map[string]string{})
			require.NoError(t, err, "method %s", method)
			if method != "HEAD" && method != "OPTIONS" {
				assert.Equal(t, method, methodReceived)
			}
		}
	})

	t.Run("includes request body", func(t *testing.T) {
		bodyReceived := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				buf := new(strings.Builder)
				_, _ = io.Copy(buf, r.Body)
				bodyReceived = buf.String()
			}
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := NewClient()
		body := "test request body"
		response, err := client.MakeRequestWithTrace(context.Background(), "POST", server.URL, body, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, response.StatusCode)
		assert.Equal(t, body, bodyReceived)
	})

	t.Run("handles server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}))
		defer server.Close()

		client := NewClient()
		response, err := client.MakeRequestWithTrace(context.Background(), "GET", server.URL, "", map[string]string{})
		require.Error(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})

	t.Run("enables tracing on request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		middleware := &mockMiddleware{}
		client.SetMiddlewares(middleware)

		response, err := client.MakeRequestWithTrace(context.Background(), "GET", server.URL, "", map[string]string{})
		require.NoError(t, err)
		require.NotNil(t, response)
		require.True(t, middleware.afterRequestCalled)
		assert.Positive(t, middleware.requestInfo.TraceInfo.TotalTime, "TraceInfo.TotalTime should be populated when trace is enabled")
	})
}
