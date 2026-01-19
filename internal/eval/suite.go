package eval

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SuiteConfig represents the configuration for an evaluation suite
type SuiteConfig struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	Requirements string      `yaml:"requirements"`
	Language     string      `yaml:"language"`
	Timeout      string      `yaml:"timeout"`
	Setup        []string    `yaml:"setup"`
	Tests        TestsConfig `yaml:"tests"`
}

// TestsConfig represents the test configuration section
type TestsConfig struct {
	Shared []string `yaml:"shared"`
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
