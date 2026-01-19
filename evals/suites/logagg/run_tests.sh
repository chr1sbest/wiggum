#!/bin/bash

# Log Aggregator Eval Test Suite
# Usage: ./run_tests.sh <project_dir>
#
# Tests:
# 1. Build - does 'go build' succeed?
# 2. Parse JSON - can it parse JSON logs?
# 3. Parse Apache - can it parse Apache logs?
# 4. Filter by level - does --level filter work?
# 5. Stats - does stats command give correct counts?
# 6. Query - does SQL-like query work?

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="${1:-}"

if [ -z "$PROJECT_DIR" ]; then
    echo "Usage: $0 <project_dir>"
    exit 1
fi

if [ ! -d "$PROJECT_DIR" ]; then
    echo "Error: Project directory not found: $PROJECT_DIR"
    exit 1
fi

cd "$PROJECT_DIR"

PASSED=0
FAILED=0
TOTAL=0

# Helper function
run_test() {
    local name="$1"
    local cmd="$2"
    local expected="$3"
    
    TOTAL=$((TOTAL + 1))
    echo -n "  $name... "
    
    OUTPUT=$(eval "$cmd" 2>&1) || true
    
    if echo "$OUTPUT" | grep -q "$expected"; then
        echo "✅ PASS"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo "❌ FAIL"
        echo "    Expected to find: $expected"
        echo "    Got: ${OUTPUT:0:200}"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

run_test_exit_code() {
    local name="$1"
    local cmd="$2"
    local expected_code="$3"
    
    TOTAL=$((TOTAL + 1))
    echo -n "  $name... "
    
    set +e
    eval "$cmd" > /dev/null 2>&1
    ACTUAL_CODE=$?
    set -e
    
    if [ "$ACTUAL_CODE" -eq "$expected_code" ]; then
        echo "✅ PASS"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo "❌ FAIL (exit code $ACTUAL_CODE, expected $expected_code)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║            Log Aggregator Eval Suite                         ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Project: $PROJECT_DIR"
echo ""

# =============================================================================
# TEST 1: Build
# =============================================================================
echo "🔨 Build Tests"

# Find main.go location
if [ -f "cmd/logagg/main.go" ]; then
    BUILD_PATH="./cmd/logagg"
elif [ -f "main.go" ]; then
    BUILD_PATH="."
else
    echo "  ❌ FAIL: Cannot find main.go"
    FAILED=$((FAILED + 1))
    TOTAL=$((TOTAL + 1))
    BUILD_PATH=""
fi

if [ -n "$BUILD_PATH" ]; then
    run_test_exit_code "go build succeeds" "go build -o logagg $BUILD_PATH" 0
fi

# Check binary exists
if [ -f "logagg" ]; then
    BINARY="./logagg"
elif [ -f "bin/logagg" ]; then
    BINARY="./bin/logagg"
else
    echo "  ❌ FAIL: Binary not found after build"
    FAILED=$((FAILED + 1))
    TOTAL=$((TOTAL + 1))
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Results: $PASSED passed, $FAILED failed out of $TOTAL"
    echo "═══════════════════════════════════════════════════════════════"
    exit 1
fi

# =============================================================================
# TEST 2: Help/Usage
# =============================================================================
echo ""
echo "📖 Help Tests"

run_test "help command shows usage" "$BINARY --help" "Usage"
run_test "parse subcommand exists" "$BINARY parse --help 2>&1 || $BINARY --help" "parse"

# =============================================================================
# TEST 3: Parse JSON Logs
# =============================================================================
echo ""
echo "📄 JSON Parse Tests"

FIXTURES="$SCRIPT_DIR/fixtures"

run_test "parse JSON logs" "$BINARY parse $FIXTURES/json.log --format json 2>&1 || $BINARY parse $FIXTURES/json.log 2>&1" "Application started"
run_test "parse JSON shows error level" "$BINARY parse $FIXTURES/json.log --format json 2>&1 || $BINARY parse $FIXTURES/json.log 2>&1" "error"
run_test "parse JSON shows timestamp" "$BINARY parse $FIXTURES/json.log --format json 2>&1 || $BINARY parse $FIXTURES/json.log 2>&1" "2024"

# =============================================================================
# TEST 4: Parse Apache Logs
# =============================================================================
echo ""
echo "🌐 Apache Parse Tests"

run_test "parse Apache logs" "$BINARY parse $FIXTURES/apache.log --format apache 2>&1 || $BINARY parse $FIXTURES/apache.log 2>&1" "GET"
run_test "parse Apache shows status" "$BINARY parse $FIXTURES/apache.log --format apache 2>&1 || $BINARY parse $FIXTURES/apache.log 2>&1" "200"

# =============================================================================
# TEST 5: Filter by Level
# =============================================================================
echo ""
echo "🔍 Filter Tests"

run_test "filter by error level" "$BINARY filter $FIXTURES/json.log --level error 2>&1" "error"
run_test "filter excludes info when filtering error" "$BINARY filter $FIXTURES/json.log --level error 2>&1 | grep -c 'info' || echo '0'" "0"

# =============================================================================
# TEST 6: Stats
# =============================================================================
echo ""
echo "📊 Stats Tests"

run_test "stats command runs" "$BINARY stats $FIXTURES/json.log 2>&1" ""
run_test "stats shows count" "$BINARY stats $FIXTURES/json.log 2>&1" "10\|count"

# =============================================================================
# TEST 7: Query (if implemented)
# =============================================================================
echo ""
echo "🔎 Query Tests"

# Query is advanced - test if it exists
if $BINARY query --help 2>&1 | grep -q "query\|SELECT"; then
    run_test "query SELECT works" "$BINARY query $FIXTURES/json.log \"SELECT level FROM logs\" 2>&1 || $BINARY query $FIXTURES/json.log 'SELECT level FROM logs' 2>&1" "info\|error\|warn"
else
    echo "  ⏭️  SKIP: query command not implemented"
fi

# =============================================================================
# Summary
# =============================================================================
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Results: $PASSED passed, $FAILED failed out of $TOTAL"
echo "═══════════════════════════════════════════════════════════════"

# Output JSON results
cat > .eval_results.json << EOF
{
  "suite": "logagg",
  "passed": $PASSED,
  "failed": $FAILED,
  "total": $TOTAL,
  "score": $(echo "scale=2; $PASSED / $TOTAL * 100" | bc 2>/dev/null || echo "0")
}
EOF

echo ""
echo "Results saved to: .eval_results.json"

# Exit with failure if any tests failed
[ "$FAILED" -eq 0 ]
