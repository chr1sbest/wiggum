# Ralph Evals

Evaluation framework for comparing agent frameworks on code generation tasks.

## Design Principles

When writing evals, follow these principles:

- **Determine winner between multiple agent frameworks** given the same input requirements
- **Scripts to run any framework** - oneshot, ralph, or other agent frameworks
- **Outputs metrics and a working product** - not just metrics, but actual runnable code
- **Shared test framework** - same tests run against all implementations for fair comparison

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
Lists all available evaluation suites by scanning `evals/suites/*/suite.yaml` files.

### `ralph eval run <suite> [flags]`
Runs an evaluation suite with the specified approach and model.

**Flags:**
- `--approach` - Approach to use: `ralph` or `oneshot` (default: ralph)
- `--model` - Model to use: `sonnet`, `opus`, or `haiku` (default: sonnet)

**Examples:**
```bash
ralph eval run flask --approach ralph --model sonnet
ralph eval run tasktracker --approach oneshot --model opus
```

### `ralph eval compare <suite>`
Compares the most recent ralph and oneshot results for a suite, displaying metrics side-by-side.

## Suite Configuration Format

Each evaluation suite is defined by a `suite.yaml` file in `evals/suites/<suite-name>/`. The YAML format specifies all information needed to run and test the suite.

### Suite YAML Schema

```yaml
name: string              # Suite identifier (usually matches directory name)
description: string       # Brief description of what's being built
requirements: string      # Path to requirements.md file (relative to repo root)
language: string          # Primary language: go, python, etc.
timeout: string          # Max time allowed (e.g., "30m", "1h", "2h")

setup:                   # Optional setup commands (run before tests)
  - command1
  - command2

tests:
  shared:                # Shared tests run against both ralph and oneshot outputs
    - test_command1
    - test_command2
```

### Example: Flask Suite

```yaml
name: flask
description: Simple Flask web server showing current day with basic HTML and styling
requirements: examples/flask_requirements.md
language: python
timeout: 30m

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install flask pytest requests

tests:
  shared:
    - source venv/bin/activate && pytest evals/suites/flask/tests/ -v
```

### Example: Task Tracker Suite

```yaml
name: tasktracker
description: REST API with JWT auth, 4 models, ~15 endpoints
requirements: examples/task_tracker_requirements.md
language: python
timeout: 2h

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install -r requirements.txt

tests:
  shared:
    - source venv/bin/activate && pytest evals/suites/tasktracker/tests/ -v
```

### Example: Log Aggregator Suite

```yaml
name: logagg
description: Log aggregator CLI with parsing, filtering, stats, and query capabilities
requirements: examples/log_aggregator_requirements.md
language: go
timeout: 1h

tests:
  shared:
    - evals/suites/logagg/run_tests.sh
```

## Result JSON Schema

Evaluation results are saved as JSON files in `evals/results/` with the naming pattern:
```
<suite>-<approach>-<model>-<timestamp>.json
```

### Result File Format

```json
{
  "suite": "string",              // Suite name (e.g., "tasktracker")
  "approach": "string",           // "ralph" or "oneshot"
  "model": "string",              // Model used (e.g., "sonnet")
  "timestamp": "string",          // ISO 8601 timestamp (e.g., "2025-01-19T12:34:56Z")
  "duration_seconds": number,     // Total elapsed time in seconds
  "total_calls": number,          // Number of Claude API calls made
  "input_tokens": number,         // Total input tokens consumed
  "output_tokens": number,        // Total output tokens generated
  "total_tokens": number,         // Sum of input + output tokens
  "cost_usd": number,            // Estimated cost in USD
  "shared_tests_passed": number,  // Number of shared tests that passed
  "shared_tests_total": number,   // Total number of shared tests
  "files_generated": number,      // Number of files created
  "lines_generated": number,      // Total lines of code generated
  "output_dir": "string"         // Path to generated project directory
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

To add a new evaluation suite:

### 1. Create Requirements Document

Create a requirements file in `examples/`:
```bash
touch examples/my_feature_requirements.md
```

Write clear, detailed requirements for what should be built.

### 2. Create Suite Directory

```bash
mkdir -p evals/suites/myfeature/tests
```

### 3. Create suite.yaml

Create `evals/suites/myfeature/suite.yaml`:
```yaml
name: myfeature
description: Brief description of what's being built
requirements: examples/my_feature_requirements.md
language: python  # or go, etc.
timeout: 1h

setup:
  - python3 -m venv venv
  - source venv/bin/activate && pip install -r requirements.txt

tests:
  shared:
    - source venv/bin/activate && pytest evals/suites/myfeature/tests/ -v
```

### 4. Write Shared Tests

Create implementation-agnostic tests in `evals/suites/myfeature/tests/`:

**conftest.py** - Pytest fixtures:
```python
import pytest
import requests

@pytest.fixture
def base_url():
    return "http://localhost:8000"

@pytest.fixture
def api_client(base_url):
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})
    return session
```

**test_feature.py** - Actual tests:
```python
def test_basic_functionality(api_client, base_url):
    response = api_client.get(f"{base_url}/endpoint")
    assert response.status_code == 200
    # Add more assertions...
```

### 5. Test Your Suite

```bash
ralph eval run myfeature --approach ralph --model sonnet
```

### Guidelines for Writing Tests

- **Test the contract, not the implementation** - Tests should work regardless of how the feature is implemented
- **Use fixtures for setup** - Keep tests DRY by using pytest fixtures for common setup
- **Be explicit about requirements** - Each test should clearly validate one specific requirement
- **Fail fast with clear messages** - Use descriptive assertion messages
- **Avoid implementation details** - Don't test internal function names, variable names, or file structure

## Key Concept: Shared Test Suites

The framework uses **shared test suites** in `suites/<name>/tests/` that are run against BOTH ralph and oneshot outputs. This ensures fair comparison - both implementations are tested against the same requirements.

```
evals/
├── suites/
│   ├── flask/
│   │   ├── suite.yaml       # Configuration
│   │   └── tests/           # Shared test suite
│   │       ├── conftest.py  # Fixtures
│   │       └── test_app.py  # Tests
│   ├── tasktracker/
│   │   ├── suite.yaml
│   │   └── tests/
│   │       ├── conftest.py
│   │       ├── test_auth.py
│   │       └── test_tasks.py
│   └── logagg/
│       ├── suite.yaml
│       └── run_tests.sh     # Can use scripts instead of pytest
├── results/                 # Generated result JSON files
├── run.sh                   # Core evaluation runner (called by CLI)
└── compare_evals.sh         # Comparison tool (called by CLI)
```

## What's Measured

- **Duration**: Total elapsed time in seconds
- **API Calls**: Number of Claude API calls made
- **Tokens**: Input, output, and total token usage
- **Cost**: Estimated cost in USD based on token usage
- **Test Results**: Number of shared tests passed vs. total
- **Code Generated**: File count and line count

## Results Storage

Results are saved to `evals/results/` and gitignored to prevent bloat. Each run generates a single JSON file with all metrics.
