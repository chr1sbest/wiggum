package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateProjectDir(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Create a temporary test directory
	tempDir := t.TempDir()
	// Resolve symlinks (macOS uses /private/var)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	testWiggumDir := filepath.Join(tempDir, "wiggum")
	if err := os.MkdirAll(testWiggumDir, 0755); err != nil {
		t.Fatalf("failed to create test wiggum directory: %v", err)
	}

	// Change to the test wiggum directory
	if err := os.Chdir(testWiggumDir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	tests := []struct {
		name          string
		config        *ProjectDirConfig
		expectNested  bool
		wantDirPrefix string
	}{
		{
			name: "ralph approach creates nested structure",
			config: &ProjectDirConfig{
				Approach:  ApproachRalph,
				SuiteName: "flask",
				Model:     "sonnet",
				Timestamp: time.Unix(1234567890, 0),
			},
			expectNested:  true,
			wantDirPrefix: "eval-ralph-flask-sonnet-1234567890",
		},
		{
			name: "oneshot approach creates flat structure",
			config: &ProjectDirConfig{
				Approach:  ApproachOneshot,
				SuiteName: "logagg",
				Model:     "opus",
				Timestamp: time.Unix(1234567890, 0),
			},
			expectNested:  false,
			wantDirPrefix: "eval-oneshot-logagg-opus-1234567890",
		},
		{
			name: "different timestamp creates unique directory",
			config: &ProjectDirConfig{
				Approach:  ApproachRalph,
				SuiteName: "tasktracker",
				Model:     "haiku",
				Timestamp: time.Unix(9876543210, 0),
			},
			expectNested:  true,
			wantDirPrefix: "eval-ralph-tasktracker-haiku-9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir, err := CreateProjectDir(tt.config)
			if err != nil {
				t.Fatalf("CreateProjectDir() error = %v", err)
			}

			// Verify the directory exists
			if _, err := os.Stat(workingDir); os.IsNotExist(err) {
				t.Errorf("expected directory to exist at %s", workingDir)
			}

			// Verify directory naming
			projectRoot := GetProjectRootDir(workingDir, tt.config.Approach)
			if !strings.HasSuffix(projectRoot, tt.wantDirPrefix) {
				t.Errorf("project root directory = %s, want suffix %s", projectRoot, tt.wantDirPrefix)
			}

			// Verify directory is created at parent level (sibling to wiggum/)
			expectedParent := tempDir
			actualParent := filepath.Dir(projectRoot)
			if actualParent != expectedParent {
				t.Errorf("project parent directory = %s, want %s", actualParent, expectedParent)
			}

			// Verify nested structure for ralph
			if tt.expectNested {
				expectedWorkingDir := filepath.Join(projectRoot, tt.config.SuiteName)
				if workingDir != expectedWorkingDir {
					t.Errorf("working directory = %s, want %s", workingDir, expectedWorkingDir)
				}

				// Verify suite subdirectory exists
				if _, err := os.Stat(workingDir); os.IsNotExist(err) {
					t.Errorf("expected suite subdirectory to exist at %s", workingDir)
				}
			} else {
				// For oneshot, working dir should be the project root
				if workingDir != projectRoot {
					t.Errorf("working directory = %s, want %s (project root)", workingDir, projectRoot)
				}
			}

			// Cleanup
			if err := CleanupProjectDir(projectRoot); err != nil {
				t.Errorf("CleanupProjectDir() error = %v", err)
			}

			// Verify cleanup worked
			if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
				t.Errorf("expected directory to be removed at %s", projectRoot)
			}
		})
	}
}

func TestCleanupProjectDir(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
		errMsg  string
	}{
		{
			name: "cleanup valid eval directory",
			setup: func() string {
				dir := filepath.Join(tempDir, "eval-ralph-test-sonnet-123")
				os.MkdirAll(dir, 0755)
				// Create some content
				os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644)
				os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
				return dir
			},
			wantErr: false,
		},
		{
			name: "cleanup nested eval directory",
			setup: func() string {
				dir := filepath.Join(tempDir, "eval-ralph-nested-opus-456", "suite")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
				// Return the root, not the nested directory
				return filepath.Join(tempDir, "eval-ralph-nested-opus-456")
			},
			wantErr: false,
		},
		{
			name: "reject empty path",
			setup: func() string {
				return ""
			},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "reject non-eval directory",
			setup: func() string {
				dir := filepath.Join(tempDir, "not-an-eval-dir")
				os.MkdirAll(dir, 0755)
				return dir
			},
			wantErr: true,
			errMsg:  "doesn't look like an eval project",
		},
		{
			name: "handle non-existent directory gracefully",
			setup: func() string {
				return filepath.Join(tempDir, "eval-nonexistent-test-sonnet-999")
			},
			wantErr: false, // os.RemoveAll succeeds even if path doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup()
			err := CleanupProjectDir(dir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CleanupProjectDir() expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CleanupProjectDir() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CleanupProjectDir() unexpected error = %v", err)
				}
				// Verify directory was removed
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Errorf("expected directory to be removed at %s", dir)
				}
			}
		})
	}
}

func TestGetProjectRootDir(t *testing.T) {
	tests := []struct {
		name       string
		workingDir string
		approach   string
		want       string
	}{
		{
			name:       "ralph approach returns parent directory",
			workingDir: "/tmp/eval-ralph-test-sonnet-123/flask",
			approach:   ApproachRalph,
			want:       "/tmp/eval-ralph-test-sonnet-123",
		},
		{
			name:       "oneshot approach returns same directory",
			workingDir: "/tmp/eval-oneshot-test-opus-456",
			approach:   ApproachOneshot,
			want:       "/tmp/eval-oneshot-test-opus-456",
		},
		{
			name:       "ralph with nested path",
			workingDir: "/home/user/projects/eval-ralph-logagg-haiku-789/logagg",
			approach:   ApproachRalph,
			want:       "/home/user/projects/eval-ralph-logagg-haiku-789",
		},
		{
			name:       "oneshot with absolute path",
			workingDir: "/var/tmp/eval-oneshot-tasktracker-sonnet-999",
			approach:   ApproachOneshot,
			want:       "/var/tmp/eval-oneshot-tasktracker-sonnet-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetProjectRootDir(tt.workingDir, tt.approach)
			if got != tt.want {
				t.Errorf("GetProjectRootDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectDirConfig_Integration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Create a temporary test directory
	tempDir := t.TempDir()
	// Resolve symlinks (macOS uses /private/var)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	testWiggumDir := filepath.Join(tempDir, "wiggum")
	if err := os.MkdirAll(testWiggumDir, 0755); err != nil {
		t.Fatalf("failed to create test wiggum directory: %v", err)
	}

	// Change to the test wiggum directory
	if err := os.Chdir(testWiggumDir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	// Test full workflow: create, verify structure, cleanup
	config := &ProjectDirConfig{
		Approach:  ApproachRalph,
		SuiteName: "integration-test",
		Model:     "sonnet",
		Timestamp: time.Now(),
	}

	// Create project directory
	workingDir, err := CreateProjectDir(config)
	if err != nil {
		t.Fatalf("CreateProjectDir() error = %v", err)
	}

	// Get project root
	projectRoot := GetProjectRootDir(workingDir, config.Approach)

	// Verify structure
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		t.Errorf("project root doesn't exist: %s", projectRoot)
	}
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		t.Errorf("working directory doesn't exist: %s", workingDir)
	}

	// Create some test files
	testFile := filepath.Join(workingDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Verify test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("test file doesn't exist: %s", testFile)
	}

	// Cleanup
	if err := CleanupProjectDir(projectRoot); err != nil {
		t.Fatalf("CleanupProjectDir() error = %v", err)
	}

	// Verify everything is removed
	if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
		t.Errorf("project root should be removed: %s", projectRoot)
	}
	if _, err := os.Stat(workingDir); !os.IsNotExist(err) {
		t.Errorf("working directory should be removed: %s", workingDir)
	}
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("test file should be removed: %s", testFile)
	}
}
