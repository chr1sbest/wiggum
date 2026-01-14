package steps

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitCommitStepCommitsWhenChangesExist(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(b))
		}
	}

	run("init")
	run("config", "user.email", "ralph@local")
	run("config", "user.name", "Ralph")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commitMsg := "chore: progress - llm summary\n\nWhy: explain what changed\nWhat: describe the diff\n"
	if err := os.WriteFile(filepath.Join(dir, "commit_message.txt"), []byte(commitMsg), 0644); err != nil {
		t.Fatal(err)
	}

	step := NewGitCommitStep()
	cfg := map[string]any{"repo_dir": dir, "enabled": true, "commit_message_file": "commit_message.txt", "message_template": "chore: progress - {{task}}"}
	raw, _ := json.Marshal(cfg)
	if err := step.Execute(context.Background(), raw); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "commit_message.txt")); err == nil {
		t.Fatalf("expected commit_message.txt to be removed after commit")
	}

	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v\n%s", err, string(b))
	}
	if string(b) == "1\n" {
		t.Fatalf("expected a new commit, still at 1")
	}

	msgCmd := exec.Command("git", "log", "-1", "--pretty=%B")
	msgCmd.Dir = dir
	mb, err := msgCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, string(mb))
	}
	if !strings.Contains(string(mb), "llm summary") {
		t.Fatalf("expected commit message to include llm summary, got:\n%s", string(mb))
	}
}
