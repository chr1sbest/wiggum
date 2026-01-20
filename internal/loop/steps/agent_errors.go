package steps

import (
	"errors"
	"strings"

	"github.com/chr1sbest/wiggum/internal/agent"
)

// ClaudeUsageError indicates Claude is unavailable due to quota/usage limits.
// This is surfaced to the CLI for clearer, actionable messaging.
type ClaudeUsageError struct {
	Details string
}

func (e *ClaudeUsageError) Error() string {
	if e == nil {
		return "claude usage error"
	}
	if strings.TrimSpace(e.Details) == "" {
		return "claude usage limit reached"
	}
	return "claude usage limit reached: " + strings.TrimSpace(e.Details)
}

func isClaudeUsageLimitText(s string) bool {
	msg := strings.ToLower(s)
	return strings.Contains(msg, "out of extra usage") ||
		strings.Contains(msg, "out of usage") ||
		strings.Contains(msg, "usage limit") ||
		strings.Contains(msg, "quota") ||
		strings.Contains(msg, "resets")
}

// AgentExitError indicates the agent has determined work is complete.
// This is a SUCCESS signal, not a failure - use it to exit the loop gracefully.
type AgentExitError struct {
	Reason agent.ExitReason
}

func (e *AgentExitError) Error() string {
	return "agent exit: " + string(e.Reason)
}

// IsAgentExitError checks if an error is an agent exit signal.
// Uses errors.As to unwrap any wrapped errors (e.g., PermanentError).
func IsAgentExitError(err error) (*AgentExitError, bool) {
	if err == nil {
		return nil, false
	}
	var exitErr *AgentExitError
	if errors.As(err, &exitErr) {
		return exitErr, true
	}
	return nil, false
}
