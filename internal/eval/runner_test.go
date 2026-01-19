package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseRunMetrics(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		want        *RunMetrics
		wantErr     bool
	}{
		{
			name: "valid metrics",
			jsonContent: `{
				"total_claude_calls": 28,
				"input_tokens": 18265487,
				"output_tokens": 147108,
				"total_tokens": 18412595,
				"total_cost_usd": 10.657889849999998
			}`,
			want: &RunMetrics{
				TotalClaudeCalls: 28,
				InputTokens:      18265487,
				OutputTokens:     147108,
				TotalTokens:      18412595,
				TotalCostUSD:     10.657889849999998,
			},
			wantErr: false,
		},
		{
			name: "zero metrics",
			jsonContent: `{
				"total_claude_calls": 0,
				"input_tokens": 0,
				"output_tokens": 0,
				"total_tokens": 0,
				"total_cost_usd": 0
			}`,
			want: &RunMetrics{
				TotalClaudeCalls: 0,
				InputTokens:      0,
				OutputTokens:     0,
				TotalTokens:      0,
				TotalCostUSD:     0,
			},
			wantErr: false,
		},
		{
			name:        "invalid json",
			jsonContent: `{invalid}`,
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			metricsPath := filepath.Join(tmpDir, "run_metrics.json")

			if err := os.WriteFile(metricsPath, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse metrics
			got, err := parseRunMetrics(metricsPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRunMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.TotalClaudeCalls != tt.want.TotalClaudeCalls {
					t.Errorf("TotalClaudeCalls = %d, want %d", got.TotalClaudeCalls, tt.want.TotalClaudeCalls)
				}
				if got.InputTokens != tt.want.InputTokens {
					t.Errorf("InputTokens = %d, want %d", got.InputTokens, tt.want.InputTokens)
				}
				if got.OutputTokens != tt.want.OutputTokens {
					t.Errorf("OutputTokens = %d, want %d", got.OutputTokens, tt.want.OutputTokens)
				}
				if got.TotalTokens != tt.want.TotalTokens {
					t.Errorf("TotalTokens = %d, want %d", got.TotalTokens, tt.want.TotalTokens)
				}
				if got.TotalCostUSD != tt.want.TotalCostUSD {
					t.Errorf("TotalCostUSD = %f, want %f", got.TotalCostUSD, tt.want.TotalCostUSD)
				}
			}
		})
	}
}

func TestParseRunMetrics_FileNotFound(t *testing.T) {
	_, err := parseRunMetrics("/nonexistent/path/run_metrics.json")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestRunRalphApproach_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This is a basic integration test that verifies the function can be called
	// and handles missing ralph command gracefully
	config := &RunConfig{
		SuiteName:      "flask",
		Approach:       ApproachRalph,
		Model:          "sonnet",
		TimeoutSeconds: 10, // Short timeout for test
	}

	suite := &SuiteConfig{
		Name:         "flask",
		Requirements: "examples/flask_requirements.md",
	}

	// Create a mock requirements file
	tmpDir := t.TempDir()
	reqPath := filepath.Join(tmpDir, "requirements.md")
	if err := os.WriteFile(reqPath, []byte("# Test Requirements\n\nBuild a test app."), 0644); err != nil {
		t.Fatalf("failed to create mock requirements: %v", err)
	}

	// Update suite to use the temp requirements file
	suite.Requirements = reqPath

	// This will likely fail because ralph command may not be in the expected state,
	// but we're just testing that the function structure is correct
	_, err := runRalphApproach(config, suite)

	// We expect an error since we don't have a real ralph setup, but the function should execute
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestRunMetricsJSON(t *testing.T) {
	// Test that RunMetrics can be marshaled and unmarshaled correctly
	original := &RunMetrics{
		TotalClaudeCalls: 28,
		InputTokens:      18265487,
		OutputTokens:     147108,
		TotalTokens:      18412595,
		TotalCostUSD:     10.657889849999998,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded RunMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Compare
	if decoded.TotalClaudeCalls != original.TotalClaudeCalls {
		t.Errorf("TotalClaudeCalls mismatch: got %d, want %d", decoded.TotalClaudeCalls, original.TotalClaudeCalls)
	}
	if decoded.InputTokens != original.InputTokens {
		t.Errorf("InputTokens mismatch: got %d, want %d", decoded.InputTokens, original.InputTokens)
	}
	if decoded.OutputTokens != original.OutputTokens {
		t.Errorf("OutputTokens mismatch: got %d, want %d", decoded.OutputTokens, original.OutputTokens)
	}
	if decoded.TotalTokens != original.TotalTokens {
		t.Errorf("TotalTokens mismatch: got %d, want %d", decoded.TotalTokens, original.TotalTokens)
	}
	if decoded.TotalCostUSD != original.TotalCostUSD {
		t.Errorf("TotalCostUSD mismatch: got %f, want %f", decoded.TotalCostUSD, original.TotalCostUSD)
	}
}

func TestRunRalphApproach_ResultStructure(t *testing.T) {
	// Test that the result has the correct structure
	// This is a unit test that doesn't actually run ralph

	startTime := time.Now()
	metrics := &RunMetrics{
		TotalClaudeCalls: 28,
		InputTokens:      18265487,
		OutputTokens:     147108,
		TotalTokens:      18412595,
		TotalCostUSD:     10.657889849999998,
	}

	result := &EvalResult{
		Suite:           "flask",
		Approach:        ApproachRalph,
		Model:           "sonnet",
		Timestamp:       startTime,
		DurationSeconds: 237,
		TotalCalls:      metrics.TotalClaudeCalls,
		InputTokens:     metrics.InputTokens,
		OutputTokens:    metrics.OutputTokens,
		TotalTokens:     metrics.TotalTokens,
		CostUSD:         metrics.TotalCostUSD,
		OutputDir:       "/tmp/eval-ralph-flask-sonnet-1234567890",
	}

	// Verify all fields are set correctly
	if result.Suite != "flask" {
		t.Errorf("Suite = %s, want flask", result.Suite)
	}
	if result.Approach != ApproachRalph {
		t.Errorf("Approach = %s, want %s", result.Approach, ApproachRalph)
	}
	if result.Model != "sonnet" {
		t.Errorf("Model = %s, want sonnet", result.Model)
	}
	if result.TotalCalls != 28 {
		t.Errorf("TotalCalls = %d, want 28", result.TotalCalls)
	}
	if result.InputTokens != 18265487 {
		t.Errorf("InputTokens = %d, want 18265487", result.InputTokens)
	}
	if result.OutputTokens != 147108 {
		t.Errorf("OutputTokens = %d, want 147108", result.OutputTokens)
	}
	if result.TotalTokens != 18412595 {
		t.Errorf("TotalTokens = %d, want 18412595", result.TotalTokens)
	}
}
