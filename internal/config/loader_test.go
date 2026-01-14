package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "test.json")
	configContent := `{
		"name": "test-config",
		"description": "Test configuration",
		"steps": [
			{"type": "command", "name": "test-step", "config": {"command": "echo hello"}}
		]
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	loader := NewLoader(dir)
	cfg, err := loader.LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if cfg.Name != "test-config" {
		t.Errorf("expected name 'test-config', got %s", cfg.Name)
	}
	if len(cfg.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(cfg.Steps))
	}
	if cfg.Steps[0].Type != "command" {
		t.Errorf("expected step type 'command', got %s", cfg.Steps[0].Type)
	}
}

func TestLoadDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create two config files
	configs := []struct {
		name    string
		content string
	}{
		{"a.json", `{"name": "config-a", "steps": []}`},
		{"b.json", `{"name": "config-b", "steps": []}`},
	}

	for _, c := range configs {
		path := filepath.Join(dir, c.name)
		if err := os.WriteFile(path, []byte(c.content), 0644); err != nil {
			t.Fatalf("Failed to create config %s: %v", c.name, err)
		}
	}

	loader := NewLoader(dir)
	cfgs, err := loader.LoadDirectory(dir)
	if err != nil {
		t.Fatalf("LoadDirectory failed: %v", err)
	}

	if len(cfgs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(cfgs))
	}
}

func TestStepConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{"nil enabled defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := StepConfig{Enabled: tt.enabled}
			if got := sc.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
