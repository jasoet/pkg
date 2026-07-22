package server

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
)

func ExampleNew() {
	// Port 0 asks the OS for an ephemeral port; Addr() reports it once Start
	// has bound the listener.
	srv, err := New(WithPort(0))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Exercise the built-in health endpoint through the Echo instance without
	// binding a real listener.
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	srv.Echo().ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)
	fmt.Println(rec.Result().StatusCode)
	fmt.Println(strings.TrimSpace(string(body)))

	// Output:
	// 200
	// {"status":"UP"}
}

func ExampleServer_Shutdown() {
	srv, err := New(WithPort(0))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Shutdown from another goroutine; Start then returns nil once the server
	// has drained.
	go func() {
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.Start(); err != nil {
		fmt.Println("error:", err)
	}
}
