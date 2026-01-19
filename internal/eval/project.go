package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProjectDirConfig contains configuration for creating a project directory.
type ProjectDirConfig struct {
	Approach  string
	SuiteName string
	Model     string
	Timestamp time.Time
}

// CreateProjectDir creates a unique project directory for an eval run.
// Directory naming: eval-{approach}-{suite}-{model}-{timestamp}
// For ralph approach: creates nested suite subdirectory
// For oneshot approach: uses flat structure
// Returns the project directory path and any error.
func CreateProjectDir(config *ProjectDirConfig) (string, error) {
	// Generate unique directory name
	dirName := fmt.Sprintf("eval-%s-%s-%s-%d",
		config.Approach,
		config.SuiteName,
		config.Model,
		config.Timestamp.Unix(),
	)

	// Get the wiggum project root (parent of current directory)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Project directory is created at the same level as wiggum/
	// This matches the run.sh behavior: PROJECT_DIR="$WIGGUM_DIR/$PROJECT"
	projectDir := filepath.Join(filepath.Dir(cwd), dirName)

	// Create the project directory
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	// For ralph approach, create nested suite subdirectory
	if config.Approach == ApproachRalph {
		suiteDir := filepath.Join(projectDir, config.SuiteName)
		if err := os.MkdirAll(suiteDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create suite subdirectory: %w", err)
		}
		// Return the suite subdirectory as the working directory for ralph
		return suiteDir, nil
	}

	// For oneshot, return the project directory directly
	return projectDir, nil
}

// CleanupProjectDir removes a project directory and all its contents.
// Use with caution - this permanently deletes the directory.
func CleanupProjectDir(projectDir string) error {
	if projectDir == "" {
		return fmt.Errorf("project directory path cannot be empty")
	}

	// Safety check: ensure the path looks like an eval project directory
	dirName := filepath.Base(projectDir)
	if len(dirName) < 5 || dirName[:5] != "eval-" {
		return fmt.Errorf("refusing to delete directory that doesn't look like an eval project: %s", projectDir)
	}

	// Remove the directory and all contents
	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("failed to remove project directory: %w", err)
	}

	return nil
}

// GetProjectRootDir returns the root project directory for a given working directory.
// For ralph approach (nested): returns the parent directory
// For oneshot approach (flat): returns the directory itself
func GetProjectRootDir(workingDir, approach string) string {
	if approach == ApproachRalph {
		// For ralph, the working directory is the suite subdirectory,
		// so we need to go up one level to get the project root
		return filepath.Dir(workingDir)
	}
	// For oneshot, the working directory is the project root
	return workingDir
}
