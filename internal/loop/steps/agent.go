package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
	"github.com/chr1sbest/wiggum/internal/tracker"
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

// AgentConfig holds configuration for the agent step
type AgentConfig struct {
	// PromptFile is the path to PROMPT.md (default: "PROMPT.md")
	PromptFile string `json:"prompt_file,omitempty"`
	// PrdFile is the path to prd.json (default: "prd.json")
	PrdFile string `json:"prd_file,omitempty"`
	// Model is the Claude model to use (optional)
	Model string `json:"model,omitempty"`
	// MarkerFile is an optional file path. If it exists, the agent step is skipped.
	// If set and the step runs successfully, the marker file will be created.
	MarkerFile string `json:"marker_file,omitempty"`
	// AllowedTools is a comma-separated list of tools Claude can use
	AllowedTools string `json:"allowed_tools,omitempty"`
	// Timeout is the max execution time (default: "15m")
	Timeout string `json:"timeout,omitempty"`
	// SessionFile is where to store session state
	SessionFile string `json:"session_file,omitempty"`
	// SessionExpiryHours is how long sessions last (default: 24)
	SessionExpiryHours int `json:"session_expiry_hours,omitempty"`
	// ClaudeBinary is the path to claude CLI (default: "claude")
	ClaudeBinary string `json:"claude_binary,omitempty"`
	// OutputFormat is json or text (default: "json")
	OutputFormat string `json:"output_format,omitempty"`
	// AppendSystemPrompt is extra context to add to the prompt
	AppendSystemPrompt string `json:"append_system_prompt,omitempty"`
	// LogDir is where to save Claude output logs
	LogDir string `json:"log_dir,omitempty"`
}

// DefaultAgentConfig returns sensible defaults
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		PromptFile:         "PROMPT.md",
		PrdFile:            "prd.json",
		Model:              "sonnet",
		MarkerFile:         "",
		AllowedTools:       "Write,Read,Edit,Glob,Grep,Bash,Task,TodoWrite,WebFetch,WebSearch",
		Timeout:            "15m",
		SessionFile:        ".ralph_session",
		SessionExpiryHours: 24,
		ClaudeBinary:       "claude",
		OutputFormat:       "json",
		LogDir:             "logs",
	}
}

// AgentStep executes Claude Code to work on tasks
type AgentStep struct {
	name         string
	session      *agent.SessionManager
	exitDetector *agent.ExitDetector
	loopCount    int
}

// NewAgentStep creates a new agent step
func NewAgentStep() *AgentStep {
	return &AgentStep{
		name:         "agent",
		exitDetector: agent.NewExitDetector(),
	}
}

func (s *AgentStep) Name() string { return s.name }
func (s *AgentStep) Type() string { return "agent" }

// Execute runs the Claude Code agent
func (s *AgentStep) Execute(ctx context.Context, rawConfig json.RawMessage) error {
	start := time.Now()
	if err := os.MkdirAll(".ralph", 0755); err != nil {
		log.Printf("warning: failed to create .ralph directory: %v", err)
	}
	trackerWriter := tracker.NewWriter(".ralph")
	writeTracking := func(status string, prdStatus *agent.PRDStatus, sessionState *agent.SessionState, currentStep string, err error) {
		currentTask := "Working"
		completed := 0
		pending := 0
		if prdStatus != nil {
			completed = prdStatus.CompletedTasks
			pending = prdStatus.IncompleteTasks
			if prdStatus.CurrentTask != "" {
				currentTask = prdStatus.CurrentTask
			}
		}

		loopCount := 0
		sessionID := ""
		if sessionState != nil {
			loopCount = sessionState.LoopCount
			sessionID = sessionState.SessionID
		}

		elapsed := int(time.Since(start).Seconds())
		if err := trackerWriter.WriteStatus(tracker.Status{
			Timestamp:      time.Now(),
			LoopCount:      loopCount,
			CurrentTask:    currentTask,
			CompletedTasks: completed,
			PendingTasks:   pending,
			Status:         status,
			ElapsedSeconds: elapsed,
			SessionID:      sessionID,
			CurrentStep:    currentStep,
		}); err != nil {
			log.Printf("warning: failed to write status: %v", err)
		}

		if err := trackerWriter.WriteProgress(tracker.Progress{
			Status:         status,
			Indicator:      "-",
			ElapsedSeconds: elapsed,
			CurrentTask:    currentTask,
			CompletedCount: completed,
			PendingCount:   pending,
			Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		}); err != nil {
			log.Printf("warning: failed to write progress: %v", err)
		}
	}

	// Parse config with defaults
	cfg := DefaultAgentConfig()
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return fmt.Errorf("failed to parse agent config: %w", err)
		}
	}

	if cfg.MarkerFile != "" {
		if _, err := os.Stat(cfg.MarkerFile); err == nil {
			return nil
		}
	}

	// Initialize session manager
	if s.session == nil {
		s.session = agent.NewSessionManager(
			cfg.SessionFile,
			cfg.SessionFile+"_history",
			cfg.SessionExpiryHours,
		)
	}

	// Get or create session
	sessionState, isNew, err := s.session.GetOrCreate()
	if err != nil {
		return fmt.Errorf("failed to manage session: %w", err)
	}
	s.loopCount = sessionState.LoopCount

	if isNew {
		s.exitDetector.Reset()
	}

	prdStatus, err := agent.LoadPRDStatus(cfg.PrdFile)
	if err != nil {
		return fmt.Errorf("failed to load prd status: %w", err)
	}

	writeTracking("executing", prdStatus, sessionState, "run-claude", nil)

	// Check if we should exit before starting
	completedBefore := 0
	if prdStatus != nil {
		completedBefore = prdStatus.CompletedTasks
	}
	if exitReason := s.exitDetector.Check(prdStatus != nil && prdStatus.IsComplete(), completedBefore); exitReason != agent.ExitReasonNone {
		writeTracking("complete", prdStatus, sessionState, "run-claude", nil)
		return &AgentExitError{Reason: exitReason}
	}

	// Exit early if no actionable tasks (all done or failed, none todo)
	if prdStatus != nil && !prdStatus.HasActionableTasks() && prdStatus.TotalTasks > 0 {
		writeTracking("complete", prdStatus, sessionState, "run-claude", nil)
		return &AgentExitError{Reason: agent.ExitReasonNoActionableTasks}
	}

	// Read prompt file
	promptContent, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		writeTracking("error", prdStatus, sessionState, "run-claude", err)
		return fmt.Errorf("failed to read prompt file %s: %w", cfg.PromptFile, err)
	}

	// Build loop context
	loopContext := s.buildLoopContext(cfg, prdStatus, sessionState)

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 15 * time.Minute
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build and execute Claude command
	output, err := s.executeClaudeCode(execCtx, cfg, string(promptContent), loopContext)
	if err != nil {
		// Save output even on failure
		s.saveOutput(cfg.LogDir, output, s.loopCount)
		writeTracking("error", prdStatus, sessionState, "run-claude", err)
		return fmt.Errorf("claude execution failed: %w", err)
	}

	// Save output
	s.saveOutput(cfg.LogDir, output, s.loopCount)

	if cfg.MarkerFile != "" {
		_ = os.WriteFile(cfg.MarkerFile, []byte(time.Now().Format(time.RFC3339)), 0644)
	}

	// Best-effort: parse token/cost usage from Claude JSON output and persist run metrics.
	if delta, ok := tracker.ParseClaudeUsageFromOutput(output); ok {
		runID := ""
		if rs, err := trackerWriter.LoadRunState(); err == nil && rs != nil {
			runID = rs.RunID
		}
		if runID == "" {
			runID = sessionState.SessionID
		}
		trackerWriter.AddUsage(runID, tracker.UsageDelta{
			InputTokens:  delta.InputTokens,
			OutputTokens: delta.OutputTokens,
			TotalTokens:  delta.TotalTokens,
			CostUSD:      delta.CostUSD,
		})
	}

	// Check exit conditions
	// Re-load prd status after the agent may have updated it.
	prdStatusAfter, _ := agent.LoadPRDStatus(cfg.PrdFile)
	planComplete := prdStatusAfter != nil && prdStatusAfter.IsComplete()
	completedAfter := 0
	if prdStatusAfter != nil {
		completedAfter = prdStatusAfter.CompletedTasks
	}
	if exitReason := s.exitDetector.Check(planComplete, completedAfter); exitReason != agent.ExitReasonNone {
		writeTracking("complete", prdStatusAfter, sessionState, "run-claude", nil)
		return &AgentExitError{Reason: exitReason}
	}

	writeTracking("executing", prdStatusAfter, sessionState, "run-claude", nil)

	return nil
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
		if prdStatus.CurrentTask != "" {
			// Truncate long task descriptions
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

// saveOutput saves Claude's output to structured log files
func (s *AgentStep) saveOutput(logDir, output string, loopCount int) {
	if logDir == "" {
		return
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("warning: failed to create log directory %s: %v", logDir, err)
		return
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		// Valid JSON - save to loop_N.json
		jsonPath := filepath.Join(logDir, fmt.Sprintf("loop_%d.json", loopCount))
		if err := os.WriteFile(jsonPath, []byte(output), 0644); err != nil {
			log.Printf("warning: failed to write JSON log %s: %v", jsonPath, err)
		}

		// Extract 'result' field and save to loop_N.md
		if result, ok := jsonData["result"].(string); ok && result != "" {
			mdPath := filepath.Join(logDir, fmt.Sprintf("loop_%d.md", loopCount))
			if err := os.WriteFile(mdPath, []byte(result), 0644); err != nil {
				log.Printf("warning: failed to write markdown log %s: %v", mdPath, err)
			}
		}
	} else {
		// Not valid JSON - fall back to .log file
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("claude_output_%s_loop%d.log", timestamp, loopCount)
		path := filepath.Join(logDir, filename)
		if err := os.WriteFile(path, []byte(output), 0644); err != nil {
			log.Printf("warning: failed to write fallback log %s: %v", path, err)
		}
	}
}

// AgentExitError indicates the agent has determined work is complete
type AgentExitError struct {
	Reason agent.ExitReason
}

func (e *AgentExitError) Error() string {
	return fmt.Sprintf("agent exit: %s", e.Reason)
}

// IsAgentExitError checks if an error is an agent exit signal
func IsAgentExitError(err error) (*AgentExitError, bool) {
	if err == nil {
		return nil, false
	}
	exitErr, ok := err.(*AgentExitError)
	return exitErr, ok
}
