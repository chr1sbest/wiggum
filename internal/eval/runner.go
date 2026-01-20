package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chr1sbest/wiggum/internal/tracker"
)

// RunMetrics represents the metrics stored in .ralph/run_metrics.json
type RunMetrics struct {
	TotalClaudeCalls int     `json:"total_claude_calls"`
	TotalTurns       int     `json:"total_turns"`
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
		TotalTurns:      metrics.TotalTurns,
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

// ClaudeOutput represents the JSON output from Claude CLI
type ClaudeOutput struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Turns        int     `json:"num_turns"`
}

// runOneshotApproach executes the oneshot approach for an evaluation.
// It creates a project directory, builds a prompt from requirements, runs Claude once, and collects metrics.
func runOneshotApproach(config *RunConfig, suite *SuiteConfig) (*EvalResult, error) {
	startTime := time.Now()

	// Create project directory
	projectConfig := &ProjectDirConfig{
		Approach:  ApproachOneshot,
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

	// Read requirements content
	requirementsContent, err := os.ReadFile(requirementsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements file: %w", err)
	}

	// Build the prompt matching run.sh format
	prompt := fmt.Sprintf(`You are building a complete application from scratch. Create ALL necessary files to fully implement this specification.

%s

IMPORTANT:
- Create every file mentioned in the requirements
- Include all dependencies (go.mod, requirements.txt, etc.)
- Write complete implementations, not stubs
- Include comprehensive tests
- Make sure the code compiles/runs

Create all the files now.`, string(requirementsContent))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run claude with JSON output for token tracking
	fmt.Printf("Running: claude --model %s --dangerously-skip-permissions --output-format json\n", config.Model)
	cmd := exec.CommandContext(ctx, "claude", "--model", config.Model, "--dangerously-skip-permissions", "--output-format", "json")
	cmd.Dir = workingDir

	// Create a pipe to send the prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture stdout and stderr
	outputPath := filepath.Join(workingDir, "_claude_output.json")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude command: %w", err)
	}

	// Write the prompt to stdin and close it
	if _, err := stdin.Write([]byte(prompt)); err != nil {
		return nil, fmt.Errorf("failed to write prompt to stdin: %w", err)
	}
	stdin.Close()

	// Wait for the command to complete
	cmdErr := cmd.Wait()

	// Close the output file so we can read it
	outputFile.Close()

	// Check for timeout or other errors
	if cmdErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("WARNING: claude command timed out after %d seconds\n", config.TimeoutSeconds)
		} else {
			fmt.Printf("WARNING: claude command exited with error: %v\n", cmdErr)
		}
	}

	// Parse the JSON output
	claudeOutput, err := parseClaudeOutput(outputPath)
	if err != nil {
		fmt.Printf("WARNING: failed to parse claude output: %v\n", err)
		// Use zero metrics if parsing fails
		claudeOutput = &ClaudeOutput{}
	}

	// Calculate duration
	duration := int(time.Since(startTime).Seconds())

	// Calculate total tokens
	totalTokens := claudeOutput.InputTokens + claudeOutput.OutputTokens

	// Create result
	result := &EvalResult{
		Suite:           config.SuiteName,
		Approach:        ApproachOneshot,
		Model:           config.Model,
		Timestamp:       startTime,
		DurationSeconds: duration,
		TotalCalls:      1, // Oneshot always has exactly 1 call
		TotalTurns:      claudeOutput.Turns,
		InputTokens:     claudeOutput.InputTokens,
		OutputTokens:    claudeOutput.OutputTokens,
		TotalTokens:     totalTokens,
		CostUSD:         claudeOutput.TotalCostUSD,
		OutputDir:       workingDir,
		// Test and code metrics will be filled in later by the unified runner
	}

	return result, nil
}

// parseClaudeOutput reads and parses the JSON output from Claude CLI.
// Uses the shared tracker.ParseClaudeUsageFromOutput helper which handles
// stderr filtering, nested JSON structures, and cache tokens.
func parseClaudeOutput(outputPath string) (*ClaudeOutput, error) {
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	usage, ok := tracker.ParseClaudeUsageFromOutput(string(data))
	if !ok {
		return nil, fmt.Errorf("no usage data found in output")
	}

	return &ClaudeOutput{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalCostUSD: usage.CostUSD,
		Turns:        usage.Turns,
	}, nil
}

// Run executes a complete evaluation run with the given configuration.
// It orchestrates loading the suite, running the appropriate approach,
// running tests, collecting metrics, and saving results.
func Run(config *RunConfig) (*EvalResult, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Print banner
	printBanner(config)

	// Load suite configuration
	suite, err := LoadSuite(config.SuiteName)
	if err != nil {
		return nil, fmt.Errorf("failed to load suite: %w", err)
	}

	// Run the appropriate approach
	var result *EvalResult
	if config.IsRalphApproach() {
		fmt.Println("Running Ralph...")
		fmt.Println("")
		result, err = runRalphApproach(config, suite)
	} else {
		fmt.Println("Running One-Shot Claude...")
		fmt.Println("")
		result, err = runOneshotApproach(config, suite)
	}

	if err != nil {
		return nil, fmt.Errorf("approach execution failed: %w", err)
	}

	fmt.Println("")
	fmt.Printf("=== Generation completed in %ds ===\n", result.DurationSeconds)

	// Run shared tests
	fmt.Println("")
	fmt.Println("=== Running Test Suite ===")
	fmt.Println("")

	testResult, err := RunSharedTests(result.OutputDir, suite, 8000)
	if err != nil {
		fmt.Printf("WARNING: test execution failed: %v\n", err)
		// Continue with zero test results
		testResult = &TestResult{}
	}

	// Update result with test metrics
	result.SharedTestsPassed = testResult.Passed
	result.SharedTestsTotal = testResult.Total

	// Collect code metrics
	metrics, err := CollectCodeMetrics(result.OutputDir)
	if err != nil {
		fmt.Printf("WARNING: failed to collect code metrics: %v\n", err)
		metrics = &CodeMetrics{}
	}

	result.FilesGenerated = metrics.FilesGenerated
	result.LinesGenerated = metrics.LinesGenerated

	// Save result to file
	resultPath, err := result.SaveToFile()
	if err != nil {
		return nil, fmt.Errorf("failed to save results: %w", err)
	}

	// Print summary
	printSummary(result, resultPath)

	return result, nil
}

// printBanner displays the evaluation run banner
func printBanner(config *RunConfig) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                    EVAL: %-32s║\n", config.SuiteName)
	fmt.Printf("║  Approach: %-8s | Model: %-28s║\n", config.Approach, config.Model)
	fmt.Printf("║  Started: %-46s║\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("║  Timeout: 2 hours                                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
}

// printSummary displays the results summary
func printSummary(result *EvalResult, resultPath string) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         SUMMARY                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("%-20s %s\n", "Project:", filepath.Base(result.OutputDir))
	fmt.Printf("%-20s %ds\n", "Time:", result.DurationSeconds)
	fmt.Printf("%-20s %d\n", "Claude Calls:", result.TotalCalls)
	fmt.Printf("%-20s %d\n", "Total Tokens:", result.TotalTokens)
	fmt.Printf("%-20s $%.4f\n", "Cost:", result.CostUSD)
	fmt.Printf("%-20s %d/%d\n", "Tests:", result.SharedTestsPassed, result.SharedTestsTotal)
	fmt.Printf("%-20s %d files, %d lines\n", "Code:", result.FilesGenerated, result.LinesGenerated)
	fmt.Println("")
	fmt.Printf("Results: %s\n", resultPath)
	fmt.Println("")
}
