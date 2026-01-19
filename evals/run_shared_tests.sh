#!/bin/bash

# ============================================================================
# DEPRECATED: This shell script is superseded by the Go implementation
# Use the Go test runner (internal/eval/testrunner.go) via 'ralph eval run'
# This script is kept for backwards compatibility and fallback purposes only
# ============================================================================

# Run shared e2e tests against a project
# Usage: ./run_shared_tests.sh <project_dir> <suite>
#
# This script:
# 1. Starts the app in the background
# 2. Waits for it to be ready
# 3. Runs the shared test suite
# 4. Kills the app
# 5. Reports results

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

PROJECT_DIR="${1:-}"
SUITE="${2:-tasktracker}"
PORT="${3:-8000}"

if [ -z "$PROJECT_DIR" ]; then
    echo "Usage: $0 <project_dir> [suite] [port]"
    echo ""
    echo "Example:"
    echo "  $0 eval-ralph-tasktracker-sonnet-123456 tasktracker 8000"
    exit 1
fi

if [ ! -d "$PROJECT_DIR" ]; then
    echo "Error: Project directory not found: $PROJECT_DIR"
    exit 1
fi

SUITE_DIR="$SCRIPT_DIR/suites/$SUITE"
if [ ! -d "$SUITE_DIR" ]; then
    echo "Error: Suite not found: $SUITE_DIR"
    exit 1
fi

# Find the actual app directory (handle nested dirs)
APP_DIR="$PROJECT_DIR"
if [ -d "$PROJECT_DIR/$PROJECT_DIR" ]; then
    # Nested structure like eval-ralph-.../eval-ralph-...
    INNER=$(ls -d "$PROJECT_DIR"/*/ 2>/dev/null | head -1)
    if [ -n "$INNER" ] && [ -f "$INNER/app.py" -o -d "$INNER/app" ]; then
        APP_DIR="$INNER"
    fi
fi

# Check for app.py or run.py
if [ ! -f "$APP_DIR/app.py" ] && [ ! -f "$APP_DIR/run.py" ] && [ ! -d "$APP_DIR/app" ]; then
    echo "Error: No app.py, run.py, or app/ found in $APP_DIR"
    exit 1
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║            Running Shared E2E Tests                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Project: $APP_DIR"
echo "Suite:   $SUITE"
echo "Port:    $PORT"
echo ""

cd "$APP_DIR"

# Set up venv if needed
if [ -f "requirements.txt" ] && [ ! -d "venv" ]; then
    echo "Setting up venv..."
    python3 -m venv venv
fi

if [ -d "venv" ]; then
    source venv/bin/activate
    pip install -q requests pytest 2>/dev/null || true
    pip install -q -r requirements.txt 2>/dev/null || true
fi

# Set up environment if .env doesn't exist
if [ -f ".env.example" ] && [ ! -f ".env" ]; then
    echo "Copying .env.example to .env..."
    cp .env.example .env
fi

# Start docker services if docker-compose.yml exists
if [ -f "docker-compose.yml" ]; then
    echo "Starting Docker services (Redis, MinIO)..."
    docker-compose up -d 2>/dev/null || true
    sleep 3  # Wait for services to start
fi

# Kill any existing process on the port
lsof -ti:$PORT | xargs kill -9 2>/dev/null || true

# Start the app
echo "Starting app on port $PORT..."
if [ -f "run.py" ]; then
    python run.py &
elif [ -f "app.py" ]; then
    python app.py &
else
    # Try Flask run
    FLASK_APP=app flask run --port $PORT &
fi
APP_PID=$!

# Wait for app to be ready
echo "Waiting for app to start..."
for i in {1..30}; do
    if curl -s "http://localhost:$PORT" > /dev/null 2>&1 || curl -s "http://localhost:$PORT/auth/login" > /dev/null 2>&1; then
        echo "App is ready!"
        break
    fi
    sleep 1
done

# Check if app started
if ! kill -0 $APP_PID 2>/dev/null; then
    echo "Error: App failed to start"
    exit 1
fi

echo ""
echo "Running tests..."
echo ""

# Run the shared tests
export EVAL_BASE_URL="http://localhost:$PORT"
PYTEST_OUT=$(pytest "$SUITE_DIR" --tb=short -v 2>&1) || true

echo "$PYTEST_OUT"

# Parse results
PASSED=$(echo "$PYTEST_OUT" | grep -oE "[0-9]+ passed" | grep -oE "[0-9]+" || echo "0")
FAILED=$(echo "$PYTEST_OUT" | grep -oE "[0-9]+ failed" | grep -oE "[0-9]+" || echo "0")
SKIPPED=$(echo "$PYTEST_OUT" | grep -oE "[0-9]+ skipped" | grep -oE "[0-9]+" || echo "0")

# Kill the app
kill $APP_PID 2>/dev/null || true

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Results: $PASSED passed, $FAILED failed, $SKIPPED skipped"
echo "═══════════════════════════════════════════════════════════════"

# Return exit code based on failures
[ "$FAILED" -eq 0 ]
