package steps

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chr1sbest/wiggum/internal/agent"
)

// executeClaudeCode runs the claude CLI
func (s *AgentStep) executeClaudeCode(ctx context.Context, cfg AgentConfig, prompt, loopContext string) (string, error) {
	args := []string{}

	// Model
	if strings.TrimSpace(cfg.Model) != "" {
		args = append(args, "--model", strings.TrimSpace(cfg.Model))
	}

	// Output format
	if cfg.OutputFormat == "json" {
		args = append(args, "--output-format", "json")
	}

	// Allowed tools
	if cfg.AllowedTools != "" {
		args = append(args, "--allowedTools")
		tools := strings.Split(cfg.AllowedTools, ",")
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				args = append(args, tool)
			}
		}
	}

	// Skip permission prompts for autonomous operation
	args = append(args, "--dangerously-skip-permissions")

	// Add loop context
	if loopContext != "" {
		args = append(args, "--append-system-prompt", loopContext)
	}

	// Add the prompt
	args = append(args, "-p", prompt)

	// Create command
	cmd := exec.CommandContext(ctx, cfg.ClaudeBinary, args...)
	cmd.Dir, _ = os.Getwd()

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n--- STDERR ---\n" + stderr.String()
	}

	if err != nil {
		// Preserve combined output in error classification.
		combinedText := strings.TrimSpace(output)
		if isClaudeUsageLimitText(combinedText) {
			return output, &ClaudeUsageError{Details: combinedText}
		}

		// Check for specific error types
		if ctx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("timeout after %s", cfg.Timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Some Claude CLI errors show up on stdout; include combined output.
			if combinedText == "" {
				combinedText = strings.TrimSpace(stderr.String())
			}
			return output, fmt.Errorf("exit code %d: %s", exitErr.ExitCode(), combinedText)
		}
		return output, err
	}

	return output, nil
}

// buildLoopContext creates context string for Claude
func (s *AgentStep) buildLoopContext(cfg AgentConfig, prdStatus *agent.PRDStatus, session *agent.SessionState) string {
	var parts []string

	// Loop number
	parts = append(parts, fmt.Sprintf("Loop #%d.", s.loopCount))

	// Task progress
	if prdStatus != nil && prdStatus.TotalTasks > 0 {
		parts = append(parts, fmt.Sprintf("Tasks: %s (%d remaining).",
			prdStatus.Progress(), prdStatus.IncompleteTasks))

		// Show current task(s) - prefer multi-task view
		if len(prdStatus.CurrentTasks) > 0 {
			// Truncate long task descriptions
			firstTask := prdStatus.CurrentTasks[0]
			if len(firstTask) > 100 {
				firstTask = firstTask[:100] + "..."
			}
			parts = append(parts, fmt.Sprintf("Current: %s", firstTask))
		} else if prdStatus.CurrentTask != "" {
			// Backward compatibility: single task
			next := prdStatus.CurrentTask
			if len(next) > 100 {
				next = next[:100] + "..."
			}
			parts = append(parts, fmt.Sprintf("Current: %s", next))
		}
	}

	// Include learnings from previous sessions
	learningsPath := ".ralph/learnings.md"
	if learnings, err := os.ReadFile(learningsPath); err == nil && len(learnings) > 0 {
		content := strings.TrimSpace(string(learnings))
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		if content != "" {
			parts = append(parts, fmt.Sprintf("\n\nPrevious learnings:\n%s", content))
		}
	}

	// Append custom context
	if cfg.AppendSystemPrompt != "" {
		parts = append(parts, cfg.AppendSystemPrompt)
	}

	return strings.Join(parts, " ")
}
