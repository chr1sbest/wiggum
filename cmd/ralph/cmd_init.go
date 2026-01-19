package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func newProjectCmd(args []string) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`init 🖍️  Initialize Ralph in the current directory

Usage:
  ralph init <requirements.md>
  ralph init -requirements <file> [-model <model>]

Flags:
  -requirements   Path to requirements.md file (required)
  -model          Claude model to use (default: opus)

Examples:
  ralph init requirements.md
  ralph init -requirements requirements.md -model opus
`)
	}
	reqFile := fs.String("requirements", "", "Path to requirements.md file (required)")
	model := fs.String("model", "", "Claude model to use")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		os.Exit(1)
	}

	pos := fs.Args()
	if *reqFile == "" && len(pos) >= 1 {
		*reqFile = pos[0]
	}
	if len(pos) >= 2 {
		fmt.Fprintln(os.Stderr, "Too many arguments.")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  ralph init <requirements.md>")
		fmt.Fprintln(os.Stderr, "  ralph init -requirements <file>")
		os.Exit(1)
	}

	// Get project name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}
	projectName := filepath.Base(cwd)

	if *reqFile == "" {
		fmt.Fprintln(os.Stderr, "Requirements file is required:")
		fmt.Fprintln(os.Stderr, "  ralph init <requirements.md>")
		fmt.Fprintln(os.Stderr, "  ralph init -requirements <file>")
		fmt.Fprintln(os.Stderr, "\nCreate a requirements.md with:")
		fmt.Fprintln(os.Stderr, "  - What you want to build")
		fmt.Fprintln(os.Stderr, "  - Key features or functionality")
		fmt.Fprintln(os.Stderr, "  - Any technical constraints")
		os.Exit(1)
	}

	reqContent, err := os.ReadFile(*reqFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read requirements file: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(string(reqContent)) == "" {
		fmt.Fprintln(os.Stderr, "Requirements file is empty.")
		os.Exit(1)
	}

	if err := validateClaudePreflight(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	configModel := strings.TrimSpace(*model)
	analysisModel := configModel
	if analysisModel == "" {
		analysisModel = "opus"
	}

	fmt.Printf("Analyzing requirements with Claude (model: %s)...\n", analysisModel)

	prompt, err := renderNewProjectPrompt(projectName, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build Claude prompt: %v\n", err)
		os.Exit(1)
	}

	result, err := runClaudeOnceWithModel(prompt, analysisModel)
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

	if strings.HasPrefix(strings.TrimSpace(result), "INSUFFICIENT:") {
		fmt.Fprintln(os.Stderr, "Requirements need clarification:")
		fmt.Fprintln(os.Stderr, strings.TrimPrefix(strings.TrimSpace(result), "INSUFFICIENT:"))
		os.Exit(1)
	}

	prdContent := parseGeneratedPRD(result)
	if prdContent == "" {
		fmt.Fprintln(os.Stderr, "Failed to parse Claude's response.")
		os.Exit(1)
	}

	ralphDir := ".ralph"

	dirs := []string{
		filepath.Join(ralphDir, "configs"),
		filepath.Join(ralphDir, "logs"),
		filepath.Join(ralphDir, "prompts"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", d, err)
			os.Exit(1)
		}
	}

	configContent, err := renderDefaultLoopConfig(DefaultLoopConfigOptions{
		RepoDir:           ".",
		SetupPromptFile:   ".ralph/prompts/SETUP_PROMPT.md",
		LoopPromptFile:    ".ralph/prompts/LOOP_PROMPT.md",
		MarkerFile:        ".ralph/.ralph_setup_done",
		LogDir:            ".ralph/logs",
		PrdFile:           ".ralph/prd.json",
		CommitMessageFile: "../.ralph/commit_message.txt",
		Model:             configModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render default config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "configs", "default.json"), []byte(configContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create default config: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(ralphDir, "requirements.md"), reqContent, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save requirements: %v\n", err)
		os.Exit(1)
	}

	setupPrompt, err := renderSetupPrompt(projectName, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render SETUP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	loopPrompt, err := renderLoopPrompt(projectName, string(reqContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render LOOP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "prompts", "SETUP_PROMPT.md"), []byte(setupPrompt), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SETUP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "prompts", "LOOP_PROMPT.md"), []byte(loopPrompt), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LOOP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "prd.json"), []byte(prdContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create prd.json: %v\n", err)
		os.Exit(1)
	}

	// Append .ralph/ to .gitignore if not already present
	appendToGitignore(".ralph/")

	taskCount := strings.Count(prdContent, "\"id\"")

	fmt.Printf("Initialized Ralph in: %s\n", cwd)
	fmt.Printf("Tasks: %d\n", taskCount)
	fmt.Printf("Tasks file: %s\n", filepath.Join(ralphDir, "prd.json"))
	fmt.Println("\nNext step:")
	fmt.Println("  ralph run")
}

func parseGeneratedPRD(response string) string {
	marker := "---FILE: prd.json---"
	idx := strings.Index(response, marker)
	if idx == -1 {
		return ""
	}
	content := strings.TrimSpace(response[idx+len(marker):])
	return stripJSONFences(content)
}

// appendToGitignore adds an entry to .gitignore if not already present
func appendToGitignore(entry string) {
	gitignorePath := ".gitignore"
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return // Can't read, skip silently
	}

	// Check if entry already exists
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return // Already present
		}
	}

	// Append entry
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Can't open, skip silently
	}
	defer f.Close()

	// Add newline before if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}
