package eval

import (
	"fmt"
	"strings"
)

// Default configuration values
const (
	DefaultTimeoutSeconds = 7200 // 2 hours
	ApproachRalph         = "ralph"
	ApproachOneshot       = "oneshot"
)

// RunConfig contains configuration for running an evaluation.
type RunConfig struct {
	SuiteName      string
	Approach       string
	Model          string
	TimeoutSeconds int
	OutputDir      string
}

// NewRunConfig creates a new RunConfig with default values.
func NewRunConfig(suiteName, approach, model string) *RunConfig {
	return &RunConfig{
		SuiteName:      suiteName,
		Approach:       approach,
		Model:          model,
		TimeoutSeconds: DefaultTimeoutSeconds,
		OutputDir:      "",
	}
}

// Validate checks if the config is valid and returns an error if not.
func (c *RunConfig) Validate() error {
	if c.SuiteName == "" {
		return fmt.Errorf("suite name cannot be empty")
	}

	if err := c.ValidateApproach(); err != nil {
		return err
	}

	if c.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	if c.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be positive, got %d", c.TimeoutSeconds)
	}

	return nil
}

// ValidateApproach checks if the approach is valid.
func (c *RunConfig) ValidateApproach() error {
	normalized := strings.ToLower(strings.TrimSpace(c.Approach))
	if normalized != ApproachRalph && normalized != ApproachOneshot {
		return fmt.Errorf("invalid approach '%s': must be '%s' or '%s'", c.Approach, ApproachRalph, ApproachOneshot)
	}
	c.Approach = normalized
	return nil
}

// IsRalphApproach returns true if the approach is ralph.
func (c *RunConfig) IsRalphApproach() bool {
	return strings.ToLower(strings.TrimSpace(c.Approach)) == ApproachRalph
}

// IsOneshotApproach returns true if the approach is oneshot.
func (c *RunConfig) IsOneshotApproach() bool {
	return strings.ToLower(strings.TrimSpace(c.Approach)) == ApproachOneshot
}
