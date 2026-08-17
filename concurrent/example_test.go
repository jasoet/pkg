package concurrent_test

import (
	"context"
	"fmt"

	"github.com/jasoet/pkg/v3/concurrent"
)

// ExecuteConcurrently runs named functions in parallel and returns their
// results keyed by name.
func ExampleExecuteConcurrently() {
	funcs := map[string]concurrent.Func[string]{
		"greeting": func(ctx context.Context) (string, error) {
			return "hello", nil
		},
		"name": func(ctx context.Context) (string, error) {
			return "world", nil
		},
	}

	results, err := concurrent.ExecuteConcurrently(context.Background(), funcs)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(results["greeting"], results["name"])

	// Output:
	// hello world
}

// ExecuteConcurrentlyTyped adds a result builder on top of
// ExecuteConcurrently. Type parameters are result-first:
// ExecuteConcurrentlyTyped[Output, Input].
func ExampleExecuteConcurrentlyTyped() {
	funcs := map[string]concurrent.Func[int]{
		"users": func(ctx context.Context) (int, error) {
			return 10, nil
		},
		"orders": func(ctx context.Context) (int, error) {
			return 32, nil
		},
	}

	summary, err := concurrent.ExecuteConcurrentlyTyped[string, int](
		context.Background(),
		func(results map[string]int) (string, error) {
			return fmt.Sprintf("users=%d orders=%d", results["users"], results["orders"]), nil
		},
		funcs,
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(summary)

	// Output:
	// users=10 orders=32
}
