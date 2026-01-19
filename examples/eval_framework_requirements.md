# Eval Framework Requirements

Build a structured evaluation framework for comparing agent approaches (Ralph loop vs one-shot vs future approaches) on code generation tasks.

## Goals

1. **Fair comparison** - Same requirements, same tests, reproducible results
2. **Extensible** - Easy to add new suites and new approaches
3. **Statistical rigor** - Multiple runs, variance tracking, significance testing
4. **Actionable output** - Clear reports showing which approach wins and why

## Current State

The existing `evals/` directory has bash scripts that work but are hard to extend:
- `run.sh` - Unified runner for ralph/oneshot
- `compare_evals.sh` - Compares two eval runs
- `suites/` - Test suites (only logagg currently)

## Requirements

### 1. Suite Configuration (suite.yaml)

Each eval suite should be defined by a YAML config file rather than hardcoded paths.

**File:** `evals/suites/<name>/suite.yaml`

**Schema:**
```yaml
name: tasktracker
description: REST API with JWT auth, 4 models, ~15 endpoints
requirements: examples/task_tracker_requirements.md
language: python  # or "go", "javascript"
timeout: 2h

# Commands to run after code generation
setup:
  - pip install -r requirements.txt

# How to start the server for functional tests
server:
  command: python -m uvicorn main:app --port 8000
  health_check: http://localhost:8000/health
  startup_timeout: 30s

# Test definitions
tests:
  # Shared tests run against ALL approaches for fair comparison
  shared:
    - pytest evals/suites/tasktracker/tests/ -v
  
  # Functional tests (server must be running)
  functional:
    - curl -sf http://localhost:8000/health
    - curl -sf http://localhost:8000/docs

# Optional: code quality metrics to collect
metrics:
  - lines_of_code
  - file_count
```

**Acceptance Criteria:**
- [ ] Create `evals/pkg/suite/loader.go` that parses suite.yaml
- [ ] Create `Suite` struct with all fields from schema
- [ ] Validate required fields (name, requirements, tests.shared)
- [ ] Support timeout parsing (e.g., "2h", "30m")
- [ ] Add suite discovery: find all `evals/suites/*/suite.yaml`

### 2. Result Schema

Standardize eval results in a typed Go struct.

**File:** `evals/pkg/results/result.go`

**Schema:**
```go
type EvalResult struct {
    // Identity
    ID        string    `json:"id"`         // UUID
    Suite     string    `json:"suite"`      // e.g., "tasktracker"
    Approach  string    `json:"approach"`   // e.g., "ralph", "oneshot"
    Model     string    `json:"model"`      // e.g., "sonnet", "opus"
    Timestamp time.Time `json:"timestamp"`
    
    // Timing
    DurationSeconds float64 `json:"duration_seconds"`
    
    // Cost
    TotalCalls   int     `json:"total_calls"`
    InputTokens  int     `json:"input_tokens"`
    OutputTokens int     `json:"output_tokens"`
    TotalTokens  int     `json:"total_tokens"`
    CostUSD      float64 `json:"cost_usd"`
    
    // Quality
    SharedTestsPassed int `json:"shared_tests_passed"`
    SharedTestsTotal  int `json:"shared_tests_total"`
    OwnTestsPassed    int `json:"own_tests_passed"`
    OwnTestsTotal     int `json:"own_tests_total"`
    
    // Code metrics
    FilesGenerated int `json:"files_generated"`
    LinesGenerated int `json:"lines_generated"`
    
    // Artifacts
    OutputDir string `json:"output_dir"`
    
    // Errors
    Error string `json:"error,omitempty"`
}
```

**Acceptance Criteria:**
- [ ] Create `EvalResult` struct with JSON tags
- [ ] Create `SaveResult(result EvalResult, dir string) error`
- [ ] Create `LoadResult(path string) (*EvalResult, error)`
- [ ] Results saved to `evals/results/<id>.json`

### 3. Approach Interface

Define a pluggable interface for different code generation approaches.

**File:** `evals/pkg/approach/approach.go`

**Interface:**
```go
type Approach interface {
    Name() string
    Run(ctx context.Context, suite *Suite, model string, outputDir string) error
}
```

**Implementations needed:**
1. `RalphApproach` - Runs `ralph init` + `ralph run`
2. `OneshotApproach` - Single Claude prompt with full requirements

**Acceptance Criteria:**
- [ ] Define `Approach` interface
- [ ] Implement `RalphApproach` in `approach/ralph.go`
- [ ] Implement `OneshotApproach` in `approach/oneshot.go`
- [ ] Create approach registry: `GetApproach(name string) Approach`

### 4. CLI Commands

Add eval subcommands to the ralph CLI.

**Commands:**

```bash
# Run an eval
ralph eval run <suite> --approach <name> --model <model>
ralph eval run tasktracker --approach ralph --model sonnet
ralph eval run tasktracker --approach oneshot --model sonnet

# Compare latest results for a suite
ralph eval compare <suite>
ralph eval compare tasktracker

# Compare specific result files
ralph eval compare --files result1.json,result2.json

# List available suites
ralph eval list

# Show leaderboard (aggregated scores across runs)
ralph eval leaderboard
```

**Acceptance Criteria:**
- [ ] Add `eval` subcommand to `cmd/ralph/main.go`
- [ ] Create `cmd/ralph/cmd_eval.go` with subcommand routing
- [ ] Implement `eval run` - runs approach on suite, saves result
- [ ] Implement `eval compare` - loads results, prints comparison table
- [ ] Implement `eval list` - discovers and lists available suites

### 5. Comparison Report

Generate clear comparison output.

**Terminal Output Example:**
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
│ Own Tests       │ 25/25        │ 0/0          │ Ralph           │
│ Files Generated │ 12           │ 8            │ Ralph (+50%)    │
└─────────────────┴──────────────┴──────────────┴─────────────────┘

Summary: Ralph produces higher quality code (90% vs 70% test pass rate)
         but costs 3x more in time and tokens.
```

**Acceptance Criteria:**
- [ ] Create `evals/pkg/report/compare.go`
- [ ] Load two EvalResult structs
- [ ] Calculate deltas and determine winner per metric
- [ ] Print formatted table to terminal
- [ ] Support `--format json` for machine-readable output

### 6. Test Suite for tasktracker

Create a shared test suite for the Task Tracker API eval.

**Directory:** `evals/suites/tasktracker/`

**Files:**
- `suite.yaml` - Suite configuration
- `tests/conftest.py` - Pytest fixtures (base URL, auth helpers)
- `tests/test_auth.py` - JWT auth tests
- `tests/test_tasks.py` - CRUD operations on tasks
- `tests/test_validation.py` - Input validation tests

**Test Requirements:**
- Tests should be implementation-agnostic
- Use only the API contract from requirements
- Include both positive and negative test cases
- At least 15 test cases covering core functionality

**Acceptance Criteria:**
- [ ] Create `evals/suites/tasktracker/suite.yaml`
- [ ] Create pytest test files with 15+ test cases
- [ ] Tests pass when run against a correct implementation
- [ ] Tests fail appropriately for missing/broken features

### 7. Multi-Run Variance (Stretch Goal)

Support running the same eval multiple times to measure variance.

```bash
ralph eval run tasktracker --approach ralph --runs 3
```

**Output includes:**
- Mean, stddev, min, max for each metric
- Confidence interval for test pass rate

**Acceptance Criteria:**
- [ ] Add `--runs N` flag to `eval run`
- [ ] Run eval N times with fresh output dirs
- [ ] Aggregate results and compute statistics
- [ ] Report variance in comparison output

## Directory Structure

```
evals/
├── cmd/                    # CLI entry if separate binary needed
├── pkg/
│   ├── suite/              # Suite loader (suite.yaml parsing)
│   ├── approach/           # Approach interface + implementations
│   ├── results/            # Result schema and persistence
│   └── report/             # Comparison report generation
├── suites/
│   ├── tasktracker/
│   │   ├── suite.yaml
│   │   └── tests/
│   │       ├── conftest.py
│   │       ├── test_auth.py
│   │       ├── test_tasks.py
│   │       └── test_validation.py
│   ├── logagg/
│   │   ├── suite.yaml
│   │   └── tests/
│   └── flask/
│       ├── suite.yaml
│       └── tests/
├── results/                # Gitignored, stores eval results
├── run.sh                  # Legacy script (keep for compatibility)
└── README.md               # Updated documentation
```

## Implementation Order

1. **Suite loader** - Parse suite.yaml, validate schema
2. **Result schema** - Define struct, save/load functions
3. **Approach interface** - Define interface, implement ralph + oneshot
4. **CLI commands** - eval run, eval compare, eval list
5. **Comparison report** - Terminal table output
6. **tasktracker test suite** - Shared pytest tests
7. **Multi-run variance** - Stretch goal

## Success Criteria

- [ ] `ralph eval list` shows available suites
- [ ] `ralph eval run flask --approach ralph` works end-to-end
- [ ] `ralph eval run flask --approach oneshot` works end-to-end
- [ ] `ralph eval compare flask` shows formatted comparison
- [ ] Adding a new suite requires only creating suite.yaml + tests
- [ ] Adding a new approach requires only implementing the interface

## Non-Goals

- Web UI / dashboards (terminal output is fine)
- Database storage (JSON files are fine)
- Parallel execution (sequential is fine)
- Support for non-Claude LLMs (Claude only for now)
