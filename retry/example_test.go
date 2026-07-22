package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jasoet/pkg/v3/retry"
)

// New starts from DefaultConfig and applies each option in order.
func ExampleNew() {
	cfg := retry.New(
		retry.WithName("db.connect"),
		retry.WithMaxRetries(3),
		retry.WithInitialInterval(100*time.Millisecond),
	)

	fmt.Println("name:", cfg.Name)
	fmt.Println("maxRetries:", cfg.MaxRetries)
	fmt.Println("initialInterval:", cfg.InitialInterval)
	// Untouched fields keep their defaults:
	fmt.Println("maxInterval:", cfg.MaxInterval)
	fmt.Println("multiplier:", cfg.Multiplier)

	// Output:
	// name: db.connect
	// maxRetries: 3
	// initialInterval: 100ms
	// maxInterval: 1m0s
	// multiplier: 2
}

// Do retries the operation with exponential backoff until it succeeds.
// Here the operation fails twice, then succeeds on the third attempt.
func ExampleDo() {
	cfg := retry.New(
		retry.WithName("flaky.op"),
		retry.WithMaxRetries(5),
		retry.WithInitialInterval(time.Millisecond),
		retry.WithRandomizationFactor(0), // disable jitter for deterministic timing
	)

	attempts := 0
	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	fmt.Println("attempts:", attempts)
	fmt.Println("err:", err)

	// Output:
	// attempts: 3
	// err: <nil>
}

// Permanent marks an error as non-retryable: Do stops after the first
// attempt. The returned error wraps the original one, so errors.Is/As
// still match it.
func ExamplePermanent() {
	cfg := retry.New(
		retry.WithName("validate.input"),
		retry.WithMaxRetries(5),
		retry.WithInitialInterval(time.Millisecond),
	)

	attempts := 0
	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return retry.Permanent(errors.New("invalid input"))
	})

	fmt.Println("attempts:", attempts)
	fmt.Println("err:", err)

	// Output:
	// attempts: 1
	// err: validate.input failed after 1 attempts (1 initial + 0 retries): invalid input
}

// DoWithNotify calls the notify function before each retry wait. With
// RandomizationFactor 0 the backoff durations are exact powers of the
// multiplier: 10ms, then 20ms.
func ExampleDoWithNotify() {
	cfg := retry.New(
		retry.WithName("notify.op"),
		retry.WithMaxRetries(3),
		retry.WithInitialInterval(10*time.Millisecond),
		retry.WithMultiplier(2.0),
		retry.WithRandomizationFactor(0),
	)

	attempts := 0
	err := retry.DoWithNotify(context.Background(), cfg,
		func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("failure #%d", attempts)
			}
			return nil
		},
		func(err error, backoff time.Duration) {
			fmt.Printf("retrying in %v: %v\n", backoff, err)
		},
	)

	fmt.Println("attempts:", attempts)
	fmt.Println("err:", err)

	// Output:
	// retrying in 10ms: failure #1
	// retrying in 20ms: failure #2
	// attempts: 3
	// err: <nil>
}
