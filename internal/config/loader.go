package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles loading configuration files.
type Loader struct {
	configDir string
}

// NewLoader creates a new config loader.
func NewLoader(configDir string) *Loader {
	return &Loader{configDir: configDir}
}

// LoadFile loads a configuration from a specific file path.
// Environment variables in the config are expanded before parsing.
// Supports ${VAR} and ${VAR:-default} syntax.
func (l *Loader) LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables before parsing JSON
	data = ExpandEnvVarsBytes(data)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &cfg, nil
}

// LoadAndValidate loads and validates a config file against known step types.
func (l *Loader) LoadAndValidate(path string, knownStepTypes []string) (*Config, error) {
	cfg, err := l.LoadFile(path)
	if err != nil {
		return nil, err
	}

	if err := ValidateConfig(cfg, knownStepTypes); err != nil {
		return nil, fmt.Errorf("config validation failed for %s:\n%w", path, err)
	}

	return cfg, nil
}

// LoadDirectory scans a directory for JSON config files and loads them all.
func (l *Loader) LoadDirectory(dir string) ([]*Config, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []*Config
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		cfg, err := l.LoadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

// LoadDefault loads the default configuration from the config directory.
func (l *Loader) LoadDefault() (*Config, error) {
	defaultPath := filepath.Join(l.configDir, "default.json")
	return l.LoadFile(defaultPath)
}
