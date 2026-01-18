package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRunPreflight(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	promptsDir := filepath.Join(ralphDir, "prompts")
	configsDir := filepath.Join(ralphDir, "configs")

	// Create required directories
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("Failed to create configs dir: %v", err)
	}

	// Create required files
	files := map[string]string{
		filepath.Join(ralphDir, "prd.json"):          `{"version":1,"tasks":[]}`,
		filepath.Join(ralphDir, "requirements.md"):   "# Requirements",
		filepath.Join(promptsDir, "SETUP_PROMPT.md"): "Setup prompt",
		filepath.Join(promptsDir, "LOOP_PROMPT.md"):  "Loop prompt",
		filepath.Join(configsDir, "default.json"):    `{"name":"test"}`,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test with all files present
	configFile := ".ralph/configs/default.json"
	if err := validateRunPreflight(configFile); err != nil {
		t.Errorf("validateRunPreflight() returned error with all files present: %v", err)
	}

	// Test with missing config file
	if err := validateRunPreflight(".ralph/configs/nonexistent.json"); err == nil {
		t.Error("validateRunPreflight() should return error for missing config file")
	}
}

func TestMustGetwd(t *testing.T) {
	wd := mustGetwd()
	if wd == "" {
		t.Error("mustGetwd() returned empty string")
	}
}
