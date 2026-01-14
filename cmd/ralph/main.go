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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chris/go_ralph/internal/banner"
	"github.com/chris/go_ralph/internal/config"
	"github.com/chris/go_ralph/internal/logger"
	"github.com/chris/go_ralph/internal/loop"
	"github.com/chris/go_ralph/internal/loop/steps"
	"github.com/chris/go_ralph/internal/tracker"
)

const version = "1.0.0"

type prdFile struct {
	Version jsonInt   `json:"version"`
	Tasks   []prdTask `json:"tasks"`
}

type jsonInt int

func (i *jsonInt) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*i = 0
		return nil
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		unq, err := strconv.Unquote(s)
		if err != nil {
			return err
		}
		s = strings.TrimSpace(unq)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = jsonInt(n)
	return nil
}

type prdTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Details  string `json:"details,omitempty"`
	Priority string `json:"priority,omitempty"`
	Status   string `json:"status,omitempty"`
	Tests    string `json:"tests,omitempty"`
}

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		if len(lines) > 0 {
			last := strings.TrimSpace(lines[len(lines)-1])
			if last == "```" {
				lines = lines[:len(lines)-1]
			}
		}
		s = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return s
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printUsage()
		os.Exit(0)
	}
	if os.Args[1] == "--version" {
		fmt.Printf("ralph version %s\n", version)
		os.Exit(0)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(runCmd(os.Args[2:]))
	case "new-project":
		newProjectCmd(os.Args[2:])
	case "new-work":
		newWorkCmd(os.Args[2:])
	case "init-config":
		initConfigCmd(os.Args[2:])
	case "version":
		fmt.Printf("ralph version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
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
	// Prefer the nested code directory (default scaffold layout).
	if st, err := os.Stat(codeDir); err == nil && st.IsDir() {
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  See README.md for how to run/test the app")
		return
	}
	// Fallback: project root contains the app.
	fmt.Println("  See README.md for how to run/test the app")
}

func validateRunPreflight(configFile string) error {
	// We intentionally validate required files in the *current working directory*.
	// This prevents confusing behavior when `ralph run` is invoked from the wrong folder.
	required := []string{
		"prd.json",
		"requirements.md",
		"SETUP_PROMPT.md",
		"LOOP_PROMPT.md",
		"configs",
	}
	for _, p := range required {
		st, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("missing required project file: %s\n\nRun `ralph run` from your project root (the folder created by `ralph new-project`) containing prd.json, requirements.md, configs/, SETUP_PROMPT.md, LOOP_PROMPT.md", p)
		}
		if p == "configs" && !st.IsDir() {
			return fmt.Errorf("expected configs/ to be a directory\n\nRun `ralph run` from your project root (the folder created by `ralph new-project`) containing prd.json, requirements.md, configs/, SETUP_PROMPT.md, LOOP_PROMPT.md")
		}
	}

	// Also ensure the provided config file exists.
	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	return nil
}

func printUsage() {
	fmt.Println(`ralph - Modular automation loop

Usage:
  ralph <command> [flags]

Commands:
  run          Run the main loop
  new-project  Initialize a new Ralph project
  new-work     Add new work context to existing project
  init-config  Generate a default configuration file
  version      Show version information
  help         Show this help message

Run 'ralph <command> -h' for more information on a command.`)
}

func runCmd(args []string) int {
	// Accept `--once` as an alias for Go's standard `-once` flag.
	for i := range args {
		if args[i] == "--once" {
			args[i] = "-once"
		}
	}
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("config", "configs/default.json", "Path to config file")
	logFile := fs.String("log", "ralph.log", "Path to log file")
	once := fs.Bool("once", false, "Run loop only once")
	fs.Parse(args)

	if err := validateRunPreflight(*configFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Set up file logger (stdout handled by status display)
	fileLogger, err := logger.NewFileLogger(*logFile, logger.LevelDebug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create file logger: %v\n", err)
		return 1
	}
	defer fileLogger.Close()

	// Set up step registry
	registry := loop.NewStepRegistry()
	registry.Register("command", func() loop.Step { return steps.NewCommandStep() })
	registry.Register("noop", func() loop.Step { return steps.NewNoopStep() })
	registry.Register("readme-check", func() loop.Step { return steps.NewReadmeCheckStep() })
	registry.Register("agent", func() loop.Step { return steps.NewAgentStep() })
	registry.Register("git-commit", func() loop.Step { return steps.NewGitCommitStep() })

	// Load and validate config
	loader := config.NewLoader("configs")
	cfg, err := loader.LoadAndValidate(*configFile, registry.RegisteredTypes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Print startup banner
	b := banner.New()
	b.Print(cfg)

	// Create loop
	mainLoop := loop.NewLoop(cfg, registry, fileLogger)

	// Prevent concurrent runs and persist run state for abrupt stop + restart.
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

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Run loop
	if *once {
		if err := mainLoop.RunOnce(ctx); err != nil && err != context.Canceled {
			// Check for agent exit (graceful completion)
			if _, ok := steps.IsAgentExitError(err); ok {
				trk.MarkComplete(runID)
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
	} else {
		if err := mainLoop.Run(ctx); err != nil && err != context.Canceled {
			// Check for agent exit (graceful completion)
			if _, ok := steps.IsAgentExitError(err); ok {
				trk.MarkComplete(runID)
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
	}

	return 0
}

func newProjectCmd(args []string) {
	fs := flag.NewFlagSet("new-project", flag.ExitOnError)
	name := fs.String("name", "", "Project name")
	dir := fs.String("dir", ".", "Project directory")
	reqFile := fs.String("requirements", "", "Path to requirements.md file (required)")
	fs.Parse(args)

	// Positional args support:
	//   ralph new-project <name> <requirements.md> [dir]
	pos := fs.Args()
	if *name == "" && len(pos) >= 1 {
		*name = pos[0]
	}
	if *reqFile == "" && len(pos) >= 2 {
		*reqFile = pos[1]
	}
	if *dir == "." && len(pos) >= 3 {
		*dir = pos[2]
	}

	if *name == "" {
		fmt.Fprintln(os.Stderr, "Project name is required:")
		fmt.Fprintln(os.Stderr, "  ralph new-project <name> <requirements.md> [dir]")
		fmt.Fprintln(os.Stderr, "  ralph new-project -name <name> -requirements <file> [-dir <dir>]")
		os.Exit(1)
	}

	if *reqFile == "" {
		fmt.Fprintln(os.Stderr, "Requirements file is required:")
		fmt.Fprintln(os.Stderr, "  ralph new-project <name> <requirements.md> [dir]")
		fmt.Fprintln(os.Stderr, "  ralph new-project -name <name> -requirements <file> [-dir <dir>]")
		fmt.Fprintln(os.Stderr, "\nCreate a requirements.md with:")
		fmt.Fprintln(os.Stderr, "  - What you want to build")
		fmt.Fprintln(os.Stderr, "  - Key features or functionality")
		fmt.Fprintln(os.Stderr, "  - Any technical constraints")
		os.Exit(1)
	}

	// Read requirements
	reqContent, err := os.ReadFile(*reqFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read requirements file: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(string(reqContent)) == "" {
		fmt.Fprintln(os.Stderr, "Requirements file is empty.")
		os.Exit(1)
	}

	fmt.Println("Analyzing requirements with Claude...")

	// Use Claude to interpret requirements and generate project files
	prompt, err := renderNewProjectPrompt(*name, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build Claude prompt: %v\n", err)
		os.Exit(1)
	}

	// Run Claude to interpret. If Claude can't run, fail fast without creating any project files.
	result, err := runClaudeOnce(prompt)
	if err != nil {
		if isClaudeRateLimitError(err) {
			fmt.Fprintln(os.Stderr, "Claude is unavailable (usage limit / rate limit).")
			details := claudeActionableDetails(err)
			if details != "" {
				fmt.Fprintf(os.Stderr, "\nDetails:\n%s\n", details)
			}
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "Claude analysis failed.")
		details := claudeActionableDetails(err)
		if details != "" {
			fmt.Fprintf(os.Stderr, "\nDetails:\n%s\n", details)
		}
		os.Exit(1)
	}

	// Check if requirements are insufficient
	if strings.HasPrefix(strings.TrimSpace(result), "INSUFFICIENT:") {
		fmt.Fprintln(os.Stderr, "Requirements need clarification:")
		fmt.Fprintln(os.Stderr, strings.TrimPrefix(strings.TrimSpace(result), "INSUFFICIENT:"))
		os.Exit(1)
	}

	// Parse the generated files
	prdContent := parseGeneratedPRD(result)
	if prdContent == "" {
		fmt.Fprintln(os.Stderr, "Failed to parse Claude's response.")
		os.Exit(1)
	}

	projectDir := filepath.Join(*dir, *name)
	codeDir := filepath.Join(projectDir, *name)

	// Create directory structure
	dirs := []string{
		projectDir,
		filepath.Join(projectDir, "configs"),
		filepath.Join(projectDir, "logs"),
		codeDir,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", d, err)
			os.Exit(1)
		}
	}

	// Create default config (git repo lives in nested codeDir)
	configContent, err := renderDefaultLoopConfig(*name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render default config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "configs", "default.json"), []byte(configContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create default config: %v\n", err)
		os.Exit(1)
	}

	// Save original requirements
	if err := os.WriteFile(filepath.Join(projectDir, "requirements.md"), reqContent, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save requirements: %v\n", err)
		os.Exit(1)
	}

	readmeContent, err := renderReadme(*name, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render README.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(codeDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create README.md: %v\n", err)
		os.Exit(1)
	}

	setupPrompt, err := renderSetupPrompt(*name, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render SETUP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	loopPrompt, err := renderLoopPrompt(*name, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render LOOP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "SETUP_PROMPT.md"), []byte(setupPrompt), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SETUP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "LOOP_PROMPT.md"), []byte(loopPrompt), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LOOP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "prd.json"), []byte(prdContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create prd.json: %v\n", err)
		os.Exit(1)
	}

	// Best-effort: initialize a git repo and create an initial scaffold commit (inside codeDir).
	gitInitAndInitialCommit(codeDir)

	// Count tasks (best-effort)
	taskCount := strings.Count(prdContent, "\"id\"")

	fmt.Printf("Created project: %s\n", projectDir)
	fmt.Printf("Tasks: %d\n", taskCount)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectDir)
	fmt.Println("  ralph run")
}

// parseGeneratedPRD extracts prd.json from Claude's response
func parseGeneratedPRD(response string) string {
	marker := "---FILE: prd.json---"
	idx := strings.Index(response, marker)
	if idx == -1 {
		return ""
	}
	content := strings.TrimSpace(response[idx+len(marker):])
	return stripJSONFences(content)
}

func parseNewTasks(response string) string {
	marker := "---NEW_TASKS---"
	idx := strings.Index(response, marker)
	if idx == -1 {
		return ""
	}
	content := strings.TrimSpace(response[idx+len(marker):])
	return stripJSONFences(content)
}

func newWorkCmd(args []string) {
	fs := flag.NewFlagSet("new-work", flag.ExitOnError)
	description := fs.String("desc", "", "Work description")
	filePath := fs.String("file", "", "Path to markdown file with work description")
	fs.Parse(args)

	// Positional args support:
	//   ralph new-work <file.md>
	//   ralph new-work <description...>
	pos := fs.Args()
	if *filePath == "" && *description == "" && len(pos) > 0 {
		if len(pos) == 1 {
			if _, err := os.Stat(pos[0]); err == nil {
				*filePath = pos[0]
			} else {
				*description = pos[0]
			}
		} else {
			*description = strings.Join(pos, " ")
		}
	}

	// Read work description from file if provided
	workDesc := *description
	if *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file %s: %v\n", *filePath, err)
			os.Exit(1)
		}
		workDesc = string(data)
	}

	if workDesc == "" {
		fmt.Fprintln(os.Stderr, "Work description is required:")
		fmt.Fprintln(os.Stderr, "  ralph new-work <file.md>")
		fmt.Fprintln(os.Stderr, "  ralph new-work \"description...\"")
		fmt.Fprintln(os.Stderr, "  ralph new-work -file work.md")
		fmt.Fprintln(os.Stderr, "  ralph new-work -desc \"description\"")
		os.Exit(1)
	}

	// Check if prd.json and requirements.md exist
	prdPath := "prd.json"
	prdBytes, err := os.ReadFile(prdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read prd.json - are you in a Ralph project? Error: %v\n", err)
		os.Exit(1)
	}
	reqBytes, err := os.ReadFile("requirements.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read requirements.md - are you in a Ralph project? Error: %v\n", err)
		os.Exit(1)
	}

	projectName := filepath.Base(mustGetwd())
	prompt, err := renderNewWorkPrompt(projectName, string(reqBytes), string(prdBytes), workDesc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build Claude prompt: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Calling Claude to translate into tasks...")
	result, err := runClaudeOnce(prompt)
	if err != nil {
		if isClaudeRateLimitError(err) {
			fmt.Fprintln(os.Stderr, "Claude is unavailable (usage limit / rate limit).")
			details := claudeActionableDetails(err)
			if details != "" {
				fmt.Fprintf(os.Stderr, "\nDetails:\n%s\n", details)
			}
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "Claude analysis failed.")
		details := claudeActionableDetails(err)
		if details != "" {
			fmt.Fprintf(os.Stderr, "\nDetails:\n%s\n", details)
		}
		os.Exit(1)
	}

	updatedPRD := parseGeneratedPRD(result)
	if updatedPRD != "" {
		// Best-effort: print which tasks were added by diffing IDs.
		var before prdFile
		_ = json.Unmarshal(prdBytes, &before)
		oldIDs := map[string]struct{}{}
		for _, t := range before.Tasks {
			if strings.TrimSpace(t.ID) != "" {
				oldIDs[t.ID] = struct{}{}
			}
		}

		var after prdFile
		_ = json.Unmarshal([]byte(updatedPRD), &after)
		added := make([]prdTask, 0)
		for _, t := range after.Tasks {
			id := strings.TrimSpace(t.ID)
			if id == "" {
				continue
			}
			if _, ok := oldIDs[id]; !ok {
				added = append(added, t)
			}
		}

		if err := os.WriteFile(prdPath, []byte(updatedPRD), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update prd.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("New tasks:")
		if len(added) == 0 {
			fmt.Println("  (unable to determine added tasks; prd.json was updated)")
		} else {
			for _, t := range added {
				id := strings.TrimSpace(t.ID)
				title := strings.TrimSpace(t.Title)
				prio := strings.TrimSpace(t.Priority)
				if prio == "" {
					prio = "(no priority)"
				}
				fmt.Printf("  - [%s] %s (%s)\n", id, title, prio)
			}
		}
		fmt.Println("\nNext step:")
		fmt.Println("  ralph run")
		return
	}

	newTasksJSON := parseNewTasks(result)
	if newTasksJSON == "" {
		fmt.Fprintln(os.Stderr, "Failed to parse new tasks from Claude's response.")
		os.Exit(1)
	}

	var newTasks []prdTask
	if err := json.Unmarshal([]byte(newTasksJSON), &newTasks); err != nil {
		fmt.Fprintf(os.Stderr, "New tasks are not valid JSON: %v\n", err)
		os.Exit(1)
	}
	if len(newTasks) == 0 {
		fmt.Fprintln(os.Stderr, "No new tasks returned.")
		os.Exit(1)
	}

	var existing prdFile
	if err := json.Unmarshal(prdBytes, &existing); err != nil {
		fmt.Fprintf(os.Stderr, "Existing prd.json is not valid JSON: %v\n", err)
		os.Exit(1)
	}
	if existing.Version == 0 {
		existing.Version = 1
	}

	existing.Tasks = append(newTasks, existing.Tasks...)

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serialize updated prd.json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(prdPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update prd.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("New tasks:")
	for _, t := range newTasks {
		id := strings.TrimSpace(t.ID)
		title := strings.TrimSpace(t.Title)
		prio := strings.TrimSpace(t.Priority)
		if prio == "" {
			prio = "(no priority)"
		}
		fmt.Printf("  - [%s] %s (%s)\n", id, title, prio)
	}
	fmt.Println("\nNext step:")
	fmt.Println("  ralph run")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func initConfigCmd(args []string) {
	fs := flag.NewFlagSet("init-config", flag.ExitOnError)
	output := fs.String("output", "configs/default.json", "Output file path")
	force := fs.Bool("force", false, "Overwrite existing file")
	fs.Parse(args)

	// Check if file exists
	if _, err := os.Stat(*output); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "Config file already exists: %s\n", *output)
		fmt.Fprintln(os.Stderr, "Use -force to overwrite.")
		os.Exit(1)
	}

	// Ensure directory exists
	dir := filepath.Dir(*output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	configContent, err := renderDefaultLoopConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render default config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, []byte(configContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created config file: %s\n", *output)
	fmt.Println("\nAvailable step types:")
	fmt.Println("  - agent:        Run Claude Code to work on tasks from prd.json")
	fmt.Println("  - command:      Execute a shell command")
	fmt.Println("  - readme-check: Check and update README.md")
	fmt.Println("  - noop:         Do nothing (for testing)")
	fmt.Println("\nEnvironment variables are supported:")
	fmt.Println("  - ${VAR}          - Replace with VAR's value")
	fmt.Println("  - ${VAR:-default} - Use 'default' if VAR is not set")
}
