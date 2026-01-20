package resilience

import (
	"context"
	"time"
)

/*
RETRY POLICY DOCUMENTATION

This package defines retry behavior for transient failures.

## What is Transient vs Permanent?

TRANSIENT (retryable):
- Network timeouts
- Connection refused (server might come back up)
- Connection reset
- Rate limiting (429 responses)
- Temporary service unavailable (503)

PERMANENT (not retryable):
- File not found (ENOENT)
- Permission denied (EACCES, EPERM)
- DNS lookup failure (host doesn't exist)
- Invalid configuration
- Context cancelled/deadline exceeded
- Authentication failures

## Default Behavior

Errors are assumed TRANSIENT by default to allow recovery from temporary issues.
Use NewPermanentError() to explicitly mark an error as non-retryable.
Use NewTransientError() to explicitly mark an error as retryable.

## Retry Configuration

- MaxRetries: Maximum number of retry attempts (0 = no retries)
- InitDelay: Initial delay before first retry
- MaxDelay: Maximum delay between retries
- Multiplier: Exponential backoff multiplier
- Jitter: Random jitter factor (0.0-1.0) to prevent thundering herd
*/

// RetryPolicy defines a named, documented retry configuration
type RetryPolicy struct {
	// Name identifies this policy for logging/debugging
	Name string

	// MaxRetries is the maximum number of retry attempts (0 = no retries)
	MaxRetries int

	// InitDelay is the initial delay before first retry
	InitDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier for exponential backoff (e.g., 2.0 = double each time)
	Multiplier float64

	// Jitter adds randomness to prevent thundering herd (0.0-1.0)
	Jitter float64

	// ShouldRetry is an optional function to determine if an error should be retried
	// If nil, uses the default IsPermanentError check
	ShouldRetry func(error) bool
}

// Predefined policies for common use cases
var (
	// NoRetry disables retries entirely
	NoRetry = RetryPolicy{
		Name:       "no-retry",
		MaxRetries: 0,
	}

	// QuickRetry for fast operations that may have transient failures
	QuickRetry = RetryPolicy{
		Name:       "quick-retry",
		MaxRetries: 3,
		InitDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}

	// StandardRetry for typical operations
	StandardRetry = RetryPolicy{
		Name:       "standard-retry",
		MaxRetries: 3,
		InitDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}

	// AggressiveRetry for critical operations that must succeed
	AggressiveRetry = RetryPolicy{
		Name:       "aggressive-retry",
		MaxRetries: 5,
		InitDelay:  1 * time.Second,
		MaxDelay:   60 * time.Second,
		Multiplier: 1.5,
		Jitter:     0.2,
	}
)

// ToConfig converts a RetryPolicy to RetryConfig for use with Retry functions
func (p RetryPolicy) ToConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: p.MaxRetries,
		InitDelay:  p.InitDelay,
		MaxDelay:   p.MaxDelay,
		Multiplier: p.Multiplier,
		Jitter:     p.Jitter,
	}
}

// Execute runs a function with this retry policy
func (p RetryPolicy) Execute(ctx context.Context, fn RetryFunc) error {
	shouldRetry := p.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = func(err error) bool {
			return !IsPermanentError(err)
		}
	}

	return RetryWithCheck(ctx, p.ToConfig(), fn, shouldRetry)
}

// RetryWithCheck executes with retry and a custom should-retry check
func RetryWithCheck(ctx context.Context, cfg RetryConfig, fn RetryFunc, shouldRetry func(error) bool) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if !shouldRetry(err) {
			return err
		}

		if attempt >= cfg.MaxRetries {
			break
		}

		delay := calculateDelay(cfg, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}
