package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func runClaudeOnce(prompt string) (string, error) {
	return runClaudeOnceWithModel(prompt, "")
}

func runClaudeOnceWithModel(prompt string, model string) (string, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return "", fmt.Errorf("claude not found in PATH")
	}
	args := []string{"--dangerously-skip-permissions"}
	if strings.TrimSpace(model) != "" {
		args = append(args, "--model", strings.TrimSpace(model))
	}
	args = append(args, "-p", prompt)
	cmd := exec.Command("claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		// Only include stderr when Claude fails (stderr frequently contains non-fatal warnings)
		if stderr.Len() > 0 {
			out += "\n--- STDERR ---\n" + stderr.String()
		}
		return out, err
	}
	// On success, return stdout only to keep JSON outputs clean.
	return out, nil
}

func isClaudeRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "rate limit") || strings.Contains(s, "usage limit") || strings.Contains(s, "429")
}

func claudeActionableDetails(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
