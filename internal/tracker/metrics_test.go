package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMetricsAccumulateAndPersist(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	w.AddUsage("run1", UsageDelta{InputTokens: 1, OutputTokens: 2, TotalTokens: 3, CostUSD: 0.01})
	w.AddUsage("run1", UsageDelta{InputTokens: 10, OutputTokens: 20, TotalTokens: 30, CostUSD: 0.02})

	b, err := os.ReadFile(filepath.Join(dir, "run_metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m RunMetrics
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m.TotalClaudeCalls != 2 {
		t.Fatalf("expected 2 calls, got %d", m.TotalClaudeCalls)
	}
	if m.TotalTokens != 33 {
		t.Fatalf("expected 33 tokens, got %d", m.TotalTokens)
	}
	if m.InputTokens != 11 || m.OutputTokens != 22 {
		t.Fatalf("unexpected split: in=%d out=%d", m.InputTokens, m.OutputTokens)
	}
}
