package main

import (
	"bytes"
	"fmt"
	"text/template"
)

type defaultConfigTemplateData struct {
	RepoDir           string
	SetupPromptFile   string
	LoopPromptFile    string
	MarkerFile        string
	LogDir            string
	PrdFile           string
	CommitMessageFile string
	Model             string
}

type DefaultLoopConfigOptions struct {
	RepoDir           string
	SetupPromptFile   string
	LoopPromptFile    string
	MarkerFile        string
	LogDir            string
	PrdFile           string
	CommitMessageFile string
	Model             string
}

const defaultLoopConfigTemplate = `{
  "name": "default-loop",
  "description": "Ralph automation loop with Claude agent",
  "max_loops_per_task": 10,
  "steps": [
    {
      "type": "agent",
      "name": "setup",
      "config": {
        "prompt_file": "{{.SetupPromptFile}}",
        "prd_file": "{{.PrdFile}}",
{{- if .Model }}
        "model": "{{.Model}}",
{{- end }}
        "session_file": ".ralph/.ralph_session",
        "marker_file": "{{.MarkerFile}}",
        "allowed_tools": "Write,Read,Edit,Glob,Grep,Bash,Task,TodoWrite,WebFetch,WebSearch",
        "timeout": "15m",
        "log_dir": "{{.LogDir}}"
      },
      "timeout": "20m",
      "max_retries": 1,
      "retry_delay": "30s"
    },
    {
      "type": "agent",
      "name": "run-claude",
      "config": {
        "prompt_file": "{{.LoopPromptFile}}",
        "prd_file": "{{.PrdFile}}",
{{- if .Model }}
        "model": "{{.Model}}",
{{- end }}
        "session_file": ".ralph/.ralph_session",
        "allowed_tools": "Write,Read,Edit,Glob,Grep,Bash,Task,TodoWrite,WebFetch,WebSearch",
        "timeout": "15m",
        "log_dir": "{{.LogDir}}"
      },
      "timeout": "20m",
      "max_retries": 1,
      "retry_delay": "30s"
    },
    {
      "type": "git-commit",
      "name": "commit-progress",
      "continue_on_error": true,
      "config": {
        "enabled": true,
        "repo_dir": "{{.RepoDir}}",
        "prd_file": "{{.PrdFile}}",
        "commit_message_file": "{{.CommitMessageFile}}",
        "message_template": "{{"{{task_id}}"}}: {{"{{task}}"}}"
      }
    }
  ]
}
`

func renderDefaultLoopConfig(opts DefaultLoopConfigOptions) (string, error) {
	if opts.RepoDir == "" {
		opts.RepoDir = "."
	}
	if opts.SetupPromptFile == "" {
		opts.SetupPromptFile = ".ralph/prompts/SETUP_PROMPT.md"
	}
	if opts.LoopPromptFile == "" {
		opts.LoopPromptFile = ".ralph/prompts/LOOP_PROMPT.md"
	}
	if opts.MarkerFile == "" {
		opts.MarkerFile = ".ralph/.ralph_setup_done"
	}
	if opts.LogDir == "" {
		opts.LogDir = ".ralph/logs"
	}
	if opts.PrdFile == "" {
		opts.PrdFile = ".ralph/prd.json"
	}
	if opts.CommitMessageFile == "" {
		opts.CommitMessageFile = "../.ralph/commit_message.txt"
	}

	data := defaultConfigTemplateData{
		RepoDir:           opts.RepoDir,
		SetupPromptFile:   opts.SetupPromptFile,
		LoopPromptFile:    opts.LoopPromptFile,
		MarkerFile:        opts.MarkerFile,
		LogDir:            opts.LogDir,
		PrdFile:           opts.PrdFile,
		CommitMessageFile: opts.CommitMessageFile,
		Model:             opts.Model,
	}

	tmpl, err := template.New("default_config").Option("missingkey=error").Parse(defaultLoopConfigTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	out := buf.String()
	if out == "" {
		return "", fmt.Errorf("rendered default config is empty")
	}
	return out, nil
}
