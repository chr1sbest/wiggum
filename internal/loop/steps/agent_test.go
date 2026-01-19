package steps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveOutput_JSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	step := NewAgentStep()

	// Sample JSON output (simplified from actual Claude output)
	jsonOutput := `{"type":"result","result":"This is the result text","session_id":"test-123"}`

	// Save output
	step.saveOutput(logDir, jsonOutput, 1)

	// Check that loop_1.json was created
	jsonPath := filepath.Join(logDir, "loop_1.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("Expected loop_1.json to exist")
	}

	// Check that loop_1.md was created
	mdPath := filepath.Join(logDir, "loop_1.md")
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Errorf("Expected loop_1.md to exist")
	}

	// Verify JSON content
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}
	if string(jsonContent) != jsonOutput {
		t.Errorf("JSON content mismatch. Expected %q, got %q", jsonOutput, string(jsonContent))
	}

	// Verify markdown content contains just the result
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}
	expected := "This is the result text"
	if string(mdContent) != expected {
		t.Errorf("Markdown content mismatch. Expected %q, got %q", expected, string(mdContent))
	}
}

func TestSaveOutput_NonJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	step := NewAgentStep()

	// Non-JSON output
	plainOutput := "This is plain text output, not JSON"

	// Save output
	step.saveOutput(logDir, plainOutput, 2)

	// Check that a .log file was created (with timestamp)
	files, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	foundLog := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".log" {
			foundLog = true
			// Verify content
			content, err := os.ReadFile(filepath.Join(logDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}
			if string(content) != plainOutput {
				t.Errorf("Log content mismatch. Expected %q, got %q", plainOutput, string(content))
			}
		}
	}

	if !foundLog {
		t.Errorf("Expected a .log file to be created for non-JSON output")
	}

	// Verify no JSON or markdown files were created
	jsonPath := filepath.Join(logDir, "loop_2.json")
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Errorf("Did not expect loop_2.json to exist for non-JSON output")
	}

	mdPath := filepath.Join(logDir, "loop_2.md")
	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Errorf("Did not expect loop_2.md to exist for non-JSON output")
	}
}

func TestSaveOutput_JSONWithoutResult(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	step := NewAgentStep()

	// JSON without result field
	jsonOutput := `{"type":"error","message":"Something went wrong"}`

	// Save output
	step.saveOutput(logDir, jsonOutput, 3)

	// Check that loop_3.json was created
	jsonPath := filepath.Join(logDir, "loop_3.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("Expected loop_3.json to exist")
	}

	// Check that loop_3.md was NOT created (no result field)
	mdPath := filepath.Join(logDir, "loop_3.md")
	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Errorf("Did not expect loop_3.md to exist when JSON has no result field")
	}
}

func TestSaveOutput_EmptyLogDir(t *testing.T) {
	step := NewAgentStep()

	// Should not panic or error when logDir is empty
	step.saveOutput("", "some output", 1)

	// No files should be created (test passes if no panic)
}
