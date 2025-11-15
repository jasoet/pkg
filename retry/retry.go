package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	pkgotel "github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Operation is a function that may fail and should be retried.
// Return nil to indicate success, or an error to trigger a retry.
type Operation func(ctx context.Context) error

// Config holds retry configuration with exponential backoff.
type Config struct {
	// MaxRetries is the maximum number of retry attempts (0 means unlimited).
	// Default: 5
	MaxRetries uint64

	// InitialInterval is the initial retry interval.
	// Default: 500ms
	InitialInterval time.Duration

	// MaxInterval caps the maximum retry interval.
	// Default: 60s
	MaxInterval time.Duration

	// Multiplier is the exponential backoff multiplier.
	// Default: 2.0 (each retry waits 2x longer)
	Multiplier float64

	// OperationName is used for logging and tracing.
	// Default: "retry.operation"
	OperationName string

	// OTelConfig enables OpenTelemetry tracing and logging.
	// Optional: if nil, no OTel instrumentation
	OTelConfig *pkgotel.Config
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:      5,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     60 * time.Second,
		Multiplier:      2.0,
		OperationName:   "retry.operation",
	}
}

// WithOTel adds OpenTelemetry configuration to the retry config.
func (c Config) WithOTel(otelConfig *pkgotel.Config) Config {
	c.OTelConfig = otelConfig
	return c
}

// WithName sets the operation name for logging and tracing.
func (c Config) WithName(name string) Config {
	c.OperationName = name
	return c
}

// WithMaxRetries sets the maximum number of retry attempts.
func (c Config) WithMaxRetries(maxRetries uint64) Config {
	c.MaxRetries = maxRetries
	return c
}

// WithInitialInterval sets the initial retry interval.
func (c Config) WithInitialInterval(interval time.Duration) Config {
	c.InitialInterval = interval
	return c
}

// WithMaxInterval sets the maximum retry interval.
func (c Config) WithMaxInterval(interval time.Duration) Config {
	c.MaxInterval = interval
	return c
}

// WithMultiplier sets the exponential backoff multiplier.
func (c Config) WithMultiplier(multiplier float64) Config {
	c.Multiplier = multiplier
	return c
}

// Do executes the operation with retry logic using exponential backoff.
// It returns nil if the operation succeeds, or the last error if all retries are exhausted.
//
// Example:
//
//	cfg := retry.DefaultConfig().
//		WithName("database.connect").
//		WithMaxRetries(3).
//		WithOTel(otelConfig)
//
//	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
//		return db.Ping()
//	})
func Do(ctx context.Context, cfg Config, operation Operation) error {
	// Create backoff strategy
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.InitialInterval
	expBackoff.MaxInterval = cfg.MaxInterval
	expBackoff.Multiplier = cfg.Multiplier
	expBackoff.MaxElapsedTime = 0 // No time limit, only MaxRetries

	// Wrap with max retries
	var strategy backoff.BackOff = expBackoff
	if cfg.MaxRetries > 0 {
		strategy = backoff.WithMaxRetries(expBackoff, cfg.MaxRetries)
	}

	// Wrap with context
	strategy = backoff.WithContext(strategy, ctx)

	// Setup OTel if configured
	var span trace.Span
	var logger *pkgotel.LogHelper
	if cfg.OTelConfig != nil && cfg.OTelConfig.IsTracingEnabled() {
		tracer := cfg.OTelConfig.GetTracer("github.com/jasoet/pkg/v2/retry")
		ctx, span = tracer.Start(ctx, cfg.OperationName,
			trace.WithAttributes(
				attribute.Int64("retry.max_retries", int64(cfg.MaxRetries)),
				attribute.String("retry.initial_interval", cfg.InitialInterval.String()),
				attribute.String("retry.max_interval", cfg.MaxInterval.String()),
				attribute.Float64("retry.multiplier", cfg.Multiplier),
			),
		)
		defer span.End()

		logger = pkgotel.NewLogHelper(ctx, cfg.OTelConfig, "github.com/jasoet/pkg/v2/retry", cfg.OperationName)
	}

	// Use backoff.Retry for simpler implementation
	attempt := uint64(0)
	var lastErr error

	retryFunc := func() error {
		attempt++

		// Log attempt if OTel is configured
		if logger != nil {
			logger.Debug("Executing operation",
				pkgotel.F("attempt", attempt),
				pkgotel.F("max_retries", cfg.MaxRetries),
			)
		}

		// Execute operation
		err := operation(ctx)
		if err != nil {
			lastErr = err
			// Log retry if not last attempt
			if logger != nil {
				logger.Warn("Operation failed",
					pkgotel.F("attempt", attempt),
					pkgotel.F("max_retries", cfg.MaxRetries),
					pkgotel.F("error", err.Error()),
				)
			}
		}
		return err
	}

	err := backoff.Retry(retryFunc, strategy)

	if err == nil {
		// Success
		if span != nil {
			span.SetStatus(codes.Ok, "Operation succeeded")
			span.SetAttributes(attribute.Int64("retry.attempts", int64(attempt)))
		}
		if logger != nil && attempt > 1 {
			logger.Info("Operation succeeded after retry",
				pkgotel.F("attempts", attempt),
			)
		}
		return nil
	}

	// Handle context cancellation
	if ctx.Err() != nil {
		if span != nil {
			span.SetStatus(codes.Error, "Operation cancelled")
			span.RecordError(ctx.Err())
		}
		if logger != nil {
			logger.Error(ctx.Err(), "Operation cancelled",
				pkgotel.F("attempts", attempt),
			)
		}
		return fmt.Errorf("%s cancelled after %d attempts: %w", cfg.OperationName, attempt, ctx.Err())
	}

	// Failed after retries
	if span != nil {
		span.SetStatus(codes.Error, "Operation failed after all retries")
		span.RecordError(lastErr)
		span.SetAttributes(attribute.Int64("retry.attempts", int64(attempt)))
	}
	if logger != nil {
		logger.Error(lastErr, "Operation failed after all retries",
			pkgotel.F("attempts", attempt),
			pkgotel.F("max_retries", cfg.MaxRetries),
		)
	}
	return fmt.Errorf("%s failed after %d attempts: %w", cfg.OperationName, attempt, lastErr)
}

// DoWithNotify is like Do but calls notifyFunc on each retry with the error and backoff duration.
// This is useful for custom logging or metrics.
//
// Example:
//
//	err := retry.DoWithNotify(ctx, cfg, operation, func(err error, backoff time.Duration) {
//		log.Printf("Retry after error: %v (waiting %v)", err, backoff)
//	})
func DoWithNotify(
	ctx context.Context,
	cfg Config,
	operation Operation,
	notifyFunc func(error, time.Duration),
) error {
	// Create backoff strategy
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.InitialInterval
	expBackoff.MaxInterval = cfg.MaxInterval
	expBackoff.Multiplier = cfg.Multiplier
	expBackoff.MaxElapsedTime = 0

	var strategy backoff.BackOff = expBackoff
	if cfg.MaxRetries > 0 {
		strategy = backoff.WithMaxRetries(expBackoff, cfg.MaxRetries)
	}
	strategy = backoff.WithContext(strategy, ctx)

	// Use backoff.RetryNotify
	err := backoff.RetryNotify(
		func() error {
			return operation(ctx)
		},
		strategy,
		func(err error, duration time.Duration) {
			notifyFunc(err, duration)
		},
	)

	if err != nil {
		return fmt.Errorf("%s failed: %w", cfg.OperationName, err)
	}
	return nil
}

// Permanent wraps an error to indicate it should not be retried.
// Use this when an error is not transient and retrying would be pointless.
//
// Example:
//
//	return retry.Permanent(fmt.Errorf("invalid configuration"))
func Permanent(err error) error {
	return backoff.Permanent(err)
}
