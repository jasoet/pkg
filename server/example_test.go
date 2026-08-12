package server

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"time"
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

	// Start blocks, so run it in a goroutine and capture its result.
	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	// Wait until the listener is actually bound before shutting down. Without
	// this, Shutdown can win the race and run as a no-op (the server was never
	// started), leaving Start blocking forever.
	for srv.Addr() == "" {
		time.Sleep(time.Millisecond)
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		fmt.Println("shutdown error:", err)
	}
	if err := <-startErr; err != nil {
		fmt.Println("start error:", err)
	}

	fmt.Println("stopped cleanly")
	// Output:
	// stopped cleanly
}
