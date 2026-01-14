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

	s := Status{
		Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LoopCount:      1,
		CurrentTask:    "Working",
		CompletedTasks: 0,
		PendingTasks:   2,
		Status:         "executing",
		ElapsedSeconds: 3,
	}
	if err := w.WriteStatus(s); err != nil {
		t.Fatalf("WriteStatus error: %v", err)
	}

	p := Progress{
		Status:         "executing",
		Indicator:      "-",
		ElapsedSeconds: 3,
		CurrentTask:    "Working",
		CompletedCount: 0,
		PendingCount:   2,
		Timestamp:      "2026-01-01 00:00:03",
	}
	if err := w.WriteProgress(p); err != nil {
		t.Fatalf("WriteProgress error: %v", err)
	}

	for _, name := range []string{"status.json", "progress.json"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s invalid json: %v", name, err)
		}
	}
}
