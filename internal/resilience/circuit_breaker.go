package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation, requests flow through
	CircuitOpen                         // Failures exceeded threshold, requests blocked
	CircuitHalfOpen                     // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	Threshold  int           // Number of consecutive failures before opening
	ResetAfter time.Duration // Time to wait before attempting half-open
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Threshold:  5,
		ResetAfter: 30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu            sync.RWMutex
	config        CircuitBreakerConfig
	state         CircuitState
	failures      int
	lastFailure   time.Time
	onStateChange func(from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: cfg,
		state:  CircuitClosed,
	}
}

// OnStateChange sets a callback for state changes.
func (cb *CircuitBreaker) OnStateChange(fn func(from, to CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Failures returns the current consecutive failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Execute runs the function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check if we can proceed
	if !cb.canExecute() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn(ctx)

	// Record the result
	cb.recordResult(err)

	return err
}

// canExecute checks if the circuit allows execution.
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if enough time has passed to try half-open
		if time.Since(cb.lastFailure) >= cb.config.ResetAfter {
			cb.setState(CircuitHalfOpen)
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow single request through for probing
		return true

	default:
		return false
	}
}

// recordResult records success or failure.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		cb.recordSuccess()
	} else {
		cb.recordFailure()
	}
}

// recordSuccess handles a successful execution.
func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case CircuitClosed:
		// Reset failure count on success
		cb.failures = 0

	case CircuitHalfOpen:
		// Success in half-open means we can close
		cb.failures = 0
		cb.setState(CircuitClosed)
	}
}

// recordFailure handles a failed execution.
func (cb *CircuitBreaker) recordFailure() {
	cb.lastFailure = time.Now()
	cb.failures++

	switch cb.state {
	case CircuitClosed:
		// Check if we've hit the threshold
		if cb.failures >= cb.config.Threshold {
			cb.setState(CircuitOpen)
		}

	case CircuitHalfOpen:
		// Any failure in half-open opens the circuit
		cb.setState(CircuitOpen)
	}
}

// setState transitions to a new state.
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil {
		// Call in goroutine to avoid blocking
		go cb.onStateChange(oldState, newState)
	}
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.lastFailure = time.Time{}
	cb.setState(CircuitClosed)
}

// CircuitBreakerRegistry manages circuit breakers per step.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	defaults CircuitBreakerConfig
}

// NewCircuitBreakerRegistry creates a new registry.
func NewCircuitBreakerRegistry(defaults CircuitBreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		defaults: defaults,
	}
}

// Get retrieves or creates a circuit breaker for the given step.
func (r *CircuitBreakerRegistry) Get(stepName string, cfg *CircuitBreakerConfig) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, exists := r.breakers[stepName]; exists {
		return cb
	}

	// Use provided config or defaults
	useCfg := r.defaults
	if cfg != nil {
		useCfg = *cfg
	}

	cb := NewCircuitBreaker(useCfg)
	r.breakers[stepName] = cb
	return cb
}

// State returns the state of a specific step's circuit breaker.
func (r *CircuitBreakerRegistry) State(stepName string) (CircuitState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cb, exists := r.breakers[stepName]; exists {
		return cb.State(), true
	}
	return CircuitClosed, false
}

// ResetAll resets all circuit breakers.
func (r *CircuitBreakerRegistry) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cb := range r.breakers {
		cb.Reset()
	}
}
