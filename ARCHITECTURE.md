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

---

## Contributing: Where to Change What

This section helps contributors understand where to make common modifications.

### Adding a New Command

**Location:** `cmd/ralph/`

**Files to modify:**
1. `main.go` - Add case to switch statement and call your command function
2. Create `cmd_yourcommand.go` - Implement command logic following pattern:
   ```go
   func yourCommandCmd(args []string) {
       fs := flag.NewFlagSet("yourcommand", flag.ExitOnError)
       // Add flags
       fs.Parse(args)
       // Implement command
   }
   ```
3. Update `printUsage()` in `main.go` to document your command

**Example:** See `cmd_fix.go` or `cmd_add.go`

### Adding a New Step Type

**Location:** `internal/loop/steps/`

**Files to modify:**
1. Create `yourstep.go` implementing the `Step` interface:
   ```go
   type Step interface {
       Name() string
       Execute(ctx context.Context, rawConfig json.RawMessage) error
   }
   ```
2. Register in `cmd/ralph/cmd_run.go`:
   ```go
   registry.Register("yourstep", func() loop.Step {
       return steps.NewYourStep()
   })
   ```
3. Add step to config files (`configs/default.json` or user's `.ralph/configs/`)

**Example:** See `readme_check.go` or `noop.go`

### Modifying Loop Behavior

**Location:** `internal/loop/loop.go`

**Common changes:**
- **Exit conditions:** Check `Run()` and `RunOnce()` methods
- **Step execution order:** Modify `Run()` to change how steps are iterated
- **Circuit breaker logic:** See `circuitBreakers` field and `executeStepWithResilience()`
- **Task tracking:** Modify `currentTaskID` and `loopsOnTask` logic

**Files to understand:**
- `internal/loop/loop.go` - Main orchestration
- `internal/resilience/circuit_breaker.go` - Fault tolerance
- `internal/config/types.go` - Configuration structure

### Changing Prompts

**Location:** `.ralph/prompts/` (per-project)

**Files:**
- `SETUP_PROMPT.md` - First-time setup instructions (runs once)
- `LOOP_PROMPT.md` - Main loop instructions (runs each iteration)

**Templates:** `configs/LOOP_PROMPT.md` and `configs/SETUP_PROMPT.md` (defaults used by `ralph init`)

**Key points:**
- Prompts are project-specific (in `.ralph/prompts/`)
- Template defaults live in `configs/` directory at repo root
- Changes to templates only affect new projects
- To update existing project, edit `.ralph/prompts/` files directly

### Modifying Configuration Schema

**Location:** `internal/config/types.go`

**Files to modify:**
1. `types.go` - Add fields to `Config` or `StepConfig` structs
2. `loader.go` - Update validation logic if needed
3. `configs/default.json` - Update template with new fields
4. Documentation - Update this file and README

**Safety:** Always maintain backward compatibility for existing config files

### Changing Task Status Logic

**Location:** `internal/agent/prd_status.go`

**Common changes:**
- Task state transitions (todo → in_progress → done → failed)
- Task filtering (finding next task to work on)
- Completion detection (when to exit loop)

**Related files:**
- `internal/agent/exit_detector.go` - Detects when to stop loop
- `internal/loop/loop.go` - Enforces max loops per task

### Adding Telemetry/Metrics

**Location:** `internal/tracker/`

**Files:**
- `metrics.go` - Add new metric fields to `Metrics` struct
- `loop.go` - Update `RecordClaudeCall()` or add new recording methods
- `cmd_run.go` - Update `printRunMetrics()` to display new metrics

**Persistence:** Metrics are written to `.ralph/aggregate.json`

### Modifying Claude Invocation

**Location:** `internal/loop/steps/agent.go`

**Key areas:**
- `Execute()` method - Main Claude invocation logic
- `buildClaudeCommand()` - Command-line flag construction
- `AgentConfig` struct - Configuration options
- Exit detection - Integrated with `agent.ExitDetector`

**Related:**
- `cmd/ralph/claude_helpers.go` - Shared Claude CLI utilities
- `.ralph/prompts/LOOP_PROMPT.md` - Instructions given to Claude

---

## Entrypoint Flow

Understanding the code path from `ralph run` to Claude execution:

```
ralph run
    ↓
main.go (main)
    ↓
cmd_run.go (runCmd)
    ├── Parse flags (--config, --model, --once)
    ├── Validate preflight checks
    │   ├── Check required files exist
    │   └── Verify claude CLI is installed
    ├── Load PRD and check task status
    ├── Initialize components
    │   ├── StepRegistry (register all step types)
    │   ├── Config loader (.ralph/configs/default.json)
    │   ├── Loop executor
    │   └── Tracker (locking, metrics)
    ├── Acquire lock (.ralph/.ralph_lock)
    ├── Print banner
    └── Execute run mode
        ├── runOnce (--once flag) → Loop.RunOnce()
        └── runContinuous (default) → Loop.Run()
            ↓
        internal/loop/loop.go (Run/RunOnce)
            ↓
        For each step in config:
            ├── Check circuit breaker
            ├── Execute step with timeout/retry
            │   ↓
            │   internal/loop/steps/agent.go (Execute)
            │       ├── Load PRD status
            │       ├── Check exit conditions
            │       ├── Build Claude command
            │       │   ├── Read prompt file (.ralph/prompts/LOOP_PROMPT.md)
            │       │   ├── Append system context (loop #, progress)
            │       │   └── Add flags (--append-system-prompt, --model, etc.)
            │       ├── Execute: claude -p <prompt> [flags]
            │       ├── Parse output (JSON or text)
            │       ├── Record metrics (tokens, cost)
            │       └── Check exit again
            ├── Record step result
            └── Check for agent exit
                ├── All tasks done → Exit loop
                └── Stuck on task → Mark failed, continue
            ↓
        Write metrics, release lock, exit
```

**Key decision points:**
1. **Preflight validation** (`cmd_run.go:229-270`) - Ensures environment is ready
2. **Lock acquisition** (`cmd_run.go:123`) - Prevents concurrent runs
3. **Step execution** (`loop.go:145-220`) - Orchestrates steps with resilience
4. **Exit detection** (`agent.go`, `agent/exit_detector.go`) - Determines when to stop

---

## Configuration and Prompts

### Configuration Files

**Location:** `.ralph/configs/`

**Structure:**
```json
{
  "name": "default-loop",
  "description": "Description of config",
  "max_loops_per_task": 10,  // Optional: limit iterations per task
  "steps": [
    {
      "type": "agent",           // Step type (must be registered)
      "name": "run-claude",      // Display name
      "enabled": true,           // Can disable without removing
      "timeout": "20m",          // Max execution time
      "max_retries": 1,          // Retry failed steps
      "retry_delay": "30s",      // Wait between retries
      "continue_on_error": false, // Keep going if step fails
      "circuit_breaker": {       // Optional fault tolerance
        "threshold": 3,          // Failures before opening circuit
        "reset_after": "60s"     // Cool-down period
      },
      "config": {                // Step-specific config (varies by type)
        "prompt_file": ".ralph/prompts/LOOP_PROMPT.md",
        "prd_file": ".ralph/prd.json",
        "model": "sonnet",
        "timeout": "15m",
        "allowed_tools": "Write,Read,Edit,Glob,Grep,Bash,Task,TodoWrite"
      }
    }
  ]
}
```

**Default template:** `configs/default.json` (repo root) - copied during `ralph init`

**Environment substitution:** Config loader supports `${ENV_VAR}` syntax

### Prompt Files

**Location:** `.ralph/prompts/`

**Files:**
- **`SETUP_PROMPT.md`** - One-time setup instructions
  - Runs only once (creates `.ralph/.ralph_setup_done` marker)
  - Purpose: Initialize project structure, README, basic files
  - Typically shorter, focused on scaffolding

- **`LOOP_PROMPT.md`** - Main work instructions
  - Runs every loop iteration
  - Purpose: Task execution, coding, testing, documentation
  - Contains full "Ralph Loop" instructions

**Context injected via `--append-system-prompt`:**
- Loop number
- Task progress (completed / total)
- Current task details
- Recent learnings from `.ralph/learnings.md`

**Templates:** Default versions in `configs/` directory (used by `ralph init`)

**Customization:** Edit project-specific files in `.ralph/prompts/` to change behavior

---

## Safety Guardrails

Ralph includes multiple safety mechanisms to prevent runaway processes and ensure safe operation.

### 1. File Locking

**Location:** `internal/tracker/lock.go`

**Mechanism:**
- Creates `.ralph/.ralph_lock` file when `ralph run` starts
- Contains process ID and run ID
- Prevents concurrent `ralph run` processes in same project
- Automatically released on exit (or manual cleanup if process crashes)

**Override:** Remove `.ralph/.ralph_lock` manually if lock is stale (e.g., after force-kill)

### 2. Loop Limits

**Configuration:** `max_loops_per_task` in config file

**Behavior:**
- Tracks how many loop iterations have been attempted on the same task
- If limit exceeded, marks task as "failed" and moves to next task
- Prevents infinite loops on stuck tasks
- Default: 0 (no limit - not recommended for production)
- Recommended: 5-10 iterations per task

**Location:** `internal/loop/loop.go` - `currentTaskID` and `loopsOnTask` fields

**Override:** Set `max_loops_per_task: 0` in config (disables limit)

### 3. Timeouts

**Levels:**
1. **Step timeout** (`config.steps[].timeout`) - Max time for entire step
2. **Agent timeout** (`config.steps[].config.timeout`) - Max time for Claude execution
3. **Command context timeout** - OS-level process timeout

**Default:** 15-20 minutes per step

**Behavior:** If exceeded, step is cancelled and potentially retried (if `max_retries > 0`)

**Location:**
- `internal/config/types.go` - `StepConfig.GetTimeout()`
- `internal/loop/loop.go` - Timeout enforcement in `executeStepWithResilience()`

### 4. Circuit Breakers

**Location:** `internal/resilience/circuit_breaker.go`

**Mechanism:**
- Tracks consecutive failures per step
- After N failures (threshold), "opens circuit" - step is skipped temporarily
- After cool-down period (reset_after), attempts "half-open" - tries one execution
- If succeeds, "closes circuit" - normal operation resumes
- If fails, reopens circuit for another cool-down period

**Configuration:**
```json
{
  "circuit_breaker": {
    "threshold": 3,      // Failures before opening
    "reset_after": "60s" // Cool-down duration
  }
}
```

**Purpose:** Prevents repeated failures from wasting time/resources

### 5. Exit Detection

**Location:** `internal/agent/exit_detector.go`

**Conditions that stop the loop:**
1. **All tasks complete** - Every task in `prd.json` has status "done"
2. **Stuck detection** - Same task attempted multiple times without progress
3. **Explicit failure** - Task marked as "failed" and no more todos
4. **User interrupt** - SIGINT (Ctrl+C) or SIGTERM

**Check frequency:** After every step execution

**Location:** `internal/loop/steps/agent.go` - Checks before and after Claude execution

### 6. Stuck Task Detection

**Logic:**
- If a task remains "in_progress" for multiple iterations
- And no code changes are detected (via git status)
- Task is marked "failed" after max loops per task
- Loop continues with next task

**Related:** Works in conjunction with `max_loops_per_task` limit

### 7. Safe Mode (Default Behavior)

**Restrictions:**
- Only modifies files in current directory and subdirectories
- Requires git repository for tracking changes
- Validates PRD structure before starting
- Checks Claude CLI is installed and working

**Preflight checks** (`cmd_run.go:229-270`):
- `.ralph/prd.json` exists and has tasks
- `.ralph/requirements.md` exists
- Prompt files exist
- Config file is valid
- Claude binary is available and runnable

### 8. Cost Tracking

**Location:** `internal/tracker/metrics.go`

**Metrics collected:**
- Total Claude API calls
- Input/output tokens
- Estimated cost (USD)
- Elapsed time

**Purpose:** Helps users monitor spend and prevent surprise bills

**Output:** Displayed at end of run, written to `.ralph/aggregate.json`

### Unsafe Mode (Not Yet Implemented)

**Planned features:**
- Skip certain validations
- Allow operations outside project directory
- Disable loop limits

**Current status:** All runs operate in "safe mode" - no unsafe mode available yet

**If needed:** Modify code directly or adjust config timeouts/limits

---

## Configuration Best Practices

For contributors and advanced users:

1. **Start with defaults** - `configs/default.json` is well-tested
2. **Set loop limits** - Always use `max_loops_per_task: 5-10` in production
3. **Configure timeouts** - Adjust based on project complexity (simple: 5m, complex: 20m)
4. **Enable circuit breakers** - Prevent wasted iterations on consistently failing steps
5. **Use retries sparingly** - `max_retries: 1-2` for transient failures only
6. **Test config changes** - Use `ralph run --once` to validate before full run
7. **Monitor costs** - Check `.ralph/aggregate.json` after runs
