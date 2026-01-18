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
# Create a new project using pre-defined requirements
ralph init myproject requirements.md

# Run the Ralph loop
cd myproject
ralph run
```

### Examples

- `examples/flask_requirements.md` - minimal Flask webapp to display the current day

```bash
ralph init flasky examples/flask_requirements.md
cd flasky
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

Projects created by `ralph init` include:
- `.ralph/prd.json` - Task tracking (source of truth)
- `.ralph/prd_archive.json` - Archived completed tasks (created/updated by `ralph add`)
- `.ralph/requirements.md` - Project requirements
- `.ralph/learnings.md` - Persistent learnings across loop iterations (created/updated over time)
- `.ralph/prompts/SETUP_PROMPT.md` - Setup prompt (initial scaffolding / planning)
- `.ralph/prompts/LOOP_PROMPT.md` - Worker loop prompt (iterative implementation)
- `.ralph/configs/default.json` - Loop configuration
- `.ralph/logs/` - Claude output logs
- `.ralph/run_state.json` - Current run state
- `.ralph/run_metrics.json` - Token/cost/time metrics
- `.ralph/progress.json` - Status/progress for the terminal UI
- `.ralph/.ralph_lock` - Prevents concurrent runs
- `./<project>/<project>/.git` - Git repository for application code

## FAQ

### Do I have to pass a requirements file to `init`?

Yes. `init` requires a markdown requirements file.

### I ran `ralph run` and it says files are missing

Run `ralph run` from the project root created by `ralph init` (the directory containing `.ralph/` and the nested application code directory).

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