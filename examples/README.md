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

## Example: Flask Day Server

```bash
ralph init flasky examples/flask_requirements.md
cd flasky
ralph run
```

After Ralph finishes, follow the generated app README:

```bash
cd flasky
cat flasky/README.md
```
