package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCalcWinner(t *testing.T) {
	tests := []struct {
		name           string
		ralphVal       int
		oneshotVal     int
		higherIsBetter bool
		want           string
	}{
		{
			name:           "ralph wins - lower is better",
			ralphVal:       100,
			oneshotVal:     200,
			higherIsBetter: false,
			want:           "Ralph -50.00%",
		},
		{
			name:           "oneshot wins - lower is better",
			ralphVal:       200,
			oneshotVal:     100,
			higherIsBetter: false,
			want:           "Oneshot -100.00%",
		},
		{
			name:           "ralph wins - higher is better",
			ralphVal:       200,
			oneshotVal:     100,
			higherIsBetter: true,
			want:           "Ralph +100.00%",
		},
		{
			name:           "oneshot wins - higher is better",
			ralphVal:       100,
			oneshotVal:     200,
			higherIsBetter: true,
			want:           "Oneshot +50.00%",
		},
		{
			name:           "tie",
			ralphVal:       100,
			oneshotVal:     100,
			higherIsBetter: false,
			want:           "Tie",
		},
		{
			name:           "oneshot is zero",
			ralphVal:       100,
			oneshotVal:     0,
			higherIsBetter: false,
			want:           "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcWinner(tt.ralphVal, tt.oneshotVal, tt.higherIsBetter)
			if got != tt.want {
				t.Errorf("calcWinner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalcWinnerFloat(t *testing.T) {
	tests := []struct {
		name           string
		ralphVal       float64
		oneshotVal     float64
		higherIsBetter bool
		want           string
	}{
		{
			name:           "ralph wins - lower is better",
			ralphVal:       1.50,
			oneshotVal:     3.00,
			higherIsBetter: false,
			want:           "Ralph -50.00%",
		},
		{
			name:           "oneshot wins - lower is better",
			ralphVal:       3.00,
			oneshotVal:     1.50,
			higherIsBetter: false,
			want:           "Oneshot -100.00%",
		},
		{
			name:           "tie",
			ralphVal:       1.50,
			oneshotVal:     1.50,
			higherIsBetter: false,
			want:           "Tie",
		},
		{
			name:           "oneshot is zero",
			ralphVal:       1.50,
			oneshotVal:     0,
			higherIsBetter: false,
			want:           "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcWinnerFloat(tt.ralphVal, tt.oneshotVal, tt.higherIsBetter)
			if got != tt.want {
				t.Errorf("calcWinnerFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	// Create temporary results directory
	tmpDir := t.TempDir()
	resultsDir := filepath.Join(tmpDir, "evals", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Fatalf("failed to create results dir: %v", err)
	}

	// Change working directory to temp dir
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create sample result files
	ralphResult := &EvalResult{
		Suite:             "test-suite",
		Approach:          "ralph",
		Model:             "test-model",
		Timestamp:         time.Unix(1000000000, 0),
		DurationSeconds:   100,
		TotalCalls:        5,
		InputTokens:       1000,
		OutputTokens:      500,
		TotalTokens:       1500,
		CostUSD:           0.50,
		SharedTestsPassed: 10,
		SharedTestsTotal:  10,
		FilesGenerated:    5,
		LinesGenerated:    100,
		OutputDir:         "/tmp/test",
	}

	oneshotResult := &EvalResult{
		Suite:             "test-suite",
		Approach:          "oneshot",
		Model:             "test-model",
		Timestamp:         time.Unix(1000000100, 0),
		DurationSeconds:   200,
		TotalCalls:        1,
		InputTokens:       2000,
		OutputTokens:      1000,
		TotalTokens:       3000,
		CostUSD:           1.00,
		SharedTestsPassed: 8,
		SharedTestsTotal:  10,
		FilesGenerated:    3,
		LinesGenerated:    80,
		OutputDir:         "/tmp/test2",
	}

	// Save result files
	ralphFile := filepath.Join(resultsDir, "test-suite-ralph-test-model-1000000000.json")
	oneshotFile := filepath.Join(resultsDir, "test-suite-oneshot-test-model-1000000100.json")

	ralphData, _ := json.MarshalIndent(ralphResult, "", "  ")
	if err := os.WriteFile(ralphFile, ralphData, 0644); err != nil {
		t.Fatalf("failed to write ralph result: %v", err)
	}

	oneshotData, _ := json.MarshalIndent(oneshotResult, "", "  ")
	if err := os.WriteFile(oneshotFile, oneshotData, 0644); err != nil {
		t.Fatalf("failed to write oneshot result: %v", err)
	}

	// Test Compare function
	err = Compare("test-suite", "test-model")
	if err != nil {
		t.Errorf("Compare() failed: %v", err)
	}
}

func TestCompareMissingFiles(t *testing.T) {
	// Create temporary results directory
	tmpDir := t.TempDir()
	resultsDir := filepath.Join(tmpDir, "evals", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Fatalf("failed to create results dir: %v", err)
	}

	// Change working directory to temp dir
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test with missing files
	err = Compare("nonexistent", "test-model")
	if err == nil {
		t.Error("Compare() should fail with missing files")
	}
}
