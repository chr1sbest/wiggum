package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// CommandConfig holds configuration for command step.
type CommandConfig struct {
	Command string `json:"command"`
	Timeout string `json:"timeout,omitempty"`
}

// CommandStep executes shell commands.
type CommandStep struct {
	name string
}

// NewCommandStep creates a new command step.
func NewCommandStep() *CommandStep {
	return &CommandStep{name: "command"}
}

func (s *CommandStep) Name() string { return s.name }
func (s *CommandStep) Type() string { return "command" }

func (s *CommandStep) Execute(ctx context.Context, rawConfig json.RawMessage) error {
	var cfg CommandConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return fmt.Errorf("failed to parse command config: %w", err)
	}

	if cfg.Command == "" {
		return fmt.Errorf("command is required")
	}

	timeout := 5 * time.Minute
	if cfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
