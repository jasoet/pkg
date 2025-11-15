package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, uint64(5), cfg.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, cfg.InitialInterval)
	assert.Equal(t, 60*time.Second, cfg.MaxInterval)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Equal(t, "retry.operation", cfg.OperationName)
	assert.Nil(t, cfg.OTelConfig)
}

func TestConfigWithMethods(t *testing.T) {
	cfg := DefaultConfig().
		WithName("test.operation").
		WithMaxRetries(3).
		WithInitialInterval(100 * time.Millisecond).
		WithMaxInterval(10 * time.Second).
		WithMultiplier(1.5)

	assert.Equal(t, "test.operation", cfg.OperationName)
	assert.Equal(t, uint64(3), cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialInterval)
	assert.Equal(t, 10*time.Second, cfg.MaxInterval)
	assert.Equal(t, 1.5, cfg.Multiplier)
}

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().WithName("test.success")

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		return nil
	}

	err := Do(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.retry").
		WithMaxRetries(3).
		WithInitialInterval(10 * time.Millisecond)

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	start := time.Now()
	err := Do(ctx, cfg, operation)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	// Should have waited at least 10ms (first retry) + 20ms (second retry)
	assert.Greater(t, elapsed, 30*time.Millisecond)
}

func TestDo_FailsAfterMaxRetries(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.fail").
		WithMaxRetries(3).
		WithInitialInterval(10 * time.Millisecond)

	attempts := 0
	expectedErr := errors.New("permanent error")
	operation := func(ctx context.Context) error {
		attempts++
		return expectedErr
	}

	err := Do(ctx, cfg, operation)
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Contains(t, err.Error(), "failed after 4 attempts") // 1 initial + 3 retries
	assert.Equal(t, 4, attempts)
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := DefaultConfig().
		WithName("test.cancel").
		WithMaxRetries(5).
		WithInitialInterval(100 * time.Millisecond)

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("error")
	}

	err := Do(ctx, cfg, operation)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "cancelled")
	// Should stop after cancellation
	assert.LessOrEqual(t, attempts, 3)
}

func TestDo_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := DefaultConfig().
		WithName("test.timeout").
		WithMaxRetries(5).
		WithInitialInterval(100 * time.Millisecond)

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	}

	err := Do(ctx, cfg, operation)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	// Should not retry much due to timeout
	assert.LessOrEqual(t, attempts, 2)
}

func TestDo_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.backoff").
		WithMaxRetries(3).
		WithInitialInterval(10 * time.Millisecond).
		WithMultiplier(2.0)

	var intervals []time.Duration
	lastTime := time.Now()

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts > 1 {
			intervals = append(intervals, time.Since(lastTime))
		}
		lastTime = time.Now()
		return errors.New("error")
	}

	_ = Do(ctx, cfg, operation)

	// Verify exponential backoff: each interval should be roughly 2x the previous
	assert.Len(t, intervals, 3) // 3 retries

	// First retry should be around 10ms (with some jitter, backoff can be 0.5x-1.5x the interval)
	assert.GreaterOrEqual(t, intervals[0], 5*time.Millisecond)
	assert.LessOrEqual(t, intervals[0], 50*time.Millisecond)

	// Second retry should be around 20ms
	assert.GreaterOrEqual(t, intervals[1], 10*time.Millisecond)
	assert.LessOrEqual(t, intervals[1], 100*time.Millisecond)

	// Third retry should be around 40ms
	assert.GreaterOrEqual(t, intervals[2], 20*time.Millisecond)
}

func TestDo_UnlimitedRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := DefaultConfig().
		WithName("test.unlimited").
		WithMaxRetries(0). // Unlimited
		WithInitialInterval(2 * time.Millisecond)

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	}

	err := Do(ctx, cfg, operation)
	assert.Error(t, err)
	// Should have made several attempts before timeout
	assert.GreaterOrEqual(t, attempts, 3)
}

func TestDoWithNotify(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.notify").
		WithMaxRetries(3).
		WithInitialInterval(10 * time.Millisecond)

	var notifications []error
	notifyFunc := func(err error, backoff time.Duration) {
		notifications = append(notifications, err)
	}

	attempts := 0
	expectedErr := errors.New("test error")
	operation := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return expectedErr
		}
		return nil
	}

	err := DoWithNotify(ctx, cfg, operation, notifyFunc)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.Len(t, notifications, 2) // Notified on first 2 failures
	assert.Equal(t, expectedErr, notifications[0])
	assert.Equal(t, expectedErr, notifications[1])
}

func TestDoWithNotify_AllFailed(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.notify.fail").
		WithMaxRetries(2).
		WithInitialInterval(10 * time.Millisecond)

	var notifications []error
	notifyFunc := func(err error, backoff time.Duration) {
		notifications = append(notifications, err)
	}

	expectedErr := errors.New("persistent error")
	operation := func(ctx context.Context) error {
		return expectedErr
	}

	err := DoWithNotify(ctx, cfg, operation, notifyFunc)
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Len(t, notifications, 2) // Notified on both retries
}

func TestPermanent(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.permanent").
		WithMaxRetries(5).
		WithInitialInterval(10 * time.Millisecond)

	attempts := 0
	permanentErr := errors.New("permanent error")
	operation := func(ctx context.Context) error {
		attempts++
		return Permanent(permanentErr)
	}

	err := Do(ctx, cfg, operation)
	assert.Error(t, err)
	assert.ErrorIs(t, err, permanentErr)
	assert.Equal(t, 1, attempts) // Should not retry
}

func TestDo_MaxIntervalCap(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithName("test.maxinterval").
		WithMaxRetries(10).
		WithInitialInterval(10 * time.Millisecond).
		WithMaxInterval(50 * time.Millisecond). // Cap at 50ms
		WithMultiplier(2.0)

	var intervals []time.Duration
	lastTime := time.Now()

	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		if attempts > 1 {
			intervals = append(intervals, time.Since(lastTime))
		}
		lastTime = time.Now()

		if attempts >= 6 {
			return nil // Success after 6 attempts
		}
		return errors.New("error")
	}

	err := Do(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 6, attempts)

	// Later intervals should be capped at ~50ms
	for i := 3; i < len(intervals); i++ {
		assert.LessOrEqual(t, intervals[i], 100*time.Millisecond)
	}
}
