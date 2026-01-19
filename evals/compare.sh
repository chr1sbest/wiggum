#!/bin/bash

# Compare eval runs by reading .ralph/run_metrics.json from each project
# Usage: ./compare.sh [suite]           - auto-find latest ralph/oneshot for suite
#        ./compare.sh [ralph] [oneshot] - compare specific dirs

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WIGGUM_DIR="$SCRIPT_DIR/.."

# If single arg and not a directory, treat as suite name
if [ $# -eq 1 ] && [ ! -d "$1" ]; then
    SUITE="$1"
    RALPH_DIR=$(ls -dt "$WIGGUM_DIR"/eval-ralph-${SUITE}-* 2>/dev/null | head -1)
    ONESHOT_DIR=$(ls -dt "$WIGGUM_DIR"/eval-oneshot-${SUITE}-* 2>/dev/null | head -1)
else
    # Auto-find latest eval dirs if not specified
    RALPH_DIR="${1:-$(ls -dt "$WIGGUM_DIR"/eval-ralph-* 2>/dev/null | head -1)}"
    ONESHOT_DIR="${2:-$(ls -dt "$WIGGUM_DIR"/eval-oneshot-* 2>/dev/null | head -1)}"
fi

if [ -z "$RALPH_DIR" ] || [ -z "$ONESHOT_DIR" ]; then
    echo "Usage: $0 [suite]              - auto-find latest for suite"
    echo "       $0 [ralph_dir] [oneshot_dir]"
    echo ""
    echo "Examples:"
    echo "  $0 logagg"
    echo "  $0 eval-ralph-logagg-sonnet-123 eval-oneshot-logagg-sonnet-456"
    echo ""
    echo "Available eval dirs:"
    ls -d "$WIGGUM_DIR"/eval-* 2>/dev/null | xargs -I{} basename {} | head -10
    exit 1
fi

RALPH_METRICS="$RALPH_DIR/.ralph/run_metrics.json"
ONESHOT_METRICS="$ONESHOT_DIR/.ralph/run_metrics.json"

# Helper to extract JSON values
get_val() {
    local file="$1" key="$2"
    grep -oE "\"$key\"[[:space:]]*:[[:space:]]*[0-9.]+" "$file" 2>/dev/null | grep -oE '[0-9.]+$' || echo "0"
}

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              EVAL COMPARISON: Ralph vs One-Shot              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Ralph:   $(basename "$RALPH_DIR")"
echo "Oneshot: $(basename "$ONESHOT_DIR")"
echo ""

# Check files exist
if [ ! -f "$RALPH_METRICS" ]; then
    echo "❌ Missing: $RALPH_METRICS"
    exit 1
fi
if [ ! -f "$ONESHOT_METRICS" ]; then
    echo "❌ Missing: $ONESHOT_METRICS"
    echo "   (Run oneshot eval with updated script to generate metrics)"
    exit 1
fi

# Extract all values
R_TIME=$(get_val "$RALPH_METRICS" "elapsed_seconds")
R_CALLS=$(get_val "$RALPH_METRICS" "total_claude_calls")
R_INPUT=$(get_val "$RALPH_METRICS" "input_tokens")
R_OUTPUT=$(get_val "$RALPH_METRICS" "output_tokens")
R_COST=$(get_val "$RALPH_METRICS" "total_cost_usd")
R_TESTS=$(get_val "$RALPH_METRICS" "tests_passed")
R_TESTS_TOTAL=$(get_val "$RALPH_METRICS" "tests_total")

O_TIME=$(get_val "$ONESHOT_METRICS" "elapsed_seconds")
O_CALLS=$(get_val "$ONESHOT_METRICS" "total_claude_calls")
O_INPUT=$(get_val "$ONESHOT_METRICS" "input_tokens")
O_OUTPUT=$(get_val "$ONESHOT_METRICS" "output_tokens")
O_TOTAL=$(get_val "$ONESHOT_METRICS" "total_tokens")
O_COST=$(get_val "$ONESHOT_METRICS" "total_cost_usd")
O_TESTS=$(get_val "$ONESHOT_METRICS" "tests_passed")
O_TESTS_TOTAL=$(get_val "$ONESHOT_METRICS" "tests_total")
O_FILES=$(get_val "$ONESHOT_METRICS" "files_generated")
O_LINES=$(get_val "$ONESHOT_METRICS" "lines_generated")

# Calculate ralph totals
R_TOTAL=$((R_INPUT + R_OUTPUT))

# Print comparison table
printf "┌────────────────────┬─────────────────┬─────────────────┐\n"
printf "│ %-18s │ %-15s │ %-15s │\n" "Metric" "Ralph" "One-Shot"
printf "├────────────────────┼─────────────────┼─────────────────┤\n"
printf "│ %-18s │ %13ss │ %13ss │\n" "Time" "$R_TIME" "$O_TIME"
printf "│ %-18s │ %15s │ %15s │\n" "Claude Calls" "$R_CALLS" "$O_CALLS"
printf "│ %-18s │ %15s │ %15s │\n" "Input Tokens" "$R_INPUT" "$O_INPUT"
printf "│ %-18s │ %15s │ %15s │\n" "Output Tokens" "$R_OUTPUT" "$O_OUTPUT"
printf "│ %-18s │ %15s │ %15s │\n" "Total Tokens" "$R_TOTAL" "$O_TOTAL"
printf "│ %-18s │ %14s │ %14s │\n" "Cost" "\$$R_COST" "\$$O_COST"
printf "├────────────────────┼─────────────────┼─────────────────┤\n"
printf "│ %-18s │ %11s/%s │ %11s/%s │\n" "Tests Passed" "$R_TESTS" "$R_TESTS_TOTAL" "$O_TESTS" "$O_TESTS_TOTAL"
printf "└────────────────────┴─────────────────┴─────────────────┘\n"

echo ""

# Determine winner
echo "═══════════════════════════════════════════════════════════════"
if [ "$R_TESTS" -gt "$O_TESTS" ]; then
    echo "🏆 WINNER: Ralph ($R_TESTS vs $O_TESTS tests passed)"
elif [ "$O_TESTS" -gt "$R_TESTS" ]; then
    echo "🏆 WINNER: One-Shot ($O_TESTS vs $R_TESTS tests passed)"
else
    echo "🤝 TIE on tests ($R_TESTS each)"
    if [ "$R_COST" \< "$O_COST" ]; then
        echo "   Ralph wins on cost (\$$R_COST vs \$$O_COST)"
    else
        echo "   One-shot wins on cost (\$$O_COST vs \$$R_COST)"
    fi
fi
echo "═══════════════════════════════════════════════════════════════"
