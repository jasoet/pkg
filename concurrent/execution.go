package concurrent

import (
	"context"
	"sync"
)

// Func is a generic type for repository functions that return a value of type T and an error
type Func[T any] func(ctx context.Context) (T, error)

// Result ConcurrentResult holds the result of a concurrent repository call
type Result[T any] struct {
	Value T
	Err   error
}

// ExecuteConcurrently executes multiple repository functions concurrently
// It returns a map of results indexed by the provided keys, and the first error encountered (if any)
func ExecuteConcurrently[T any](ctx context.Context, funcs map[string]Func[T]) (map[string]T, error) {
	// Create a cancelable context
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(len(funcs))

	// Create a channel for results
	resultCh := make(chan struct {
		key   string
		value T
		err   error
	}, len(funcs))

	// Launch goroutines for each repository call
	for key, fn := range funcs {
		go func(key string, fn Func[T]) {
			defer wg.Done()
			value, err := fn(ctxWithCancel)
			resultCh <- struct {
				key   string
				value T
				err   error
			}{key, value, err}
			if err != nil {
				cancel() // Cancel other operations if this one fails
			}
		}(key, fn)
	}

	// Close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results from the channel
	results := make(map[string]T)
	for result := range resultCh {
		if result.err != nil {
			return nil, result.err
		}
		results[result.key] = result.value
	}

	return results, nil
}

// ExecuteConcurrentlyTyped executes multiple repository functions concurrently and returns the results in a typed struct
// This is a more type-safe alternative to ExecuteConcurrently when you know the exact structure of the results
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
