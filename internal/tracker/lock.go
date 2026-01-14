package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

type Lock struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	RunID     string    `json:"run_id"`
}

var ErrLockHeld = errors.New("ralph lock is held")

func (w *Writer) AcquireLock(runID string) (func() error, error) {
	pid := os.Getpid()

	// If lock exists, decide whether it's stale.
	if b, err := os.ReadFile(w.LockPath); err == nil {
		var existing Lock
		if json.Unmarshal(b, &existing) == nil && existing.PID > 0 {
			if processAlive(existing.PID) {
				return nil, fmt.Errorf("%w by pid %d (run_id=%s)", ErrLockHeld, existing.PID, existing.RunID)
			}
		}
		_ = os.Remove(w.LockPath)
	}

	l := Lock{PID: pid, StartedAt: time.Now(), RunID: runID}
	if err := writeJSONAtomic(w.LockPath, l); err != nil {
		return nil, err
	}

	release := func() error {
		return os.Remove(w.LockPath)
	}
	return release, nil
}

func processAlive(pid int) bool {
	// On unix, signal 0 checks existence/permission.
	err := syscall.Kill(pid, 0)
	return err == nil
}
