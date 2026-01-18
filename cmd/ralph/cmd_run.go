package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
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
	configFile := fs.String("config", ".ralph/configs/default.json", "Path to config file")
	logFile := fs.String("log", ".ralph/ralph.log", "Path to log file")
	model := fs.String("model", "", "Claude model to use (overrides agent step config)")
	once := fs.Bool("once", false, "Run loop only once")
	fs.Parse(args)

	if err := validateRunPreflight(*configFile); err != nil {
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
		fmt.Println("All tasks are complete. I don't know what that means.")
		return 0
	}

	fileLogger, err := logger.NewFileLogger(*logFile, logger.LevelDebug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create file logger: %v\n", err)
		return 1
	}
	defer fileLogger.Close()

	registry := loop.NewStepRegistry()
	registry.Register("command", func() loop.Step { return steps.NewCommandStep() })
	registry.Register("noop", func() loop.Step { return steps.NewNoopStep() })
	registry.Register("readme-check", func() loop.Step { return steps.NewReadmeCheckStep() })
	registry.Register("agent", func() loop.Step { return steps.NewAgentStep() })
	registry.Register("git-commit", func() loop.Step { return steps.NewGitCommitStep() })

	loader := config.NewLoader(".ralph/configs")
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

	mainLoop := loop.NewLoop(cfg, registry, fileLogger)
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
		return runOnce(ctx, mainLoop, trk, runID)
	}
	return runContinuous(ctx, mainLoop, trk, runID)
}

func runOnce(ctx context.Context, mainLoop *loop.Loop, trk *tracker.Writer, runID string) int {
	if err := mainLoop.RunOnce(ctx); err != nil && err != context.Canceled {
		if _, ok := steps.IsAgentExitError(err); ok {
			trk.MarkComplete(runID)
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

func runContinuous(ctx context.Context, mainLoop *loop.Loop, trk *tracker.Writer, runID string) int {
	if err := mainLoop.Run(ctx); err != nil && err != context.Canceled {
		if _, ok := steps.IsAgentExitError(err); ok {
			trk.MarkComplete(runID)
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
		".ralph/configs",
	}
	for _, p := range required {
		st, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("missing required project file: %s\n\nRun `ralph run` from your project root (the folder created by `ralph init`) containing .ralph/prd.json, .ralph/requirements.md, .ralph/configs/, .ralph/prompts/SETUP_PROMPT.md, .ralph/prompts/LOOP_PROMPT.md", p)
		}
		if p == ".ralph/configs" && !st.IsDir() {
			return fmt.Errorf("expected .ralph/configs/ to be a directory\n\nRun `ralph run` from your project root (the folder created by `ralph init`) containing .ralph/prd.json, .ralph/requirements.md, .ralph/configs/, .ralph/prompts/SETUP_PROMPT.md, .ralph/prompts/LOOP_PROMPT.md")
		}
	}

	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	return nil
}
