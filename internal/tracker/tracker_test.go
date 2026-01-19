package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	rs := RunState{
		RunID:     "test-run-123",
		PID:       12345,
		StartedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 3, 0, time.UTC),
		Status:    "running",
	}
	if err := w.WriteRunState(rs); err != nil {
		t.Fatalf("WriteRunState error: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "run_state.json"))
	if err != nil {
		t.Fatalf("read run_state.json: %v", err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("run_state.json invalid json: %v", err)
	}
}
