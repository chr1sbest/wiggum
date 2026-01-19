package eval

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromFile(t *testing.T) {
	// Test loading an actual result file to ensure format matches
	resultsDir := filepath.Join("..", "..", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		t.Skip("No results directory found")
	}

	// Find a JSON file
	var testFile string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			testFile = filepath.Join(resultsDir, entry.Name())
			break
		}
	}

	if testFile == "" {
		t.Skip("No JSON files found in results directory")
	}

	result, err := LoadFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to load result file: %v", err)
	}

	// Verify all expected fields are present
	if result.Suite == "" {
		t.Error("Suite field is empty")
	}
	if result.Approach == "" {
		t.Error("Approach field is empty")
	}
	if result.Model == "" {
		t.Error("Model field is empty")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	os.Chdir(tmpDir)

	result := &EvalResult{
		Suite:             "test-suite",
		Approach:          "ralph",
		Model:             "sonnet",
		Timestamp:         time.Now(),
		DurationSeconds:   300,
		TotalCalls:        5,
		InputTokens:       1000,
		OutputTokens:      500,
		TotalTokens:       1500,
		CostUSD:           0.75,
		SharedTestsPassed: 10,
		SharedTestsTotal:  12,
		FilesGenerated:    5,
		LinesGenerated:    100,
		OutputDir:         "/tmp/eval-test",
	}

	// Save to file
	filePath, err := result.SaveToFile()
	if err != nil {
		t.Fatalf("Failed to save result: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Result file was not created: %s", filePath)
	}

	// Load from file
	loaded, err := LoadFromFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load result: %v", err)
	}

	// Compare values
	if loaded.Suite != result.Suite {
		t.Errorf("Suite mismatch: got %s, want %s", loaded.Suite, result.Suite)
	}
	if loaded.Approach != result.Approach {
		t.Errorf("Approach mismatch: got %s, want %s", loaded.Approach, result.Approach)
	}
	if loaded.Model != result.Model {
		t.Errorf("Model mismatch: got %s, want %s", loaded.Model, result.Model)
	}
	if loaded.DurationSeconds != result.DurationSeconds {
		t.Errorf("DurationSeconds mismatch: got %d, want %d", loaded.DurationSeconds, result.DurationSeconds)
	}
	if loaded.TotalCalls != result.TotalCalls {
		t.Errorf("TotalCalls mismatch: got %d, want %d", loaded.TotalCalls, result.TotalCalls)
	}
	if loaded.InputTokens != result.InputTokens {
		t.Errorf("InputTokens mismatch: got %d, want %d", loaded.InputTokens, result.InputTokens)
	}
	if loaded.OutputTokens != result.OutputTokens {
		t.Errorf("OutputTokens mismatch: got %d, want %d", loaded.OutputTokens, result.OutputTokens)
	}
	if loaded.TotalTokens != result.TotalTokens {
		t.Errorf("TotalTokens mismatch: got %d, want %d", loaded.TotalTokens, result.TotalTokens)
	}
	if loaded.CostUSD != result.CostUSD {
		t.Errorf("CostUSD mismatch: got %f, want %f", loaded.CostUSD, result.CostUSD)
	}
	if loaded.SharedTestsPassed != result.SharedTestsPassed {
		t.Errorf("SharedTestsPassed mismatch: got %d, want %d", loaded.SharedTestsPassed, result.SharedTestsPassed)
	}
	if loaded.SharedTestsTotal != result.SharedTestsTotal {
		t.Errorf("SharedTestsTotal mismatch: got %d, want %d", loaded.SharedTestsTotal, result.SharedTestsTotal)
	}
	if loaded.FilesGenerated != result.FilesGenerated {
		t.Errorf("FilesGenerated mismatch: got %d, want %d", loaded.FilesGenerated, result.FilesGenerated)
	}
	if loaded.LinesGenerated != result.LinesGenerated {
		t.Errorf("LinesGenerated mismatch: got %d, want %d", loaded.LinesGenerated, result.LinesGenerated)
	}
	if loaded.OutputDir != result.OutputDir {
		t.Errorf("OutputDir mismatch: got %s, want %s", loaded.OutputDir, result.OutputDir)
	}
}

func TestFindLatestResult(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	os.Chdir(tmpDir)

	// Create multiple result files with different timestamps
	timestamps := []time.Time{
		time.Unix(1000000, 0),
		time.Unix(2000000, 0),
		time.Unix(3000000, 0),
	}

	for _, ts := range timestamps {
		result := &EvalResult{
			Suite:     "test-suite",
			Approach:  "ralph",
			Model:     "sonnet",
			Timestamp: ts,
		}
		_, err := result.SaveToFile()
		if err != nil {
			t.Fatalf("Failed to save result: %v", err)
		}
	}

	// Find the latest result
	filePath, err := FindLatestResult("test-suite", "ralph", "sonnet")
	if err != nil {
		t.Fatalf("Failed to find latest result: %v", err)
	}

	// Verify it's the one with the latest timestamp
	result, err := LoadFromFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load result: %v", err)
	}

	if result.Timestamp.Unix() != 3000000 {
		t.Errorf("Expected latest timestamp 3000000, got %d", result.Timestamp.Unix())
	}
}

func TestFindLatestResult_NotFound(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	os.Chdir(tmpDir)
	os.MkdirAll("evals/results", 0755)

	// Try to find a result that doesn't exist
	_, err := FindLatestResult("nonexistent", "ralph", "sonnet")
	if err == nil {
		t.Error("Expected error for nonexistent result, got nil")
	}
}
