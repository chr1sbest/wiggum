# Eval Framework Improvements

Improve the existing `evals/` directory to be more structured, extensible, and easier to use. This is NOT a new service—it's refactoring the existing bash scripts into a cleaner architecture that lives within the ralph repository.

## Goals

1. **Keep it simple** - Improve existing scripts, don't over-engineer
2. **Config-driven suites** - Define evals via YAML instead of hardcoded paths
3. **Unified CLI** - Run evals via `ralph eval` instead of separate bash scripts
4. **Fair comparison** - Same tests run against all approaches
5. **Clear output** - Terminal tables showing winners per metric

## Non-Goals

- Building a separate service or binary
- Web UI or dashboards
- Database storage (JSON files are fine)
- Parallel execution
- Support for non-Claude LLMs

## Current State

The existing `evals/` directory works but is hard to extend:
- `run.sh` - Unified runner for ralph/oneshot
- `compare_evals.sh` - Compares two eval runs  
- `suites/` - Test suites (only logagg currently)
- Hardcoded paths and bash string parsing

## Requirements

### 1. Suite Configuration (suite.yaml)

Replace hardcoded paths with YAML config files.

**File:** `evals/suites/<name>/suite.yaml`

```yaml
name: tasktracker
description: REST API with JWT auth, 4 models, ~15 endpoints
requirements: examples/task_tracker_requirements.md
language: python
timeout: 2h

setup:
  - pip install -r requirements.txt

tests:
  shared:
    - pytest evals/suites/tasktracker/tests/ -v
```

**Acceptance Criteria:**
- [ ] Create simple YAML loader in Go (can use existing yaml library)
- [ ] Update `run.sh` to read from suite.yaml instead of case statement
- [ ] Create suite.yaml for existing logagg suite
- [ ] Create suite.yaml for flask suite

### 2. Add `ralph eval` Subcommand

Wrap existing bash scripts with a Go CLI for convenience.

```bash
# These just call the existing bash scripts under the hood
ralph eval run <suite> --approach <ralph|oneshot> --model <model>
ralph eval compare <suite>
ralph eval list
```

**Acceptance Criteria:**
- [ ] Add `eval` case to `cmd/ralph/main.go`
- [ ] Create `cmd/ralph/cmd_eval.go`
- [ ] `ralph eval run` calls `evals/run.sh` with proper args
- [ ] `ralph eval compare` calls `evals/compare_evals.sh`
- [ ] `ralph eval list` discovers and prints available suites

### 3. Standardize Result Schema

Define a consistent JSON schema for eval results.

**File:** `evals/results/<suite>-<approach>-<model>-<timestamp>.json`

```json
{
  "suite": "tasktracker",
  "approach": "ralph",
  "model": "sonnet",
  "timestamp": "2026-01-19T13:00:00Z",
  "duration_seconds": 1143,
  "total_calls": 3,
  "input_tokens": 757038,
  "output_tokens": 6283,
  "total_tokens": 763321,
  "cost_usd": 0.43,
  "shared_tests_passed": 18,
  "shared_tests_total": 20,
  "files_generated": 12,
  "lines_generated": 450,
  "output_dir": "eval-ralph-tasktracker-sonnet-20260119"
}
```

**Acceptance Criteria:**
- [ ] Update `run.sh` to output this JSON format
- [ ] Update `compare_evals.sh` to read this format
- [ ] Results saved to `evals/results/` directory

### 4. Comparison Table Output

Improve the comparison output to be clearer.

```
┌─────────────────────────────────────────────────────────────────┐
│ Eval Comparison: tasktracker                                    │
├─────────────────┬──────────────┬──────────────┬─────────────────┤
│ Metric          │ Ralph        │ Oneshot      │ Winner          │
├─────────────────┼──────────────┼──────────────┼─────────────────┤
│ Duration        │ 12m 34s      │ 2m 15s       │ Oneshot (-82%)  │
│ Total Tokens    │ 245,000      │ 89,000       │ Oneshot (-64%)  │
│ Cost            │ $0.52        │ $0.18        │ Oneshot (-65%)  │
│ Shared Tests    │ 18/20 (90%)  │ 14/20 (70%)  │ Ralph (+20%)    │
└─────────────────┴──────────────┴──────────────┴─────────────────┘
```

**Acceptance Criteria:**
- [ ] Update `compare_evals.sh` to print formatted table
- [ ] Show percentage difference and winner per metric
- [ ] Print summary line at bottom

### 5. Create tasktracker Test Suite

Add shared tests for the Task Tracker API eval.

**Directory:** `evals/suites/tasktracker/`

**Files:**
- `suite.yaml` - Config
- `tests/conftest.py` - Pytest fixtures
- `tests/test_auth.py` - JWT auth tests
- `tests/test_tasks.py` - CRUD tests

**Acceptance Criteria:**
- [ ] Create suite.yaml pointing to task_tracker_requirements.md
- [ ] Create 10-15 pytest tests covering core API functionality
- [ ] Tests are implementation-agnostic (only test API contract)

### 6. Create flask Test Suite

Add shared tests for the simple Flask eval.

**Directory:** `evals/suites/flask/`

**Acceptance Criteria:**
- [ ] Create suite.yaml pointing to flask_requirements.md
- [ ] Create 5-10 pytest tests for Flask app

## Updated Directory Structure

```
evals/
├── run.sh                  # Keep existing, update to use suite.yaml
├── compare_evals.sh        # Keep existing, improve output format
├── suites/
│   ├── logagg/
│   │   ├── suite.yaml      # NEW: config file
│   │   ├── run_tests.sh    # Existing
│   │   └── fixtures/
│   ├── tasktracker/
│   │   ├── suite.yaml      # NEW
│   │   └── tests/
│   │       ├── conftest.py
│   │       └── test_*.py
│   └── flask/
│       ├── suite.yaml      # NEW
│       └── tests/
├── results/                # Gitignored, stores result JSON files
└── README.md               # Update docs
```

## Implementation Order

1. **suite.yaml for existing suites** - Create config files for logagg, flask
2. **Standardize result JSON** - Update scripts to output consistent format
3. **`ralph eval` CLI** - Thin wrapper around existing scripts
4. **Improve comparison output** - Better formatted tables
5. **tasktracker test suite** - Shared pytest tests
6. **flask test suite** - Shared pytest tests

## Success Criteria

- [ ] `ralph eval list` shows available suites
- [ ] `ralph eval run flask --approach ralph` works
- [ ] `ralph eval run flask --approach oneshot` works  
- [ ] `ralph eval compare flask` shows formatted comparison
- [ ] Adding a new suite = create suite.yaml + tests directory
- [ ] Existing bash scripts still work for backwards compatibility
