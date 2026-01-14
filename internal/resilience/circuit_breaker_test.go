package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  3,
		ResetAfter: 100 * time.Millisecond,
	})

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state, got %v", cb.State())
	}

	// Execute successfully
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state, got %v", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  3,
		ResetAfter: 100 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Fail threshold times to open circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected open state after %d failures, got %v", 3, cb.State())
	}

	// Next call should return ErrCircuitOpen
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterReset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  2,
		ResetAfter: 50 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected open state, got %v", cb.State())
	}

	// Wait for reset period
	time.Sleep(60 * time.Millisecond)

	// Next execution should allow through (half-open)
	executed := false
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("expected function to execute in half-open state")
	}
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after successful half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  2,
		ResetAfter: 50 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for reset period
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open state
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	// Should be back to open
	if cb.State() != CircuitOpen {
		t.Errorf("expected open state after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  2,
		ResetAfter: 1 * time.Hour, // Long reset to ensure manual reset is what closes it
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected open state, got %v", cb.State())
	}

	// Manually reset
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after reset, got %v", cb.State())
	}
	if cb.Failures() != 0 {
		t.Errorf("expected 0 failures after reset, got %d", cb.Failures())
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  2,
		ResetAfter: 50 * time.Millisecond,
	})

	stateChanges := make([]CircuitState, 0)
	cb.OnStateChange(func(from, to CircuitState) {
		stateChanges = append(stateChanges, to)
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for callback goroutine
	time.Sleep(10 * time.Millisecond)

	if len(stateChanges) < 1 || stateChanges[0] != CircuitOpen {
		t.Errorf("expected state change to open, got %v", stateChanges)
	}
}

func TestCircuitBreakerRegistry_GetOrCreate(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	cb1 := registry.Get("step1", nil)
	cb2 := registry.Get("step1", nil)

	if cb1 != cb2 {
		t.Error("expected same circuit breaker instance for same step")
	}

	cb3 := registry.Get("step2", nil)
	if cb1 == cb3 {
		t.Error("expected different circuit breaker for different step")
	}
}

func TestCircuitBreakerRegistry_CustomConfig(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	customCfg := &CircuitBreakerConfig{
		Threshold:  10,
		ResetAfter: 1 * time.Minute,
	}

	cb := registry.Get("custom-step", customCfg)

	// Verify custom threshold by failing 5 times (less than threshold)
	for i := 0; i < 5; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	if cb.State() != CircuitClosed {
		t.Error("circuit should still be closed with custom higher threshold")
	}
}

func TestCircuitBreakerRegistry_ResetAll(t *testing.T) {
	registry := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		Threshold:  1,
		ResetAfter: 1 * time.Hour,
	})

	cb1 := registry.Get("step1", nil)
	cb2 := registry.Get("step2", nil)

	testErr := errors.New("error")

	// Open both circuits
	_ = cb1.Execute(context.Background(), func(ctx context.Context) error { return testErr })
	_ = cb2.Execute(context.Background(), func(ctx context.Context) error { return testErr })

	if cb1.State() != CircuitOpen || cb2.State() != CircuitOpen {
		t.Error("both circuits should be open")
	}

	registry.ResetAll()

	if cb1.State() != CircuitClosed || cb2.State() != CircuitClosed {
		t.Error("both circuits should be closed after ResetAll")
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("state %d: expected %q, got %q", tt.state, tt.expected, tt.state.String())
		}
	}
}
