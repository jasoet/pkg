package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// instrumentationName is the OpenTelemetry instrumentation scope for this package.
const instrumentationName = "github.com/jasoet/pkg/v3/retry"

// Operation is a function that may fail and should be retried.
// Return nil to indicate success, or an error to trigger a retry.
type Operation func(ctx context.Context) error

// Config holds retry configuration with exponential backoff.
type Config struct {
	// MaxRetries is the maximum number of retries after the initial attempt
	// (0 means unlimited retries). With MaxRetries = N the operation is called
	// at most N+1 times: 1 initial attempt plus up to N retries.
	// Default: 5
	MaxRetries uint64 `yaml:"maxRetries" mapstructure:"maxRetries"`

	// InitialInterval is the initial retry interval.
	// Default: 500ms
	InitialInterval time.Duration `yaml:"initialInterval" mapstructure:"initialInterval"`

	// MaxInterval caps the maximum retry interval.
	// Default: 60s
	MaxInterval time.Duration `yaml:"maxInterval" mapstructure:"maxInterval"`

	// Multiplier is the exponential backoff multiplier. Must be > 1.
	// Default: 2.0 (each retry waits 2x longer)
	Multiplier float64 `yaml:"multiplier" mapstructure:"multiplier"`

	// RandomizationFactor adds jitter to backoff intervals to prevent thundering herd.
	// Must be in [0, 1]. 0.0 means no randomization, 0.5 means +/-50% jitter.
	// Default: 0.5
	RandomizationFactor float64 `yaml:"randomizationFactor" mapstructure:"randomizationFactor"`

	// Name is the operation name used for logging and tracing.
	// Default: "retry.operation"
	Name string `yaml:"name" mapstructure:"name"`

	// OTelConfig enables OpenTelemetry tracing and logging.
	// Optional: if nil, no OTel instrumentation.
	OTelConfig *pkgotel.Config `yaml:"-" mapstructure:"-"` // Not serializable from config files
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:          5,
		InitialInterval:     500 * time.Millisecond,
		MaxInterval:         60 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5,
		Name:                "retry.operation",
	}
}

// Option configures a Config. Options never panic; invalid values are
// reported as an error by Do and DoWithNotify before the first attempt.
type Option func(*Config)

// New returns a Config starting from DefaultConfig with each option applied
// in order.
func New(opts ...Option) Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithOTelConfig adds OpenTelemetry configuration to the retry config.
func WithOTelConfig(otelConfig *pkgotel.Config) Option {
	return func(c *Config) {
		c.OTelConfig = otelConfig
	}
}

// WithName sets the operation name for logging and tracing.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithMaxRetries sets the maximum number of retries after the initial attempt.
func WithMaxRetries(maxRetries uint64) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithInitialInterval sets the initial retry interval.
func WithInitialInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.InitialInterval = interval
	}
}

// WithMaxInterval sets the maximum retry interval.
func WithMaxInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.MaxInterval = interval
	}
}

// WithMultiplier sets the exponential backoff multiplier. Must be > 1;
// invalid values make Do and DoWithNotify return an error.
func WithMultiplier(multiplier float64) Option {
	return func(c *Config) {
		c.Multiplier = multiplier
	}
}

// WithRandomizationFactor sets the jitter factor for backoff intervals.
// A value of 0.5 means intervals will vary by +/-50%. Set to 0.0 to disable
// jitter. Must be in [0, 1]; invalid values make Do and DoWithNotify return
// an error.
func WithRandomizationFactor(factor float64) Option {
	return func(c *Config) {
		c.RandomizationFactor = factor
	}
}

// validate reports whether the Config is usable. It returns an error
// describing the first invalid field.
func (c Config) validate() error {
	if c.Multiplier <= 1 {
		return fmt.Errorf("retry: multiplier must be > 1, got %v", c.Multiplier)
	}
	if c.InitialInterval <= 0 {
		return fmt.Errorf("retry: initial interval must be > 0, got %v", c.InitialInterval)
	}
	if c.MaxInterval < c.InitialInterval {
		return fmt.Errorf("retry: max interval (%v) must be >= initial interval (%v)",
			c.MaxInterval, c.InitialInterval)
	}
	if c.RandomizationFactor < 0 || c.RandomizationFactor > 1 {
		return fmt.Errorf("retry: randomization factor must be in [0, 1], got %v", c.RandomizationFactor)
	}
	return nil
}

// doRetry is the shared implementation for Do and DoWithNotify.
// When notifyFunc is non-nil, backoff.RetryNotify is used; otherwise backoff.Retry.
// The span (if non-nil) is ended via defer in the caller before this function returns.
func doRetry(ctx context.Context, cfg Config, operation Operation, notifyFunc func(error, time.Duration)) error {
	// Setup OTel tracing if enabled.
	var span trace.Span
	if cfg.OTelConfig != nil && cfg.OTelConfig.IsTracingEnabled() {
		tracer := cfg.OTelConfig.GetTracer(instrumentationName)
		ctx, span = tracer.Start(ctx, cfg.Name,
			trace.WithAttributes(
				attribute.Int64("retry.max_retries", int64(cfg.MaxRetries)),
				attribute.String("retry.initial_interval", cfg.InitialInterval.String()),
				attribute.String("retry.max_interval", cfg.MaxInterval.String()),
				attribute.Float64("retry.multiplier", cfg.Multiplier),
			),
		)
		// Span is ended via defer so it is always closed before this function returns.
		defer span.End()
	}

	// Setup OTel logging independently of tracing.
	var logger *pkgotel.LogHelper
	if cfg.OTelConfig != nil {
		logger = pkgotel.NewLogHelper(ctx, cfg.OTelConfig, instrumentationName, cfg.Name)
	}

	// Create backoff strategy.
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.InitialInterval
	expBackoff.MaxInterval = cfg.MaxInterval
	expBackoff.Multiplier = cfg.Multiplier
	expBackoff.RandomizationFactor = cfg.RandomizationFactor
	expBackoff.MaxElapsedTime = 0 // No time limit, only MaxRetries

	// Wrap with max retries.
	var strategy backoff.BackOff = expBackoff
	if cfg.MaxRetries > 0 {
		strategy = backoff.WithMaxRetries(expBackoff, cfg.MaxRetries)
	}

	// Wrap with context (uses span-enriched ctx when OTel is enabled).
	strategy = backoff.WithContext(strategy, ctx)

	attempt := uint64(0)
	var lastErr error

	retryFunc := func() error {
		attempt++

		if logger != nil {
			logger.Debug("Executing operation",
				pkgotel.F("attempt", attempt),
				pkgotel.F("max_retries", cfg.MaxRetries),
			)
		}

		err := operation(ctx)
		if err != nil {
			lastErr = err
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

	var err error
	if notifyFunc != nil {
		err = backoff.RetryNotify(retryFunc, strategy, notifyFunc)
	} else {
		err = backoff.Retry(retryFunc, strategy)
	}

	if err == nil {
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

	// Handle context cancellation.
	if ctx.Err() != nil {
		if span != nil {
			span.SetStatus(codes.Error, "Operation canceled")
			span.RecordError(ctx.Err())
		}
		if logger != nil {
			logger.Error(ctx.Err(), "Operation canceled",
				pkgotel.F("attempts", attempt),
			)
		}
		return fmt.Errorf("%s canceled after %d attempts: %w", cfg.Name, attempt, ctx.Err())
	}

	// Failed after retries.
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
	return fmt.Errorf("%s failed after %d attempts (1 initial + %d retries): %w",
		cfg.Name, attempt, attempt-1, lastErr)
}

// Do executes the operation with retry logic using exponential backoff.
// It returns nil if the operation succeeds, or the last error if all retries are exhausted.
// An invalid Config (see Config field docs) is reported as an error before the
// first attempt; Do never panics.
//
// Example:
//
//	cfg := retry.New(
//		retry.WithName("database.connect"),
//		retry.WithMaxRetries(3),
//		retry.WithOTelConfig(otelConfig),
//	)
//
//	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
//		return db.Ping()
//	})
func Do(ctx context.Context, cfg Config, operation Operation) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	return doRetry(ctx, cfg, operation, nil)
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
	if err := cfg.validate(); err != nil {
		return err
	}
	return doRetry(ctx, cfg, operation, notifyFunc)
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
