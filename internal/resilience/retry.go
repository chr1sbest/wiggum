package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries int           // Maximum number of retry attempts
	InitDelay  time.Duration // Initial delay between retries
	MaxDelay   time.Duration // Maximum delay cap
	Multiplier float64       // Backoff multiplier (e.g., 2.0 for doubling)
	Jitter     float64       // Jitter factor (0.0 to 1.0)
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		InitDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}
}

// RetryFunc is the function signature for operations that can be retried.
type RetryFunc func(ctx context.Context) error

// RetryCallback is called before each retry attempt.
type RetryCallback func(attempt int, err error, nextDelay time.Duration)

// Retry executes the operation with exponential backoff and jitter.
// Returns the last error if all retries fail.
func Retry(ctx context.Context, cfg RetryConfig, fn RetryFunc) error {
	return RetryWithCallback(ctx, cfg, fn, nil)
}

// RetryWithCallback executes with retry and calls back before each attempt.
func RetryWithCallback(ctx context.Context, cfg RetryConfig, fn RetryFunc, callback RetryCallback) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is permanent (non-retryable)
		if IsPermanentError(err) {
			return err
		}

		// No more retries
		if attempt >= cfg.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(cfg, attempt)

		// Notify callback before waiting
		if callback != nil {
			callback(attempt+1, err, delay)
		}

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// calculateDelay computes the delay for a given attempt with jitter.
func calculateDelay(cfg RetryConfig, attempt int) time.Duration {
	// Exponential backoff: initDelay * multiplier^attempt
	delay := float64(cfg.InitDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter: delay * (1 ± jitter)
	if cfg.Jitter > 0 {
		jitterRange := delay * cfg.Jitter
		delay = delay - jitterRange + (rand.Float64() * 2 * jitterRange)
	}

	return time.Duration(delay)
}
