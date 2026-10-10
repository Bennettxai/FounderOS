#!/bin/bash

# =============================================================================
# OSA Queue Worker Test Runner
# =============================================================================
# This script automates testing of the OSA queue worker implementation
# =============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOG_FILE="$SCRIPT_DIR/osa_worker_test_$(date +%Y%m%d_%H%M%S).log"
REPORT_FILE="$PROJECT_ROOT/../../OSA_WORKER_TEST_REPORT.md"

# Database connection from .env
if [ -f "$PROJECT_ROOT/.env" ]; then
    source "$PROJECT_ROOT/.env"
fi

# Extract connection details from DATABASE_URL
# Format: postgresql://user:pass@host:port/dbname?params
DB_URL="${DATABASE_URL}"

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}OSA Queue Worker Test Runner${NC}"
echo -e "${BLUE}==============================================================================${NC}"
echo ""
echo "Project Root: $PROJECT_ROOT"
echo "Log File: $LOG_FILE"
echo "Report File: $REPORT_FILE"
echo ""

# Function to log messages
log() {
    echo -e "$1" | tee -a "$LOG_FILE"
}

# Function to run SQL and capture output
run_sql() {
    local sql="$1"
    local description="$2"

    log "${YELLOW}▶ $description${NC}"

    if [ -z "$DB_URL" ]; then
        log "${RED}❌ DATABASE_URL not set in .env${NC}"
        return 1
    fi

    # Run SQL using psql via docker or direct connection
    # This assumes psql is available or docker postgres client
    echo "$sql" | psql "$DB_URL" 2>&1 | tee -a "$LOG_FILE"

    if [ $? -eq 0 ]; then
        log "${GREEN}✅ Success${NC}\n"
        return 0
    else
        log "${RED}❌ Failed${NC}\n"
        return 1
    fi
}

# Function to check if server is running
check_server() {
    if curl -s http://localhost:8001/health > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# =============================================================================
# Test Steps
# =============================================================================

log "${BLUE}=== Step 1: Check Migration Status ===${NC}"
run_sql "SELECT
    CASE
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'app_templates')
        THEN 'app_templates exists'
        ELSE 'app_templates NOT FOUND - run migration!'
    END as app_templates_status,
    CASE
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'app_generation_queue')
        THEN 'app_generation_queue exists'
        ELSE 'app_generation_queue NOT FOUND - run migration!'
    END as queue_status;" "Check tables exist"

log "${BLUE}=== Step 2: Run Migration (if needed) ===${NC}"
log "Running migrations..."
cd "$PROJECT_ROOT"

# Check if goose is available
if command -v goose &> /dev/null; then
    goose -dir internal/database/migrations postgres "$DB_URL" up 2>&1 | tee -a "$LOG_FILE"
else
    log "${YELLOW}⚠ goose not found, attempting to run via go run${NC}"
    # Alternative: run migrations via Go code if available
    if [ -f "cmd/migrate/main.go" ]; then
        go run cmd/migrate/main.go up 2>&1 | tee -a "$LOG_FILE"
    else
        log "${RED}❌ Cannot run migrations - goose not available${NC}"
    fi
fi

log "${BLUE}=== Step 3: Seed Test Data ===${NC}"
log "Inserting test templates..."

# Read and execute the SQL test script
if [ -f "$SCRIPT_DIR/test_osa_worker.sql" ]; then
    cat "$SCRIPT_DIR/test_osa_worker.sql" | psql "$DB_URL" 2>&1 | tee -a "$LOG_FILE"
else
    log "${RED}❌ test_osa_worker.sql not found${NC}"
    exit 1
fi

log "${BLUE}=== Step 4: Check Server Status ===${NC}"
if check_server; then
    log "${YELLOW}⚠ Server is already running${NC}"
    log "Will monitor existing server..."
    SERVER_ALREADY_RUNNING=1
else
    log "${GREEN}✅ Server not running, will start it${NC}"
    SERVER_ALREADY_RUNNING=0
fi

# Start server if not running
if [ $SERVER_ALREADY_RUNNING -eq 0 ]; then
    log "${BLUE}=== Step 5: Start Server ===${NC}"
    log "Building server..."
    cd "$PROJECT_ROOT"
    go build -o bin/server-test ./cmd/server 2>&1 | tee -a "$LOG_FILE"

    if [ $? -ne 0 ]; then
        log "${RED}❌ Build failed${NC}"
        exit 1
    fi

    log "Starting server in background..."
    ./bin/server-test > "$SCRIPT_DIR/server_output.log" 2>&1 &
    SERVER_PID=$!
    log "Server PID: $SERVER_PID"

    # Wait for server to start
    log "Waiting for server to start..."
    for i in {1..30}; do
        if check_server; then
            log "${GREEN}✅ Server started successfully${NC}"
            break
        fi
        sleep 1
        echo -n "."
    done
    echo ""

    if ! check_server; then
        log "${RED}❌ Server failed to start${NC}"
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
fi

log "${BLUE}=== Step 6: Monitor Worker Activity ===${NC}"
log "Watching for worker activity for 60 seconds..."
log "Worker polls every 5 seconds..."

START_TIME=$(date +%s)
MONITOR_DURATION=60

while [ $(($(date +%s) - START_TIME)) -lt $MONITOR_DURATION ]; do
    log "\n${YELLOW}--- Status Check ($(date +%H:%M:%S)) ---${NC}"

    # Query queue status
    run_sql "SELECT
        status,
        COUNT(*) as count,
        STRING_AGG(generation_context->>'app_name', ', ') as apps
    FROM app_generation_queue
    GROUP BY status
    ORDER BY
        CASE status
            WHEN 'processing' THEN 1
            WHEN 'pending' THEN 2
            WHEN 'completed' THEN 3
            WHEN 'failed' THEN 4
        END;" "Queue status"

    # Check for completed items
    COMPLETED_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM app_generation_queue WHERE status IN ('completed', 'failed');" 2>/dev/null | xargs)

    if [ "$COMPLETED_COUNT" -gt 0 ]; then
        log "${GREEN}✅ Found $COMPLETED_COUNT completed/failed items${NC}"
        break
    fi

    sleep 10
done

log "${BLUE}=== Step 7: Final Results ===${NC}"

# Get detailed results
run_sql "SELECT
    q.id,
    q.generation_context->>'app_name' as app_name,
    t.template_name,
    q.status,
    q.started_at,
    q.completed_at,
    CASE
        WHEN q.completed_at IS NOT NULL AND q.started_at IS NOT NULL THEN
            EXTRACT(EPOCH FROM (q.completed_at - q.started_at)) || ' seconds'
        ELSE
            'N/A'
    END as duration,
    SUBSTRING(q.error_message, 1, 100) as error_preview
FROM app_generation_queue q
LEFT JOIN app_templates t ON q.template_id = t.id
ORDER BY q.created_at DESC
LIMIT 10;" "Detailed results"

# Stop server if we started it
if [ $SERVER_ALREADY_RUNNING -eq 0 ] && [ ! -z "$SERVER_PID" ]; then
    log "${BLUE}=== Stopping Test Server ===${NC}"
    kill $SERVER_PID 2>/dev/null || true
    log "Server stopped"
fi

log "${BLUE}=== Step 8: Check Server Logs ===${NC}"
log "Last 50 lines of server output:"
if [ -f "$SCRIPT_DIR/server_output.log" ]; then
    tail -n 50 "$SCRIPT_DIR/server_output.log" | tee -a "$LOG_FILE"
else
    log "${YELLOW}⚠ Server log file not found${NC}"
fi

# Search for worker-related log entries
log "\n${YELLOW}Worker-related log entries:${NC}"
grep -i "osa.*worker\|queue.*worker\|processing.*queue" "$SCRIPT_DIR/server_output.log" 2>/dev/null | tail -n 20 | tee -a "$LOG_FILE" || log "No worker logs found"

log "${BLUE}==============================================================================${NC}"
log "${GREEN}Test completed!${NC}"
log "Full log: $LOG_FILE"
log "Report: $REPORT_FILE"
log "${BLUE}==============================================================================${NC}"

# Generate report
log "\nGenerating test report..."
bash "$SCRIPT_DIR/generate_osa_worker_report.sh" "$LOG_FILE" "$REPORT_FILE"

log "${GREEN}✅ All done!${NC}"
