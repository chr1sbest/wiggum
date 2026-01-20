package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
	"github.com/chr1sbest/wiggum/internal/tracker"
)

// AgentStep executes Claude Code to work on tasks.
// This is the core orchestration - other concerns are split into:
//   - agent_config.go:  configuration
//   - agent_errors.go:  error types
//   - agent_claude.go:  Claude CLI execution
//   - agent_logging.go: output logging
//   - agent_status.go:  terminal status display
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
	if err := os.MkdirAll(".ralph", 0755); err != nil {
		log.Printf("warning: failed to create .ralph directory: %v", err)
	}
	trackerWriter := tracker.NewWriter(".ralph")

	// Parse config with defaults
	cfg := DefaultAgentConfig()
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return fmt.Errorf("failed to parse agent config: %w", err)
		}
	}

	// Check marker file (skip if already done)
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

	// Load PRD status for context building (exit check is handled by loop.go preflight)
	prdStatus, _ := agent.LoadPRDStatus(cfg.PrdFile)

	// Read prompt file
	promptContent, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
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

	// Start status refresh ticker for animation
	stopRefresh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopRefresh:
				return
			case <-ticker.C:
				s.refreshStatus(cfg.PrdFile)
			}
		}
	}()

	// Execute Claude
	output, err := s.executeClaudeCode(execCtx, cfg, string(promptContent), loopContext)
	close(stopRefresh)
	if err != nil {
		s.saveOutput(cfg.LogDir, output, s.loopCount)
		return fmt.Errorf("claude execution failed: %w", err)
	}

	// Save output
	s.saveOutput(cfg.LogDir, output, s.loopCount)

	// Write marker file if configured
	if cfg.MarkerFile != "" {
		_ = os.WriteFile(cfg.MarkerFile, []byte(time.Now().Format(time.RFC3339)), 0644)
	}

	// Track usage metrics
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

	// Check exit conditions after execution
	prdStatusAfter, _ := agent.LoadPRDStatus(cfg.PrdFile)
	planComplete := prdStatusAfter != nil && prdStatusAfter.IsComplete()
	completedAfter := 0
	if prdStatusAfter != nil {
		completedAfter = prdStatusAfter.CompletedTasks
	}

	// Mark loop complete for no-progress tracking (once per loop, not per check)
	s.exitDetector.MarkLoopComplete(completedAfter)

	if exitReason := s.exitDetector.Check(planComplete, completedAfter); exitReason != agent.ExitReasonNone {
		return &AgentExitError{Reason: exitReason}
	}

	return nil
}
