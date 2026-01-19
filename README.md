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

- Claude Code must be installed and available on your `PATH` as `claude`.
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
