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
  "name": "default",
  "max_loops_per_task": 10,
  "steps": [
    {
      "type": "agent",
      "name": "claude",
      "config": {
        "prompt_file": "{{.LoopPromptFile}}",
        "prd_file": "{{.PrdFile}}",
{{- if .Model }}
        "model": "{{.Model}}",
{{- end }}
        "log_dir": "{{.LogDir}}"
      },
      "timeout": "20m"
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
