package rest_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/jasoet/pkg/v3/rest"
)

// NewClient builds a client from functional options. Fields not set in the
// provided Config keep their zero values, so start from DefaultRestConfig
// when you only want to tweak a few fields.
func ExampleNewClient() {
	cfg := rest.DefaultRestConfig()
	cfg.RetryCount = 3
	cfg.Timeout = 10 * time.Second

	client := rest.NewClient(rest.WithRestConfig(*cfg))

	actual := client.GetRestConfig()
	fmt.Println("retryCount:", actual.RetryCount)
	fmt.Println("retryWaitTime:", actual.RetryWaitTime)
	fmt.Println("timeout:", actual.Timeout)
	fmt.Println("maxResponseBodyLog:", actual.MaxResponseBodyLog)

	// Output:
	// retryCount: 3
	// retryWaitTime: 2s
	// timeout: 10s
	// maxResponseBodyLog: 1024
}

// MakeRequest returns the library-owned *rest.Response with status
// predicates. Non-2xx responses also produce a typed error suitable for
// errors.As.
func ExampleClient_MakeRequest() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Replace the default LoggingMiddleware to keep the example output clean.
	client := rest.NewClient(rest.WithMiddlewares(rest.NewNoOpMiddleware()))
	ctx := context.Background()

	resp, err := client.MakeRequest(ctx, http.MethodGet, server.URL+"/users", "", nil)
	fmt.Println("err:", err)
	fmt.Println("status:", resp.StatusCode)
	fmt.Println("isSuccess:", resp.IsSuccess())
	fmt.Println("isError:", resp.IsError())
	fmt.Println("body:", resp.Body)

	// Non-2xx responses return both the Response and a typed error.
	resp, err = client.MakeRequest(ctx, http.MethodGet, server.URL+"/missing", "", nil)
	var notFound *rest.ResourceNotFoundError
	fmt.Println("notFoundErr:", errors.As(err, &notFound))
	fmt.Println("isNotFound:", resp.IsNotFound())

	// Output:
	// err: <nil>
	// status: 200
	// isSuccess: true
	// isError: false
	// body: {"status":"ok"}
	// notFoundErr: true
	// isNotFound: true
}
