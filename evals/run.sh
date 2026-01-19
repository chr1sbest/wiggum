#!/bin/bash

# Unified Eval Runner
# Usage: ./run.sh <suite> <approach> [model]
#
# Suites: logagg, flask
# Approaches: ralph, oneshot
# Models: sonnet (default), opus, haiku
#
# Timeout: 2 hours

set -e

# 2 hour timeout (7200 seconds)
TIMEOUT_SECONDS=7200

# Check if timeout command exists (not on macOS by default)
if command -v timeout &> /dev/null; then
    TIMEOUT_CMD="timeout $TIMEOUT_SECONDS"
elif command -v gtimeout &> /dev/null; then
    TIMEOUT_CMD="gtimeout $TIMEOUT_SECONDS"
else
    echo "Note: 'timeout' not found, running without timeout"
    TIMEOUT_CMD=""
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WIGGUM_DIR="$SCRIPT_DIR/.."

SUITE="${1:-}"
APPROACH="${2:-}"
MODEL="${3:-sonnet}"

if [ -z "$SUITE" ] || [ -z "$APPROACH" ]; then
    echo "Usage: $0 <suite> <approach> [model]"
    echo ""
    echo "Suites:"
    echo "  logagg      - Log Aggregator CLI (Go)"
    echo "  tasktracker - Task Tracker API (Python)"
    echo "  flask       - Simple Flask server (Python)"
    echo ""
    echo "Approaches:"
    echo "  ralph   - Multi-turn iterative loop"
    echo "  oneshot - Single Claude prompt"
    echo ""
    echo "Models: sonnet (default), opus, haiku"
    echo ""
    echo "Examples:"
    echo "  $0 logagg ralph sonnet"
    echo "  $0 logagg oneshot sonnet"
    exit 1
fi

# Map suite to requirements file and test script
SUITE_DIR="$SCRIPT_DIR/suites/$SUITE"
SUITE_YAML="$SUITE_DIR/suite.yaml"

# Try to read from suite.yaml first
if [ -f "$SUITE_YAML" ]; then
    # Parse requirements path from suite.yaml
    REQUIREMENTS_REL=$(grep "^requirements:" "$SUITE_YAML" | head -1 | sed 's/^requirements:[[:space:]]*//')
    if [ -z "$REQUIREMENTS_REL" ]; then
        echo "Error: Could not parse requirements from $SUITE_YAML"
        exit 1
    fi
    REQUIREMENTS="$WIGGUM_DIR/$REQUIREMENTS_REL"

    # Parse test command from suite.yaml (first shared test)
    TEST_SCRIPT=$(grep -A 10 "^tests:" "$SUITE_YAML" | grep -A 10 "shared:" | grep "^[[:space:]]*-" | head -1 | sed 's/^[[:space:]]*-[[:space:]]*//')

else
    # Fallback to hardcoded paths for backwards compatibility
    echo "Note: suite.yaml not found at $SUITE_YAML, using hardcoded configuration"

    case "$SUITE" in
        logagg)
            REQUIREMENTS="$WIGGUM_DIR/examples/log_aggregator_requirements.md"
            TEST_SCRIPT="evals/suites/logagg/run_tests.sh"
            ;;
        tasktracker)
            REQUIREMENTS="$WIGGUM_DIR/examples/task_tracker_requirements.md"
            TEST_SCRIPT=""
            ;;
        flask)
            REQUIREMENTS="$WIGGUM_DIR/examples/flask_requirements.md"
            TEST_SCRIPT=""
            ;;
        *)
            echo "Error: Unknown suite '$SUITE'"
            exit 1
            ;;
    esac
fi

if [ ! -f "$REQUIREMENTS" ]; then
    echo "Error: Requirements file not found: $REQUIREMENTS"
    exit 1
fi

if [ ! -d "$SUITE_DIR" ]; then
    echo "Warning: No test suite found at $SUITE_DIR"
fi

TIMESTAMP=$(date +%s)
PROJECT="eval-${APPROACH}-${SUITE}-${MODEL}-${TIMESTAMP}"
RESULTS_DIR="$SCRIPT_DIR/results"
mkdir -p "$RESULTS_DIR"

START_TIME=$(date "+%Y-%m-%d %H:%M:%S")

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    EVAL: $SUITE"
echo "║  Approach: $APPROACH | Model: $MODEL"
echo "║  Started: $START_TIME"
echo "║  Timeout: 2 hours"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Create project directory at wiggum level (not inside wiggum/)
PROJECT_DIR="$WIGGUM_DIR/$PROJECT"
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

START=$(date +%s)

# =============================================================================
# Run the approach
# =============================================================================

if [ "$APPROACH" == "ralph" ]; then
    echo "Running Ralph..."
    echo ""
    
    # Create project directory and cd into it
    mkdir -p "$SUITE"
    cd "$SUITE"
    
    # Initialize ralph in the project directory
    $TIMEOUT_CMD ralph init "$REQUIREMENTS" || {
        echo "ERROR: ralph init timed out or failed"
        exit 1
    }
    
    # Run ralph loop with timeout
    $TIMEOUT_CMD ralph run -model "$MODEL" || {
        if [ $? -eq 124 ]; then
            echo "ERROR: ralph run timed out after 2 hours"
        fi
    }
    
    # Get metrics from ralph
    if [ -f ".ralph/run_metrics.json" ]; then
        INPUT_TOKENS=$(grep -oE '"input_tokens"[[:space:]]*:[[:space:]]*[0-9]+' .ralph/run_metrics.json | grep -oE '[0-9]+' || echo "0")
        OUTPUT_TOKENS=$(grep -oE '"output_tokens"[[:space:]]*:[[:space:]]*[0-9]+' .ralph/run_metrics.json | grep -oE '[0-9]+' || echo "0")
        TOTAL_TOKENS=$(grep -oE '"total_tokens"[[:space:]]*:[[:space:]]*[0-9]+' .ralph/run_metrics.json | grep -oE '[0-9]+' || echo "0")
        COST_USD=$(grep -oE '"total_cost_usd"[[:space:]]*:[[:space:]]*[0-9.]+' .ralph/run_metrics.json | grep -oE '[0-9.]+' || echo "0")
        CLAUDE_CALLS=$(grep -oE '"total_claude_calls"[[:space:]]*:[[:space:]]*[0-9]+' .ralph/run_metrics.json | grep -oE '[0-9]+' || echo "0")
    else
        INPUT_TOKENS=0
        OUTPUT_TOKENS=0
        TOTAL_TOKENS=0
        COST_USD=0
        CLAUDE_CALLS=0
    fi

elif [ "$APPROACH" == "oneshot" ]; then
    echo "Running One-Shot Claude..."
    echo ""
    
    REQUIREMENTS_CONTENT=$(cat "$REQUIREMENTS")
    
    PROMPT="You are building a complete application from scratch. Create ALL necessary files to fully implement this specification.

$REQUIREMENTS_CONTENT

IMPORTANT:
- Create every file mentioned in the requirements
- Include all dependencies (go.mod, requirements.txt, etc.)
- Write complete implementations, not stubs
- Include comprehensive tests
- Make sure the code compiles/runs

Create all the files now."

    # Run claude with JSON output for token tracking (with timeout)
    # Write prompt to temp file to avoid shell escaping issues
    echo "$PROMPT" > _prompt.txt
    $TIMEOUT_CMD claude --model "$MODEL" --dangerously-skip-permissions --output-format json < _prompt.txt > _claude_output.json 2>&1 || true
    
    # Parse token usage from JSON output
    if [ -f "_claude_output.json" ]; then
        JSON_LINE=$(grep '^{' "_claude_output.json" | tail -1)
        if [ -n "$JSON_LINE" ]; then
            INPUT_TOKENS=$(echo "$JSON_LINE" | grep -oE '"input_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | head -1 || echo "0")
            OUTPUT_TOKENS=$(echo "$JSON_LINE" | grep -oE '"output_tokens"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | head -1 || echo "0")
            COST_USD=$(echo "$JSON_LINE" | grep -oE '"total_cost_usd"[[:space:]]*:[[:space:]]*[0-9.]+' | grep -oE '[0-9.]+' | head -1 || echo "0")
            TOTAL_TOKENS=$((INPUT_TOKENS + OUTPUT_TOKENS))
        fi
    fi
    CLAUDE_CALLS=1
    
    : "${INPUT_TOKENS:=0}"
    : "${OUTPUT_TOKENS:=0}"
    : "${TOTAL_TOKENS:=0}"
    : "${COST_USD:=0}"

else
    echo "Error: Unknown approach '$APPROACH'. Use 'ralph' or 'oneshot'."
    exit 1
fi

END=$(date +%s)
ELAPSED=$((END - START))

echo ""
echo "=== Generation completed in ${ELAPSED}s ==="

# =============================================================================
# Run test suite
# =============================================================================

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Determine how to run tests
if [ -n "$TEST_SCRIPT" ]; then
    echo ""
    echo "=== Running Test Suite ==="
    echo ""

    # Check if TEST_SCRIPT is a path to a shell script
    if [[ "$TEST_SCRIPT" == *".sh"* ]]; then
        # Treat as a script path
        if [[ "$TEST_SCRIPT" != /* ]]; then
            TEST_SCRIPT_PATH="$WIGGUM_DIR/$TEST_SCRIPT"
        else
            TEST_SCRIPT_PATH="$TEST_SCRIPT"
        fi

        if [ -f "$TEST_SCRIPT_PATH" ]; then
            # Run the test script
            "$TEST_SCRIPT_PATH" "$PROJECT_DIR" || true
        else
            echo "⚠️  Test script not found: $TEST_SCRIPT_PATH"
        fi
    else
        # Treat as a command to execute
        eval "$TEST_SCRIPT" || true
    fi

    # Read results if available
    if [ -f ".eval_results.json" ]; then
        TESTS_PASSED=$(grep -oE '"passed"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
        TESTS_FAILED=$(grep -oE '"failed"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
        TESTS_TOTAL=$(grep -oE '"total"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
    fi
elif [ -f "$SUITE_DIR/run_tests.sh" ]; then
    # Fallback to legacy location
    echo ""
    echo "=== Running Test Suite ==="
    echo ""

    # Run the test suite
    "$SUITE_DIR/run_tests.sh" "$PROJECT_DIR" || true

    # Read results if available
    if [ -f ".eval_results.json" ]; then
        TESTS_PASSED=$(grep -oE '"passed"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
        TESTS_FAILED=$(grep -oE '"failed"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
        TESTS_TOTAL=$(grep -oE '"total"[[:space:]]*:[[:space:]]*[0-9]+' .eval_results.json | grep -oE '[0-9]+' || echo "0")
    fi
else
    echo ""
    echo "⚠️  No test suite found for $SUITE"
fi

# =============================================================================
# Count generated files
# =============================================================================

FILE_COUNT=$(find . -type f \( -name "*.go" -o -name "*.py" -o -name "*.md" -o -name "*.yaml" -o -name "*.yml" -o -name "*.mod" -o -name "*.sum" \) -not -path "./.ralph/*" -not -path "./venv/*" 2>/dev/null | wc -l | tr -d ' ')
LINE_COUNT=$(find . -type f \( -name "*.go" -o -name "*.py" \) -not -path "./.ralph/*" -not -path "./venv/*" -exec cat {} \; 2>/dev/null | wc -l | tr -d ' ')

# =============================================================================
# Save metrics
# =============================================================================

mkdir -p .ralph
cat > .ralph/run_metrics.json << EOF
{
  "suite": "$SUITE",
  "approach": "$APPROACH",
  "model": "$MODEL",
  "project": "$PROJECT",
  "timestamp": "$(date -Iseconds)",
  "elapsed_seconds": $ELAPSED,
  "total_claude_calls": $CLAUDE_CALLS,
  "input_tokens": $INPUT_TOKENS,
  "output_tokens": $OUTPUT_TOKENS,
  "total_tokens": $TOTAL_TOKENS,
  "total_cost_usd": $COST_USD,
  "tests_passed": $TESTS_PASSED,
  "tests_failed": $TESTS_FAILED,
  "tests_total": $TESTS_TOTAL,
  "files_generated": $FILE_COUNT,
  "lines_generated": $LINE_COUNT
}
EOF

# Also save to results dir
cp .ralph/run_metrics.json "$RESULTS_DIR/${PROJECT}_metrics.json"

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                         SUMMARY                              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
printf "%-20s %s\n" "Project:" "$PROJECT"
printf "%-20s %ss\n" "Time:" "$ELAPSED"
printf "%-20s %s\n" "Claude Calls:" "$CLAUDE_CALLS"
printf "%-20s %s\n" "Total Tokens:" "$TOTAL_TOKENS"
printf "%-20s \$%s\n" "Cost:" "$COST_USD"
printf "%-20s %s/%s\n" "Tests:" "$TESTS_PASSED" "$TESTS_TOTAL"
printf "%-20s %s files, %s lines\n" "Code:" "$FILE_COUNT" "$LINE_COUNT"
echo ""
echo "Results: $RESULTS_DIR/${PROJECT}_metrics.json"
echo ""
