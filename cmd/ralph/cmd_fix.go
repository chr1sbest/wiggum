package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func fixCmd(args []string) {
	fs := flag.NewFlagSet("fix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`fix 🔧  Create tasks from a GitHub issue

Usage:
  ralph fix --issue <number>
  ralph fix <github-issue-url>

Flags:
  -issue    GitHub issue number (infers repo from git remote)
  -repo     Override repository (owner/repo format)
  -model    Claude model to use

Examples:
  ralph fix --issue 42
  ralph fix https://github.com/owner/repo/issues/42
  ralph fix --issue 42 --repo owner/repo
`)
	}

	issueNum := fs.Int("issue", 0, "GitHub issue number")
	repoOverride := fs.String("repo", "", "Repository (owner/repo)")
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

	// Handle positional arg (URL)
	pos := fs.Args()
	if *issueNum == 0 && len(pos) > 0 {
		parsed := parseIssueURL(pos[0])
		if parsed != nil {
			*issueNum = parsed.Number
			if *repoOverride == "" {
				*repoOverride = parsed.Repo
			}
		}
	}

	if *issueNum == 0 {
		fmt.Fprintln(os.Stderr, "Issue number is required:")
		fmt.Fprintln(os.Stderr, "  ralph fix --issue 42")
		fmt.Fprintln(os.Stderr, "  ralph fix https://github.com/owner/repo/issues/42")
		os.Exit(1)
	}

	// Preflight checks
	fmt.Println("Preflight checks...")

	if err := checkClaudeAvailable(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Claude: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Claude CLI available")

	if err := checkGitHubAuth(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ GitHub: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ GitHub CLI authenticated")

	// Determine repo
	repo := *repoOverride
	if repo == "" {
		var err error
		repo, err = getGitHubRepo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not detect GitHub repo: %v\n", err)
			fmt.Fprintln(os.Stderr, "Use --repo owner/repo to specify manually")
			os.Exit(1)
		}
	}
	fmt.Printf("  ✓ Repository: %s\n", repo)

	// Fetch issue
	fmt.Printf("\nFetching issue #%d from %s...\n", *issueNum, repo)
	issue, err := fetchGitHubIssue(repo, *issueNum)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ %s\n", issue.Title)

	if issue.State == "closed" {
		fmt.Printf("  ⚠️  Issue is closed (state: %s)\n", issue.State)
	}

	// Check we're in a Ralph project
	prdPath := filepath.Join(".ralph", "prd.json")
	prdBytes, err := os.ReadFile(prdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read .ralph/prd.json - are you in a Ralph project? Error: %v\n", err)
		os.Exit(1)
	}
	reqBytes, err := os.ReadFile(filepath.Join(".ralph", "requirements.md"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read .ralph/requirements.md - are you in a Ralph project? Error: %v\n", err)
		os.Exit(1)
	}

	// Format issue as work description
	workDesc := formatIssueAsWork(issue)

	chosenModel := strings.TrimSpace(*model)
	if chosenModel == "" {
		chosenModel = "default"
	}

	// Archive completed tasks and compact learnings before adding new work
	archiveCompletedTasks()
	compactLearnings(chosenModel)

	projectName := filepath.Base(mustGetwd())
	prompt, err := renderNewWorkPrompt(projectName, string(reqBytes), string(prdBytes), workDesc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build Claude prompt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nCalling Claude to create tasks (model: %s)...\n", chosenModel)
	result, err := runClaudeOnceWithModel(prompt, chosenModel)
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

	// Try parsing as full PRD first (same logic as cmd_add.go)
	updatedPRD := parseGeneratedPRD(result)
	if updatedPRD != "" {
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
			fmt.Fprintf(os.Stderr, "Failed to update .ralph/prd.json: %v\n", err)
			os.Exit(1)
		}
		printAddedTasks(added, issue)
		return
	}

	// Parse as new tasks
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
		fmt.Fprintf(os.Stderr, "Existing .ralph/prd.json is not valid JSON: %v\n", err)
		os.Exit(1)
	}
	if existing.Version == 0 {
		existing.Version = 1
	}

	existing.Tasks = append(newTasks, existing.Tasks...)

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serialize updated .ralph/prd.json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(prdPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update .ralph/prd.json: %v\n", err)
		os.Exit(1)
	}

	printAddedTasks(newTasks, issue)
}

func printAddedTasks(tasks []prdTask, issue *GitHubIssue) {
	fmt.Printf("\nTasks created for issue #%d:\n", issue.Number)
	if len(tasks) == 0 {
		fmt.Println("  (unable to determine added tasks; .ralph/prd.json was updated)")
	} else {
		for _, t := range tasks {
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
}

type parsedIssueURL struct {
	Repo   string
	Number int
}

func parseIssueURL(url string) *parsedIssueURL {
	// Match: https://github.com/owner/repo/issues/123
	re := regexp.MustCompile(`github\.com/([^/]+/[^/]+)/issues/(\d+)`)
	m := re.FindStringSubmatch(url)
	if m == nil {
		return nil
	}
	num, err := strconv.Atoi(m[2])
	if err != nil {
		return nil
	}
	return &parsedIssueURL{
		Repo:   m[1],
		Number: num,
	}
}
