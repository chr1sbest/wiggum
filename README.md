# Wiggum 🍩🤖

<p align="left">
  <img src="assets/ralph.png" alt="ralph" width="250" />
</p>

Hi! I'm Ralph Wiggum and I help your codebase do stuff. Sometimes I even do the *right* stuff.

This project is a "Ralph Loop" (aka `while :; do cat PROMPT.md | claude-code ; done`)… **with some bells and whistles**.

## ELI5: What's the Ralph Loop?
You give Ralph a goal. Ralph tries. Then Ralph tries again.  
Ralph keeps going until the work is actually done (or until the guardrails say "nap time").

### What Ralph is good at
- **Autonomous coding:** He can iterate on a project instead of stopping after one attempt.
- **Tiny brain on purpose:** Ralph "forgets" between runs — fresh starts, less context rot, git/files become the memory.
- **Breaking big work into chunks:** Ralph tracks tasks in `prd.json`, so big scary stuff becomes little checkbox stuff.
- **Guardrails:** Locking + run artifacts live in `.ralph/` so Ralph doesn't stampede your terminal forever.

## Install

### Homebrew

```bash
brew tap chr1sbest/tap
brew install ralph
```

### Requirements

- **Claude Code CLI** must be installed and available on your `PATH` as `claude`.
  - Install: https://code.claude.com/docs/en/setup
  - Quick install: `curl -fsSL https://claude.ai/install.sh | bash`
  - Verify: `claude --help`
- Ralph invokes Claude Code with **tool access enabled** (unsafe mode).

## Usage

Ralph works like `git init` — run it in the directory where you want to work. It creates a `.ralph/` folder with your task list and configuration, and adds `.ralph/` to your `.gitignore`.

### New Projects

Write requirements in a markdown file (see `examples/`) then have Ralph build it from scratch:

```bash
mkdir myproject && cd myproject
ralph init requirements.md    # Creates .ralph/ and generates tasks
ralph run                     # Ralph builds the project
```

**Example:**

```bash
mkdir flasky && cd flasky
ralph init ../examples/flask_requirements.md
ralph run
```

### Existing Projects

Ralph works with existing codebases too. Run `ralph init` without a requirements file — Ralph will explore the codebase and generate a summary:

```bash
cd my-existing-repo
ralph init                    # Ralph explores and summarizes
ralph add "Add unit tests"    # Add work items
ralph run
```

### Add new work

Use `add` to translate a work request into new tasks and append them to `.ralph/prd.json`.

```bash
cd myproject

# Option A: from a markdown file
ralph add new-work.md

# Option B: inline
ralph add "Add an endpoint that returns the user's country based on IP"

# Option C: from a GitHub issue
ralph fix --issue 42
```

`add` and `fix` will:
- call Claude to translate your request into tasks
- update `.ralph/prd.json` (also conditionally compact and archive)
- print the new tasks to stdout

When using `fix`, tasks include the issue reference so commits automatically close the GitHub issue with "Fixes #N".

Next step:

```bash
ralph run
```

### Open a pull request

Use `pr` to push your branch and open a pull request with an auto-generated summary:

```bash
# Create a feature branch
git checkout -b feature/my-feature

# Do work
ralph run

# Push and open PR
ralph pr
```

Options:
- `--draft` - Create as draft PR
- `--title "Custom title"` - Override auto-generated title
- `--base develop` - Target a different base branch (default: main)

The PR body is auto-generated from `.ralph/prd.json`, listing completed tasks and any linked GitHub issues.

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `ralph init [requirements.md]` | Initialize project with generated task list | `-requirements`, `-model` |
| `ralph run` | Execute the autonomous loop | None |
| `ralph add <work>` | Add new work items to PRD | `-file`, `-desc`, `-model` |
| `ralph fix --issue N` | Create tasks from GitHub issue | `--issue`, `--repo`, `-model` |
| `ralph pr` | Push branch and open PR with auto-generated summary | `--title`, `--draft`, `--base` |
| `ralph upgrade` | Check for updates and upgrade Ralph | `-yes` |
| `ralph version` | Show Ralph's version number | None |

For detailed help on any command, run `ralph <command> -h`.

## Architecture

```
cmd/ralph/           # CLI entry point and commands
internal/
├── loop/            # Loop execution engine with pluggable steps
│   └── steps/       # Step implementations (agent, git-commit, command, readme-check, noop)
├── agent/           # Session and PRD management
├── banner/          # Welcome banner display
├── config/          # JSON config loading with hot-reload
├── tracker/         # Run state, metrics, and file locking
├── resilience/      # Circuit breaker and retry logic
├── status/          # Terminal UI progress display
└── logger/          # Structured logging
configs/             # Default configuration templates
examples/            # Sample requirements files
evals/               # Evaluation/testing framework
```

### Key Components

- **cmd/ralph/main.go** - CLI dispatcher routing commands (run, init, add, fix, upgrade)
- **internal/loop/loop.go** - Core loop orchestrator executing steps iteratively
- **internal/loop/steps/agent.go** - Claude Code invocation step with exit detection
- **internal/agent/prd_status.go** - PRD task parsing (todo → in_progress → done → failed)
- **internal/config/loader.go** - Configuration management with JSON config files
- **internal/tracker/lock.go** - File-based locking preventing concurrent runs
- **internal/resilience/circuit_breaker.go** - Fault tolerance with exponential backoff

## Development

### Installation from source

```bash
go install github.com/chr1sbest/wiggum/cmd/ralph@latest
```

For local iteration:

```bash
go install ./cmd/ralph
```

If you build from source locally, `ralph version` may show `dev`. Official releases stamp the version at build time.

### Upgrade behavior

- If installed via `go install`, use `ralph upgrade`.
- If installed via Homebrew, upgrade with:

```bash
brew update
brew upgrade ralph
```

### Project Structure

`ralph init` creates a `.ralph/` directory in your project:

```
your-project/
├── .ralph/
│   ├── prd.json              # Task list (source of truth)
│   ├── requirements.md       # Original requirements
│   ├── configs/default.json  # Loop configuration
│   ├── prompts/              # SETUP_PROMPT.md, LOOP_PROMPT.md
│   ├── logs/                 # Claude output logs
│   ├── run_state.json        # Current run state
│   ├── run_metrics.json      # Token/cost/time metrics
│   └── .ralph_lock           # Prevents concurrent runs
├── your code files...        # Application code goes here
└── README.md
```

### Task Statuses

Tasks in `prd.json` progress through these statuses:

- **`todo`** - Task has not been started yet
- **`in_progress`** - Task is currently being worked on
- **`done`** - Task completed successfully
- **`failed`** - Task exceeded max_loops_per_task without completion

Ralph automatically picks the first `todo` task and moves it through these states. If a task takes too many loop iterations without completing, it's marked as `failed` to prevent infinite loops.

### Session Management

Ralph maintains session persistence to optimize Claude Code interactions and provide crash recovery:

#### Session Persistence (`.ralph_session`)

Ralph tracks Claude Code sessions across loop iterations to maintain context continuity:

- **Session tracking:** Each session has a unique ID, creation timestamp, and loop count
- **Auto-expiry:** Sessions expire after a configurable period (default: 24 hours) to prevent context staleness
- **Fresh starts:** When a session expires, Ralph creates a new session ID and resets the exit detector
- **Session history:** Session transitions are logged to `.ralph_session_history` (last 50 transitions kept)

This allows Ralph to maintain context across related work while automatically resetting when appropriate. The session file is configured per-step in `.ralph/configs/default.json` via the `session_file` parameter.

#### Run State Persistence (`.ralph/run_state.json`)

Ralph maintains detailed run state for crash recovery and monitoring:

- **Process tracking:** Records PID, run ID, start time, and last update timestamp
- **Step tracking:** Tracks current step name, step start time, and last successful step
- **Task tracking:** Records current task ID and description being worked on
- **Error tracking:** Stores last error message for debugging failed runs
- **Atomic writes:** Uses temporary files with atomic rename to prevent corruption

If Ralph crashes or is killed, the run state file contains enough information to understand what was being worked on and where the failure occurred. The loop number and current step help identify exactly where execution stopped.

## FAQ

### Do I have to pass a requirements file to `init`?

No. For new projects, pass a requirements file. For existing projects, run `ralph init` without arguments — Ralph will explore and summarize the codebase.

### I ran `ralph run` and it says files are missing

Run `ralph run` from the directory where you ran `ralph init` (the directory containing `.ralph/`).

### Ralph says the lock is held

Ralph uses a lock file to prevent concurrent runs. The lock is stored in `.ralph/.ralph_lock`. If a previous run crashed, the lock may be stale. You can remove it:

```bash
rm -f .ralph/.ralph_lock
```

### What is `.ralph/`?

`.ralph/` contains run artifacts (run state, metrics, status/progress, lock) so the project root stays clean.

### Where do Claude logs go?

Claude output logs are written to `.ralph/logs/`:
- `loop_N.json` - Full Claude output (tokens, cost, session info, result)
- `loop_N.md` - Clean markdown summary of what Claude accomplished
- If output isn't valid JSON, falls back to timestamped `.log` files

### Claude usage limit / rate limit

If you hit a quota limit, wait for your quota to reset and rerun `ralph run`.
