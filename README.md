# wiggum

<p align="center">
  <img src="assets/ralph.png" alt="ralph" width="250" />
</p>

Ralph Loop (`while :; do cat PROMPT.md | claude-code ; done`) with some bells and whistles.

## Install

```bash
go install github.com/chris/go_ralph/cmd/ralph@v1.0.0
```

## Usage

Write requirements in an .md file (see /examples) then have Ralph get to work.

```bash
# Create a new project using pre-defined requirements
ralph new-project myproject requirements.md

# Run the Ralph loop
cd myproject
ralph run
```

### Examples

- `examples/flask_requirements.md` - minimal Flask example requirements file

```bash
ralph new-project flasky examples/flask_requirements.md .
```

### Add new work

Use `new-work` to translate a work request into new tasks and append them to `prd.json`.

```bash
cd myproject

# Option A: from a markdown file
ralph new-work ../work.md

# Option B: inline
ralph new-work "Add an endpoint that returns the user's country based on IP"
```

`new-work` will:
- call Claude to translate your request into tasks
- update `prd.json`
- print the new tasks to stdout

Next step:

```bash
ralph run
```

## Project Structure

Projects created by `ralph new-project` include:
- `prd.json` - Task tracking (source of truth)
- `requirements.md` - Project requirements
- `SETUP_PROMPT.md` - Setup prompt (initial scaffolding / planning)
- `LOOP_PROMPT.md` - Worker loop prompt (iterative implementation)
- `configs/default.json` - Loop configuration
- `logs/` - Claude output logs

## FAQ

### Do I have to pass a requirements file to `new-project`?

Yes. `new-project` requires a markdown requirements file:

```bash
ralph new-project myproject requirements.md
```

### I ran `ralph run` and it says files are missing

Run `ralph run` from the project root created by `ralph new-project` (the directory containing `prd.json`, `requirements.md`, and `configs/`).

### Ralph says the lock is held

Ralph uses a lock file to prevent concurrent runs. The lock is stored in `.ralph/.ralph_lock`.

If a previous run crashed, the lock may be stale. You can remove it:

```bash
rm -f .ralph/.ralph_lock
```

### What is `.ralph/`?

`.ralph/` contains run artifacts (run state, metrics, status/progress, lock) so the project root stays clean.

### Where do Claude logs go?

Claude output logs are written to `logs/`.

### Claude usage limit / rate limit

If you hit a quota limit, wait for your quota to reset and rerun `ralph run`.
