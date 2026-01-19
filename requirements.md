# Wiggum (Ralph)

## Overview
Ralph is a fully autonomous AI coding assistant that leverages Claude Code for continuous software development. It implements a "Ralph Loop" architecture that avoids context rot by running fresh Claude sessions in repeated iterations—each session focuses on one task, then exits cleanly. Named after Ralph Wiggum from The Simpsons with the tagline "I'm a helper!"

## Tech Stack
- Go 1.23.1 (primary language)
- Claude Code CLI (external dependency for AI execution)
- GitHub API (for issue-to-task conversion)
- Git (version control and commit automation)
- fsnotify (file system watching for config hot-reload)
- Distribution targets: macOS/Linux × ARM64/AMD64

## Architecture
The codebase is organized into:
- `cmd/ralph/` - CLI command implementations (main entry point, command routing)
- `internal/loop/` - Core execution loop engine with step orchestration
- `internal/loop/steps/` - Step implementations (agent, git-commit, command, readme)
- `internal/agent/` - Claude session management and task tracking
- `internal/config/` - Configuration system with hot-reload support
- `internal/tracker/` - Run state, metrics, and file locking
- `internal/resilience/` - Retry logic and circuit breaker patterns
- `internal/status/` - Terminal UI progress display
- `examples/` - Example project requirements
- `.ralph/` - Per-project state directory (prd.json, learnings.md, configs, logs)

## Key Components
- **run command** - Execute the main autonomous loop, iterating through configured steps
- **init command** - Initialize Ralph in a directory, analyze requirements or explore existing repo
- **add command** - Add new work items, translate descriptions into tasks
- **fix command** - Convert GitHub issues into actionable tasks
- **upgrade command** - Self-update via GitHub releases or Homebrew
- **Loop Engine** - Orchestrates step execution with circuit breakers and task-level loop counting
- **Agent Step** - Executes Claude Code with context injection (loop #, task progress, learnings)
- **Git Commit Step** - Automatically commits changes after task completion
- **Tracker** - Persists run state, token usage, and cost metrics
- **Config System** - JSON-based configuration with environment variable substitution
