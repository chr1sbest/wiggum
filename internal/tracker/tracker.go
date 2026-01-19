package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Writer struct {
	Dir          string
	RunStatePath string
	LockPath     string
	MetricsPath  string
}

func NewWriter(dir string) *Writer {
	return &Writer{
		Dir:          dir,
		RunStatePath: filepath.Join(dir, "run_state.json"),
		LockPath:     filepath.Join(dir, ".ralph_lock"),
		MetricsPath:  filepath.Join(dir, "run_metrics.json"),
	}
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
