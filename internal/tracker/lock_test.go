package tracker

import "testing"

func TestAcquireLockBlocksSecondAcquire(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	runID := "test-run"

	release, err := w.AcquireLock(runID)
	if err != nil {
		t.Fatalf("AcquireLock error: %v", err)
	}
	defer func() { _ = release() }()

	if _, err := w.AcquireLock("other-run"); err == nil {
		t.Fatalf("expected second AcquireLock to fail")
	}

	if err := release(); err != nil {
		t.Fatalf("release error: %v", err)
	}

	if _, err := w.AcquireLock("third-run"); err != nil {
		t.Fatalf("expected AcquireLock after release to succeed, got: %v", err)
	}
}
