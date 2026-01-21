# Ralph Evals

Evaluation harness for comparing agent harnesses on code generation tasks.

## Latest Results

See **[RESULTS.md](RESULTS.md)** for capability eval results.

| Suite | Ralph | Oneshot | Δ Tasks |
|-------|-------|---------|---------|
| workflow | 41/48 (85%) | 40/48 (83%) | +2.5% |
| tasktracker | 26/28 (93%) | 23/28 (82%) | +13% |

**Key finding:** Ralph costs 1.5-5x more and takes longer, but passes 2-13% more tasks. The tradeoff favors Ralph when correctness matters more than speed or cost.

## Terminology

Following [Anthropic's eval framework](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents):

| Term | Definition |
|------|------------|
| **Task** | A single test case with defined inputs and success criteria |
| **Trial** | One attempt at a task (we run one trial per task currently) |
| **Grader** | Logic that scores the agent's outcome (we use deterministic graders) |
| **Outcome** | The final state: the generated code that graders evaluate |
| **Transcript** | Complete record of a trial including all tool calls and responses |
| **Evaluation suite** | Collection of tasks measuring specific capabilities |
| **Evaluation harness** | Infrastructure that runs evals end-to-end (this repo) |
| **Agent harness** | System that orchestrates the model (Ralph or Oneshot) |

## Design Principles

- **Compare agent harnesses fairly** - Same tasks, same graders, same model
- **Grade outcomes, not paths** - Test what the agent produced, not how it got there
- **Deterministic graders** - Code-based tests that pass or fail objectively
- **Isolated trials** - Each trial starts from a clean environment

## Quick Start

```bash
# List available evaluation suites
ralph eval list

# Run an evaluation suite
ralph eval run tasktracker --approach ralph --model sonnet
ralph eval run tasktracker --approach oneshot --model sonnet

# Compare results between approaches
ralph eval compare tasktracker
```

## CLI Commands

### `ralph eval list`
Lists all available evaluation suites.

### `ralph eval run <suite> [flags]`
Runs all tasks in a suite with the specified agent harness and model.

**Flags:**
- `--approach` - Agent harness: `ralph` or `oneshot` (default: ralph)
- `--model` - Model: `sonnet`, `opus`, or `haiku` (default: sonnet)

**Examples:**
```bash
ralph eval run flask --approach ralph --model sonnet
ralph eval run tasktracker --approach oneshot --model opus
```

### `ralph eval compare <suite>`
Compares the most recent Ralph and Oneshot results, showing tasks passed and tracked metrics.

## Suite Configuration Format

Each evaluation suite is defined by a `suite.yaml` file in `evals/suites/<suite-name>/`.

### Suite YAML Schema

```yaml
name: string              # Suite identifier
description: string       # Brief description of what's being built
requirements: string      # Path to task specification (requirements.md)
language: string          # Primary language: go, python, etc.
type: string              # Suite type: "web" or "cli"
timeout: string           # Max time for trial (e.g., "45m")

setup:                    # Optional setup commands (run before graders)
  - command1
  - command2
```

**Suite Types:**
- `web` - Web applications. Graders are pytest tests in `tests/` directory.
- `cli` - CLI tools. Graders are Go-based tests in `internal/eval/`.

### Example: Flask Suite (Web App)

```yaml
name: flask
description: Simple Flask web server showing current day with basic HTML and styling
requirements: examples/flask_requirements.md
language: python
type: web
timeout: 30m

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install flask pytest requests
```

### Example: Task Tracker Suite (Web App)

```yaml
name: tasktracker
description: REST API with JWT auth, 4 models, ~15 endpoints
requirements: examples/task_tracker_requirements.md
language: python
type: web
timeout: 2h

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install -r requirements.txt
```

### Example: Log Aggregator Suite (CLI Tool)

```yaml
name: logagg
description: Log aggregator CLI with parsing, filtering, stats, and query capabilities
requirements: examples/log_aggregator_requirements.md
language: go
type: cli
timeout: 1h
```

## Trial Result Schema

Trial results are saved as JSON in `evals/results/`:
```
<suite>-<approach>-<model>-<timestamp>.json
```

### Result File Format

```json
{
  "suite": "string",              // Evaluation suite name
  "approach": "string",           // Agent harness: "ralph" or "oneshot"
  "model": "string",              // Model used (e.g., "sonnet")
  "timestamp": "string",          // ISO 8601 timestamp
  "duration_seconds": number,     // Trial duration
  "total_calls": number,          // Claude API calls in transcript
  "input_tokens": number,         // Input tokens consumed
  "output_tokens": number,        // Output tokens generated
  "total_tokens": number,         // Total tokens
  "cost_usd": number,             // Estimated cost
  "shared_tests_passed": number,  // Tasks passed by graders
  "shared_tests_total": number,   // Total tasks in suite
  "files_generated": number,      // Files in outcome
  "lines_generated": number,      // Lines of code in outcome
  "output_dir": "string"          // Path to outcome directory
}
```

### Example Result File

```json
{
  "suite": "flask",
  "approach": "ralph",
  "model": "sonnet",
  "timestamp": "2025-01-19T14:23:45Z",
  "duration_seconds": 127,
  "total_calls": 8,
  "input_tokens": 45230,
  "output_tokens": 12450,
  "total_tokens": 57680,
  "cost_usd": 0.87,
  "shared_tests_passed": 7,
  "shared_tests_total": 7,
  "files_generated": 4,
  "lines_generated": 234,
  "output_dir": "/tmp/eval-ralph-flask-sonnet-20250119-142345/flask"
}
```

## Adding a New Evaluation Suite

### 1. Create Task Specification

Create a requirements file in `examples/`:
```bash
touch examples/my_feature_requirements.md
```

Write clear requirements for what the agent should build.

### 2. Create Suite Directory

```bash
mkdir -p evals/suites/myfeature/tests
```

### 3. Create suite.yaml

```yaml
name: myfeature
description: Brief description of what's being built
requirements: examples/my_feature_requirements.md
language: python  # or go
type: web         # or cli
timeout: 45m

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install -r requirements.txt
```

### 4. Write Graders

**For web suites (`type: web`)**: Create pytest tests in `evals/suites/myfeature/tests/`

**For CLI suites (`type: cli`)**: Create Go tests in `internal/eval/`. Example:

```go
// internal/eval/myfeature_tests.go
func RunMyFeatureTests(appDir, fixturesDir string) (*TestResult, error) {
    r := NewCLITestRunner(appDir, "myfeature")
    
    // Build the binary
    r.RunTestExitCode("build succeeds", "go build -o myfeature .", 0)
    
    // Test functionality
    r.RunTest("basic command works", "./myfeature run input.txt", "expected output")
    
    return &TestResult{Passed: r.Passed, Failed: r.Failed, Total: r.GetTotal()}, nil
}
```

### 5. Run the Suite

```bash
ralph eval run myfeature --approach ralph --model sonnet
```

### Guidelines for Writing Graders

- **Grade outcomes, not paths** - Test what the agent produced, not how it got there
- **Use deterministic graders** - Objective pass/fail based on the outcome
- **One task per test** - Each test should validate one specific capability
- **Avoid implementation details** - Don't test internal names or file structure

## Architecture

```
evals/
├── suites/
│   ├── flask/
│   │   ├── suite.yaml       # Suite config (type: web)
│   │   └── tests/           # Pytest graders
│   ├── tasktracker/
│   │   ├── suite.yaml       # Suite config (type: web)
│   │   └── tests/           # Pytest graders
│   └── workflow/
│       ├── suite.yaml       # Suite config (type: cli)
│       └── fixtures/        # Test fixtures for Go graders
└── results/                 # Trial results (JSON)

internal/eval/
├── workflow_tests.go        # Go graders for workflow CLI
├── cli_tests.go             # Go graders for logagg CLI
└── tasktracker_tests.go     # Go graders for tasktracker API
```

## Tracked Metrics

| Metric | Description |
|--------|-------------|
| **Tasks passed** | Number of tasks where graders passed |
| **Duration** | Total trial time in seconds |
| **Tokens** | Input, output, and total token usage |
| **Cost** | Estimated cost in USD |
| **Code** | Files and lines in outcome |

## Results Storage

Trial results are saved to `evals/results/` (gitignored). Each trial generates one JSON file with all metrics and can be compared across agent harnesses.
