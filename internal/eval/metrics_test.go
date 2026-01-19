package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectCodeMetrics(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string) error
		wantFiles int
		wantLines int
		wantErr   bool
	}{
		{
			name: "empty directory",
			setup: func(dir string) error {
				return nil
			},
			wantFiles: 0,
			wantLines: 0,
			wantErr:   false,
		},
		{
			name: "single go file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
			},
			wantFiles: 1,
			wantLines: 3,
			wantErr:   false,
		},
		{
			name: "single python file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "app.py"), []byte("def main():\n    pass\n"), 0644)
			},
			wantFiles: 1,
			wantLines: 2,
			wantErr:   false,
		},
		{
			name: "multiple file types",
			setup: func(dir string) error {
				files := map[string]string{
					"main.go":     "package main\n\nfunc main() {}\n", // 3 lines
					"app.py":      "def main():\n    pass\n",          // 2 lines
					"README.md":   "# Project\n\nDescription\n",       // counted but not for lines
					"config.yaml": "key: value\n",                     // counted but not for lines
					"config.yml":  "another: value\n",                 // counted but not for lines
					"go.mod":      "module test\n",                    // counted but not for lines
					"go.sum":      "checksum here\n",                  // counted but not for lines
					"other.txt":   "not counted\n",                    // not counted
				}
				for name, content := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			wantFiles: 7, // go, py, md, yaml, yml, mod, sum
			wantLines: 5, // only go and py files
			wantErr:   false,
		},
		{
			name: "exclude .ralph directory",
			setup: func(dir string) error {
				// Create files in main dir
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
					return err
				}
				// Create .ralph dir with files
				ralphDir := filepath.Join(dir, ".ralph")
				if err := os.MkdirAll(ralphDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(ralphDir, "prd.go"), []byte("package ralph\n"), 0644); err != nil {
					return err
				}
				return nil
			},
			wantFiles: 1, // only main.go, not prd.go
			wantLines: 1, // only main.go lines
			wantErr:   false,
		},
		{
			name: "exclude venv directory",
			setup: func(dir string) error {
				// Create files in main dir
				if err := os.WriteFile(filepath.Join(dir, "app.py"), []byte("def main():\n    pass\n"), 0644); err != nil {
					return err
				}
				// Create venv dir with files
				venvDir := filepath.Join(dir, "venv")
				if err := os.MkdirAll(venvDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(venvDir, "lib.py"), []byte("import sys\n"), 0644); err != nil {
					return err
				}
				return nil
			},
			wantFiles: 1, // only app.py, not lib.py
			wantLines: 2, // only app.py lines
			wantErr:   false,
		},
		{
			name: "nested directories",
			setup: func(dir string) error {
				// Create nested structure
				srcDir := filepath.Join(dir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(srcDir, "lib.go"), []byte("package lib\n"), 0644); err != nil {
					return err
				}

				testDir := filepath.Join(dir, "tests")
				if err := os.MkdirAll(testDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(testDir, "test.py"), []byte("def test():\n    pass\n"), 0644); err != nil {
					return err
				}

				return nil
			},
			wantFiles: 2, // lib.go and test.py
			wantLines: 3, // 1 from lib.go, 2 from test.py
			wantErr:   false,
		},
		{
			name: "file without trailing newline",
			setup: func(dir string) error {
				// Create file without trailing newline
				return os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
			},
			wantFiles: 1,
			wantLines: 0, // no newlines = 0 lines
			wantErr:   false,
		},
		{
			name: "mixed nested and excluded directories",
			setup: func(dir string) error {
				// Create src/main.go
				srcDir := filepath.Join(dir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
					return err
				}

				// Create .ralph/config.yaml (should be excluded)
				ralphDir := filepath.Join(dir, ".ralph")
				if err := os.MkdirAll(ralphDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("key: value\n"), 0644); err != nil {
					return err
				}

				// Create venv/lib.py (should be excluded)
				venvDir := filepath.Join(dir, "venv", "lib")
				if err := os.MkdirAll(venvDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(venvDir, "lib.py"), []byte("import sys\n"), 0644); err != nil {
					return err
				}

				// Create tests/test.py
				testDir := filepath.Join(dir, "tests")
				if err := os.MkdirAll(testDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(testDir, "test.py"), []byte("def test():\n    pass\n"), 0644); err != nil {
					return err
				}

				return nil
			},
			wantFiles: 2, // src/main.go and tests/test.py (excludes .ralph and venv)
			wantLines: 5, // 3 from main.go, 2 from test.py
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Setup test files
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Run metrics collection
			metrics, err := CollectCodeMetrics(tmpDir)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("CollectCodeMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check files count
			if metrics.FilesGenerated != tt.wantFiles {
				t.Errorf("FilesGenerated = %d, want %d", metrics.FilesGenerated, tt.wantFiles)
			}

			// Check lines count
			if metrics.LinesGenerated != tt.wantLines {
				t.Errorf("LinesGenerated = %d, want %d", metrics.LinesGenerated, tt.wantLines)
			}
		})
	}
}

func TestCollectCodeMetrics_NonexistentDirectory(t *testing.T) {
	_, err := CollectCodeMetrics("/nonexistent/directory/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantLines int
	}{
		{
			name:      "empty file",
			content:   "",
			wantLines: 0,
		},
		{
			name:      "single line with newline",
			content:   "line 1\n",
			wantLines: 1,
		},
		{
			name:      "single line without newline",
			content:   "line 1",
			wantLines: 0,
		},
		{
			name:      "multiple lines",
			content:   "line 1\nline 2\nline 3\n",
			wantLines: 3,
		},
		{
			name:      "multiple lines without trailing newline",
			content:   "line 1\nline 2\nline 3",
			wantLines: 2,
		},
		{
			name:      "empty lines",
			content:   "\n\n\n",
			wantLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile := filepath.Join(t.TempDir(), "test.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Count lines
			lines, err := countLines(tmpFile)
			if err != nil {
				t.Fatalf("countLines() error = %v", err)
			}

			if lines != tt.wantLines {
				t.Errorf("countLines() = %d, want %d", lines, tt.wantLines)
			}
		})
	}
}

func TestCountLines_NonexistentFile(t *testing.T) {
	_, err := countLines("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}
