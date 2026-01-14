package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteRunStateWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	rs := RunState{
		RunID:      "abc",
		PID:        123,
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		LoopNumber: 7,
		Status:     "running",
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
		t.Fatalf("invalid json: %v", err)
	}
}
