package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryPolicy_NoRetry(t *testing.T) {
	calls := 0
	err := NoRetry.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return errors.New("always fails")
	})

	if err == nil {
		t.Error("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call with NoRetry, got %d", calls)
	}
}

func TestRetryPolicy_SuccessOnFirstTry(t *testing.T) {
	calls := 0
	err := QuickRetry.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryPolicy_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := QuickRetry.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryPolicy_PermanentErrorNoRetry(t *testing.T) {
	calls := 0
	permErr := NewPermanentError(errors.New("permanent failure"))

	err := QuickRetry.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return permErr
	})

	if err == nil {
		t.Error("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries for permanent error), got %d", calls)
	}
}

func TestRetryPolicy_CustomShouldRetry(t *testing.T) {
	calls := 0
	customErr := errors.New("custom non-retryable")

	policy := RetryPolicy{
		Name:       "custom",
		MaxRetries: 3,
		InitDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
		ShouldRetry: func(err error) bool {
			// Only retry if NOT our custom error
			return err.Error() != "custom non-retryable"
		},
	}

	err := policy.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return customErr
	})

	if err == nil {
		t.Error("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (custom shouldRetry returned false), got %d", calls)
	}
}

func TestRetryPolicy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := StandardRetry.Execute(ctx, func(ctx context.Context) error {
		calls++
		return errors.New("keep failing")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetryPolicy_ExhaustsRetries(t *testing.T) {
	policy := RetryPolicy{
		Name:       "test",
		MaxRetries: 2,
		InitDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	err := policy.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return errors.New("always fails")
	})

	if err == nil {
		t.Error("expected error after exhausting retries")
	}
	// 1 initial + 2 retries = 3 total calls
	if calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func TestRetryWithCheck(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		InitDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	t.Run("retries when shouldRetry returns true", func(t *testing.T) {
		calls := 0
		err := RetryWithCheck(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			if calls < 2 {
				return errors.New("retry me")
			}
			return nil
		}, func(err error) bool {
			return true // always retry
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("no retry when shouldRetry returns false", func(t *testing.T) {
		calls := 0
		err := RetryWithCheck(context.Background(), cfg, func(ctx context.Context) error {
			calls++
			return errors.New("don't retry me")
		}, func(err error) bool {
			return false // never retry
		})

		if err == nil {
			t.Error("expected error")
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})
}
