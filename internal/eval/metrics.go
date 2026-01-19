package eval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CodeMetrics represents code generation metrics
type CodeMetrics struct {
	FilesGenerated int
	LinesGenerated int
}

// CollectCodeMetrics counts files and lines of code in a project directory
// Counts files matching: *.go, *.py, *.md, *.yaml, *.yml, *.mod, *.sum
// Counts lines in: *.go and *.py files
// Excludes: .ralph/ and venv/ directories
func CollectCodeMetrics(projectDir string) (*CodeMetrics, error) {
	metrics := &CodeMetrics{}

	// File extensions to count
	fileExts := map[string]bool{
		".go":   true,
		".py":   true,
		".md":   true,
		".yaml": true,
		".yml":  true,
		".mod":  true,
		".sum":  true,
	}

	// Extensions to count lines
	lineExts := map[string]bool{
		".go": true,
		".py": true,
	}

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip .ralph and venv directories
			if info.Name() == ".ralph" || info.Name() == "venv" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path for checking exclusions
		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}

		// Skip files in .ralph or venv directories
		if strings.HasPrefix(relPath, ".ralph/") || strings.HasPrefix(relPath, "venv/") {
			return nil
		}

		// Get file extension
		ext := filepath.Ext(info.Name())

		// Count files
		if fileExts[ext] {
			metrics.FilesGenerated++
		}

		// Count lines
		if lineExts[ext] {
			lines, err := countLines(path)
			if err != nil {
				// Don't fail on individual file errors, just skip
				return nil
			}
			metrics.LinesGenerated += lines
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return metrics, nil
}

// countLines counts the number of lines in a file
func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buf := make([]byte, 32*1024)

	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		// Count newlines in buffer
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				count++
			}
		}

		if err == io.EOF {
			break
		}
	}

	return count, nil
}
