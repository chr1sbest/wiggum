package steps

import (
	"context"
	"encoding/json"
)

// NoopStep is a step that does nothing (useful for testing).
type NoopStep struct {
	name string
}

// NewNoopStep creates a new noop step.
func NewNoopStep() *NoopStep {
	return &NoopStep{name: "noop"}
}

func (s *NoopStep) Name() string { return s.name }
func (s *NoopStep) Type() string { return "noop" }

func (s *NoopStep) Execute(ctx context.Context, config json.RawMessage) error {
	return nil
}
