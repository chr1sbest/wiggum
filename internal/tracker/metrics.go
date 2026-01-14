package tracker

import (
	"encoding/json"
	"os"
	"time"
)

type RunMetrics struct {
	StartedAt         time.Time  `json:"started_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	TotalClaudeCalls  int        `json:"total_claude_calls"`
	InputTokens       int        `json:"input_tokens"`
	OutputTokens      int        `json:"output_tokens"`
	TotalTokens       int        `json:"total_tokens"`
	TotalCostUSD      float64    `json:"total_cost_usd,omitempty"`
	LastRunID         string     `json:"last_run_id,omitempty"`
	LastClaudeSession string     `json:"last_claude_session,omitempty"`
}

type UsageDelta struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

func (w *Writer) LoadMetrics() (*RunMetrics, error) {
	b, err := os.ReadFile(w.MetricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m RunMetrics
	if err := json.Unmarshal(b, &m); err != nil {
		// Corrupted metrics file: treat as no metrics.
		return nil, nil
	}
	return &m, nil
}

func (w *Writer) SaveMetrics(m *RunMetrics) error {
	return writeJSONAtomic(w.MetricsPath, m)
}

func (w *Writer) LoadOrInitMetrics(runID string) (*RunMetrics, error) {
	m, err := w.LoadMetrics()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if m == nil {
		m = &RunMetrics{StartedAt: now}
	}
	if m.StartedAt.IsZero() {
		m.StartedAt = now
	}
	m.UpdatedAt = now
	m.LastRunID = runID
	if err := w.SaveMetrics(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (w *Writer) AddUsage(runID string, delta UsageDelta) {
	m, err := w.LoadOrInitMetrics(runID)
	if err != nil || m == nil {
		return
	}
	m.TotalClaudeCalls++
	m.InputTokens += delta.InputTokens
	m.OutputTokens += delta.OutputTokens
	m.TotalTokens += delta.TotalTokens
	m.TotalCostUSD += delta.CostUSD
	m.UpdatedAt = time.Now()
	m.LastRunID = runID
	_ = w.SaveMetrics(m)
}

func (w *Writer) MarkComplete(runID string) {
	m, err := w.LoadOrInitMetrics(runID)
	if err != nil || m == nil {
		return
	}
	if m.CompletedAt == nil {
		now := time.Now()
		m.CompletedAt = &now
	}
	m.UpdatedAt = time.Now()
	m.LastRunID = runID
	_ = w.SaveMetrics(m)
}
