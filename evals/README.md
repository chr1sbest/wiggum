# Ralph Evals

Evaluation framework for comparing agent frameworks on code generation tasks.

## Design Principles

When writing evals, follow these principles:

- **Determine winner between multiple agent frameworks** given the same input requirements
- **Scripts to run any framework** - oneshot, ralph, or other agent frameworks
- **Outputs metrics and a working product** - not just metrics, but actual runnable code
- **Shared test framework** - same tests run against all implementations for fair comparison

## Quick Start (New Unified Framework)

```bash
# Run both approaches on the same suite
./run_eval.sh tasktracker ralph sonnet
./run_eval.sh tasktracker oneshot sonnet

# Compare results (uses shared test suite for fair comparison)
./compare_evals.sh tasktracker
```

## Key Concept: Shared Test Suites

The framework uses **shared test suites** in `suites/<name>/` that are run against BOTH ralph and oneshot outputs. This ensures fair comparison - both implementations are tested against the same requirements.

```
evals/
├── suites/
│   └── tasktracker/       # Shared test suite
│       ├── conftest.py    # Fixtures
│       ├── test_api.py    # API tests
│       └── test_validation.py
├── run_eval.sh            # Unified runner
└── compare_evals.sh       # Comparison tool
```

## Legacy Scripts

```bash
# Run Ralph eval (legacy)
./run_ralph_eval.sh sonnet

# Run one-shot eval (legacy)
./run_oneshot_eval.sh sonnet

# Validate both (after they complete)
./validate.sh ../eval-ralph-tasktracker-sonnet-<timestamp>/tasktracker
./validate.sh ../eval-oneshot-tasktracker-sonnet-<timestamp>/tasktracker

# Compare results (legacy)
./compare.sh eval-ralph-tasktracker-sonnet-<timestamp> eval-oneshot-tasktracker-sonnet-<timestamp>
```

## Scripts

| Script | Purpose |
|--------|---------|
| `run_ralph_eval.sh [model]` | Run Ralph on task_tracker_requirements.md |
| `run_oneshot_eval.sh [model]` | Run single Claude prompt on same requirements |
| `validate.sh <project_dir>` | Test if generated app works (endpoints, tests) |
| `compare.sh <ralph> <oneshot>` | Compare two eval runs side-by-side |

## What's Measured

- **Time**: Total elapsed seconds
- **Validation**: Automated checks (app starts, endpoints work, tests pass)
- **Tokens**: Input/output token usage (Ralph only, via run_metrics.json)
- **Cost**: Estimated USD (Ralph only)

## Eval Project: Task Tracker API

A REST API with:
- JWT authentication
- 4 data models (User, Project, Task, Comment)
- ~15 endpoints with permission checks
- Input validation
- Comprehensive tests

See `examples/task_tracker_requirements.md` for full spec.

## Results

Results are saved to `evals/results/`:
- `*_meta.json` - Run metadata (model, time, timestamp)
- `*_metrics.json` - Token usage (Ralph only)
- `*_validation.json` - Validation check results
- `*_output.txt` - Claude output (one-shot only)
