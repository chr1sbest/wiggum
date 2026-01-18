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

	// Try to create lock file exclusively (O_EXCL fails if file exists)
	l := Lock{PID: pid, StartedAt: time.Now(), RunID: runID}
	data, err := json.MarshalIndent(l, "", "    ")
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(w.LockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// Lock file exists - check if stale
			if b, readErr := os.ReadFile(w.LockPath); readErr == nil {
				var existing Lock
				if json.Unmarshal(b, &existing) == nil && existing.PID > 0 {
					if processAlive(existing.PID) {
						return nil, fmt.Errorf("%w by pid %d (run_id=%s)", ErrLockHeld, existing.PID, existing.RunID)
					}
					// Process is dead, remove stale lock and retry once
					if removeErr := os.Remove(w.LockPath); removeErr == nil {
						return w.AcquireLock(runID)
					}
				}
			}
			return nil, fmt.Errorf("%w (lock file exists)", ErrLockHeld)
		}
		return nil, err
	}

	// Write lock data with fsync
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(w.LockPath)
		return nil, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(w.LockPath)
		return nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(w.LockPath)
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
