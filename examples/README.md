# Examples

This folder contains sample `requirements.md` files you can use with `ralph init`.

## Quick start

From the repo root:

```bash
ralph init myproject examples/flask_requirements.md
cd myproject
ralph run
```

## What gets created

`ralph init <project> <requirements.md>` creates:

- `.ralph/` - Ralph scaffolding and run artifacts
  - `.ralph/prd.json` - task list (source of truth)
  - `.ralph/requirements.md` - a copy of your requirements
  - `.ralph/prompts/` - prompts used by the loop
  - `.ralph/configs/default.json` - default loop config
  - `.ralph/logs/` - Claude output logs
- `./<project>/` - application code directory (this is the git repo)
  - `./<project>/.git`

You should run `ralph run` from the **project root** (the directory created by `ralph init`, i.e. the folder containing `.ralph/` and `./<project>/`).

## Available Examples

| File | Description | Complexity |
|------|-------------|------------|
| `flask_requirements.md` | Simple Flask app displaying current day | Low (~3 tasks) |
| `log_aggregator_requirements.md` | Go CLI for parsing, filtering, alerting on logs | High (~18 tasks) |
| `static_site_generator_requirements.md` | Python CLI static site builder with templates | High (~20 tasks) |

## Example: Flask Day Server (simple)

```bash
ralph init flasky examples/flask_requirements.md
cd flasky
ralph run
```

## Example: Log Aggregator CLI (complex)

A Go CLI tool for parsing, filtering, and alerting on log files. Good for comparing Ralph vs one-shot prompts.

```bash
ralph init logagg examples/log_aggregator_requirements.md
cd logagg
ralph run
```

See `evals/` for scripts to benchmark Ralph against single-shot Claude prompts.

## After Ralph finishes

Follow the generated app README:

```bash
cd <project>/<project>
cat README.md
```
