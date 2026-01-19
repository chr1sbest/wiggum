#!/bin/bash

# Compare eval results from the unified eval framework
# Usage: ./compare_evals.sh <suite> [model]
#    or: ./compare_evals.sh [ralph_result.json] [oneshot_result.json]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"

# Helper to extract JSON values
get_val() {
    local file="$1" key="$2"
    grep -oE "\"$key\"[[:space:]]*:[[:space:]]*[0-9.]+" "$file" 2>/dev/null | grep -oE '[0-9.]+$' || echo "0"
}

# Find latest result file for suite/approach/model
find_latest_result() {
    local suite="$1" approach="$2" model="${3:-sonnet-4-5}"
    ls -t "$RESULTS_DIR"/${suite}-${approach}-${model}-*.json 2>/dev/null | head -1
}

# Parse arguments
if [ $# -eq 1 ]; then
    # Single argument: suite name, use default model
    SUITE="$1"
    MODEL="sonnet-4-5"
    RALPH_METRICS=$(find_latest_result "$SUITE" "ralph" "$MODEL")
    ONESHOT_METRICS=$(find_latest_result "$SUITE" "oneshot" "$MODEL")
elif [ $# -eq 2 ] && [ ! -f "$1" ]; then
    # Two arguments, first is not a file: suite name and model
    SUITE="$1"
    MODEL="$2"
    RALPH_METRICS=$(find_latest_result "$SUITE" "ralph" "$MODEL")
    ONESHOT_METRICS=$(find_latest_result "$SUITE" "oneshot" "$MODEL")
else
    # Legacy mode: explicit paths to result files
    RALPH_METRICS="$1"
    ONESHOT_METRICS="$2"
fi

if [ -z "$RALPH_METRICS" ] || [ ! -f "$RALPH_METRICS" ]; then
    echo "Usage: $0 <suite> [model]"
    echo "   or: $0 <ralph_result.json> <oneshot_result.json>"
    echo ""
    echo "Available result files:"
    ls "$RESULTS_DIR"/*.json 2>/dev/null | xargs -I{} basename {}
    exit 1
fi

if [ -z "$ONESHOT_METRICS" ] || [ ! -f "$ONESHOT_METRICS" ]; then
    echo "❌ Missing oneshot result file"
    echo "Ralph result: $RALPH_METRICS"
    exit 1
fi

# Extract values (new standardized format)
R_TIME=$(get_val "$RALPH_METRICS" "duration_seconds")
R_CALLS=$(get_val "$RALPH_METRICS" "total_calls")
R_INPUT=$(get_val "$RALPH_METRICS" "input_tokens")
R_OUTPUT=$(get_val "$RALPH_METRICS" "output_tokens")
R_TOTAL=$(get_val "$RALPH_METRICS" "total_tokens")
R_COST=$(get_val "$RALPH_METRICS" "cost_usd")
R_SHARED=$(get_val "$RALPH_METRICS" "shared_tests_passed")
R_SHARED_TOTAL=$(get_val "$RALPH_METRICS" "shared_tests_total")
R_FILES=$(get_val "$RALPH_METRICS" "files_generated")
R_LINES=$(get_val "$RALPH_METRICS" "lines_generated")

O_TIME=$(get_val "$ONESHOT_METRICS" "duration_seconds")
O_CALLS=$(get_val "$ONESHOT_METRICS" "total_calls")
O_INPUT=$(get_val "$ONESHOT_METRICS" "input_tokens")
O_OUTPUT=$(get_val "$ONESHOT_METRICS" "output_tokens")
O_TOTAL=$(get_val "$ONESHOT_METRICS" "total_tokens")
O_COST=$(get_val "$ONESHOT_METRICS" "cost_usd")
O_SHARED=$(get_val "$ONESHOT_METRICS" "shared_tests_passed")
O_SHARED_TOTAL=$(get_val "$ONESHOT_METRICS" "shared_tests_total")
O_FILES=$(get_val "$ONESHOT_METRICS" "files_generated")
O_LINES=$(get_val "$ONESHOT_METRICS" "lines_generated")

echo ""
echo "╔══════════════════════════════════════════════════════════════════════╗"
echo "║                    EVAL COMPARISON: Ralph vs One-Shot                ║"
echo "╚══════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Ralph:   $(basename "$RALPH_METRICS")"
echo "Oneshot: $(basename "$ONESHOT_METRICS")"
echo ""

# Print comparison table
printf "┌──────────────────────┬───────────────────┬───────────────────┐\n"
printf "│ %-20s │ %-17s │ %-17s │\n" "Metric" "Ralph" "One-Shot"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %15ss │ %15ss │\n" "Duration" "$R_TIME" "$O_TIME"
printf "│ %-20s │ %17s │ %17s │\n" "Claude Calls" "$R_CALLS" "$O_CALLS"
printf "│ %-20s │ %17s │ %17s │\n" "Total Tokens" "$R_TOTAL" "$O_TOTAL"
printf "│ %-20s │ %16s │ %16s │\n" "Cost" "\$$R_COST" "\$$O_COST"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %13s/%s │ %13s/%s │\n" "Tests Passed" "$R_SHARED" "$R_SHARED_TOTAL" "$O_SHARED" "$O_SHARED_TOTAL"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %17s │ %17s │\n" "Files Generated" "$R_FILES" "$O_FILES"
printf "│ %-20s │ %17s │ %17s │\n" "Lines Generated" "$R_LINES" "$O_LINES"
printf "└──────────────────────┴───────────────────┴───────────────────┘\n"

echo ""

# Determine winner based on tests passed (the fair comparison)
echo "═══════════════════════════════════════════════════════════════════════"
if [ "$R_SHARED" -gt "$O_SHARED" ]; then
    echo "🏆 WINNER: Ralph ($R_SHARED vs $O_SHARED tests passed)"
elif [ "$O_SHARED" -gt "$R_SHARED" ]; then
    echo "🏆 WINNER: One-Shot ($O_SHARED vs $R_SHARED tests passed)"
else
    echo "🤝 TIE on tests passed ($R_SHARED each)"
    # Break tie with cost
    R_COST_INT=$(echo "$R_COST" | cut -d. -f1)
    O_COST_INT=$(echo "$O_COST" | cut -d. -f1)
    if [ "$R_COST_INT" -lt "$O_COST_INT" ]; then
        echo "   Ralph wins on cost (\$$R_COST vs \$$O_COST)"
    else
        echo "   One-shot wins on cost (\$$O_COST vs \$$R_COST)"
    fi
fi
echo "═══════════════════════════════════════════════════════════════════════"
echo ""
