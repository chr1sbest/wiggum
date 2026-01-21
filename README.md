# Wiggum 🤖

*"I'm in danger"* - Ralph Wiggum, and every software engineer in 2026

<p align="left">
  <img src="assets/ralph.png" alt="ralph" width="250" />
</p>

## Install

```bash
brew tap chr1sbest/tap
brew install ralph
```

## What's the Ralph Wiggum Loop?

The Ralph Wiggum Loop is `while :; do cat PROMPT.md | claude-code ; done`

This simple bash script attempts to enable fully-automated development while addressing several problems cited by Anthropic's ["Effective harnesses for long-running agents."](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)

Most of the magic is within `PROMPT.md`, but the surrounding `while` loop aims to address the context rot of long-running agents by resetting sessions between each attempt. Through tight scoping and well-defined success criteria (guided by `PROMPT.md`), Ralph can iteratively work through problems, resetting and rebuilding context for each task.

## What's the purpose of this project?

This project aims to minimally build upon Ralph's intentions of *simplicity* and *fully automated development* while addressing common pitfalls of agent orchestration. This implementation adds guardrails, evaluation frameworks, and monitoring for time and cost.

In short, this implementation:

1. Can be used in existing or new codebases with simple commands

2. Can reliably execute and finish work without human intervention

3. Answers the question of "is the tradeoff of context reset worth the extra token cost?"

## Comparisons

### Official Claude Ralph Loop Plugin

The "Official" [ralph-wiggum](https://github.com/anthropics/claude-code/tree/main/plugins/ralph-wiggum) plugin for Claude maintains a session *across* runs instead of resetting context *between* runs. This plugin is not technically a "Ralph Loop", but it effectively applies best practices around scoping and well-defined success criteria.

**Official Plugin**
- No Context Resets
- Callable from within Claude only, limited monitoring
- LLM-triggered exit conditions

**This Project**
- Default to Context Resets
- Callable from Command line with monitoring
- Deterministic Exit Conditions

### Simple Bash Script

Originally published by [Geoffrey Huntley](https://ghuntley.com/ralph/), this simple `while :; do cat PROMPT.md | claude-code ; done` script will run infinitely until a human intervenes.

**Bash Script**
- Requires human intervention to stop
- No guardrails
- No tracking or metriccs
- Infinite cost

**This Project**
- Has clear exit conditions to end automation
- Has guardrails, circuit breakers, and timeouts
- Monitors cost and token usage

## Usage

Ralph works like `git init` — run it in the directory where you want to work. It creates a `.ralph/` folder with your task list and configuration, and adds `.ralph/` to your `.gitignore`.

### Requirements

- **[Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code/overview)** must be installed and available on your `PATH` as `claude`.
- Ralph invokes Claude Code with **tool access enabled** (unsafe mode).

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

### Run evaluations

Use `eval` to run evaluation suites and compare ralph vs oneshot approaches:

```bash
# List available evaluation suites
ralph eval list

# Run an evaluation suite
ralph eval run flask --approach ralph

# Compare results between approaches
ralph eval compare flask
```

Evaluation suites are defined in `evals/suites/` and results are saved to `evals/results/`.

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

## Contributing (5 min quickstart)

We provide a Makefile for common dev tasks. Run `make help` to see all available targets.

**Key directories:**
- `cmd/ralph/` - CLI entry point and command routing
- `internal/` - Core logic (loop engine, PRD management, session control)

See [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines including PR process and code style requirements.

## License

MIT License - see [LICENSE](LICENSE) file for details.
