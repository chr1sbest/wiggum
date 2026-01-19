# Ralph Architecture

## Overview

Ralph is an autonomous coding assistant that executes tasks in a loop. Each loop iteration is a fresh Claude session that works on one task, then exits. This design ensures no context accumulation across iterations.

```
┌─────────────────────────────────────────────────────────────┐
│                        ralph run                            │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐           │
│  │   setup   │ -> │   agent   │ -> │git-commit │ -> loop   │
│  └───────────┘    └───────────┘    └───────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Loop Engine (`internal/loop/`)

The loop orchestrates step execution:

```go
// loop.go
type Loop struct {
    config   *config.Config
    registry *StepRegistry
    state    *RunState
    status   *status.Writer
}
```

- **Steps** are registered in a `StepRegistry` and instantiated per-run
- **Config** (`default.json`) defines which steps run and their settings
- **State** tracks loop number, current step, errors

### 2. Steps (`internal/loop/steps/`)

Each step implements:

```go
type Step interface {
    Name() string
    Execute(ctx context.Context, rawConfig json.RawMessage) error
}
```

| Step | Purpose |
|------|---------|
| `agent` | Runs Claude to work on a task |
| `git-commit` | Commits changes after task completion |
| `command` | Runs arbitrary shell commands |
| `readme-check` | Validates README exists |
| `noop` | Does nothing (for testing) |

### 3. Agent Step (`internal/loop/steps/agent.go`)

The core step that invokes Claude:

```
Execute()
  ├── Load PRD status from .ralph/prd.json
  ├── Check exit conditions (all done? stuck?)
  ├── Read prompt file (LOOP_PROMPT.md)
  ├── Build context (loop #, task progress, learnings)
  ├── Execute: claude -p <prompt> --append-system-prompt <context>
  ├── Parse output, track tokens/cost
  └── Check exit conditions again
```

**Key design: Fresh session each iteration** — no `--resume`, so no context rot.

### 4. Tracker (`internal/tracker/`)

Persists run state and metrics:

```
.ralph/
  ├── run_state.json      # Current run ID, loop count, status
  ├── aggregate.json      # Final output with version, model, metrics
  └── .ralph_lock         # Prevents concurrent runs
```

### 5. Status Writer (`internal/status/`)

Terminal UI showing progress:

```
[████████░░] 4/6 Implementing geo endpoint
```

Reads task completion from `.ralph/prd.json`.

## Execution Flow

### `ralph init`

```
1. Parse requirements.md
2. Call Claude (opus) to analyze requirements → generate tasks
3. Create project structure:
   .ralph/
     ├── prd.json           # Task list (source of truth)
     ├── requirements.md    # Original requirements
     ├── prompts/
     │   ├── SETUP_PROMPT.md
     │   └── LOOP_PROMPT.md
     └── configs/
         └── default.json   # Step configuration
   <project>/
     └── .git/              # Git repo for app code
```

### `ralph run`

```
1. Validate preflight (config exists, prd.json has tasks)
2. Acquire lock (.ralph/.ralph_lock)
3. Print banner
4. Loop:
   a. For each enabled step in config:
      - Execute step
      - If agent exits (all done / stuck), break
   b. Repeat until exit condition
5. Print metrics (tokens, cost, time)
6. Write aggregate.json
7. Release lock
```

### `ralph add`

```
1. Archive completed tasks → .ralph/prd_archive.json
2. Compact learnings.md if large (Claude summarizes)
3. Read current prd.json
4. Call Claude (opus) with work description → new tasks
5. Append new tasks to prd.json
6. Print new tasks
```

## Task Lifecycle

```
                    ┌──────────────────┐
                    │   requirements   │
                    │      .md         │
                    └────────┬─────────┘
                             │ ralph init
                             ▼
┌────────────┐     ┌──────────────────┐     ┌────────────────┐
│ work.md    │ --> │   .ralph/        │ --> │  prd_archive   │
│ (ralph add)│     │   prd.json       │     │  .json         │
└────────────┘     │   (tasks)        │     │  (completed)   │
                   └────────┬─────────┘     └────────────────┘
                            │
                            ▼
              ┌─────────────────────────┐
              │      ralph run          │
              │  ┌───────────────────┐  │
              │  │ Loop iteration 1  │  │
              │  │ - Read prd.json   │  │
              │  │ - Work on task    │  │
              │  │ - Mark done       │  │
              │  │ - Write learnings │  │
              │  └───────────────────┘  │
              │  ┌───────────────────┐  │
              │  │ Loop iteration 2  │  │
              │  │ - Fresh session   │  │
              │  │ - Next task       │  │
              │  └───────────────────┘  │
              │          ...            │
              └─────────────────────────┘
```

## Context Flow

Each loop iteration receives:

| Source | Content | Size limit |
|--------|---------|------------|
| `LOOP_PROMPT.md` | Base instructions | Full file |
| `--append-system-prompt` | Loop #, task progress, learnings | ~2500 chars |
| Claude reads | `.ralph/prd.json` | Full file |
| Claude reads | `.ralph/learnings.md` | Full file |

**Learnings** persist across sessions. Claude writes discoveries to `.ralph/learnings.md`, and they're injected into subsequent iterations.

## Configuration

`default.json` structure:

```json
{
  "name": "default",
  "steps": [
    {
      "name": "setup",
      "type": "agent",
      "enabled": true,
      "config": {
        "prompt_file": ".ralph/prompts/SETUP_PROMPT.md",
        "prd_file": ".ralph/prd.json",
        "marker_file": ".ralph/.setup_done"
      }
    },
    {
      "name": "run-claude",
      "type": "agent",
      "enabled": true,
      "config": {
        "prompt_file": ".ralph/prompts/LOOP_PROMPT.md",
        "prd_file": ".ralph/prd.json"
      }
    },
    {
      "name": "git-commit",
      "type": "git-commit",
      "enabled": true
    }
  ]
}
```

## Exit Conditions

The agent step exits when:

1. **All tasks complete** — every task in prd.json has status "done"
2. **Stuck detection** — same task attempted multiple times without progress
3. **Timeout** — step exceeds configured timeout (default 15m)
4. **Error** — Claude CLI fails or returns error

## File Layout

```
myproject/                    # Created by ralph init
├── .ralph/
│   ├── prd.json             # Task list (source of truth)
│   ├── prd_archive.json     # Archived completed tasks
│   ├── requirements.md      # Original requirements
│   ├── learnings.md         # Cross-session context
│   ├── prompts/
│   │   ├── SETUP_PROMPT.md
│   │   └── LOOP_PROMPT.md
│   ├── configs/
│   │   └── default.json
│   ├── logs/                # Claude output logs
│   ├── run_state.json       # Run ID, loop count, status
│   ├── aggregate.json       # Final metrics (version, model, tokens, cost)
│   └── .ralph_lock
├── myproject/               # Application code (nested)
│   ├── .git/
│   ├── app.py
│   └── ...
```

## Model Defaults

| Command | Default model | Override |
|---------|--------------|----------|
| `ralph init` | opus | `--model` |
| `ralph add` | opus | `--model` |
| `ralph run` | sonnet | `--model` |
