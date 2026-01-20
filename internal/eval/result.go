package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EvalResult represents the results of an evaluation run
type EvalResult struct {
	Suite             string    `json:"suite"`
	Approach          string    `json:"approach"`
	Model             string    `json:"model"`
	Timestamp         time.Time `json:"timestamp"`
	DurationSeconds   int       `json:"duration_seconds"`
	TotalCalls        int       `json:"total_calls"`
	TotalTurns        int       `json:"total_turns"`
	InputTokens       int       `json:"input_tokens"`
	OutputTokens      int       `json:"output_tokens"`
	TotalTokens       int       `json:"total_tokens"`
	CostUSD           float64   `json:"cost_usd"`
	SharedTestsPassed int       `json:"shared_tests_passed"`
	SharedTestsTotal  int       `json:"shared_tests_total"`
	FilesGenerated    int       `json:"files_generated"`
	LinesGenerated    int       `json:"lines_generated"`
	OutputDir         string    `json:"output_dir"`
}

// SaveToFile saves the eval result to a JSON file in the evals/results directory
// Filename format: {suite}-{approach}-{model}-{timestamp}.json
func (r *EvalResult) SaveToFile() (string, error) {
	resultsDir := filepath.Join("evals", "results")

	// Ensure results directory exists
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create results directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s-%s-%s-%d.json",
		r.Suite,
		r.Approach,
		r.Model,
		r.Timestamp.Unix(),
	)
	filePath := filepath.Join(resultsDir, filename)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, append(data, '\n'), 0644); err != nil {
		return "", fmt.Errorf("failed to write result file: %w", err)
	}

	return filePath, nil
}

// LoadFromFile loads an eval result from a JSON file
func LoadFromFile(filePath string) (*EvalResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	var result EvalResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result file: %w", err)
	}

	return &result, nil
}

// FindLatestResult finds the latest result file for a given suite, approach, and model
// Returns the file path or an error if not found
func FindLatestResult(suite, approach, model string) (string, error) {
	resultsDir := filepath.Join("evals", "results")

	pattern := fmt.Sprintf("%s-%s-%s-*.json", suite, approach, model)
	matches, err := filepath.Glob(filepath.Join(resultsDir, pattern))
	if err != nil {
		return "", fmt.Errorf("failed to search for results: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no results found for suite=%s approach=%s model=%s", suite, approach, model)
	}

	// Return the last match (alphabetically, which is chronologically due to timestamp)
	return matches[len(matches)-1], nil
}
