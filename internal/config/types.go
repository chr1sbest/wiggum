package config

import (
	"encoding/json"
	"time"
)

// Config represents a loop configuration loaded from JSON.
type Config struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Steps       []StepConfig `json:"steps"`
}

// StepConfig defines a single step in the loop.
type StepConfig struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Enabled *bool           `json:"enabled,omitempty"`
	Config  json.RawMessage `json:"config"`

	// Retry configuration
	MaxRetries      int    `json:"max_retries,omitempty"`       // Maximum retry attempts (0 = no retries)
	RetryDelay      string `json:"retry_delay,omitempty"`       // Initial delay between retries (e.g., "1s", "500ms")
	ContinueOnError bool   `json:"continue_on_error,omitempty"` // Continue to next step even if this fails

	// Timeout configuration
	Timeout string `json:"timeout,omitempty"` // Step execution timeout (e.g., "30s", "5m")

	// Circuit breaker configuration
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker settings for a step.
type CircuitBreakerConfig struct {
	Threshold  int    `json:"threshold"`   // Number of failures before opening circuit
	ResetAfter string `json:"reset_after"` // Duration before attempting half-open (e.g., "30s")
}

// IsEnabled returns whether the step is enabled (defaults to true).
func (s StepConfig) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// GetRetryDelay parses and returns the retry delay duration.
func (s StepConfig) GetRetryDelay() time.Duration {
	if s.RetryDelay == "" {
		return time.Second // Default 1 second
	}
	d, err := time.ParseDuration(s.RetryDelay)
	if err != nil {
		return time.Second
	}
	return d
}

// GetTimeout parses and returns the timeout duration.
func (s StepConfig) GetTimeout() time.Duration {
	if s.Timeout == "" {
		return 0 // No timeout
	}
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return 0
	}
	return d
}

// GetCircuitBreakerResetAfter parses the circuit breaker reset duration.
func (s StepConfig) GetCircuitBreakerResetAfter() time.Duration {
	if s.CircuitBreaker == nil || s.CircuitBreaker.ResetAfter == "" {
		return 30 * time.Second // Default
	}
	d, err := time.ParseDuration(s.CircuitBreaker.ResetAfter)
	if err != nil {
		return 30 * time.Second
	}
	return d
}
