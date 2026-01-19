#!/bin/bash

# Compare eval results from the unified eval framework
# Usage: ./compare_evals.sh [ralph_dir] [oneshot_dir]
#    or: ./compare_evals.sh <suite>  (auto-finds latest for suite)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WIGGUM_DIR="$SCRIPT_DIR/.."

# Helper to extract JSON values
get_val() {
    local file="$1" key="$2"
    grep -oE "\"$key\"[[:space:]]*:[[:space:]]*[0-9.]+" "$file" 2>/dev/null | grep -oE '[0-9.]+$' || echo "0"
}

# If single arg, treat as suite name and find latest
if [ $# -eq 1 ] && [ ! -d "$1" ]; then
    SUITE="$1"
    RALPH_DIR=$(ls -dt "$WIGGUM_DIR"/eval-ralph-${SUITE}-* 2>/dev/null | head -1)
    ONESHOT_DIR=$(ls -dt "$WIGGUM_DIR"/eval-oneshot-${SUITE}-* 2>/dev/null | head -1)
else
    RALPH_DIR="${1:-$(ls -dt "$WIGGUM_DIR"/eval-ralph-* 2>/dev/null | head -1)}"
    ONESHOT_DIR="${2:-$(ls -dt "$WIGGUM_DIR"/eval-oneshot-* 2>/dev/null | head -1)}"
fi

if [ -z "$RALPH_DIR" ] || [ -z "$ONESHOT_DIR" ]; then
    echo "Usage: $0 [ralph_dir] [oneshot_dir]"
    echo "   or: $0 <suite_name>"
    echo ""
    echo "Available eval dirs:"
    ls -d "$WIGGUM_DIR"/eval-* 2>/dev/null | xargs -I{} basename {}
    exit 1
fi

RALPH_METRICS="$RALPH_DIR/.ralph/run_metrics.json"
ONESHOT_METRICS="$ONESHOT_DIR/.ralph/run_metrics.json"

if [ ! -f "$RALPH_METRICS" ]; then
    echo "❌ Missing: $RALPH_METRICS"
    exit 1
fi
if [ ! -f "$ONESHOT_METRICS" ]; then
    echo "❌ Missing: $ONESHOT_METRICS"
    exit 1
fi

# Extract values
R_TIME=$(get_val "$RALPH_METRICS" "elapsed_seconds")
R_CALLS=$(get_val "$RALPH_METRICS" "total_claude_calls")
R_INPUT=$(get_val "$RALPH_METRICS" "input_tokens")
R_OUTPUT=$(get_val "$RALPH_METRICS" "output_tokens")
R_TOTAL=$(get_val "$RALPH_METRICS" "total_tokens")
R_COST=$(get_val "$RALPH_METRICS" "total_cost_usd")
R_SHARED=$(get_val "$RALPH_METRICS" "shared_tests_passed")
R_SHARED_TOTAL=$(get_val "$RALPH_METRICS" "shared_tests_total")
R_OWN=$(get_val "$RALPH_METRICS" "own_tests_passed")
R_OWN_TOTAL=$(get_val "$RALPH_METRICS" "own_tests_total")
R_FILES=$(get_val "$RALPH_METRICS" "files_generated")
R_LINES=$(get_val "$RALPH_METRICS" "lines_generated")

O_TIME=$(get_val "$ONESHOT_METRICS" "elapsed_seconds")
O_CALLS=$(get_val "$ONESHOT_METRICS" "total_claude_calls")
O_INPUT=$(get_val "$ONESHOT_METRICS" "input_tokens")
O_OUTPUT=$(get_val "$ONESHOT_METRICS" "output_tokens")
O_TOTAL=$(get_val "$ONESHOT_METRICS" "total_tokens")
O_COST=$(get_val "$ONESHOT_METRICS" "total_cost_usd")
O_SHARED=$(get_val "$ONESHOT_METRICS" "shared_tests_passed")
O_SHARED_TOTAL=$(get_val "$ONESHOT_METRICS" "shared_tests_total")
O_OWN=$(get_val "$ONESHOT_METRICS" "own_tests_passed")
O_OWN_TOTAL=$(get_val "$ONESHOT_METRICS" "own_tests_total")
O_FILES=$(get_val "$ONESHOT_METRICS" "files_generated")
O_LINES=$(get_val "$ONESHOT_METRICS" "lines_generated")

# Fallback for old metrics format
if [ "$R_SHARED" == "0" ] && [ "$R_SHARED_TOTAL" == "0" ]; then
    R_SHARED=$(get_val "$RALPH_METRICS" "tests_passed")
    R_SHARED_TOTAL=$(get_val "$RALPH_METRICS" "tests_total")
fi
if [ "$O_SHARED" == "0" ] && [ "$O_SHARED_TOTAL" == "0" ]; then
    O_SHARED=$(get_val "$ONESHOT_METRICS" "tests_passed")
    O_SHARED_TOTAL=$(get_val "$ONESHOT_METRICS" "tests_total")
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════════════╗"
echo "║                    EVAL COMPARISON: Ralph vs One-Shot                ║"
echo "╚══════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Ralph:   $(basename "$RALPH_DIR")"
echo "Oneshot: $(basename "$ONESHOT_DIR")"
echo ""

# Print comparison table
printf "┌──────────────────────┬───────────────────┬───────────────────┐\n"
printf "│ %-20s │ %-17s │ %-17s │\n" "Metric" "Ralph" "One-Shot"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %15ss │ %15ss │\n" "Time" "$R_TIME" "$O_TIME"
printf "│ %-20s │ %17s │ %17s │\n" "Claude Calls" "$R_CALLS" "$O_CALLS"
printf "│ %-20s │ %17s │ %17s │\n" "Total Tokens" "$R_TOTAL" "$O_TOTAL"
printf "│ %-20s │ %16s │ %16s │\n" "Cost" "\$$R_COST" "\$$O_COST"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %13s/%s │ %13s/%s │\n" "Shared Tests ⭐" "$R_SHARED" "$R_SHARED_TOTAL" "$O_SHARED" "$O_SHARED_TOTAL"
printf "│ %-20s │ %13s/%s │ %13s/%s │\n" "Own Tests" "$R_OWN" "$R_OWN_TOTAL" "$O_OWN" "$O_OWN_TOTAL"
printf "├──────────────────────┼───────────────────┼───────────────────┤\n"
printf "│ %-20s │ %17s │ %17s │\n" "Files Generated" "$R_FILES" "$O_FILES"
printf "│ %-20s │ %17s │ %17s │\n" "Lines Generated" "$R_LINES" "$O_LINES"
printf "└──────────────────────┴───────────────────┴───────────────────┘\n"

echo ""
echo "⭐ Shared Tests = same test suite run against both implementations"
echo ""

# Determine winner based on shared tests (the fair comparison)
echo "═══════════════════════════════════════════════════════════════════════"
if [ "$R_SHARED" -gt "$O_SHARED" ]; then
    echo "🏆 WINNER: Ralph ($R_SHARED vs $O_SHARED shared tests passed)"
elif [ "$O_SHARED" -gt "$R_SHARED" ]; then
    echo "🏆 WINNER: One-Shot ($O_SHARED vs $R_SHARED shared tests passed)"
else
    echo "🤝 TIE on shared tests ($R_SHARED each)"
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
