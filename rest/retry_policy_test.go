package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAlways500Server returns a server that always fails with 500 and counts calls.
func newAlways500Server() (*httptest.Server, *atomic.Int32) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	return server, &calls
}

func fastRetryConfig() *Config {
	cfg := DefaultRestConfig()
	cfg.RetryCount = 2
	cfg.RetryWaitTime = time.Millisecond
	cfg.RetryMaxWaitTime = 5 * time.Millisecond
	return cfg
}

// TestRetry_IdempotentRetriedByDefault confirms GET (idempotent) is retried.
func TestRetry_IdempotentRetriedByDefault(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	client := NewClient(WithRestConfig(*fastRetryConfig()))
	_, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.Error(t, err)
	assert.Equal(t, int32(3), calls.Load(), "GET must be retried: 1 initial + 2 retries")
}

// TestRetry_NonIdempotentNotRetriedByDefault confirms POST is NOT retried by
// default, so non-idempotent side effects are not duplicated.
func TestRetry_NonIdempotentNotRetriedByDefault(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	client := NewClient(WithRestConfig(*fastRetryConfig()))
	_, err := client.MakeRequest(context.Background(), http.MethodPost, server.URL, "", nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(), "POST must not be retried by default")
}

// TestRetry_PatchNotRetriedByDefault confirms PATCH is also treated as non-idempotent.
func TestRetry_PatchNotRetriedByDefault(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	client := NewClient(WithRestConfig(*fastRetryConfig()))
	_, err := client.MakeRequest(context.Background(), http.MethodPatch, server.URL, "", nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(), "PATCH must not be retried by default")
}

// TestRetry_NonIdempotentRetriedWhenOptedIn confirms WithRetryNonIdempotent
// enables retrying POST.
func TestRetry_NonIdempotentRetriedWhenOptedIn(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	client := NewClient(WithRestConfig(*fastRetryConfig()), WithRetryNonIdempotent())
	_, err := client.MakeRequest(context.Background(), http.MethodPost, server.URL, "", nil)
	require.Error(t, err)
	assert.Equal(t, int32(3), calls.Load(), "POST must be retried once opted in")
}

// TestRetry_OptInOrderIndependent confirms WithRetryNonIdempotent survives a
// later WithRestConfig (order independence).
func TestRetry_OptInOrderIndependent(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	client := NewClient(WithRetryNonIdempotent(), WithRestConfig(*fastRetryConfig()))
	_, err := client.MakeRequest(context.Background(), http.MethodPost, server.URL, "", nil)
	require.Error(t, err)
	assert.Equal(t, int32(3), calls.Load(), "opt-in must survive a later WithRestConfig")
}

// TestRetry_PermanentError_NoRetry verifies a permanent transport error
// (unsupported scheme / malformed URL) is not retried and returns promptly
// instead of burning the full backoff budget.
func TestRetry_PermanentError_NoRetry(t *testing.T) {
	cfg := DefaultRestConfig()
	cfg.RetryCount = 2
	cfg.RetryWaitTime = 2 * time.Second // large, so retries would be obvious
	cfg.RetryMaxWaitTime = 5 * time.Second
	client := NewClient(WithRestConfig(*cfg))

	start := time.Now()
	// Relative URL with no base -> unsupported protocol scheme (permanent).
	_, err := client.MakeRequest(context.Background(), http.MethodGet, "/no-scheme", "", nil)
	elapsed := time.Since(start)

	require.Error(t, err)
	var execErr *ExecutionError
	require.ErrorAs(t, err, &execErr)
	assert.Less(t, elapsed, time.Second, "permanent error must not be retried with backoff")
}

// TestRetry_ContextCanceled_NoRetry verifies a canceled context is treated as a
// permanent error and not retried.
func TestRetry_ContextCanceled_NoRetry(t *testing.T) {
	server, calls := newAlways500Server()
	defer server.Close()

	cfg := DefaultRestConfig()
	cfg.RetryCount = 3
	cfg.RetryWaitTime = 2 * time.Second
	cfg.RetryMaxWaitTime = 5 * time.Second
	client := NewClient(WithRestConfig(*cfg))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	start := time.Now()
	_, err := client.MakeRequest(ctx, http.MethodGet, server.URL, "", nil)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, time.Second, "canceled context must not trigger retry backoff")
	assert.LessOrEqual(t, calls.Load(), int32(1), "canceled request must not hammer the server")
}

// TestRetry_429_Retried verifies 429 Too Many Requests is retryable.
func TestRetry_429_Retried(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient(WithRestConfig(*fastRetryConfig()))
	resp, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), calls.Load(), "429 must be retried: 1 initial + 1 retry")
}

// TestRetry_RetryAfterHonored verifies the Retry-After header (delta-seconds)
// controls the retry delay.
func TestRetry_RetryAfterHonored(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultRestConfig()
	cfg.RetryCount = 2
	cfg.RetryWaitTime = time.Millisecond // small min so Retry-After (1s) is not clamped up
	cfg.RetryMaxWaitTime = 30 * time.Second
	client := NewClient(WithRestConfig(*cfg))

	start := time.Now()
	resp, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), calls.Load())
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "Retry-After: 1 must delay the retry ~1s")
}

// TestParseRetryAfter unit-tests the Retry-After parser for both forms.
func TestParseRetryAfter(t *testing.T) {
	assert.Equal(t, time.Duration(0), parseRetryAfter(""))
	assert.Equal(t, 5*time.Second, parseRetryAfter("5"))
	assert.Equal(t, time.Duration(0), parseRetryAfter("-3"))
	assert.Equal(t, time.Duration(0), parseRetryAfter("garbage"))
	// HTTP-date in the past yields 0.
	assert.Equal(t, time.Duration(0), parseRetryAfter("Mon, 02 Jan 2006 15:04:05 GMT"))
	// HTTP-date in the future yields a positive duration.
	future := time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat)
	assert.Greater(t, parseRetryAfter(future), time.Hour)
}

// TestIsIdempotentMethod unit-tests the method classification.
func TestIsIdempotentMethod(t *testing.T) {
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		assert.True(t, isIdempotentMethod(m), "%s should be idempotent", m)
	}
	for _, m := range []string{http.MethodPost, http.MethodPatch, "CUSTOM"} {
		assert.False(t, isIdempotentMethod(m), "%s should not be idempotent", m)
	}
}
