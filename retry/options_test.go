package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jasoet/pkg/v3/retry"
)

func TestNew_AppliesOptions(t *testing.T) {
	cfg := retry.New(
		retry.WithName("db.connect"),
		retry.WithMaxRetries(3),
		retry.WithInitialInterval(100*time.Millisecond),
	)
	assert.Equal(t, "db.connect", cfg.Name)
	assert.Equal(t, uint64(3), cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialInterval)
}

func TestDo_InvalidConfigReturnsErrorNotPanic(t *testing.T) {
	cfg := retry.New(retry.WithMultiplier(0.5))
	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		return errors.New("boom")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiplier")
}
