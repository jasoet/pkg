// Package concurrent provides utilities for executing multiple functions concurrently
// with error propagation, context cancellation, and panic recovery.
package concurrent

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Func is a generic function type that accepts a context and returns a value of type T and an error.
type Func[T any] func(ctx context.Context) (T, error)

// ExecuteConcurrently executes multiple functions concurrently and collects their results.
//
// All functions receive a shared cancellable context. When any function returns an error
// or panics, the context is cancelled to signal other goroutines to stop.
//
// Returns a map of results indexed by the provided keys, and the first causal error
// encountered (preferring real errors over context cancellation errors).
// If a function panics, the panic is recovered and converted to an error.
func ExecuteConcurrently[T any](ctx context.Context, funcs map[string]Func[T]) (map[string]T, error) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(funcs))

	type result struct {
		key   string
		value T
		err   error
	}

	resultCh := make(chan result, len(funcs))

	for key, fn := range funcs {
		go func(key string, fn Func[T]) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					resultCh <- result{key: key, err: fmt.Errorf("panic in %q: %v", key, r)}
					cancel()
				}
			}()

			value, err := fn(ctxWithCancel)
			resultCh <- result{key: key, value: value, err: err}
			if err != nil {
				cancel()
			}
		}(key, fn)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make(map[string]T)
	var firstErr error
	for res := range resultCh {
		if res.err != nil {
			// Prefer causal errors over context cancellation errors.
			if firstErr == nil || (isContextErr(firstErr) && !isContextErr(res.err)) {
				firstErr = res.err
			}
		} else {
			results[res.key] = res.value
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// isContextErr reports whether the error is a context cancellation or deadline error.
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// ExecuteConcurrentlyTyped executes multiple functions concurrently and transforms
// the results into a typed struct using the provided resultBuilder function.
//
// This is a more type-safe alternative to ExecuteConcurrently when you know the
// exact structure of the results.
func ExecuteConcurrentlyTyped[T any, R any](
	ctx context.Context,
	resultBuilder func(map[string]T) (R, error),
	funcs map[string]Func[T],
) (R, error) {
	var zero R
	results, err := ExecuteConcurrently(ctx, funcs)
	if err != nil {
		return zero, err
	}
	return resultBuilder(results)
}
