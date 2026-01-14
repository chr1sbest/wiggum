package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadmeCheckConfig holds configuration for the readme check step.
type ReadmeCheckConfig struct {
	// ReadmePath is the path to the README file (default: README.md)
	ReadmePath string `json:"readme_path,omitempty"`
	// ProjectDir is the project root directory (default: current directory)
	ProjectDir string `json:"project_dir,omitempty"`
	// Patterns are file patterns to check for changes (default: ["*.go", "configs/*.json"])
	Patterns []string `json:"patterns,omitempty"`
}

// ReadmeCheckStep checks if README needs updating based on recent changes.
type ReadmeCheckStep struct {
	name string
}

// NewReadmeCheckStep creates a new readme check step.
func NewReadmeCheckStep() *ReadmeCheckStep {
	return &ReadmeCheckStep{name: "readme-check"}
}

func (s *ReadmeCheckStep) Name() string { return s.name }
func (s *ReadmeCheckStep) Type() string { return "readme-check" }

func (s *ReadmeCheckStep) Execute(ctx context.Context, rawConfig json.RawMessage) error {
	var cfg ReadmeCheckConfig
	if rawConfig != nil && len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return fmt.Errorf("failed to parse readme-check config: %w", err)
		}
	}

	// Set defaults
	if cfg.ReadmePath == "" {
		cfg.ReadmePath = "README.md"
	}
	if cfg.ProjectDir == "" {
		cfg.ProjectDir = "."
	}
	if len(cfg.Patterns) == 0 {
		cfg.Patterns = []string{"*.go", "cmd/**/*.go", "internal/**/*.go", "configs/*.json"}
	}

	readmePath := filepath.Join(cfg.ProjectDir, cfg.ReadmePath)

	// Check if README exists
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		return &ReadmeUpdateNeeded{
			Reason:  "README.md does not exist",
			Changes: []string{"Initial README creation needed"},
		}
	}

	// Get list of changed files using git
	changedFiles, err := s.getChangedFiles(ctx, cfg.ProjectDir)
	if err != nil {
		// If git fails, we can't determine changes - skip the check
		return nil
	}

	if len(changedFiles) == 0 {
		return nil
	}

	// Check if any significant files changed
	significantChanges := s.filterSignificantChanges(changedFiles, cfg.Patterns)
	if len(significantChanges) == 0 {
		return nil
	}

	// Check if changes might require README update
	needsUpdate := s.analyzeChanges(significantChanges)
	if needsUpdate != nil {
		return needsUpdate
	}

	return nil
}

// getChangedFiles returns files changed since last commit.
func (s *ReadmeCheckStep) getChangedFiles(ctx context.Context, dir string) ([]string, error) {
	// Get staged and unstaged changes
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) > 3 {
			// Status format: "XY filename" where XY is status codes
			file := strings.TrimSpace(line[3:])
			files = append(files, file)
		}
	}

	return files, nil
}

// filterSignificantChanges filters files matching significant patterns.
func (s *ReadmeCheckStep) filterSignificantChanges(files []string, patterns []string) []string {
	var significant []string
	for _, file := range files {
		for _, pattern := range patterns {
			matched, _ := filepath.Match(pattern, file)
			if matched {
				significant = append(significant, file)
				break
			}
			// Also check for nested patterns
			if strings.Contains(pattern, "**") {
				// Simple ** handling: check if base pattern matches
				base := strings.Replace(pattern, "**", "*", -1)
				if matched, _ := filepath.Match(base, filepath.Base(file)); matched {
					significant = append(significant, file)
					break
				}
			}
		}
	}
	return significant
}

// analyzeChanges determines if changes likely need README updates.
func (s *ReadmeCheckStep) analyzeChanges(files []string) *ReadmeUpdateNeeded {
	var reasons []string

	for _, file := range files {
		// New CLI commands
		if strings.Contains(file, "cmd/") && strings.HasSuffix(file, ".go") {
			reasons = append(reasons, fmt.Sprintf("CLI changes: %s", file))
		}
		// Config format changes
		if strings.HasSuffix(file, ".json") && strings.Contains(file, "config") {
			reasons = append(reasons, fmt.Sprintf("Config changes: %s", file))
		}
		// New packages
		if strings.Contains(file, "internal/") && strings.HasSuffix(file, ".go") {
			if strings.Contains(filepath.Base(file), "_test") {
				continue // Skip test files
			}
			reasons = append(reasons, fmt.Sprintf("Internal changes: %s", file))
		}
	}

	if len(reasons) > 0 {
		return &ReadmeUpdateNeeded{
			Reason:  "Significant code changes detected",
			Changes: reasons,
		}
	}

	return nil
}

// ReadmeUpdateNeeded is returned when README should be reviewed.
type ReadmeUpdateNeeded struct {
	Reason  string
	Changes []string
}

func (e *ReadmeUpdateNeeded) Error() string {
	return fmt.Sprintf("README review suggested: %s\nChanges:\n  - %s",
		e.Reason, strings.Join(e.Changes, "\n  - "))
}

// IsReadmeUpdateNeeded checks if an error indicates README needs updating.
func IsReadmeUpdateNeeded(err error) (*ReadmeUpdateNeeded, bool) {
	if err == nil {
		return nil, false
	}
	r, ok := err.(*ReadmeUpdateNeeded)
	return r, ok
}
