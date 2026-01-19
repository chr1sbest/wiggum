#!/bin/bash

# Validate a task tracker project
# Usage: ./validate.sh <project_dir>

PROJECT_DIR="${1:-.}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
SKIP=0

check() {
    local name="$1"
    local result="$2"
    if [ "$result" -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $name"
        ((PASS++))
    else
        echo -e "${RED}✗${NC} $name"
        ((FAIL++))
    fi
}

skip() {
    local name="$1"
    local reason="$2"
    echo -e "${YELLOW}○${NC} $name (skipped: $reason)"
    ((SKIP++))
}

cleanup() {
    # Kill any background processes we started
    if [ -n "$APP_PID" ]; then
        kill $APP_PID 2>/dev/null || true
        wait $APP_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

cd "$PROJECT_DIR" || { echo "Cannot cd to $PROJECT_DIR"; exit 1; }

echo "=== Validating Task Tracker API ==="
echo "Directory: $(pwd)"
echo ""

# 1. Check required files exist
echo "--- File Structure ---"
[ -f "app.py" ]; check "app.py exists" $?
[ -f "requirements.txt" ]; check "requirements.txt exists" $?
[ -f "README.md" ]; check "README.md exists" $?
[ -d "tests" ] || [ -f "test_app.py" ] || [ -f "tests.py" ]; check "tests exist" $?

# Advanced files
[ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ]; check "docker-compose.yml exists" $?
[ -f ".env.example" ]; check ".env.example exists" $?
[ -f "Makefile" ]; check "Makefile exists" $?

# Check requirements.txt has expected deps
if [ -f "requirements.txt" ]; then
    grep -qi "redis\|fakeredis" requirements.txt; check "requirements.txt includes Redis" $?
    grep -qi "celery" requirements.txt; check "requirements.txt includes Celery" $?
    grep -qi "boto3\|moto" requirements.txt; check "requirements.txt includes S3 libs" $?
fi

echo ""

# 2. Set up virtual environment and install deps
echo "--- Setup ---"
if [ ! -d "venv" ]; then
    python3 -m venv venv 2>/dev/null
fi
source venv/bin/activate 2>/dev/null || { echo "Failed to activate venv"; exit 1; }

pip install -r requirements.txt -q 2>/dev/null
check "dependencies installed" $?

echo ""

# 3. Start the app
echo "--- App Startup ---"
python app.py &
APP_PID=$!
sleep 3

# Check if app is running
if kill -0 $APP_PID 2>/dev/null; then
    check "app starts without crash" 0
else
    check "app starts without crash" 1
    echo "App failed to start. Remaining checks will fail."
fi

echo ""

# 4. Test endpoints
echo "--- API Endpoints ---"

# Register
REGISTER_RESP=$(curl -s -X POST http://localhost:8000/auth/register \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"testpass123"}' \
    -w "\n%{http_code}")
REGISTER_CODE=$(echo "$REGISTER_RESP" | tail -1)
[ "$REGISTER_CODE" -eq 200 ] || [ "$REGISTER_CODE" -eq 201 ]; check "POST /auth/register returns 2xx" $?

# Login
LOGIN_RESP=$(curl -s -X POST http://localhost:8000/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"testpass123"}')
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"token"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' || echo "")
if [ -z "$TOKEN" ]; then
    # Try alternate JSON key names
    TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' || echo "")
fi
[ -n "$TOKEN" ]; check "POST /auth/login returns token" $?

# Protected route without auth
NOAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/projects)
[ "$NOAUTH_CODE" -eq 401 ]; check "GET /projects without auth returns 401" $?

# Protected route with auth
if [ -n "$TOKEN" ]; then
    AUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        http://localhost:8000/projects)
    [ "$AUTH_CODE" -eq 200 ]; check "GET /projects with auth returns 200" $?

    # Create project
    CREATE_PROJ_RESP=$(curl -s -X POST http://localhost:8000/projects \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"name":"Test Project","description":"A test project"}' \
        -w "\n%{http_code}")
    CREATE_PROJ_CODE=$(echo "$CREATE_PROJ_RESP" | tail -1)
    [ "$CREATE_PROJ_CODE" -eq 200 ] || [ "$CREATE_PROJ_CODE" -eq 201 ]; check "POST /projects creates project" $?

    # Get project ID from response (try common patterns)
    PROJ_BODY=$(echo "$CREATE_PROJ_RESP" | head -n -1)
    PROJ_ID=$(echo "$PROJ_BODY" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1 || echo "1")
    [ -z "$PROJ_ID" ] && PROJ_ID="1"

    # Create task
    CREATE_TASK_RESP=$(curl -s -X POST \
        "http://localhost:8000/projects/$PROJ_ID/tasks" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"title":"Test Task","description":"A test task","priority":"high"}' \
        -w "\n%{http_code}")
    CREATE_TASK_CODE=$(echo "$CREATE_TASK_RESP" | tail -1)
    [ "$CREATE_TASK_CODE" -eq 200 ] || [ "$CREATE_TASK_CODE" -eq 201 ]; check "POST /projects/:id/tasks creates task" $?

    # Get task ID
    TASK_BODY=$(echo "$CREATE_TASK_RESP" | head -n -1)
    TASK_ID=$(echo "$TASK_BODY" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1 || echo "1")
    [ -z "$TASK_ID" ] && TASK_ID="1"

    # --- Advanced Feature Checks ---
    echo ""
    echo "--- Advanced Features ---"

    # Notifications endpoint
    NOTIF_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        http://localhost:8000/notifications)
    [ "$NOTIF_CODE" -eq 200 ]; check "GET /notifications returns 200" $?

    # Unread count
    UNREAD_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        http://localhost:8000/notifications/unread-count)
    [ "$UNREAD_CODE" -eq 200 ]; check "GET /notifications/unread-count returns 200" $?

    # Activity log
    ACTIVITY_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/projects/$PROJ_ID/activity")
    [ "$ACTIVITY_CODE" -eq 200 ]; check "GET /projects/:id/activity returns 200" $?

    # Webhooks CRUD
    WEBHOOK_LIST_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/projects/$PROJ_ID/webhooks")
    [ "$WEBHOOK_LIST_CODE" -eq 200 ]; check "GET /projects/:id/webhooks returns 200" $?

    # Task filtering
    FILTER_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/tasks?status=todo&priority=high")
    [ "$FILTER_CODE" -eq 200 ]; check "GET /tasks with filters returns 200" $?

    # Task search
    SEARCH_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/tasks?q=test")
    [ "$SEARCH_CODE" -eq 200 ]; check "GET /tasks with search returns 200" $?

    # Export CSV
    EXPORT_CSV_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/projects/$PROJ_ID/export?format=csv")
    [ "$EXPORT_CSV_CODE" -eq 200 ]; check "GET /projects/:id/export?format=csv returns 200" $?

    # Export JSON
    EXPORT_JSON_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/projects/$PROJ_ID/export?format=json")
    [ "$EXPORT_JSON_CODE" -eq 200 ]; check "GET /projects/:id/export?format=json returns 200" $?

    # Batch create tasks
    BATCH_CREATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        http://localhost:8000/tasks/batch \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"project_id\":$PROJ_ID,\"tasks\":[{\"title\":\"Batch 1\"},{\"title\":\"Batch 2\"}]}")
    [ "$BATCH_CREATE_CODE" -eq 200 ] || [ "$BATCH_CREATE_CODE" -eq 201 ]; check "POST /tasks/batch creates tasks" $?

    # Attachments list (should work even if empty)
    ATTACH_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "http://localhost:8000/tasks/$TASK_ID/attachments")
    [ "$ATTACH_CODE" -eq 200 ]; check "GET /tasks/:id/attachments returns 200" $?

    # Check X-Cache header exists (caching)
    CACHE_HEADER=$(curl -s -D - -o /dev/null \
        -H "Authorization: Bearer $TOKEN" \
        http://localhost:8000/projects 2>/dev/null | grep -i "x-cache")
    [ -n "$CACHE_HEADER" ]; check "X-Cache header present (caching)" $?

    # Rate limiting - make many requests quickly (optional, may not trigger)
    # This is more of a stress test, skip for basic validation

else
    skip "GET /projects with auth" "no token"
    skip "POST /projects" "no token"
    skip "POST /projects/:id/tasks" "no token"
    skip "Advanced features" "no token"
fi

echo ""

# 5. Run tests
echo "--- Tests ---"
# Stop app before running tests (some test frameworks start their own)
kill $APP_PID 2>/dev/null || true
sleep 1

if [ -d "tests" ]; then
    pytest tests/ -q 2>/dev/null
    check "pytest tests/ passes" $?
elif [ -f "test_app.py" ]; then
    pytest test_app.py -q 2>/dev/null
    check "pytest test_app.py passes" $?
elif [ -f "tests.py" ]; then
    pytest tests.py -q 2>/dev/null
    check "pytest tests.py passes" $?
else
    skip "pytest" "no test files found"
fi

echo ""
echo "=== Summary ==="
echo -e "${GREEN}Passed: $PASS${NC}"
echo -e "${RED}Failed: $FAIL${NC}"
echo -e "${YELLOW}Skipped: $SKIP${NC}"

# Output JSON result
RESULT_FILE="$SCRIPT_DIR/results/$(basename "$PROJECT_DIR")_validation.json"
mkdir -p "$SCRIPT_DIR/results"
cat > "$RESULT_FILE" <<EOF
{
  "project": "$(basename "$PROJECT_DIR")",
  "passed": $PASS,
  "failed": $FAIL,
  "skipped": $SKIP,
  "total": $((PASS + FAIL + SKIP))
}
EOF
echo ""
echo "Validation results: $RESULT_FILE"

# Exit with failure if any checks failed
[ $FAIL -eq 0 ]
