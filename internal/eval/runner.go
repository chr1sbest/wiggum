package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunMetrics represents the metrics stored in .ralph/run_metrics.json
type RunMetrics struct {
	TotalClaudeCalls int     `json:"total_claude_calls"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
}

// runRalphApproach executes the ralph approach for an evaluation.
// It creates a project directory, initializes ralph, runs the loop, and collects metrics.
func runRalphApproach(config *RunConfig, suite *SuiteConfig) (*EvalResult, error) {
	startTime := time.Now()

	// Create project directory
	projectConfig := &ProjectDirConfig{
		Approach:  ApproachRalph,
		SuiteName: config.SuiteName,
		Model:     config.Model,
		Timestamp: startTime,
	}

	workingDir, err := CreateProjectDir(projectConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get the requirements file path
	requirementsPath := suite.Requirements
	if !filepath.IsAbs(requirementsPath) {
		// Make it absolute relative to the wiggum directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		requirementsPath = filepath.Join(cwd, requirementsPath)
	}

	// Verify requirements file exists
	if _, err := os.Stat(requirementsPath); err != nil {
		return nil, fmt.Errorf("requirements file not found: %s: %w", requirementsPath, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run ralph init
	fmt.Printf("Running: ralph init %s\n", requirementsPath)
	initCmd := exec.CommandContext(ctx, "ralph", "init", requirementsPath)
	initCmd.Dir = workingDir
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ralph init timed out after %d seconds", config.TimeoutSeconds)
		}
		return nil, fmt.Errorf("ralph init failed: %w", err)
	}

	// Run ralph run with model
	fmt.Printf("Running: ralph run -model %s\n", config.Model)
	runCmd := exec.CommandContext(ctx, "ralph", "run", "-model", config.Model)
	runCmd.Dir = workingDir
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	// Ralph run may timeout, which is acceptable - we still collect whatever metrics we have
	if err := runCmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("WARNING: ralph run timed out after %d seconds\n", config.TimeoutSeconds)
		} else {
			// Other errors are not fatal - ralph might have completed but with non-zero exit
			fmt.Printf("WARNING: ralph run exited with error: %v\n", err)
		}
	}

	// Parse metrics from .ralph/run_metrics.json
	metricsPath := filepath.Join(workingDir, ".ralph", "run_metrics.json")
	metrics, err := parseRunMetrics(metricsPath)
	if err != nil {
		fmt.Printf("WARNING: failed to parse run_metrics.json: %v\n", err)
		// Use zero metrics if parsing fails
		metrics = &RunMetrics{}
	}

	// Calculate duration
	duration := int(time.Since(startTime).Seconds())

	// Get the project root directory for storing output
	projectRoot := GetProjectRootDir(workingDir, ApproachRalph)

	// Create result
	result := &EvalResult{
		Suite:           config.SuiteName,
		Approach:        ApproachRalph,
		Model:           config.Model,
		Timestamp:       startTime,
		DurationSeconds: duration,
		TotalCalls:      metrics.TotalClaudeCalls,
		InputTokens:     metrics.InputTokens,
		OutputTokens:    metrics.OutputTokens,
		TotalTokens:     metrics.TotalTokens,
		CostUSD:         metrics.TotalCostUSD,
		OutputDir:       projectRoot,
		// Test and code metrics will be filled in later by the unified runner
	}

	return result, nil
}

// parseRunMetrics reads and parses the run_metrics.json file
func parseRunMetrics(metricsPath string) (*RunMetrics, error) {
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	var metrics RunMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse metrics JSON: %w", err)
	}

	return &metrics, nil
}
