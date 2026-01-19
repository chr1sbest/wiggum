package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Static prd.json for explore mode - always a single completed task
const explorePRDJSON = `{
  "version": 1,
  "tasks": [
    {
      "id": "T001",
      "title": "Explore repository and summarize",
      "details": "Explored codebase structure, identified tech stack, and documented architecture",
      "priority": "high",
      "status": "done",
      "tests": "requirements.md exists with project summary"
    }
  ]
}`

func newProjectCmd(args []string) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`init 🖍️  Initialize Ralph in the current directory

Usage:
  ralph init                       (existing repo - Ralph explores and summarizes)
  ralph init <requirements.md>     (new project - generate tasks from requirements)

Flags:
  -requirements   Path to requirements.md file
  -model          Claude model to use

Examples:
  ralph init                              # existing repo
  ralph init requirements.md              # new project
  ralph init -requirements requirements.md -model sonnet
`)
	}
	reqFile := fs.String("requirements", "", "Path to requirements.md file")
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
		fmt.Fprintln(os.Stderr, "  ralph init")
		fmt.Fprintln(os.Stderr, "  ralph init <requirements.md>")
		os.Exit(1)
	}

	// Get project name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}
	projectName := filepath.Base(cwd)

	// No requirements file provided - check if existing repo
	if *reqFile == "" {
		if hasExistingCode() {
			initExistingRepo(projectName, *model)
			return
		}
		fmt.Fprintln(os.Stderr, "This folder is empty, but it could be many things.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Create a requirements.md describing what you want to build,")
		fmt.Fprintln(os.Stderr, "then run:")
		fmt.Fprintln(os.Stderr, "  ralph init requirements.md")
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
		analysisModel = "default"
	}

	fmt.Printf("Analyzing requirements with Claude...\n")

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

	// Initialize git if no .git exists (new project)
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := runGitInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize git: %v\n", err)
		}
	}

	// Add .ralph/ to .gitignore BEFORE creating files (so git never tracks them)
	appendToGitignore(".ralph/")

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
	content = stripJSONFences(content)

	// Force all task statuses to "todo" (Claude sometimes outputs "done")
	return forceTaskStatusTodo(content)
}

// forceTaskStatusTodo ensures all tasks have status "todo"
func forceTaskStatusTodo(prdJSON string) string {
	// For simple cases without tasks, return as-is
	var testStruct struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal([]byte(prdJSON), &testStruct); err == nil {
		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(prdJSON), &rawMap); err == nil {
			if _, hasTasks := rawMap["tasks"]; !hasTasks {
				return prdJSON
			}
		}
	}

	var prd prdFile
	if err := json.Unmarshal([]byte(prdJSON), &prd); err != nil {
		return prdJSON // Return as-is if can't parse
	}

	for i := range prd.Tasks {
		prd.Tasks[i].Status = "todo"
	}

	out, err := json.Marshal(prd)
	if err != nil {
		return prdJSON
	}
	return string(out)
}

// runGitInit initializes a new git repository
func runGitInit() error {
	cmd := exec.Command("git", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// hasExistingCode checks if the current directory has code files
func hasExistingCode() bool {
	codeExtensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".rb": true, ".rs": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".cs": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
		".html": true, ".css": true, ".scss": true, ".vue": true, ".svelte": true,
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		return false
	}

	for _, e := range entries {
		if e.IsDir() {
			// Skip hidden dirs and common non-code dirs
			name := e.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				continue
			}
			// Has a subdirectory - likely a project
			return true
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if codeExtensions[ext] {
			return true
		}
	}
	return false
}

// initExistingRepo handles ralph init in an existing codebase
func initExistingRepo(projectName, model string) {
	// Check if .ralph already exists
	if _, err := os.Stat(".ralph"); err == nil {
		fmt.Fprintln(os.Stderr, "Ralph is already initialized here (.ralph/ exists).")
		fmt.Fprintln(os.Stderr, "To reinitialize, remove .ralph/ first:")
		fmt.Fprintln(os.Stderr, "  rm -rf .ralph && ralph init")
		os.Exit(1)
	}

	fmt.Println("Hi code! I live here now.")

	if err := validateClaudePreflight(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	analysisModel := strings.TrimSpace(model)
	if analysisModel == "" {
		analysisModel = "default"
	}

	fmt.Printf("Exploring codebase with Claude...\n")

	prompt, err := renderExploreRepoPrompt(projectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build explore prompt: %v\n", err)
		os.Exit(1)
	}

	result, err := runClaudeOnceWithModel(prompt, analysisModel)
	if err != nil {
		if isClaudeRateLimitError(err) {
			fmt.Fprintln(os.Stderr, "Claude is unavailable (usage limit / rate limit).")
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "Claude exploration failed.")
		details := claudeActionableDetails(err)
		if details != "" {
			fmt.Fprintf(os.Stderr, "\nDetails:\n%s\n", details)
		}
		os.Exit(1)
	}

	// Parse the response - extract requirements.md only
	reqContent := parseExploreRequirements(result)

	if reqContent == "" {
		fmt.Fprintln(os.Stderr, "Failed to parse requirements from Claude's response.")
		fmt.Fprintln(os.Stderr, "\nClaude's response (first 500 chars):")
		preview := result
		if len(preview) > 500 {
			preview = preview[:500]
		}
		fmt.Fprintln(os.Stderr, preview)
		os.Exit(1)
	}

	// Use static prd.json for explore mode (always the same)
	prdContent := explorePRDJSON

	// Create .ralph directory structure
	ralphDir := ".ralph"
	for _, dir := range []string{ralphDir, filepath.Join(ralphDir, "prompts"), filepath.Join(ralphDir, "configs"), filepath.Join(ralphDir, "logs")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Add .ralph/ to .gitignore BEFORE creating files (so git never tracks them)
	appendToGitignore(".ralph/")

	// Write default config
	configContent, err := renderDefaultLoopConfig(DefaultLoopConfigOptions{
		RepoDir:           ".",
		SetupPromptFile:   ".ralph/prompts/SETUP_PROMPT.md",
		LoopPromptFile:    ".ralph/prompts/LOOP_PROMPT.md",
		LogDir:            ".ralph/logs",
		PrdFile:           ".ralph/prd.json",
		CommitMessageFile: "../.ralph/commit_message.txt",
		Model:             model,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render default config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "configs", "default.json"), []byte(configContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create default config: %v\n", err)
		os.Exit(1)
	}

	// Write requirements.md (generated from exploration)
	if err := os.WriteFile(filepath.Join(ralphDir, "requirements.md"), []byte(reqContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save requirements: %v\n", err)
		os.Exit(1)
	}

	// Write prompt templates
	setupPrompt, err := renderSetupPrompt(projectName, reqContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render SETUP_PROMPT.md: %v\n", err)
		os.Exit(1)
	}
	loopPrompt, err := renderLoopPrompt(projectName, reqContent)
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

	// Write prd.json
	if err := os.WriteFile(filepath.Join(ralphDir, "prd.json"), []byte(prdContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create prd.json: %v\n", err)
		os.Exit(1)
	}

	cwd, _ := os.Getwd()

	fmt.Printf("Initialized Ralph in: %s\n", cwd)
	fmt.Printf("Summary: .ralph/requirements.md\n")
	fmt.Printf("Tasks: .ralph/prd.json\n")
	fmt.Println("\nNext steps:")
	fmt.Println("  ralph add work.md")
	fmt.Println("  ralph add \"your task description\"")
	fmt.Println("  ralph run")
}

func parseExploreRequirements(response string) string {
	marker := "---SUMMARY---"
	idx := strings.Index(response, marker)
	if idx == -1 {
		// Fallback to old marker for compatibility
		marker = "---FILE: requirements.md---"
		idx = strings.Index(response, marker)
		if idx == -1 {
			return ""
		}
	}
	content := response[idx+len(marker):]

	// Find the end (next marker or end of string)
	for _, endMarker := range []string{"---SUMMARY---", "---FILE:"} {
		endIdx := strings.Index(content, endMarker)
		if endIdx != -1 {
			content = content[:endIdx]
			break
		}
	}

	return strings.TrimSpace(content)
}
