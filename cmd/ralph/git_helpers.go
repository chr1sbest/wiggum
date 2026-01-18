package main

import (
	"bytes"
	"os/exec"
	"strings"
)

func gitInitAndInitialCommit(repoDir string) {
	if _, err := exec.LookPath("git"); err != nil {
		return
	}
	_ = runGit(repoDir, "init")
	_ = runGit(repoDir, "add", "-A")
	// best-effort commit; may fail if user has no git config
	_ = runGit(repoDir, "commit", "-m", "chore: initial scaffold")
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		// ignore common non-fatal cases
		s := strings.ToLower(out.String())
		if strings.Contains(s, "nothing to commit") {
			return nil
		}
		return err
	}
	return nil
}
