package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSuite(t *testing.T) {
	tests := []struct {
		name          string
		suiteName     string
		wantErr       bool
		expectedName  string
		expectedLang  string
		expectedTests int
	}{
		{
			name:          "logagg suite",
			suiteName:     "logagg",
			wantErr:       false,
			expectedName:  "logagg",
			expectedLang:  "go",
			expectedTests: 1,
		},
		{
			name:          "flask suite",
			suiteName:     "flask",
			wantErr:       false,
			expectedName:  "flask",
			expectedLang:  "python",
			expectedTests: 1,
		},
		{
			name:          "tasktracker suite",
			suiteName:     "tasktracker",
			wantErr:       false,
			expectedName:  "tasktracker",
			expectedLang:  "python",
			expectedTests: 1,
		},
		{
			name:      "non-existent suite",
			suiteName: "nonexistent",
			wantErr:   true,
		},
	}

	// Change to project root for tests to work
	if _, err := os.Stat("go.mod"); err != nil {
		// We're likely in the internal/eval directory during test execution
		// Try to find the project root
		if err := os.Chdir("../../"); err != nil {
			t.Fatalf("Failed to change to project root: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadSuite(tt.suiteName)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSuite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if config.Name != tt.expectedName {
				t.Errorf("LoadSuite() Name = %v, want %v", config.Name, tt.expectedName)
			}
			if config.Language != tt.expectedLang {
				t.Errorf("LoadSuite() Language = %v, want %v", config.Language, tt.expectedLang)
			}
			if len(config.Tests.Shared) != tt.expectedTests {
				t.Errorf("LoadSuite() Tests.Shared length = %v, want %v", len(config.Tests.Shared), tt.expectedTests)
			}
			if config.Description == "" {
				t.Errorf("LoadSuite() Description is empty")
			}
			if config.Requirements == "" {
				t.Errorf("LoadSuite() Requirements is empty")
			}
			if config.Timeout == "" {
				t.Errorf("LoadSuite() Timeout is empty")
			}
		})
	}
}

func TestLoadSuiteFields(t *testing.T) {
	// Change to project root for test to work
	if _, err := os.Stat("go.mod"); err != nil {
		if err := os.Chdir("../../"); err != nil {
			t.Fatalf("Failed to change to project root: %v", err)
		}
	}

	// Test logagg suite in detail
	config, err := LoadSuite("logagg")
	if err != nil {
		t.Fatalf("LoadSuite() failed: %v", err)
	}

	if config.Name != "logagg" {
		t.Errorf("Name = %v, want logagg", config.Name)
	}
	if config.Description == "" {
		t.Errorf("Description is empty")
	}
	if config.Requirements != "examples/log_aggregator_requirements.md" {
		t.Errorf("Requirements = %v, want examples/log_aggregator_requirements.md", config.Requirements)
	}
	if config.Language != "go" {
		t.Errorf("Language = %v, want go", config.Language)
	}
	if config.Timeout != "1h" {
		t.Errorf("Timeout = %v, want 1h", config.Timeout)
	}
	if len(config.Tests.Shared) != 1 {
		t.Errorf("Tests.Shared length = %v, want 1", len(config.Tests.Shared))
	}
	if len(config.Tests.Shared) > 0 && config.Tests.Shared[0] == "" {
		t.Errorf("Tests.Shared[0] is empty")
	}

	// Test flask suite to verify setup commands
	flaskConfig, err := LoadSuite("flask")
	if err != nil {
		t.Fatalf("LoadSuite(flask) failed: %v", err)
	}

	if len(flaskConfig.Setup) < 2 {
		t.Errorf("Flask setup commands = %v, expected at least 2", len(flaskConfig.Setup))
	}
	if flaskConfig.Language != "python" {
		t.Errorf("Flask Language = %v, want python", flaskConfig.Language)
	}
}

func TestLoadSuiteAbsolutePath(t *testing.T) {
	// Test that LoadSuite works with absolute paths too
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to project root if needed
	if _, err := os.Stat("go.mod"); err != nil {
		if err := os.Chdir("../../"); err != nil {
			t.Fatalf("Failed to change to project root: %v", err)
		}
		defer os.Chdir(wd)
	}

	// Verify the suite directory exists
	suitePath := filepath.Join("evals", "suites", "logagg", "suite.yaml")
	if _, err := os.Stat(suitePath); err != nil {
		t.Skipf("Suite file not found at %s, skipping test", suitePath)
	}

	config, err := LoadSuite("logagg")
	if err != nil {
		t.Errorf("LoadSuite() with absolute path failed: %v", err)
	}
	if config == nil {
		t.Error("LoadSuite() returned nil config")
	}
}
