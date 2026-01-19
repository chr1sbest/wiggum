package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAppDirectory(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string // Returns temp dir path
		expectError bool
	}{
		{
			name: "app.py in root",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "app.py"), []byte("# app"), 0644)
				return dir
			},
			expectError: false,
		},
		{
			name: "run.py in root",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "run.py"), []byte("# run"), 0644)
				return dir
			},
			expectError: false,
		},
		{
			name: "app directory in root",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "app"), 0755)
				return dir
			},
			expectError: false,
		},
		{
			name: "nested app.py",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				nested := filepath.Join(dir, "myproject")
				os.MkdirAll(nested, 0755)
				os.WriteFile(filepath.Join(nested, "app.py"), []byte("# app"), 0644)
				return dir
			},
			expectError: false,
		},
		{
			name: "no app found",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)
			result, err := findAppDirectory(dir)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == "" {
					t.Errorf("expected non-empty result")
				}
			}
		})
	}
}

func TestSetupEnvFile(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // Returns temp dir path
		expectEnv bool
	}{
		{
			name: "copies .env.example when .env missing",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=value"), 0644)
				return dir
			},
			expectEnv: true,
		},
		{
			name: "does not copy when .env exists",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=example"), 0644)
				os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=existing"), 0644)
				return dir
			},
			expectEnv: true,
		},
		{
			name: "does nothing when no .env.example",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			expectEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)
			err := setupEnvFile(dir)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			envPath := filepath.Join(dir, ".env")
			envExists := fileExists(envPath)

			if tt.expectEnv && !envExists {
				t.Errorf("expected .env to exist but it doesn't")
			}
		})
	}
}

func TestParsePytestOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    TestResult
		wantErr bool
	}{
		{
			name:   "all passed",
			output: "test_auth.py::test_login PASSED\ntest_auth.py::test_logout PASSED\n\n===== 5 passed in 2.34s =====",
			want: TestResult{
				Passed:  5,
				Failed:  0,
				Skipped: 0,
				Total:   5,
			},
			wantErr: false,
		},
		{
			name:   "some failed",
			output: "test_auth.py::test_login FAILED\ntest_auth.py::test_logout PASSED\n\n===== 3 passed, 2 failed in 2.34s =====",
			want: TestResult{
				Passed:  3,
				Failed:  2,
				Skipped: 0,
				Total:   5,
			},
			wantErr: false,
		},
		{
			name:   "with skipped",
			output: "test_auth.py::test_login PASSED\ntest_auth.py::test_logout SKIPPED\n\n===== 10 passed, 1 failed, 2 skipped in 2.34s =====",
			want: TestResult{
				Passed:  10,
				Failed:  1,
				Skipped: 2,
				Total:   13,
			},
			wantErr: false,
		},
		{
			name:   "no tests run",
			output: "===== no tests ran in 0.01s =====",
			want: TestResult{
				Passed:  0,
				Failed:  0,
				Skipped: 0,
				Total:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp dir to simulate pytest execution
			dir := t.TempDir()
			suiteDir := filepath.Join(dir, "tests")
			os.MkdirAll(suiteDir, 0755)

			// Create a mock pytest script that outputs the test output
			mockPytest := filepath.Join(dir, "venv", "bin", "pytest")
			os.MkdirAll(filepath.Join(dir, "venv", "bin"), 0755)
			script := "#!/bin/bash\necho '" + tt.output + "'\nexit 0"
			os.WriteFile(mockPytest, []byte(script), 0755)

			result, err := runPytest(dir, suiteDir, 8000)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Passed != tt.want.Passed {
				t.Errorf("Passed: got %d, want %d", result.Passed, tt.want.Passed)
			}
			if result.Failed != tt.want.Failed {
				t.Errorf("Failed: got %d, want %d", result.Failed, tt.want.Failed)
			}
			if result.Skipped != tt.want.Skipped {
				t.Errorf("Skipped: got %d, want %d", result.Skipped, tt.want.Skipped)
			}
			if result.Total != tt.want.Total {
				t.Errorf("Total: got %d, want %d", result.Total, tt.want.Total)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	dirPath := filepath.Join(dir, "testdir")

	// Create a file and directory
	os.WriteFile(filePath, []byte("test"), 0644)
	os.MkdirAll(dirPath, 0755)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", filePath, true},
		{"directory should not be file", dirPath, false},
		{"non-existent file", filepath.Join(dir, "missing.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileExists(tt.path); got != tt.want {
				t.Errorf("fileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	dirPath := filepath.Join(dir, "testdir")

	// Create a file and directory
	os.WriteFile(filePath, []byte("test"), 0644)
	os.MkdirAll(dirPath, 0755)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing directory", dirPath, true},
		{"file should not be directory", filePath, false},
		{"non-existent directory", filepath.Join(dir, "missing"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dirExists(tt.path); got != tt.want {
				t.Errorf("dirExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
