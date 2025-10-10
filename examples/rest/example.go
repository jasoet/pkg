//go:build example

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/jasoet/pkg/v2/rest"
	"github.com/rs/zerolog"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

type APIResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error,omitempty"`
}

type AuthMiddleware struct {
	token string
}

func (m *AuthMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
	if headers != nil {
		headers["Authorization"] = "Bearer " + m.token
		headers["X-API-Version"] = "v1"
	}
	return ctx
}

func (m *AuthMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
	if info.StatusCode == 401 {
		fmt.Printf("   ⚠ Authentication failed for %s\n", info.URL)
	}
}

type MetricsMiddleware struct {
	requestCount int
	totalTime    time.Duration
	mu           sync.Mutex
}

func (m *MetricsMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
	return ctx
}

func (m *MetricsMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount++
	m.totalTime += info.Duration
}

func (m *MetricsMiddleware) GetStats() (int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount, m.totalTime
}

func main() {
	// Initialize logging
	logging.Initialize("rest-examples", true)

	fmt.Println("REST Package Examples")
	fmt.Println("====================")

	// Example 1: Basic HTTP Client
	fmt.Println("\n1. Basic HTTP Client")
	basicHTTPClientExample()

	// Example 2: Client with Custom Configuration
	fmt.Println("\n2. Client with Custom Configuration")
	customConfigurationExample()

	// Example 3: Middleware Integration
	fmt.Println("\n3. Middleware Integration")
	middlewareIntegrationExample()

	// Example 4: Error Handling
	fmt.Println("\n4. Error Handling")
	errorHandlingExample()

	// Example 5: JSON API Interactions
	fmt.Println("\n5. JSON API Interactions")
	jsonAPIExample()

	// Example 6: Retry and Timeout Patterns
	fmt.Println("\n6. Retry and Timeout Patterns")
	retryTimeoutExample()

	// Example 7: Request Tracing and Performance Monitoring
	fmt.Println("\n7. Request Tracing and Performance Monitoring")
	tracingPerformanceExample()

	// Example 8: Advanced Resty Client Usage
	fmt.Println("\n8. Advanced Resty Client Usage")
	advancedRestyExample()

	// Example 9: Integration with Other Packages
	fmt.Println("\n9. Integration with Other Packages")
	integrationExample()

	// Example 10: Production Patterns
	fmt.Println("\n10. Production Patterns")
	productionPatternsExample()
}

func basicHTTPClientExample() {
	ctx := context.Background()

	// Create client with default configuration
	client := rest.NewClient()

	fmt.Println("Creating basic HTTP client with default configuration:")
	config := client.GetRestConfig()
	fmt.Printf("- Retry Count: %d\n", config.RetryCount)
	fmt.Printf("- Timeout: %v\n", config.Timeout)
	fmt.Printf("- Retry Wait Time: %v\n", config.RetryWaitTime)

	// Create a mock server for demonstration
	server := createMockServer()
	defer server.Close()

	// Make a simple GET request
	fmt.Printf("\nMaking GET request to mock server...\n")
	response, err := client.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
	if err != nil {
		fmt.Printf("✗ Request failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Request successful:\n")
	fmt.Printf("  - Status Code: %d\n", response.StatusCode())
	fmt.Printf("  - Response Length: %d bytes\n", len(response.Body()))
	fmt.Printf("  - Content: %s\n", string(response.Body()[:min(100, len(response.Body()))]))
}

func customConfigurationExample() {
	ctx := context.Background()

	// Different configurations for different environments
	configs := map[string]*rest.Config{
		"development": {
			RetryCount:       1,
			RetryWaitTime:    1 * time.Second,
			RetryMaxWaitTime: 5 * time.Second,
			Timeout:          10 * time.Second,
		},
		"production": {
			RetryCount:       3,
			RetryWaitTime:    500 * time.Millisecond,
			RetryMaxWaitTime: 30 * time.Second,
			Timeout:          60 * time.Second,
		},
		"high-performance": {
			RetryCount:       2,
			RetryWaitTime:    100 * time.Millisecond,
			RetryMaxWaitTime: 2 * time.Second,
			Timeout:          5 * time.Second,
		},
	}

	server := createMockServer()
	defer server.Close()

	for env, config := range configs {
		fmt.Printf("\n%s Configuration:\n", env)
		fmt.Printf("- Retry Count: %d\n", config.RetryCount)
		fmt.Printf("- Timeout: %v\n", config.Timeout)
		fmt.Printf("- Retry Wait: %v - %v\n", config.RetryWaitTime, config.RetryMaxWaitTime)

		client := rest.NewClient(rest.WithRestConfig(*config))

		start := time.Now()
		response, err := client.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("✗ Request failed: %v\n", err)
		} else {
			fmt.Printf("✓ Request completed in %v (Status: %d)\n", duration, response.StatusCode())
		}
	}
}

func middlewareIntegrationExample() {
	ctx := context.Background()
	server := createMockServer()
	defer server.Close()

	// Example 1: Built-in logging middleware
	fmt.Println("Testing built-in logging middleware:")
	loggingClient := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))

	_, err := loggingClient.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
	if err != nil {
		fmt.Printf("✗ Request with logging middleware failed: %v\n", err)
	} else {
		fmt.Printf("✓ Request with logging middleware completed\n")
	}

	// Example 2: Custom authentication middleware
	fmt.Println("\nTesting custom authentication middleware:")
	authMiddleware := &AuthMiddleware{token: "example-jwt-token"}
	authClient := rest.NewClient(rest.WithMiddleware(authMiddleware))

	headers := map[string]string{}
	_, err = authClient.MakeRequest(ctx, http.MethodGet, server.URL+"/protected", "", headers)
	if err != nil {
		fmt.Printf("✗ Request with auth middleware failed: %v\n", err)
	} else {
		fmt.Printf("✓ Request with auth middleware completed\n")
	}

	// Example 3: Multiple middleware
	fmt.Println("\nTesting multiple middleware:")
	metricsMiddleware := &MetricsMiddleware{}
	multiClient := rest.NewClient(rest.WithMiddlewares(
		rest.NewLoggingMiddleware(),
		authMiddleware,
		metricsMiddleware,
	))

	// Make several requests
	for i := 1; i <= 3; i++ {
		_, err := multiClient.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
		if err != nil {
			fmt.Printf("✗ Request %d failed: %v\n", i, err)
		} else {
			fmt.Printf("✓ Request %d completed\n", i)
		}
	}

	// Show metrics
	count, totalTime := metricsMiddleware.GetStats()
	fmt.Printf("Metrics: %d requests, average time: %v\n", count, totalTime/time.Duration(count))
}

func errorHandlingExample() {
	ctx := context.Background()

	// Create a server that returns different error types
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/401":
			w.WriteHeader(401)
			w.Write([]byte(`{"error": "unauthorized"}`))
		case "/403":
			w.WriteHeader(403)
			w.Write([]byte(`{"error": "forbidden"}`))
		case "/404":
			w.WriteHeader(404)
			w.Write([]byte(`{"error": "not found"}`))
		case "/500":
			w.WriteHeader(500)
			w.Write([]byte(`{"error": "internal server error"}`))
		case "/timeout":
			time.Sleep(2 * time.Second) // Simulate slow response
			w.WriteHeader(200)
			w.Write([]byte(`{"status": "success"}`))
		}
	}))
	defer errorServer.Close()

	client := rest.NewClient(rest.WithRestConfig(rest.Config{
		RetryCount: 1,
		Timeout:    1 * time.Second, // Short timeout for demonstration
	}))

	// Test different error scenarios
	errorTests := []struct {
		path        string
		description string
	}{
		{"/401", "Unauthorized Error"},
		{"/403", "Forbidden Error"},
		{"/404", "Not Found Error"},
		{"/500", "Server Error"},
		{"/timeout", "Timeout Error"},
		{"/nonexistent", "Network Error"},
	}

	for _, test := range errorTests {
		fmt.Printf("\nTesting %s:\n", test.description)

		var url string
		if test.path == "/nonexistent" {
			url = "http://nonexistent-domain-12345.com/api"
		} else {
			url = errorServer.URL + test.path
		}

		_, err := client.MakeRequest(ctx, http.MethodGet, url, "", nil)
		if err != nil {
			handleAPIError(err)
		} else {
			fmt.Printf("✓ Request succeeded unexpectedly\n")
		}
	}
}

func handleAPIError(err error) {
	switch e := err.(type) {
	case *rest.UnauthorizedError:
		fmt.Printf("✗ Authentication Error: Status %d - %s\n", e.StatusCode, e.Error())
		fmt.Printf("  Action: Check credentials or refresh token\n")
	case *rest.ServerError:
		fmt.Printf("✗ Server Error: Status %d - %s\n", e.StatusCode, e.Error())
		fmt.Printf("  Action: Implement retry with backoff\n")
	case *rest.ResponseError:
		fmt.Printf("✗ Response Error: Status %d - %s\n", e.StatusCode, e.Error())
		fmt.Printf("  Action: Check request parameters\n")
	case *rest.ResourceNotFoundError:
		fmt.Printf("✗ Not Found Error: Status %d - %s\n", e.StatusCode, e.Error())
		fmt.Printf("  Action: Verify resource exists\n")
	case *rest.ExecutionError:
		fmt.Printf("✗ Execution Error: %s\n", e.Error())
		if e.Unwrap() != nil {
			fmt.Printf("  Underlying error: %s\n", e.Unwrap().Error())
		}
		fmt.Printf("  Action: Check network connectivity\n")
	default:
		fmt.Printf("✗ Unknown Error: %s\n", err.Error())
	}
}

func jsonAPIExample() {
	ctx := context.Background()

	// Create a mock JSON API server
	jsonServer := createJSONAPIServer()
	defer jsonServer.Close()

	client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))

	// Example 1: GET request with JSON response
	fmt.Println("Making GET request for JSON data:")
	response, err := client.MakeRequest(ctx, http.MethodGet, jsonServer.URL+"/users", "", nil)
	if err != nil {
		fmt.Printf("✗ GET request failed: %v\n", err)
		return
	}

	var users []User
	if err := json.Unmarshal(response.Body(), &users); err != nil {
		fmt.Printf("✗ JSON parsing failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Retrieved %d users:\n", len(users))
	for _, user := range users {
		fmt.Printf("  - %s (%s)\n", user.Name, user.Email)
	}

	// Example 2: POST request with JSON body
	fmt.Println("\nMaking POST request with JSON body:")
	newUser := User{Name: "John Doe", Email: "john@example.com"}
	jsonBody, _ := json.Marshal(newUser)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	response, err = client.MakeRequest(ctx, http.MethodPost, jsonServer.URL+"/users", string(jsonBody), headers)
	if err != nil {
		fmt.Printf("✗ POST request failed: %v\n", err)
		return
	}

	var createdUser User
	if err := json.Unmarshal(response.Body(), &createdUser); err != nil {
		fmt.Printf("✗ JSON parsing failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Created user: %s (ID: %d)\n", createdUser.Name, createdUser.ID)

	// Example 3: PUT request for updates
	fmt.Println("\nMaking PUT request to update user:")
	createdUser.Email = "john.doe@newdomain.com"
	updateBody, _ := json.Marshal(createdUser)

	response, err = client.MakeRequest(ctx, http.MethodPut, fmt.Sprintf("%s/users/%d", jsonServer.URL, createdUser.ID), string(updateBody), headers)
	if err != nil {
		fmt.Printf("✗ PUT request failed: %v\n", err)
		return
	}

	fmt.Printf("✓ User updated successfully (Status: %d)\n", response.StatusCode())
}

func retryTimeoutExample() {
	ctx := context.Background()

	// Create a server that simulates unreliable behavior
	unreliableServer := createUnreliableServer()
	defer unreliableServer.Close()

	fmt.Println("Testing retry behavior with unreliable server:")

	// Configuration with aggressive retries
	retryConfig := &rest.Config{
		RetryCount:       3,
		RetryWaitTime:    500 * time.Millisecond,
		RetryMaxWaitTime: 2 * time.Second,
		Timeout:          5 * time.Second,
	}

	client := rest.NewClient(rest.WithRestConfig(*retryConfig))

	// Test endpoints with different behaviors
	endpoints := []string{"/flaky", "/slow", "/eventually-success"}

	for _, endpoint := range endpoints {
		fmt.Printf("\nTesting %s:\n", endpoint)
		start := time.Now()

		response, err := client.MakeRequest(ctx, http.MethodGet, unreliableServer.URL+endpoint, "", nil)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("✗ Request failed after %v: %v\n", duration, err)
		} else {
			fmt.Printf("✓ Request succeeded after %v (Status: %d)\n", duration, response.StatusCode())
		}
	}

	// Test with context timeout
	fmt.Println("\nTesting with context timeout:")
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err := client.MakeRequest(timeoutCtx, http.MethodGet, unreliableServer.URL+"/very-slow", "", nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("✓ Request properly timed out after %v: %v\n", duration, err)
	} else {
		fmt.Printf("✗ Request should have timed out but didn't\n")
	}
}

func tracingPerformanceExample() {
	ctx := context.Background()
	server := createMockServer()
	defer server.Close()

	fmt.Println("Demonstrating request tracing and performance monitoring:")

	client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))

	// Make requests and analyze trace information
	endpoints := []string{"/users", "/posts", "/slow"}

	for _, endpoint := range endpoints {
		fmt.Printf("\nTracing request to %s:\n", endpoint)

		start := time.Now()
		response, err := client.MakeRequest(ctx, http.MethodGet, server.URL+endpoint, "", nil)
		totalDuration := time.Since(start)

		if err != nil {
			fmt.Printf("✗ Request failed: %v\n", err)
			continue
		}

		// Access trace information from the underlying Resty response
		if response != nil && response.Request != nil {
			traceInfo := response.Request.TraceInfo()

			fmt.Printf("✓ Request completed successfully:\n")
			fmt.Printf("  - Total Duration: %v\n", totalDuration)
			fmt.Printf("  - DNS Lookup: %v\n", traceInfo.DNSLookup)
			fmt.Printf("  - TCP Connection: %v\n", traceInfo.TCPConnTime)
			fmt.Printf("  - TLS Handshake: %v\n", traceInfo.TLSHandshake)
			fmt.Printf("  - Server Time: %v\n", traceInfo.ServerTime)
			fmt.Printf("  - Response Time: %v\n", traceInfo.ResponseTime)
			fmt.Printf("  - Status Code: %d\n", response.StatusCode())
			fmt.Printf("  - Response Size: %d bytes\n", len(response.Body()))
		}
	}

	// Performance benchmark
	fmt.Println("\nPerformance benchmark (10 requests):")
	benchmarkStart := time.Now()

	for i := 0; i < 10; i++ {
		_, err := client.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
		if err != nil {
			fmt.Printf("✗ Request %d failed: %v\n", i+1, err)
		}
	}

	benchmarkDuration := time.Since(benchmarkStart)
	avgDuration := benchmarkDuration / 10
	fmt.Printf("✓ 10 requests completed in %v (avg: %v per request)\n", benchmarkDuration, avgDuration)
}

func advancedRestyExample() {
	ctx := context.Background()
	server := createJSONAPIServer()
	defer server.Close()

	fmt.Println("Demonstrating advanced Resty client features:")

	client := rest.NewClient()
	restyClient := client.GetRestClient()

	// Example 1: Automatic JSON unmarshaling
	fmt.Println("\nUsing automatic JSON unmarshaling:")
	var users []User
	response, err := restyClient.R().
		SetContext(ctx).
		SetResult(&users). // Automatic JSON unmarshaling
		Get(server.URL + "/users")

	if err != nil {
		fmt.Printf("✗ Request failed: %v\n", err)
	} else {
		fmt.Printf("✓ Retrieved %d users via automatic unmarshaling\n", len(users))
	}

	// Example 2: Query parameters and headers
	fmt.Println("\nUsing query parameters and custom headers:")
	response, err = restyClient.R().
		SetContext(ctx).
		SetHeader("User-Agent", "rest-package-example/1.0").
		SetHeader("Accept", "application/json").
		SetQueryParam("limit", "5").
		SetQueryParam("sort", "name").
		Get(server.URL + "/users")

	if err != nil {
		fmt.Printf("✗ Request failed: %v\n", err)
	} else {
		fmt.Printf("✓ Request with query params completed (Status: %d)\n", response.StatusCode())
		fmt.Printf("  Request URL: %s\n", response.Request.URL)
	}

	// Example 3: Form data submission
	fmt.Println("\nSubmitting form data:")
	response, err = restyClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"name":  "Form User",
			"email": "form@example.com",
		}).
		Post(server.URL + "/users")

	if err != nil {
		fmt.Printf("✗ Form submission failed: %v\n", err)
	} else {
		fmt.Printf("✓ Form data submitted successfully (Status: %d)\n", response.StatusCode())
	}

	// Example 4: File upload simulation
	fmt.Println("\nSimulating file upload:")
	response, err = restyClient.R().
		SetContext(ctx).
		SetFileReader("file", "example.txt", strings.NewReader("This is example file content")).
		SetFormData(map[string]string{
			"description": "Example file upload",
		}).
		Post(server.URL + "/upload")

	if err != nil {
		fmt.Printf("✗ File upload failed: %v\n", err)
	} else {
		fmt.Printf("✓ File upload completed (Status: %d)\n", response.StatusCode())
	}
}

func integrationExample() {
	ctx := context.Background()

	// Integration with logging package
	logger := logging.ContextLogger(ctx, "api-integration")

	logger.Info().Msg("Starting API integration example")

	client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))
	server := createMockServer()
	defer server.Close()

	// Simulate a service that uses multiple packages
	userService := &UserService{
		client:  client,
		baseURL: server.URL,
		logger:  &logger,
	}

	// Use the service
	users, err := userService.GetUsers(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get users")
		return
	}

	logger.Info().Int("user_count", len(users)).Msg("Successfully retrieved users")

	// Create a new user
	newUser := User{Name: "Integration Test User", Email: "integration@example.com"}
	createdUser, err := userService.CreateUser(ctx, newUser)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create user")
		return
	}

	logger.Info().
		Int("user_id", createdUser.ID).
		Str("user_name", createdUser.Name).
		Msg("Successfully created user")
}

func productionPatternsExample() {
	ctx := context.Background()

	fmt.Println("Demonstrating production-ready patterns:")

	// Pattern 1: Client factory with environment configuration
	fmt.Println("\n1. Environment-based client configuration:")
	environments := []string{"development", "staging", "production"}

	for _, env := range environments {
		client := createEnvironmentClient(env)
		config := client.GetRestConfig()
		fmt.Printf("- %s: Timeout=%v, Retries=%d\n", env, config.Timeout, config.RetryCount)
	}

	// Pattern 2: Circuit breaker pattern
	fmt.Println("\n2. Circuit breaker pattern:")
	circuitBreaker := &CircuitBreaker{threshold: 3}
	client := rest.NewClient()
	server := createUnreliableServer()
	defer server.Close()

	for i := 1; i <= 5; i++ {
		if circuitBreaker.ShouldAllowRequest() {
			_, err := client.MakeRequest(ctx, http.MethodGet, server.URL+"/flaky", "", nil)
			if err != nil {
				circuitBreaker.RecordFailure()
				fmt.Printf("  Request %d failed, circuit breaker state: %s\n", i, circuitBreaker.GetState())
			} else {
				circuitBreaker.RecordSuccess()
				fmt.Printf("  Request %d succeeded\n", i)
			}
		} else {
			fmt.Printf("  Request %d blocked by circuit breaker\n", i)
		}
	}

	// Pattern 3: Graceful degradation
	fmt.Println("\n3. Graceful degradation:")
	primaryClient := rest.NewClient(rest.WithRestConfig(rest.Config{Timeout: 1 * time.Second}))
	fallbackClient := rest.NewClient(rest.WithRestConfig(rest.Config{Timeout: 5 * time.Second}))

	result := makeRequestWithFallback(ctx, primaryClient, fallbackClient, server.URL+"/slow")
	fmt.Printf("  Result: %s\n", result)
}

// Helper types and functions

type UserService struct {
	client  *rest.Client
	baseURL string
	logger  *zerolog.Logger
}

func (s *UserService) GetUsers(ctx context.Context) ([]User, error) {
	s.logger.Info().Msg("Fetching users from API")

	response, err := s.client.MakeRequest(ctx, http.MethodGet, s.baseURL+"/users", "", nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(response.Body(), &users); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) CreateUser(ctx context.Context, user User) (*User, error) {
	s.logger.Info().Str("user_name", user.Name).Msg("Creating new user")

	jsonBody, _ := json.Marshal(user)
	headers := map[string]string{"Content-Type": "application/json"}

	response, err := s.client.MakeRequest(ctx, http.MethodPost, s.baseURL+"/users", string(jsonBody), headers)
	if err != nil {
		return nil, err
	}

	var createdUser User
	if err := json.Unmarshal(response.Body(), &createdUser); err != nil {
		return nil, err
	}

	return &createdUser, nil
}

type CircuitBreaker struct {
	threshold    int
	failures     int
	lastFailTime time.Time
	state        string
}

func (cb *CircuitBreaker) ShouldAllowRequest() bool {
	if cb.failures >= cb.threshold && time.Since(cb.lastFailTime) < 30*time.Second {
		cb.state = "OPEN"
		return false
	}
	cb.state = "CLOSED"
	return true
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.failures = 0
}

func (cb *CircuitBreaker) GetState() string {
	return cb.state
}

func createEnvironmentClient(env string) *rest.Client {
	var config *rest.Config

	switch env {
	case "development":
		config = &rest.Config{
			RetryCount: 1,
			Timeout:    10 * time.Second,
		}
	case "staging":
		config = &rest.Config{
			RetryCount: 2,
			Timeout:    30 * time.Second,
		}
	case "production":
		config = &rest.Config{
			RetryCount: 3,
			Timeout:    60 * time.Second,
		}
	}

	return rest.NewClient(rest.WithRestConfig(*config))
}

func makeRequestWithFallback(ctx context.Context, primary, fallback *rest.Client, url string) string {
	// Try primary first
	_, err := primary.MakeRequest(ctx, http.MethodGet, url, "", nil)
	if err == nil {
		return "Primary service responded"
	}

	// Fallback to secondary
	_, err = fallback.MakeRequest(ctx, http.MethodGet, url, "", nil)
	if err == nil {
		return "Fallback service responded"
	}

	return "Both services failed"
}

// Mock servers for testing

func createMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request details
		fmt.Printf("   Mock server received: %s %s\n", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`[{"id":1,"name":"Alice","email":"alice@example.com"},{"id":2,"name":"Bob","email":"bob@example.com"}]`))
		case "/posts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`[{"id":1,"title":"Post 1","body":"Content 1","userId":1}]`))
		case "/slow":
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"slow response"}`))
		case "/protected":
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"authorized"}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
}

func createJSONAPIServer() *httptest.Server {
	userID := 3 // Start from 3 since mock data has users 1,2

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && r.URL.Path == "/users":
			w.WriteHeader(200)
			w.Write([]byte(`[{"id":1,"name":"Alice","email":"alice@example.com"},{"id":2,"name":"Bob","email":"bob@example.com"}]`))

		case r.Method == "POST" && r.URL.Path == "/users":
			var user User
			json.NewDecoder(r.Body).Decode(&user)
			user.ID = userID
			userID++
			response, _ := json.Marshal(user)
			w.WriteHeader(201)
			w.Write(response)

		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/users/"):
			var user User
			json.NewDecoder(r.Body).Decode(&user)
			response, _ := json.Marshal(user)
			w.WriteHeader(200)
			w.Write(response)

		case r.Method == "POST" && r.URL.Path == "/upload":
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"file uploaded successfully"}`))

		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"endpoint not found"}`))
		}
	}))
}

func createUnreliableServer() *httptest.Server {
	requestCount := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		switch r.URL.Path {
		case "/flaky":
			// Fail first 2 requests, succeed on 3rd
			if requestCount%3 != 0 {
				w.WriteHeader(500)
				w.Write([]byte(`{"error":"server error"}`))
			} else {
				w.WriteHeader(200)
				w.Write([]byte(`{"status":"success"}`))
			}

		case "/slow":
			time.Sleep(3 * time.Second)
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"slow success"}`))

		case "/very-slow":
			time.Sleep(10 * time.Second)
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"very slow success"}`))

		case "/eventually-success":
			// Fail first 2 attempts, succeed after
			if requestCount <= 2 {
				w.WriteHeader(503)
				w.Write([]byte(`{"error":"service unavailable"}`))
			} else {
				w.WriteHeader(200)
				w.Write([]byte(`{"status":"eventual success"}`))
			}

		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper interface for logger
type logger interface {
	Info() *zerolog.Event
	Error() *zerolog.Event
}
