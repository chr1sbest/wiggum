#!/bin/bash
set -e

MODEL="${1:-sonnet}"
TIMESTAMP=$(date +%s)
PROJECT="eval-oneshot-tasktracker-$MODEL-$TIMESTAMP"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
REQUIREMENTS_FILE="$SCRIPT_DIR/../examples/task_tracker_requirements.md"

mkdir -p "$RESULTS_DIR"

echo "=== One-Shot Eval: Task Tracker API ==="
echo "Model: $MODEL"
echo "Project: $PROJECT"
echo ""

cd "$SCRIPT_DIR/.."
mkdir -p "$PROJECT/$PROJECT"
cd "$PROJECT/$PROJECT"

# Read requirements
REQUIREMENTS=$(cat "$REQUIREMENTS_FILE")

# Single prompt that includes full requirements
PROMPT="You are in an empty directory. Build the complete production-ready application described below.

Create ALL files needed for a working application:
- app.py (main Flask application with all routes)
- models.py (SQLAlchemy models for all entities)
- auth.py (JWT authentication helpers)
- cache.py (Redis caching utilities)
- tasks.py (Celery background jobs)
- requirements.txt (all dependencies including redis, celery, boto3, fakeredis, moto)
- README.md (setup and run instructions)
- docker-compose.yml (Redis, optional MinIO for S3)
- .env.example (all required environment variables)
- Makefile (run, test, lint commands)
- tests/ directory with comprehensive pytest tests

Make sure:
- The app runs on port 8000
- All endpoints work correctly (auth, projects, tasks, comments, attachments, webhooks, notifications, activity log, batch ops, export)
- Tests actually pass when run with pytest (use fakeredis, moto, eager Celery)
- Database is SQLite, created automatically on first run
- Redis caching with X-Cache headers
- Rate limiting returns 429 when exceeded

Here are the full requirements:

$REQUIREMENTS

Now create all the files. Be thorough - implement every endpoint, every validation rule, every test mentioned. This is a complex application with many features."

# Run Claude Code once with JSON output for token tracking
START=$(date +%s)
claude --model "$MODEL" --dangerously-skip-permissions --output-format json -p "$PROMPT" > "$RESULTS_DIR/${PROJECT}_output.json" 2>&1 || true
END=$(date +%s)

ELAPSED=$((END - START))
echo ""
echo "=== One-shot completed in ${ELAPSED}s ==="

# Parse token usage from JSON output
OUTPUT_FILE="$RESULTS_DIR/${PROJECT}_output.json"
TOTAL_TOKENS=0
INPUT_TOKENS=0
OUTPUT_TOKENS=0
COST_USD="0"
if [ -f "$OUTPUT_FILE" ]; then
    # Extract the JSON line (skip any warning lines)
    JSON_LINE=$(grep '^{' "$OUTPUT_FILE" | tail -1)
    if [ -n "$JSON_LINE" ]; then
        INPUT_TOKENS=$(echo "$JSON_LINE" | grep -oE '"input_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' || echo "0")
        OUTPUT_TOKENS=$(echo "$JSON_LINE" | grep -oE '"output_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' || echo "0")
        CACHE_READ=$(echo "$JSON_LINE" | grep -oE '"cache_read_input_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' || echo "0")
        CACHE_CREATE=$(echo "$JSON_LINE" | grep -oE '"cache_creation_input_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' || echo "0")
        COST_USD=$(echo "$JSON_LINE" | grep -oE '"total_cost_usd"[[:space:]]*:[[:space:]]*[0-9.]+' | grep -oE '[0-9.]+' || echo "0")
        TOTAL_TOKENS=$((INPUT_TOKENS + OUTPUT_TOKENS + CACHE_READ + CACHE_CREATE))
    fi
fi

# Count generated files and lines (exclude venv)
FILE_COUNT=$(find . -type f \( -name "*.py" -o -name "*.txt" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "Makefile" \) -not -path "./venv/*" -not -path "./__pycache__/*" 2>/dev/null | wc -l | tr -d ' ')
LINE_COUNT=$(find . -type f \( -name "*.py" -o -name "*.txt" -o -name "*.md" \) -not -path "./venv/*" -not -path "./__pycache__/*" -exec cat {} \; 2>/dev/null | wc -l | tr -d ' ')

echo ""
echo "=== Running Validation ==="

# Run pytest and capture results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

if [ -f "requirements.txt" ]; then
    # Set up venv if needed
    if [ ! -d "venv" ]; then
        python3 -m venv venv 2>/dev/null || true
    fi
    source venv/bin/activate 2>/dev/null || true
    pip install -r requirements.txt -q 2>/dev/null || true
    
    # Run pytest with PYTHONPATH set to current dir
    PYTEST_OUT=$(PYTHONPATH="." pytest --tb=no -q 2>&1 || true)
    
    # Parse pytest summary line like "5 passed, 2 failed" or "10 passed"
    if echo "$PYTEST_OUT" | grep -qE "[0-9]+ passed"; then
        TESTS_PASSED=$(echo "$PYTEST_OUT" | grep -oE "[0-9]+ passed" | grep -oE "[0-9]+" || echo "0")
    fi
    if echo "$PYTEST_OUT" | grep -qE "[0-9]+ failed"; then
        TESTS_FAILED=$(echo "$PYTEST_OUT" | grep -oE "[0-9]+ failed" | grep -oE "[0-9]+" || echo "0")
    fi
    TESTS_TOTAL=$((TESTS_PASSED + TESTS_FAILED))
    
    echo "Tests: $TESTS_PASSED passed, $TESTS_FAILED failed (total: $TESTS_TOTAL)"
    deactivate 2>/dev/null || true
fi

# Create metrics JSON (comparable to ralph's run_metrics.json)
# Also create a run_metrics.json in the project dir for consistency
METRICS_JSON="{
  \"approach\": \"oneshot\",
  \"model\": \"$MODEL\",
  \"project\": \"$PROJECT\",
  \"elapsed_seconds\": $ELAPSED,
  \"timestamp\": \"$TIMESTAMP\",
  \"total_claude_calls\": 1,
  \"input_tokens\": $INPUT_TOKENS,
  \"output_tokens\": $OUTPUT_TOKENS,
  \"total_tokens\": $TOTAL_TOKENS,
  \"total_cost_usd\": $COST_USD,
  \"tests_passed\": $TESTS_PASSED,
  \"tests_failed\": $TESTS_FAILED,
  \"tests_total\": $TESTS_TOTAL,
  \"files_generated\": $FILE_COUNT,
  \"lines_generated\": $LINE_COUNT
}"
echo "$METRICS_JSON" > "$RESULTS_DIR/${PROJECT}_metrics.json"
# Also save to project dir like ralph does
mkdir -p "$SCRIPT_DIR/../$PROJECT/.ralph"
echo "$METRICS_JSON" > "$SCRIPT_DIR/../$PROJECT/.ralph/run_metrics.json"

echo ""
echo "=== Summary ==="
echo "Time: ${ELAPSED}s"
echo "Tests: $TESTS_PASSED/$TESTS_TOTAL passed"
echo "Tokens: $TOTAL_TOKENS (in: $INPUT_TOKENS, out: $OUTPUT_TOKENS)"
echo "Cost: \$$COST_USD"
echo "Files: $FILE_COUNT"
echo "Lines: $LINE_COUNT"
echo ""
echo "Results: $RESULTS_DIR/${PROJECT}_metrics.json"
