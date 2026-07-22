//go:build example

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jasoet/pkg/v3/retry"
)

func main() {
	fmt.Println("=== Retry Package Examples ===")
	fmt.Println()

	// Example 1: Basic retry with default config
	example1BasicRetry()

	// Example 2: Custom backoff configuration
	example2CustomBackoff()

	// Example 3: Permanent errors
	example3PermanentErrors()

	// Example 4: Context cancellation
	example4ContextCancellation()

	// Example 5: Unlimited retries with timeout
	example5UnlimitedRetries()

	// Example 6: Custom notifications
	example6CustomNotifications()
}

// Example 1: Basic retry with default configuration
func example1BasicRetry() {
	fmt.Println("Example 1: Basic Retry")
	fmt.Println("----------------------")

	ctx := context.Background()
	cfg := retry.New(
		retry.WithName("example.basic"),
		retry.WithMaxRetries(3),
	)
	attempts := 0
	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		fmt.Printf("  Attempt %d...\n", attempts)
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Success after %d attempts\n", attempts)
	}
	fmt.Println()
}

// Example 2: Custom backoff configuration
func example2CustomBackoff() {
	fmt.Println("Example 2: Custom Backoff")
	fmt.Println("-------------------------")

	ctx := context.Background()
	cfg := retry.New(
		retry.WithName("example.custom"),
		retry.WithMaxRetries(4),
		retry.WithInitialInterval(100*time.Millisecond),
		retry.WithMaxInterval(1*time.Second),
		retry.WithMultiplier(1.5),
		retry.WithRandomizationFactor(0), // exact intervals: 100ms, 150ms, 225ms
	)
	attempts := 0

	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		fmt.Printf("  Attempt %d\n", attempts)

		if attempts < 4 {
			return errors.New("not ready")
		}
		return nil
	})

	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Success after %d attempts\n", attempts)
	}
	fmt.Println()
}

// Example 3: Permanent errors (no retry)
func example3PermanentErrors() {
	fmt.Println("Example 3: Permanent Errors")
	fmt.Println("---------------------------")

	ctx := context.Background()
	cfg := retry.New(
		retry.WithName("example.permanent"),
		retry.WithMaxRetries(5),
	)
	// Simulate validation that should not be retried
	validateInput := func(value string) error {
		if value == "" {
			return retry.Permanent(errors.New("validation error: empty input"))
		}
		return nil
	}

	attempts := 0
	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		fmt.Printf("  Attempt %d\n", attempts)
		return validateInput("") // Invalid input
	})
	if err != nil {
		fmt.Printf("  ❌ Failed immediately (no retry): %v\n", err)
		fmt.Printf("  Total attempts: %d (expected: 1)\n", attempts)
	}
	fmt.Println()
}

// Example 4: Context cancellation
func example4ContextCancellation() {
	fmt.Println("Example 4: Context Cancellation")
	fmt.Println("-------------------------------")

	ctx, cancel := context.WithCancel(context.Background())
	cfg := retry.New(
		retry.WithName("example.cancel"),
		retry.WithMaxRetries(10),
		retry.WithInitialInterval(50*time.Millisecond),
	)
	attempts := 0

	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		fmt.Printf("  Attempt %d\n", attempts)
		if attempts == 2 {
			// Simulate an external shutdown signal.
			fmt.Println("  🛑 Cancelling context...")
			cancel()
		}
		return errors.New("still failing")
	})
	if err != nil {
		fmt.Printf("  ❌ Cancelled: %v\n", err)
		fmt.Printf("  Stopped after %d attempts\n", attempts)
	}
	fmt.Println()
}

// Example 5: Unlimited retries with timeout
func example5UnlimitedRetries() {
	fmt.Println("Example 5: Unlimited Retries with Timeout")
	fmt.Println("-----------------------------------------")

	// Set timeout instead of max retries
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cfg := retry.New(
		retry.WithName("example.unlimited"),
		retry.WithMaxRetries(0), // Unlimited!
		retry.WithInitialInterval(50*time.Millisecond),
		retry.WithMaxInterval(100*time.Millisecond),
		retry.WithRandomizationFactor(0),
	)

	attempts := 0
	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		fmt.Printf("  Attempt %d\n", attempts)
		// Simulate polling that succeeds on the 5th attempt,
		// well before the 500ms timeout (waits: 50ms, 100ms, 100ms, 100ms).
		if attempts >= 5 {
			return nil
		}
		return errors.New("not ready yet")
	})

	if err != nil {
		fmt.Printf("  ❌ Timeout: %v\n", err)
		fmt.Printf("  Made %d attempts before timeout\n", attempts)
	} else {
		fmt.Printf("  ✅ Success after %d attempts\n", attempts)
	}
	fmt.Println()
}

// Example 6: Custom notifications
func example6CustomNotifications() {
	fmt.Println("Example 6: Custom Notifications")
	fmt.Println("--------------------------------")

	ctx := context.Background()
	cfg := retry.New(
		retry.WithName("example.notify"),
		retry.WithMaxRetries(4),
		retry.WithInitialInterval(50*time.Millisecond),
		retry.WithRandomizationFactor(0), // exact backoff values for reproducible output
	)
	attempts := 0
	err := retry.DoWithNotify(ctx, cfg,
		func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("failure #%d", attempts)
			}
			return nil
		},
		func(err error, backoff time.Duration) {
			// Custom notification on each retry
			fmt.Printf("  🔄 Retry scheduled in %v due to: %v\n",
				backoff.Round(time.Millisecond), err)
		},
	)

	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Success after %d attempts\n", attempts)
	}
	fmt.Println()
}

// Simulate a flaky database connection
func simulateDBConnection(attempts *int) error {
	*attempts++
	log.Printf("Attempting database connection (attempt %d)", *attempts)

	// Succeed after 3 attempts
	if *attempts >= 3 {
		return nil
	}
	return errors.New("connection refused")
}

// Simulate an HTTP API call
func simulateAPICall(attempts *int) error {
	*attempts++

	// Simulate different HTTP status codes
	statusCode := 500
	if *attempts >= 2 {
		statusCode = 200
	}

	if statusCode >= 500 {
		return fmt.Errorf("HTTP %d: server error", statusCode)
	}
	if statusCode >= 400 {
		// Don't retry client errors
		return retry.Permanent(fmt.Errorf("HTTP %d: client error", statusCode))
	}
	return nil
}
