#!/bin/bash

# ============================================================================
# DEPRECATED: This shell script is superseded by the Go implementation
# Use 'ralph eval compare' command instead (see internal/eval/compare.go)
# This script is kept for backwards compatibility and fallback purposes only
# ============================================================================

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

# Calculate percentage differences
calc_pct_diff() {
    local ralph="$1" oneshot="$2"
    if [ "$oneshot" = "0" ] || [ -z "$oneshot" ]; then
        echo "N/A"
        return
    fi
    local diff=$(echo "scale=2; (($ralph - $oneshot) / $oneshot) * 100" | bc 2>/dev/null)
    if [ -z "$diff" ]; then
        echo "N/A"
        return
    fi
    # Determine winner based on metric type (lower is better for cost/time/tokens)
    local abs_diff=$(echo "$diff" | tr -d '-')
    if (( $(echo "$diff < 0" | bc -l) )); then
        echo "Ralph -${abs_diff}%"
    elif (( $(echo "$diff > 0" | bc -l) )); then
        echo "Oneshot -${abs_diff}%"
    else
        echo "Tie"
    fi
}

calc_pct_diff_tests() {
    local ralph="$1" oneshot="$2"
    if [ "$oneshot" = "0" ] || [ -z "$oneshot" ]; then
        echo "N/A"
        return
    fi
    local diff=$(echo "scale=2; (($ralph - $oneshot) / $oneshot) * 100" | bc 2>/dev/null)
    if [ -z "$diff" ]; then
        echo "N/A"
        return
    fi
    # For tests, higher is better
    local abs_diff=$(echo "$diff" | tr -d '-')
    if (( $(echo "$diff > 0" | bc -l) )); then
        echo "Ralph +${abs_diff}%"
    elif (( $(echo "$diff < 0" | bc -l) )); then
        echo "Oneshot +${abs_diff}%"
    else
        echo "Tie"
    fi
}

# Calculate winners
DURATION_WINNER=$(calc_pct_diff "$R_TIME" "$O_TIME")
TOKENS_WINNER=$(calc_pct_diff "$R_TOTAL" "$O_TOTAL")
COST_WINNER=$(calc_pct_diff "$R_COST" "$O_COST")
TESTS_WINNER=$(calc_pct_diff_tests "$R_SHARED" "$O_SHARED")

# Print comparison table
printf "┌──────────────────────┬───────────────────┬───────────────────┬─────────────────────┐\n"
printf "│ %-20s │ %-17s │ %-17s │ %-19s │\n" "Metric" "Ralph" "Oneshot" "Winner"
printf "├──────────────────────┼───────────────────┼───────────────────┼─────────────────────┤\n"
printf "│ %-20s │ %15ss │ %15ss │ %-19s │\n" "Duration" "$R_TIME" "$O_TIME" "$DURATION_WINNER"
printf "│ %-20s │ %17s │ %17s │ %-19s │\n" "Total Tokens" "$R_TOTAL" "$O_TOTAL" "$TOKENS_WINNER"
printf "│ %-20s │ \$%16s │ \$%16s │ %-19s │\n" "Cost" "$R_COST" "$O_COST" "$COST_WINNER"
printf "│ %-20s │ %13s/%s │ %13s/%s │ %-19s │\n" "Shared Tests" "$R_SHARED" "$R_SHARED_TOTAL" "$O_SHARED" "$O_SHARED_TOTAL" "$TESTS_WINNER"
printf "└──────────────────────┴───────────────────┴───────────────────┴─────────────────────┘\n"

echo ""

# Summary line
echo "═══════════════════════════════════════════════════════════════════════════════════════"
RALPH_WINS=0
ONESHOT_WINS=0
TIES=0

[[ "$DURATION_WINNER" == Ralph* ]] && RALPH_WINS=$((RALPH_WINS + 1))
[[ "$DURATION_WINNER" == Oneshot* ]] && ONESHOT_WINS=$((ONESHOT_WINS + 1))
[[ "$DURATION_WINNER" == "Tie" ]] && TIES=$((TIES + 1))

[[ "$TOKENS_WINNER" == Ralph* ]] && RALPH_WINS=$((RALPH_WINS + 1))
[[ "$TOKENS_WINNER" == Oneshot* ]] && ONESHOT_WINS=$((ONESHOT_WINS + 1))
[[ "$TOKENS_WINNER" == "Tie" ]] && TIES=$((TIES + 1))

[[ "$COST_WINNER" == Ralph* ]] && RALPH_WINS=$((RALPH_WINS + 1))
[[ "$COST_WINNER" == Oneshot* ]] && ONESHOT_WINS=$((ONESHOT_WINS + 1))
[[ "$COST_WINNER" == "Tie" ]] && TIES=$((TIES + 1))

[[ "$TESTS_WINNER" == Ralph* ]] && RALPH_WINS=$((RALPH_WINS + 1))
[[ "$TESTS_WINNER" == Oneshot* ]] && ONESHOT_WINS=$((ONESHOT_WINS + 1))
[[ "$TESTS_WINNER" == "Tie" ]] && TIES=$((TIES + 1))

if [ "$RALPH_WINS" -gt "$ONESHOT_WINS" ]; then
    echo "🏆 OVERALL WINNER: Ralph (${RALPH_WINS} metrics vs ${ONESHOT_WINS})"
elif [ "$ONESHOT_WINS" -gt "$RALPH_WINS" ]; then
    echo "🏆 OVERALL WINNER: Oneshot (${ONESHOT_WINS} metrics vs ${RALPH_WINS})"
else
    echo "🤝 TIE: Both approaches won ${RALPH_WINS} metrics each"
fi
[ "$TIES" -gt 0 ] && echo "   (${TIES} metric(s) tied)"
echo "═══════════════════════════════════════════════════════════════════════════════════════"
echo ""
