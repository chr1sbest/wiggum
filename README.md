# wiggum

<p align="left">
  <img src="assets/ralph.png" alt="ralph" width="250" />
</p>

Fully autonomous [Ralph Loop](https://ghuntley.com/ralph/) built to avoid context rot on Claude Code sessions.

`while :; do cat PROMPT.md | claude-code ; done`

## Install

### Homebrew

```bash
brew tap chr1sbest/tap
brew install ralph
```

### Requirements

- Claude Code must be installed and available on your `PATH` as `claude`.
- Ralph invokes Claude Code with **tool access enabled** (unsafe mode).

## Usage

Write requirements in an .md file (see /examples) then have Ralph get to work.

```bash
# Create a new project directory
mkdir myproject && cd myproject

# Initialize Ralph with your requirements
ralph init requirements.md

# Run the Ralph loop
ralph run
```

Ralph works like `git init` — run it in the directory where you want to work. It creates a `.ralph/` folder with your task list and configuration, and adds `.ralph/` to your `.gitignore`.

### Examples

- `examples/flask_requirements.md` - minimal Flask webapp to display the current day

```bash
mkdir flasky && cd flasky
ralph init ../examples/flask_requirements.md
ralph run
```

### Attach to existing project

Ralph works with existing codebases too. Just run `ralph init` without a requirements file — Ralph will explore the codebase and generate a summary:

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
```

`add` will:
- call Claude to translate your request into tasks
- update `.ralph/prd.json` (also conditionally compact and archive)
- print the new tasks to stdout

Next step:

```bash
ralph run
```
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

Yes. `init` requires a markdown requirements file.

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

Claude output logs are written to `.ralph/logs/`.

### Claude usage limit / rate limit

If you hit a quota limit, wait for your quota to reset and rerun `ralph run`.