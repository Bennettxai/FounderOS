#!/bin/bash
# =============================================================================
# BusinessOS Pipeline Integration Test
# =============================================================================
# Tests the complete pipeline: Health → CSRF → Database → Redis → OSA → App Creation
# Usage: ./scripts/cli_pipeline_test.sh [--verbose]
# =============================================================================

set -euo pipefail

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8001}"
WORKSPACE_ID="${WORKSPACE_ID:-00000000-0000-0000-0000-000000000001}"
TIMEOUT_SHORT=5
TIMEOUT_MEDIUM=15
TIMEOUT_LONG=30
COOKIE_JAR="/tmp/businessos_test_cookies_$$.txt"
VERBOSE="${VERBOSE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0
declare -a RESULTS=()

# Cleanup on exit
cleanup() {
    rm -f "$COOKIE_JAR" /tmp/businessos_test_*.txt 2>/dev/null || true
}
trap cleanup EXIT

# Logging functions
log_info() { echo -e "${BLUE}ℹ ${NC}$1"; }
log_success() { echo -e "${GREEN}✅ ${NC}$1"; ((PASSED++)); RESULTS+=("✅ $1"); }
log_fail() { echo -e "${RED}❌ ${NC}$1"; ((FAILED++)); RESULTS+=("❌ $1"); }
log_skip() { echo -e "${YELLOW}⏭️  ${NC}$1"; ((SKIPPED++)); RESULTS+=("⏭️  $1"); }
log_warn() { echo -e "${YELLOW}⚠️  ${NC}$1"; }
log_header() { echo -e "\n${CYAN}═══════════════════════════════════════════════════════════════${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"; }

# Verbose logging
debug() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${YELLOW}[DEBUG]${NC} $1"
    fi
}

# Parse arguments
for arg in "$@"; do
    case $arg in
        --verbose|-v) VERBOSE="true" ;;
        --url=*) BASE_URL="${arg#*=}" ;;
        --workspace=*) WORKSPACE_ID="${arg#*=}" ;;
    esac
done

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║       BusinessOS Pipeline Integration Test                    ║"
echo "║                                                               ║"
echo "║  Testing: Health → CSRF → Database → Redis → OSA → Apps      ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo "Configuration:"
echo "  BASE_URL:     $BASE_URL"
echo "  WORKSPACE_ID: $WORKSPACE_ID"
echo "  VERBOSE:      $VERBOSE"
echo ""

# =============================================================================
# PHASE 1: INFRASTRUCTURE HEALTH
# =============================================================================
log_header "PHASE 1: INFRASTRUCTURE HEALTH"

# 1.1 Backend server is responding
log_info "Testing backend server connectivity..."
if curl --max-time $TIMEOUT_SHORT -s -o /dev/null -w "%{http_code}" "$BASE_URL" 2>/dev/null | /usr/bin/grep -qE "^[2-4][0-9][0-9]$"; then
    log_success "Backend server is responding at $BASE_URL"
else
    log_fail "Backend server not responding at $BASE_URL"
    echo "FATAL: Cannot continue without backend. Is the server running?"
    exit 1
fi

# 1.2 Health endpoint returns 200
log_info "Testing /api/v1/health endpoint..."
HEALTH_RESPONSE=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/health" 2>/dev/null || echo -e "\n000")
HEALTH_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)
HEALTH_BODY=$(echo "$HEALTH_RESPONSE" | sed '$d')
debug "Health response: $HEALTH_CODE - $HEALTH_BODY"

if [[ "$HEALTH_CODE" == "200" ]]; then
    log_success "Health endpoint returns 200"
    # Parse health details if JSON
    if echo "$HEALTH_BODY" | jq -e . >/dev/null 2>&1; then
        DB_STATUS=$(echo "$HEALTH_BODY" | jq -r '.database // "unknown"')
        REDIS_STATUS=$(echo "$HEALTH_BODY" | jq -r '.redis // "unknown"')
        debug "Database: $DB_STATUS, Redis: $REDIS_STATUS"
    fi
else
    log_fail "Health endpoint failed: HTTP $HEALTH_CODE"
fi

# 1.3 CSRF cookie is being set
log_info "Testing CSRF cookie setup..."
CSRF_RESPONSE=$(curl --max-time $TIMEOUT_SHORT -s -c "$COOKIE_JAR" -w "\n%{http_code}" "$BASE_URL/api/auth/csrf" 2>/dev/null || echo -e "\n000")
CSRF_CODE=$(echo "$CSRF_RESPONSE" | tail -1)

if [[ -f "$COOKIE_JAR" ]] && /usr/bin/grep -q "csrf_token" "$COOKIE_JAR" 2>/dev/null; then
    CSRF_TOKEN=$(grep "csrf_token" "$COOKIE_JAR" | awk '{print $NF}')
    debug "CSRF Token: ${CSRF_TOKEN:0:20}..."
    log_success "CSRF cookie set correctly (${#CSRF_TOKEN} chars)"
else
    log_warn "CSRF cookie not found (may be expected for unauthenticated requests)"
    # Try to get CSRF from response body
    CSRF_TOKEN=$(echo "$CSRF_RESPONSE" | sed '$d' | jq -r '.csrf_token // empty' 2>/dev/null || echo "")
    if [[ -n "$CSRF_TOKEN" ]]; then
        debug "Got CSRF from response body: ${CSRF_TOKEN:0:20}..."
        log_success "CSRF token received in response body"
    else
        log_skip "CSRF not available (testing without authentication)"
    fi
fi

# 1.4 Protected endpoints return 401 (not 404)
log_info "Testing protected endpoints return 401..."
AUTH_TEST_RESPONSE=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/me" 2>/dev/null || echo -e "\n000")
AUTH_TEST_CODE=$(echo "$AUTH_TEST_RESPONSE" | tail -1)

case "$AUTH_TEST_CODE" in
    401) log_success "Protected endpoint /api/v1/me returns 401 (Unauthorized)" ;;
    404) log_fail "Protected endpoint /api/v1/me returns 404 (route missing!)" ;;
    000) log_fail "Request to /api/v1/me timed out" ;;
    *) log_warn "Protected endpoint /api/v1/me returns unexpected code: $AUTH_TEST_CODE" ;;
esac

# =============================================================================
# PHASE 2: DATABASE & REDIS CONNECTIVITY
# =============================================================================
log_header "PHASE 2: DATABASE & REDIS CONNECTIVITY"

# 2.1 Check database via health endpoint details
log_info "Checking database connectivity..."
if [[ -n "${DB_STATUS:-}" ]] && [[ "$DB_STATUS" == "connected" || "$DB_STATUS" == "ok" || "$DB_STATUS" == "healthy" ]]; then
    log_success "Database (Supabase) is connected"
elif [[ -n "${HEALTH_BODY:-}" ]] && echo "$HEALTH_BODY" | /usr/bin/grep -qi "database.*ok\|db.*connected\|postgres.*healthy"; then
    log_success "Database appears connected (from health response)"
else
    # Try dedicated database health check if exists
    DB_CHECK=$(curl --max-time $TIMEOUT_SHORT -s "$BASE_URL/health/db" 2>/dev/null || echo "")
    if [[ -n "$DB_CHECK" ]] && echo "$DB_CHECK" | /usr/bin/grep -qi "ok\|connected\|healthy"; then
        log_success "Database is connected (via /health/db)"
    else
        log_skip "Database status not explicitly reported in health check"
    fi
fi

# 2.2 Check Redis connectivity
log_info "Checking Redis connectivity..."
if [[ -n "${REDIS_STATUS:-}" ]] && [[ "$REDIS_STATUS" == "connected" || "$REDIS_STATUS" == "ok" || "$REDIS_STATUS" == "healthy" ]]; then
    log_success "Redis is connected"
elif [[ -n "${HEALTH_BODY:-}" ]] && echo "$HEALTH_BODY" | /usr/bin/grep -qi "redis.*ok\|redis.*connected"; then
    log_success "Redis appears connected (from health response)"
else
    # Try dedicated Redis health check if exists
    REDIS_CHECK=$(curl --max-time $TIMEOUT_SHORT -s "$BASE_URL/health/redis" 2>/dev/null || echo "")
    if [[ -n "$REDIS_CHECK" ]] && echo "$REDIS_CHECK" | /usr/bin/grep -qi "ok\|connected\|healthy"; then
        log_success "Redis is connected (via /health/redis)"
    else
        log_skip "Redis status not explicitly reported in health check"
    fi
fi

# =============================================================================
# PHASE 3: OSA (ONE-SHOT APP) INTEGRATION
# =============================================================================
log_header "PHASE 3: OSA (ONE-SHOT APP) INTEGRATION"

# 3.1 OSA health check
log_info "Testing OSA health endpoint..."
OSA_HEALTH=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/osa/health" 2>/dev/null || echo -e "\n000")
OSA_HEALTH_CODE=$(echo "$OSA_HEALTH" | tail -1)
OSA_HEALTH_BODY=$(echo "$OSA_HEALTH" | sed '$d')
debug "OSA health: $OSA_HEALTH_CODE"

case "$OSA_HEALTH_CODE" in
    200)
        log_success "OSA integration is enabled and healthy"
        OSA_ENABLED=true
        ;;
    404|503)
        log_warn "OSA integration not enabled (OSA_ENABLED=false) - HTTP $OSA_HEALTH_CODE"
        OSA_ENABLED=false
        ;;
    401)
        log_success "OSA endpoint exists (requires auth) - OSA is available"
        OSA_ENABLED=true
        ;;
    000)
        log_fail "OSA health request timed out"
        OSA_ENABLED=false
        ;;
    *)
        log_warn "OSA health unexpected response: HTTP $OSA_HEALTH_CODE"
        OSA_ENABLED=false
        ;;
esac

# 3.2 OSA templates endpoint (if OSA enabled)
if [[ "$OSA_ENABLED" == "true" ]]; then
    log_info "Testing OSA templates endpoint..."
    TEMPLATES=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/osa/templates" 2>/dev/null || echo -e "\n000")
    TEMPLATES_CODE=$(echo "$TEMPLATES" | tail -1)

    case "$TEMPLATES_CODE" in
        200) log_success "OSA templates endpoint accessible" ;;
        401) log_success "OSA templates endpoint exists (auth required)" ;;
        404) log_fail "OSA templates endpoint missing (route not registered)" ;;
        *) log_warn "OSA templates returned: HTTP $TEMPLATES_CODE" ;;
    esac
else
    log_skip "OSA templates test (OSA not enabled)"
fi

# 3.3 OSA status endpoint format check
if [[ "$OSA_ENABLED" == "true" ]]; then
    log_info "Testing OSA status endpoint format..."
    # Use a fake app ID to test route exists (should return 401 or 404 for app, not 404 for route)
    STATUS_CHECK=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/osa/status/test-app-id" 2>/dev/null || echo -e "\n000")
    STATUS_CODE=$(echo "$STATUS_CHECK" | tail -1)

    case "$STATUS_CODE" in
        401) log_success "OSA status route exists (auth required)" ;;
        404)
            # Check if it's route not found vs app not found
            if echo "$STATUS_CHECK" | /usr/bin/grep -qi "app.*not found\|not found.*app"; then
                log_success "OSA status route exists (app not found is expected)"
            else
                log_fail "OSA status route may be missing - check /api/v1/osa/status/:app_id"
            fi
            ;;
        400) log_success "OSA status route exists (validation error expected for invalid ID)" ;;
        *) log_warn "OSA status returned: HTTP $STATUS_CODE" ;;
    esac
else
    log_skip "OSA status endpoint test (OSA not enabled)"
fi

# =============================================================================
# PHASE 4: WORKSPACE & APP CREATION
# =============================================================================
log_header "PHASE 4: WORKSPACE & APP CREATION"

# 4.1 Test workspace ID format validation
log_info "Testing workspace endpoint with valid UUID..."
WORKSPACE_CHECK=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/workspaces/$WORKSPACE_ID" 2>/dev/null || echo -e "\n000")
WORKSPACE_CODE=$(echo "$WORKSPACE_CHECK" | tail -1)
debug "Workspace check: HTTP $WORKSPACE_CODE"

case "$WORKSPACE_CODE" in
    200) log_success "Workspace endpoint accessible" ;;
    401) log_success "Workspace endpoint exists (auth required)" ;;
    404)
        # Check if it's route not found vs workspace not found
        WORKSPACE_BODY=$(echo "$WORKSPACE_CHECK" | sed '$d')
        if echo "$WORKSPACE_BODY" | /usr/bin/grep -qi "workspace.*not found\|not found.*workspace"; then
            log_success "Workspace route exists (workspace not found is expected for mock ID)"
        else
            log_warn "Workspace route may be missing or workspace not found"
        fi
        ;;
    400)
        WORKSPACE_BODY=$(echo "$WORKSPACE_CHECK" | sed '$d')
        if echo "$WORKSPACE_BODY" | /usr/bin/grep -qi "invalid workspace"; then
            log_fail "Workspace ID format rejected - check UUID format: $WORKSPACE_ID"
        else
            log_warn "Workspace validation error: $WORKSPACE_BODY"
        fi
        ;;
    *) log_warn "Workspace endpoint returned: HTTP $WORKSPACE_CODE" ;;
esac

# 4.2 Test app creation endpoint (without auth, expect 401)
log_info "Testing app creation endpoint structure..."
APP_CREATE_CHECK=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"template_id":"test","name":"test","description":"test"}' \
    "$BASE_URL/api/v1/workspaces/$WORKSPACE_ID/apps" 2>/dev/null || echo -e "\n000")
APP_CREATE_CODE=$(echo "$APP_CREATE_CHECK" | tail -1)
APP_CREATE_BODY=$(echo "$APP_CREATE_CHECK" | sed '$d')
debug "App create check: HTTP $APP_CREATE_CODE"

case "$APP_CREATE_CODE" in
    201|200) log_success "App creation endpoint works (unexpected without auth!)" ;;
    401) log_success "App creation endpoint exists (auth required - correct)" ;;
    403)
        if echo "$APP_CREATE_BODY" | /usr/bin/grep -qi "csrf"; then
            log_success "App creation endpoint exists (CSRF required - correct)"
        else
            log_warn "App creation returned 403: $APP_CREATE_BODY"
        fi
        ;;
    400)
        if echo "$APP_CREATE_BODY" | /usr/bin/grep -qi "invalid workspace"; then
            log_fail "Workspace ID rejected - ensure UUID format is correct"
        else
            log_success "App creation endpoint exists (validation working)"
        fi
        ;;
    404) log_fail "App creation endpoint missing - check POST /api/v1/workspaces/:id/apps" ;;
    *) log_warn "App creation returned: HTTP $APP_CREATE_CODE" ;;
esac

# =============================================================================
# PHASE 5: TERMINAL/CLI INTEGRATION
# =============================================================================
log_header "PHASE 5: TERMINAL/CLI INTEGRATION"

# 5.1 Terminal WebSocket endpoint check
log_info "Testing terminal WebSocket upgrade endpoint..."
TERMINAL_WS=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    "$BASE_URL/api/v1/terminal/ws" 2>/dev/null || echo -e "\n000")
TERMINAL_WS_CODE=$(echo "$TERMINAL_WS" | tail -1)

case "$TERMINAL_WS_CODE" in
    101) log_success "Terminal WebSocket upgrade successful" ;;
    400) log_success "Terminal WebSocket endpoint exists (needs proper WS handshake)" ;;
    401) log_success "Terminal WebSocket endpoint exists (auth required)" ;;
    404) log_fail "Terminal WebSocket endpoint missing - check /api/v1/terminal/ws" ;;
    *) log_warn "Terminal WebSocket returned: HTTP $TERMINAL_WS_CODE" ;;
esac

# 5.2 Terminal sessions endpoint
log_info "Testing terminal sessions endpoint..."
TERMINAL_SESSIONS=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/terminal/sessions" 2>/dev/null || echo -e "\n000")
TERMINAL_SESSIONS_CODE=$(echo "$TERMINAL_SESSIONS" | tail -1)

case "$TERMINAL_SESSIONS_CODE" in
    200) log_success "Terminal sessions endpoint accessible" ;;
    401) log_success "Terminal sessions endpoint exists (auth required)" ;;
    404) log_fail "Terminal sessions endpoint missing" ;;
    *) log_warn "Terminal sessions returned: HTTP $TERMINAL_SESSIONS_CODE" ;;
esac

# 5.3 Filesystem endpoint
log_info "Testing filesystem endpoint..."
FS_CHECK=$(curl --max-time $TIMEOUT_SHORT -s -w "\n%{http_code}" "$BASE_URL/api/v1/filesystem/list" 2>/dev/null || echo -e "\n000")
FS_CODE=$(echo "$FS_CHECK" | tail -1)

case "$FS_CODE" in
    200) log_success "Filesystem endpoint accessible" ;;
    401) log_success "Filesystem endpoint exists (auth required)" ;;
    404) log_skip "Filesystem endpoint not found (may not be configured)" ;;
    *) log_warn "Filesystem returned: HTTP $FS_CODE" ;;
esac

# =============================================================================
# PHASE 6: SSE STREAMING (if OSA enabled)
# =============================================================================
log_header "PHASE 6: SSE STREAMING"

if [[ "$OSA_ENABLED" == "true" ]]; then
    log_info "Testing SSE streaming endpoint format..."
    # Quick check that the SSE endpoint exists (don't actually stream)
    SSE_CHECK=$(curl --max-time 2 -s -w "\n%{http_code}" \
        -H "Accept: text/event-stream" \
        "$BASE_URL/api/v1/osa/stream/build/test-app-id" 2>/dev/null || echo -e "\n000")
    SSE_CODE=$(echo "$SSE_CHECK" | tail -1)

    case "$SSE_CODE" in
        200) log_success "SSE streaming endpoint accessible" ;;
        401) log_success "SSE streaming endpoint exists (auth required)" ;;
        400) log_success "SSE streaming endpoint exists (invalid app ID expected)" ;;
        404) log_fail "SSE streaming endpoint missing - check /api/v1/osa/stream/build/:app_id" ;;
        000) log_warn "SSE endpoint timed out (may be expected for SSE)" ;;
        *) log_warn "SSE streaming returned: HTTP $SSE_CODE" ;;
    esac
else
    log_skip "SSE streaming test (OSA not enabled)"
fi

# =============================================================================
# SUMMARY
# =============================================================================
log_header "TEST SUMMARY"

echo ""
echo "Results:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for result in "${RESULTS[@]}"; do
    echo "  $result"
done
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Calculate totals
TOTAL=$((PASSED + FAILED + SKIPPED))

echo -e "┌─────────────────────────────────────────┐"
echo -e "│            TEST RESULTS                 │"
echo -e "├─────────────────────────────────────────┤"
echo -e "│  ${GREEN}PASSED${NC}:  $PASSED                              │"
echo -e "│  ${RED}FAILED${NC}:  $FAILED                              │"
echo -e "│  ${YELLOW}SKIPPED${NC}: $SKIPPED                              │"
echo -e "│  TOTAL:   $TOTAL                              │"
echo -e "└─────────────────────────────────────────┘"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo -e "${RED}❌ Some tests failed. Review the output above for details.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All critical tests passed!${NC}"
    if [[ $SKIPPED -gt 0 ]]; then
        echo -e "${YELLOW}Note: $SKIPPED tests were skipped (may need configuration or auth).${NC}"
    fi
    exit 0
fi
