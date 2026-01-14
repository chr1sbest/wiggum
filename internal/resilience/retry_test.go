package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	calls := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	calls := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ExhaustsRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		InitDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	calls := 0
	expectedErr := errors.New("persistent error")
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	// Initial call + 2 retries = 3 calls
	if calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func TestRetry_PermanentError(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	calls := 0
	permanentErr := NewPermanentError(errors.New("fatal error"))
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		return permanentErr
	})

	if err != permanentErr {
		t.Errorf("expected permanent error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries for permanent error), got %d", calls)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		calls++
		return errors.New("error")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetry_WithCallback(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	calls := 0
	callbackCalls := 0
	err := RetryWithCallback(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, func(attempt int, err error, nextDelay time.Duration) {
		callbackCalls++
		if attempt != callbackCalls {
			t.Errorf("callback attempt mismatch: expected %d, got %d", callbackCalls, attempt)
		}
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callbackCalls != 2 {
		t.Errorf("expected 2 callback calls, got %d", callbackCalls)
	}
}

func TestCalculateDelay_ExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		InitDelay:  100 * time.Millisecond,
		MaxDelay:   10 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.0, // No jitter for deterministic test
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		delay := calculateDelay(cfg, tt.attempt)
		if delay != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestCalculateDelay_MaxDelayCap(t *testing.T) {
	cfg := RetryConfig{
		InitDelay:  100 * time.Millisecond,
		MaxDelay:   500 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.0,
	}

	// Attempt 10 would be 100ms * 2^10 = 102.4s, but should be capped at 500ms
	delay := calculateDelay(cfg, 10)
	if delay != 500*time.Millisecond {
		t.Errorf("expected max delay 500ms, got %v", delay)
	}
}
