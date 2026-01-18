package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Status struct {
	Timestamp         time.Time `json:"timestamp"`
	LoopCount         int       `json:"loop_count"`
	CurrentTask       string    `json:"current_task"`
	CompletedTasks    int       `json:"completed_tasks"`
	PendingTasks      int       `json:"pending_tasks"`
	Status            string    `json:"status"`
	ElapsedSeconds    int       `json:"elapsed_seconds"`
	SessionID         string    `json:"session_id,omitempty"`
	FilesModified     int       `json:"files_modified,omitempty"`
	TestsStatus       string    `json:"tests_status,omitempty"`
	PreviousStep      string    `json:"previous_step,omitempty"`
	CurrentStep       string    `json:"current_step,omitempty"`
	CallsMadeThisHour int       `json:"calls_made_this_hour,omitempty"`
	MaxCallsPerHour   int       `json:"max_calls_per_hour,omitempty"`
	NextReset         string    `json:"next_reset,omitempty"`
}

type Progress struct {
	Status         string `json:"status"`
	Indicator      string `json:"indicator"`
	ElapsedSeconds int    `json:"elapsed_seconds"`
	CurrentTask    string `json:"current_task"`
	CompletedCount int    `json:"completed_count"`
	PendingCount   int    `json:"pending_count"`
	Timestamp      string `json:"timestamp"`
}

type Writer struct {
	Dir          string
	StatusPath   string
	ProgressPath string
	RunStatePath string
	LockPath     string
	MetricsPath  string
}

func NewWriter(dir string) *Writer {
	return &Writer{
		Dir:          dir,
		StatusPath:   filepath.Join(dir, "status.json"),
		ProgressPath: filepath.Join(dir, "progress.json"),
		RunStatePath: filepath.Join(dir, "run_state.json"),
		LockPath:     filepath.Join(dir, ".ralph_lock"),
		MetricsPath:  filepath.Join(dir, "run_metrics.json"),
	}
}

func (w *Writer) WriteStatus(s Status) error {
	return writeJSONAtomic(w.StatusPath, s)
}

func (w *Writer) WriteProgress(p Progress) error {
	return writeJSONAtomic(w.ProgressPath, p)
}

func (w *Writer) WriteRunState(s RunState) error {
	return writeJSONAtomic(w.RunStatePath, s)
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
