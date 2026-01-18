package loop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chr1sbest/wiggum/internal/config"
	"github.com/chr1sbest/wiggum/internal/logger"
)

// testStep is a simple step for testing.
type testStep struct {
	executed bool
}

func (s *testStep) Name() string { return "test" }
func (s *testStep) Type() string { return "test" }
func (s *testStep) Execute(ctx context.Context, cfg json.RawMessage) error {
	s.executed = true
	return nil
}

func TestLoopRunOnce(t *testing.T) {
	cfg := &config.Config{
		Name: "test-config",
		Steps: []config.StepConfig{
			{Type: "test", Name: "step1", Config: json.RawMessage(`{}`)},
			{Type: "test", Name: "step2", Config: json.RawMessage(`{}`)},
		},
	}

	registry := NewStepRegistry()
	step := &testStep{}
	registry.Register("test", func() Step { return step })

	log := logger.NewStdoutLogger(logger.LevelError)
	loop := NewLoop(cfg, registry, log)

	err := loop.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	state := loop.State()
	if state.LoopNumber != 1 {
		t.Errorf("expected loop number 1, got %d", state.LoopNumber)
	}
	if state.Status != StatusComplete {
		t.Errorf("expected status COMPLETE, got %s", state.Status)
	}
}

func TestStepRegistry(t *testing.T) {
	registry := NewStepRegistry()

	registry.Register("test", func() Step { return &testStep{} })

	step, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if step.Type() != "test" {
		t.Errorf("expected type 'test', got %s", step.Type())
	}

	_, err = registry.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown step type")
	}

	types := registry.RegisteredTypes()
	if len(types) != 1 || types[0] != "test" {
		t.Errorf("unexpected registered types: %v", types)
	}
}
