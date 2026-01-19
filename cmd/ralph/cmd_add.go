package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func newWorkCmd(args []string) {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`add 🖍️  Add new work for Ralph

Usage:
  ralph add <file.md>
  ralph add "description..."
  ralph add -file <file.md> [-model <model>]
  ralph add -desc "description..." [-model <model>]

Flags:
  -file   Path to markdown file with work description
  -desc   Work description
  -model  Claude model to use

Examples:
  ralph add ../work.md
  ralph add "Add an endpoint that returns the user's country based on IP"
  ralph add -file ../work.md -model sonnet
`)
	}
	description := fs.String("desc", "", "Work description")
	filePath := fs.String("file", "", "Path to markdown file with work description")
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
		fmt.Fprintln(os.Stderr, "  ralph add <file.md>")
		fmt.Fprintln(os.Stderr, "  ralph add \"description...\"")
		fmt.Fprintln(os.Stderr, "  ralph add -file work.md")
		fmt.Fprintln(os.Stderr, "  ralph add -desc \"description\"")
		os.Exit(1)
	}

	chosenModel := strings.TrimSpace(*model)
	if chosenModel == "" {
		chosenModel = "default"
	}

	// Archive completed tasks and compact learnings before adding new work
	archiveCompletedTasks()
	compactLearnings(chosenModel)

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

	projectName := filepath.Base(mustGetwd())
	prompt, err := renderNewWorkPrompt(projectName, string(reqBytes), string(prdBytes), workDesc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build Claude prompt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Calling Claude to translate into tasks (model: %s)...\n", chosenModel)
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
		fmt.Println("New tasks:")
		if len(added) == 0 {
			fmt.Println("  (unable to determine added tasks; .ralph/prd.json was updated)")
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

func parseNewTasks(response string) string {
	marker := "---NEW_TASKS---"
	idx := strings.Index(response, marker)
	if idx == -1 {
		return ""
	}
	content := strings.TrimSpace(response[idx+len(marker):])
	return stripJSONFences(content)
}
