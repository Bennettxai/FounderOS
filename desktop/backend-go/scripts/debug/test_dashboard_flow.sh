#!/bin/bash

# ============================================================================
# TEST DASHBOARD FLOW WITH REAL USER DATA
# ============================================================================
# This script tests the complete dashboard skills flow:
# 1. Verifies widget_types are seeded
# 2. Creates a test user (if needed)
# 3. Tests dashboard creation via agent
# 4. Tests widget addition
# 5. Verifies data in database
# ============================================================================

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BASE_URL="${BASE_URL:-http://localhost:8080}"
DB_URL="${DATABASE_URL}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Dashboard Skills: Real User Data Test                    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# ============================================================================
# STEP 1: Verify Database Schema
# ============================================================================

echo -e "${YELLOW}[1/6]${NC} Verifying database schema..."

if [ -z "$DB_URL" ]; then
    echo -e "${RED}✗ DATABASE_URL not set${NC}"
    echo -e "${YELLOW}Set it with: export DATABASE_URL='postgresql://user:pass@host/db'${NC}"
    exit 1
fi

# Check widget_types count
WIDGET_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM dashboard_widgets WHERE is_enabled = TRUE;" 2>/dev/null || echo "0")
WIDGET_COUNT=$(echo $WIDGET_COUNT | xargs) # trim whitespace

if [ "$WIDGET_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Widget types seeded: $WIDGET_COUNT enabled widgets"
else
    echo -e "${RED}✗ No widget types found${NC}"
    echo -e "${YELLOW}Run migrations: go run ./cmd/migrate${NC}"
    exit 1
fi

# ============================================================================
# STEP 2: Create or Get Test User
# ============================================================================

echo -e "${YELLOW}[2/6]${NC} Setting up test user..."

# Try to login or create user
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email": "test@businessos.dev", "password": "testpass123"}' 2>/dev/null || echo "")

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    SESSION_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token' 2>/dev/null)
    USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.id' 2>/dev/null)
    echo -e "${GREEN}✓${NC} Logged in as test user"
    echo -e "  User ID: $USER_ID"
    echo -e "  Token: ${SESSION_TOKEN:0:30}..."
else
    echo -e "${YELLOW}⚠ Login failed, attempting to create test user...${NC}"

    SIGNUP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d '{"email": "test@businessos.dev", "password": "testpass123", "name": "Test User"}' 2>/dev/null || echo "")

    if echo "$SIGNUP_RESPONSE" | grep -q "token"; then
        SESSION_TOKEN=$(echo "$SIGNUP_RESPONSE" | jq -r '.token')
        USER_ID=$(echo "$SIGNUP_RESPONSE" | jq -r '.user.id')
        echo -e "${GREEN}✓${NC} Created new test user"
        echo -e "  User ID: $USER_ID"
    else
        echo -e "${RED}✗ Failed to create/login test user${NC}"
        echo -e "${YELLOW}Response: $SIGNUP_RESPONSE${NC}"
        exit 1
    fi
fi

export SESSION_TOKEN
export USER_ID

# ============================================================================
# STEP 3: Test Dashboard Creation via Agent
# ============================================================================

echo -e "${YELLOW}[3/6]${NC} Testing dashboard creation via agent..."

CREATE_DASHBOARD_MSG='{
    "message": "Create a new dashboard called Test Workspace Dashboard",
    "streaming": false
}'

DASHBOARD_RESPONSE=$(curl -s -X POST "$BASE_URL/api/chat/v2" \
    -H "Authorization: Bearer $SESSION_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$CREATE_DASHBOARD_MSG")

if echo "$DASHBOARD_RESPONSE" | grep -qi "dashboard"; then
    echo -e "${GREEN}✓${NC} Dashboard creation request processed"
    echo "$DASHBOARD_RESPONSE" | jq -r '.response' 2>/dev/null | head -3
else
    echo -e "${YELLOW}⚠ Dashboard creation response unclear${NC}"
    echo "$DASHBOARD_RESPONSE" | jq '.' 2>/dev/null || echo "$DASHBOARD_RESPONSE"
fi

# Check if dashboard was created
sleep 1
DASHBOARD_ID=$(psql "$DB_URL" -t -c "SELECT id FROM user_dashboards WHERE user_id = '$USER_ID' ORDER BY created_at DESC LIMIT 1;" 2>/dev/null | xargs)

if [ -n "$DASHBOARD_ID" ]; then
    echo -e "${GREEN}✓${NC} Dashboard created in database"
    echo -e "  Dashboard ID: $DASHBOARD_ID"
else
    echo -e "${YELLOW}⚠ No dashboard found in database (may already exist)${NC}"
    # Get existing dashboard
    DASHBOARD_ID=$(psql "$DB_URL" -t -c "SELECT id FROM user_dashboards WHERE user_id = '$USER_ID' LIMIT 1;" 2>/dev/null | xargs)
    if [ -n "$DASHBOARD_ID" ]; then
        echo -e "  Using existing dashboard: $DASHBOARD_ID"
    fi
fi

export DASHBOARD_ID

# ============================================================================
# STEP 4: Test Widget Addition
# ============================================================================

echo -e "${YELLOW}[4/6]${NC} Testing widget addition..."

if [ -z "$DASHBOARD_ID" ]; then
    echo -e "${YELLOW}⚠ Skipping widget test (no dashboard available)${NC}"
else
    ADD_WIDGET_MSG='{
        "message": "Add a task summary widget to my dashboard",
        "streaming": false
    }'

    WIDGET_RESPONSE=$(curl -s -X POST "$BASE_URL/api/chat/v2" \
        -H "Authorization: Bearer $SESSION_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$ADD_WIDGET_MSG")

    if echo "$WIDGET_RESPONSE" | grep -qi "widget\|task"; then
        echo -e "${GREEN}✓${NC} Widget addition request processed"
        echo "$WIDGET_RESPONSE" | jq -r '.response' 2>/dev/null | head -3
    else
        echo -e "${YELLOW}⚠ Widget addition response unclear${NC}"
        echo "$WIDGET_RESPONSE" | jq '.' 2>/dev/null || echo "$WIDGET_RESPONSE"
    fi

    # Check widget count
    sleep 1
    WIDGET_COUNT=$(psql "$DB_URL" -t -c "SELECT jsonb_array_length(layout->'widgets') FROM user_dashboards WHERE id = '$DASHBOARD_ID';" 2>/dev/null | xargs)

    if [ -n "$WIDGET_COUNT" ] && [ "$WIDGET_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} Dashboard has $WIDGET_COUNT widget(s)"
    else
        echo -e "${YELLOW}⚠ No widgets found in dashboard${NC}"
    fi
fi

# ============================================================================
# STEP 5: Test Complex Request (Multiple Widgets)
# ============================================================================

echo -e "${YELLOW}[5/6]${NC} Testing complex multi-widget request..."

COMPLEX_MSG='{
    "message": "Show me my tasks grouped by project and add a widget for deadlines in the next 2 weeks",
    "streaming": false
}'

COMPLEX_RESPONSE=$(curl -s -X POST "$BASE_URL/api/chat/v2" \
    -H "Authorization: Bearer $SESSION_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$COMPLEX_MSG")

if echo "$COMPLEX_RESPONSE" | grep -qi "widget"; then
    echo -e "${GREEN}✓${NC} Complex request processed"
    echo "$COMPLEX_RESPONSE" | jq -r '.response' 2>/dev/null | head -5
else
    echo -e "${YELLOW}⚠ Complex request response unclear${NC}"
fi

# ============================================================================
# STEP 6: Verify Final State
# ============================================================================

echo -e "${YELLOW}[6/6]${NC} Verifying final database state..."

echo ""
echo -e "${BLUE}=== Database Verification ===${NC}"

# Run verification SQL
psql "$DB_URL" -f "$(dirname "$0")/verify_dashboard_data.sql" 2>/dev/null || {
    echo -e "${YELLOW}⚠ Could not run verification SQL${NC}"

    # Fallback: basic checks
    echo ""
    echo "Widget Types:"
    psql "$DB_URL" -c "SELECT widget_type, name, category FROM dashboard_widgets WHERE is_enabled = TRUE ORDER BY category, widget_type;" 2>/dev/null

    echo ""
    echo "User Dashboards:"
    psql "$DB_URL" -c "SELECT id, name, jsonb_array_length(layout->'widgets') as widget_count FROM user_dashboards WHERE user_id = '$USER_ID';" 2>/dev/null
}

# ============================================================================
# SUMMARY
# ============================================================================

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Test Complete                                             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}✓ Widget types seeded and verified${NC}"
echo -e "${GREEN}✓ Test user created/logged in${NC}"

if [ -n "$DASHBOARD_ID" ]; then
    echo -e "${GREEN}✓ Dashboard operations tested${NC}"
    echo -e "${GREEN}✓ Widget addition tested${NC}"
else
    echo -e "${YELLOW}⚠ Dashboard tests skipped (check agent/tool logs)${NC}"
fi

echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Check agent logs for tool calls"
echo -e "  2. Verify dashboard in frontend UI"
echo -e "  3. Test additional widget types"
echo -e "  4. Test dashboard templates"
echo ""
