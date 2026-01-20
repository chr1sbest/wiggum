package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
	"github.com/chr1sbest/wiggum/internal/banner"
	"github.com/chr1sbest/wiggum/internal/config"
	"github.com/chr1sbest/wiggum/internal/logger"
	"github.com/chr1sbest/wiggum/internal/loop"
	"github.com/chr1sbest/wiggum/internal/loop/steps"
	"github.com/chr1sbest/wiggum/internal/tracker"
)

func runCmd(args []string) int {
	for i := range args {
		if args[i] == "--once" {
			args[i] = "-once"
		}
	}
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("config", ".ralph/config.json", "Path to config file")
	model := fs.String("model", "", "Claude model to use (overrides agent step config)")
	once := fs.Bool("once", false, "Run loop only once")
	fs.Parse(args)

	if err := validateRunPreflight(*configFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := validateClaudePreflight(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	hasTasks, allComplete, err := agent.CheckPRDTasks(".ralph/prd.json")
	if err != nil {
		if errors.Is(err, agent.ErrPRDNoTasks) {
			fmt.Fprintln(os.Stderr, ".ralph/prd.json contains no tasks. Add tasks (e.g. via `ralph add`) and re-run.")
			return 1
		}
		fmt.Fprintf(os.Stderr, "Failed to read .ralph/prd.json: %v\n", err)
		return 1
	}
	if !hasTasks {
		fmt.Fprintln(os.Stderr, ".ralph/prd.json contains no tasks. Add tasks (e.g. via `ralph add`) and re-run.")
		return 1
	}
	if allComplete {
		fmt.Println("All tasks are complete!")
		fmt.Println("\nTo add more work:")
		fmt.Println("  ralph add work.md")
		fmt.Println("  ralph add \"your task description\"")
		return 0
	}

	loopLogger := logger.NewNoopLogger()

	registry := loop.NewStepRegistry()
	registry.Register("command", func() loop.Step { return steps.NewCommandStep() })
	registry.Register("noop", func() loop.Step { return steps.NewNoopStep() })
	registry.Register("readme-check", func() loop.Step { return steps.NewReadmeCheckStep() })
	registry.Register("agent", func() loop.Step { return steps.NewAgentStep() })
	registry.Register("git-commit", func() loop.Step { return steps.NewGitCommitStep() })

	loader := config.NewLoader(".ralph")
	cfg, err := loader.LoadAndValidate(*configFile, registry.RegisteredTypes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if strings.TrimSpace(*model) != "" {
		for i := range cfg.Steps {
			if cfg.Steps[i].Type != "agent" {
				continue
			}
			var stepCfgMap map[string]any
			if len(cfg.Steps[i].Config) > 0 {
				if err := json.Unmarshal(cfg.Steps[i].Config, &stepCfgMap); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to parse agent step config for %s: %v\n", cfg.Steps[i].Name, err)
					return 1
				}
			}
			if stepCfgMap == nil {
				stepCfgMap = map[string]any{}
			}
			stepCfgMap["model"] = strings.TrimSpace(*model)
			b, err := json.Marshal(stepCfgMap)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to serialize agent step config for %s: %v\n", cfg.Steps[i].Name, err)
				return 1
			}
			cfg.Steps[i].Config = b
		}
	}

	if resetCount, err := agent.ResetFailedTasks(".ralph/prd.json"); err == nil && resetCount > 0 {
		fmt.Printf("Reset %d failed task(s) to retry\n", resetCount)
	}

	b := banner.New()
	b.Print(cfg)

	mainLoop := loop.NewLoop(cfg, registry, loopLogger)
	mainLoop.SetPRDPath(".ralph/prd.json")

	trackerDir := ".ralph"
	_ = os.MkdirAll(trackerDir, 0755)
	trk := tracker.NewWriter(trackerDir)
	runID := tracker.NewRunID()
	releaseLock, err := trk.AcquireLock(runID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = releaseLock() }()
	mainLoop.EnableRunTracking(runID, trackerDir)
	_, _ = trk.LoadOrInitMetrics(runID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	if *once {
		return runOnce(ctx, mainLoop, trk, runID, cfg, *model)
	}
	return runContinuous(ctx, mainLoop, trk, runID, cfg, *model)
}

func runOnce(ctx context.Context, mainLoop *loop.Loop, trk *tracker.Writer, runID string, cfg *config.Config, modelOverride string) int {
	if err := mainLoop.RunOnce(ctx); err != nil && err != context.Canceled {
		if _, ok := steps.IsAgentExitError(err); ok {
			trk.MarkComplete(runID)
			_ = writeResultJSON(trk, cfg, modelOverride)
			printRunMetrics(trk)
			return 0
		}
		var usageErr *steps.ClaudeUsageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, "Claude usage limit reached.")
			fmt.Fprintln(os.Stderr, "Wait for your quota to reset, then re-run: ralph run")
			fmt.Fprintf(os.Stderr, "\nDetails:\n%v\n", usageErr)
			return 2
		}
		fmt.Fprintf(os.Stderr, "Loop failed: %v\n", err)
		return 1
	}
	return 0
}

func runContinuous(ctx context.Context, mainLoop *loop.Loop, trk *tracker.Writer, runID string, cfg *config.Config, modelOverride string) int {
	if err := mainLoop.Run(ctx); err != nil && err != context.Canceled {
		if _, ok := steps.IsAgentExitError(err); ok {
			trk.MarkComplete(runID)
			_ = writeResultJSON(trk, cfg, modelOverride)
			printRunMetrics(trk)
			return 0
		}
		var usageErr *steps.ClaudeUsageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, "Claude usage limit reached.")
			fmt.Fprintln(os.Stderr, "Wait for your quota to reset, then re-run: ralph run")
			fmt.Fprintf(os.Stderr, "\nDetails:\n%v\n", usageErr)
			return 2
		}
		fmt.Fprintf(os.Stderr, "Loop failed: %v\n", err)
		return 1
	}
	return 0
}

func printRunMetrics(trk *tracker.Writer) {
	if m, _ := trk.LoadMetrics(); m != nil {
		end := time.Now()
		if m.CompletedAt != nil {
			end = *m.CompletedAt
		}
		elapsed := end.Sub(m.StartedAt)
		fmt.Printf("\nRun complete in %s\n", elapsed.Round(time.Second))
		fmt.Printf("Total Claude calls: %d\n", m.TotalClaudeCalls)
		fmt.Printf("Total tokens: %d (in: %d, out: %d)\n", m.TotalTokens, m.InputTokens, m.OutputTokens)
		if m.TotalCostUSD > 0 {
			fmt.Printf("Estimated cost: $%.2f\n", m.TotalCostUSD)
		}
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func printRunInstructions() {
	projectRoot := mustGetwd()
	projectName := filepath.Base(projectRoot)
	codeDir := filepath.Join(projectRoot, projectName)

	fmt.Println("\nRun instructions:")
	if st, err := os.Stat(codeDir); err == nil && st.IsDir() {
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  See README.md for how to run/test the app")
		return
	}
	fmt.Println("  See README.md for how to run/test the app")
}

func validateRunPreflight(configFile string) error {
	required := []string{
		".ralph/prd.json",
		".ralph/requirements.md",
		".ralph/prompts/SETUP_PROMPT.md",
		".ralph/prompts/LOOP_PROMPT.md",
	}
	for _, p := range required {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("missing required project file: %s\n\nRun `ralph run` from your project root (the folder created by `ralph init`) containing .ralph/prd.json, .ralph/requirements.md, .ralph/config.json, .ralph/prompts/", p)
		}
	}

	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	return nil
}

func validateClaudePreflight() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("Claude Code is required but was not found in PATH.\n\nFix:\n  - Install Claude Code: https://code.claude.com/docs/en/setup\n  - Quick install: curl -fsSL https://claude.ai/install.sh | bash\n  - Ensure the `claude` binary is on your PATH\n  - Confirm it works: claude --help")
	}

	// Ensure the CLI is runnable (and not immediately failing due to a broken install).
	cmd := exec.Command("claude", "--version")
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("Claude Code appears to be installed, but is not working:\n%s\n\nFix:\n  - Run `claude` once interactively to complete setup/auth\n  - Then retry `ralph run`", msg)
		}
		return fmt.Errorf("Claude Code appears to be installed, but is not working (%v).\n\nFix:\n  - Run `claude` once interactively to complete setup/auth\n  - Then retry `ralph run`", err)
	}

	return nil
}

type runResult struct {
	Version string           `json:"version"`
	Model   string           `json:"model"`
	Metrics runResultMetrics `json:"metrics"`
}

type runResultMetrics struct {
	StartedAt        time.Time  `json:"started_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	TotalClaudeCalls int        `json:"total_claude_calls"`
	InputTokens      int        `json:"input_tokens"`
	OutputTokens     int        `json:"output_tokens"`
	TotalTokens      int        `json:"total_tokens"`
	TotalCostUSD     float64    `json:"total_cost_usd,omitempty"`
	LastRunID        string     `json:"last_run_id,omitempty"`
	ElapsedSec       int64      `json:"elapsed_sec,omitempty"`
}

func writeResultJSON(trk *tracker.Writer, cfg *config.Config, modelOverride string) error {
	m, err := trk.LoadMetrics()
	if err != nil {
		return err
	}
	if m == nil {
		return nil
	}

	model := strings.TrimSpace(modelOverride)
	if model == "" {
		model = strings.TrimSpace(findClaudeModelFromConfig(cfg))
	}
	if model == "" {
		model = "sonnet"
	}

	end := time.Now()
	if m.CompletedAt != nil {
		end = *m.CompletedAt
	}

	elapsedSec := int64(end.Sub(m.StartedAt).Round(time.Second) / time.Second)

	res := runResult{
		Version: versionLine(),
		Model:   model,
		Metrics: runResultMetrics{
			StartedAt:        m.StartedAt,
			UpdatedAt:        m.UpdatedAt,
			CompletedAt:      m.CompletedAt,
			TotalClaudeCalls: m.TotalClaudeCalls,
			InputTokens:      m.InputTokens,
			OutputTokens:     m.OutputTokens,
			TotalTokens:      m.TotalTokens,
			TotalCostUSD:     m.TotalCostUSD,
			LastRunID:        m.LastRunID,
			ElapsedSec:       elapsedSec,
		},
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return err
	}
	return writeFileAtomic(".ralph/aggregate.json", data, 0644)
}

func findClaudeModelFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	for _, s := range cfg.Steps {
		if s.Type != "agent" {
			continue
		}
		if len(s.Config) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(s.Config, &m); err != nil {
			continue
		}
		if v, ok := m["model"]; ok {
			if ms, ok := v.(string); ok {
				ms = strings.TrimSpace(ms)
				if ms != "" {
					return ms
				}
			}
		}
	}
	return ""
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
