package eval

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SuiteType indicates the type of project being tested
type SuiteType string

const (
	SuiteTypeWeb SuiteType = "web" // Web app - starts server, runs pytest
	SuiteTypeCLI SuiteType = "cli" // CLI tool - builds binary, runs Go tests
)

// SuiteConfig represents the configuration for an evaluation suite
type SuiteConfig struct {
	Name         string    `yaml:"name"`
	Description  string    `yaml:"description"`
	Requirements string    `yaml:"requirements"`
	Language     string    `yaml:"language"`
	Type         SuiteType `yaml:"type"` // "web" or "cli"
	Timeout      string    `yaml:"timeout"`
	Setup        []string  `yaml:"setup"`
}

// IsWebApp returns true if this is a web application suite
func (s *SuiteConfig) IsWebApp() bool {
	return s.Type == SuiteTypeWeb || s.Type == ""
}

// IsCLI returns true if this is a CLI tool suite
func (s *SuiteConfig) IsCLI() bool {
	return s.Type == SuiteTypeCLI
}

// LoadSuite loads a suite configuration from the suite.yaml file
// suiteName is the name of the suite directory (e.g., "logagg", "flask")
func LoadSuite(suiteName string) (*SuiteConfig, error) {
	suiteDir := filepath.Join("evals", "suites", suiteName)
	yamlPath := filepath.Join(suiteDir, "suite.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read suite.yaml for %s: %w", suiteName, err)
	}

	var config SuiteConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse suite.yaml for %s: %w", suiteName, err)
	}

	return &config, nil
}
