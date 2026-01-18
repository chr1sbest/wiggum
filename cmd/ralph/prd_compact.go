package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// archiveCompletedTasks moves completed tasks from prd.json to prd_archive.json
func archiveCompletedTasks() {
	prdPath := filepath.Join(".ralph", "prd.json")
	archivePath := filepath.Join(".ralph", "prd_archive.json")

	prdBytes, err := os.ReadFile(prdPath)
	if err != nil {
		return // No prd.json, nothing to archive
	}

	var prd prdFile
	if err := json.Unmarshal(prdBytes, &prd); err != nil {
		return
	}

	// Separate completed and incomplete tasks
	var completed, incomplete []prdTask
	for _, t := range prd.Tasks {
		status := strings.ToLower(strings.TrimSpace(t.Status))
		if status == "done" || status == "complete" || status == "completed" {
			completed = append(completed, t)
		} else {
			incomplete = append(incomplete, t)
		}
	}

	if len(completed) == 0 {
		return // Nothing to archive
	}

	// Load existing archive
	var archive prdArchive
	if archiveBytes, err := os.ReadFile(archivePath); err == nil {
		_ = json.Unmarshal(archiveBytes, &archive)
	}

	// Add completed tasks to archive
	archive.ArchivedAt = time.Now().Format(time.RFC3339)
	archive.Tasks = append(archive.Tasks, completed...)

	// Write updated archive
	archiveOut, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(archivePath, archiveOut, 0644); err != nil {
		return
	}

	// Update prd.json with only incomplete tasks
	prd.Tasks = incomplete
	prdOut, err := json.MarshalIndent(prd, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(prdPath, prdOut, 0644); err != nil {
		return
	}

	fmt.Printf("Archived %d completed task(s) to .ralph/prd_archive.json\n", len(completed))
}

type prdArchive struct {
	ArchivedAt string    `json:"archived_at"`
	Tasks      []prdTask `json:"tasks"`
}

// compactLearnings summarizes learnings.md if it gets too large
func compactLearnings(model string) {
	learningsPath := filepath.Join(".ralph", "learnings.md")

	content, err := os.ReadFile(learningsPath)
	if err != nil {
		return // No learnings file yet
	}

	// Only compact if file is large (> 4000 chars, roughly 1k tokens)
	if len(content) < 4000 {
		return
	}

	fmt.Printf("Compacting learnings.md (model: %s)...\n", model)

	prompt := fmt.Sprintf(`You are summarizing a project learnings document. 
Condense the following learnings into a shorter, well-organized summary.
Keep the most important patterns, gotchas, and architectural decisions.
Remove redundant or outdated information.
Output ONLY the summarized markdown content, no explanations.

---LEARNINGS---
%s
---END LEARNINGS---`, string(content))

	result, err := runClaudeOnceWithModel(prompt, model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to compact learnings: %v\n", err)
		return
	}

	// Clean up the result
	summarized := strings.TrimSpace(result)
	if summarized == "" {
		return
	}

	// Add header noting this was compacted
	header := fmt.Sprintf("<!-- Compacted on %s -->\n\n", time.Now().Format("2006-01-02"))
	summarized = header + summarized

	if err := os.WriteFile(learningsPath, []byte(summarized), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write compacted learnings: %v\n", err)
		return
	}

	fmt.Printf("Compacted learnings.md: %d -> %d chars\n", len(content), len(summarized))
}
