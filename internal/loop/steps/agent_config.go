package steps

// AgentConfig holds configuration for the agent step
type AgentConfig struct {
	// PromptFile is the path to PROMPT.md (default: "PROMPT.md")
	PromptFile string `json:"prompt_file,omitempty"`
	// PrdFile is the path to prd.json (default: "prd.json")
	PrdFile string `json:"prd_file,omitempty"`
	// Model is the Claude model to use (optional)
	Model string `json:"model,omitempty"`
	// MarkerFile is an optional file path. If it exists, the agent step is skipped.
	// If set and the step runs successfully, the marker file will be created.
	MarkerFile string `json:"marker_file,omitempty"`
	// AllowedTools is a comma-separated list of tools Claude can use
	AllowedTools string `json:"allowed_tools,omitempty"`
	// Timeout is the max execution time (default: "15m")
	Timeout string `json:"timeout,omitempty"`
	// SessionFile is where to store session state
	SessionFile string `json:"session_file,omitempty"`
	// SessionExpiryHours is how long sessions last (default: 24)
	SessionExpiryHours int `json:"session_expiry_hours,omitempty"`
	// ClaudeBinary is the path to claude CLI (default: "claude")
	ClaudeBinary string `json:"claude_binary,omitempty"`
	// OutputFormat is json or text (default: "json")
	OutputFormat string `json:"output_format,omitempty"`
	// AppendSystemPrompt is extra context to add to the prompt
	AppendSystemPrompt string `json:"append_system_prompt,omitempty"`
	// LogDir is where to save Claude output logs
	LogDir string `json:"log_dir,omitempty"`
}

// DefaultAgentConfig returns sensible defaults
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		PromptFile:         "PROMPT.md",
		PrdFile:            "prd.json",
		Model:              "sonnet",
		MarkerFile:         "",
		AllowedTools:       "Write,Read,Edit,Glob,Grep,Bash,Task,TodoWrite,WebFetch,WebSearch",
		Timeout:            "15m",
		SessionFile:        ".ralph/.ralph_session",
		SessionExpiryHours: 24,
		ClaudeBinary:       "claude",
		OutputFormat:       "json",
		LogDir:             "logs",
	}
}
