package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Step defines the interface for executable steps in the loop.
type Step interface {
	Name() string
	Type() string
	Execute(ctx context.Context, config json.RawMessage) error
}

// StepFactory creates a new step instance.
type StepFactory func() Step

// StepRegistry manages step type registrations.
type StepRegistry struct {
	mu        sync.RWMutex
	factories map[string]StepFactory
}

// NewStepRegistry creates a new step registry.
func NewStepRegistry() *StepRegistry {
	return &StepRegistry{
		factories: make(map[string]StepFactory),
	}
}

// Register adds a step factory for a given type.
func (r *StepRegistry) Register(stepType string, factory StepFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[stepType] = factory
}

// Get retrieves a step instance for the given type.
func (r *StepRegistry) Get(stepType string) (Step, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[stepType]
	if !ok {
		return nil, fmt.Errorf("unknown step type: %s", stepType)
	}
	return factory(), nil
}

// RegisteredTypes returns a list of all registered step types.
func (r *StepRegistry) RegisteredTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
