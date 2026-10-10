#!/bin/bash
# Pre-demo checklist - Run right before Orgo.ai presentation
# Usage: ./scripts/pre_demo_checklist.sh

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              PRE-DEMO CHECKLIST (Final Verification)            ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "Run this script 5 minutes before the demo to ensure everything works"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CHECKS_PASSED=0
CHECKS_FAILED=0

# Check 1: Backend server
echo "🔍 [1/10] Checking backend server..."
if curl -s http://localhost:8001/api/onboarding/status > /dev/null 2>&1; then
    echo -e "      ${GREEN}✅ Backend responding on port 8001${NC}"
    ((CHECKS_PASSED++))
else
    echo -e "      ${RED}❌ Backend NOT responding${NC}"
    echo "         Fix: cd desktop/backend-go && go run cmd/server/main.go"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 2: Frontend server
echo "🔍 [2/10] Checking frontend server..."
if curl -s http://localhost:5173 > /dev/null 2>&1; then
    echo -e "      ${GREEN}✅ Frontend responding on port 5173${NC}"
    ((CHECKS_PASSED++))
else
    echo -e "      ${RED}❌ Frontend NOT responding${NC}"
    echo "         Fix: cd frontend && npm run dev"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 3: Database connection
echo "🔍 [3/10] Checking database connection..."
if [ -f ".env" ]; then
    export $(cat .env | grep -v '^#' | xargs)

    if psql "$DATABASE_URL" -c "SELECT 1" > /dev/null 2>&1; then
        echo -e "      ${GREEN}✅ Database connected${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${RED}❌ Database connection failed${NC}"
        echo "         Fix: Check Supabase project status"
        ((CHECKS_FAILED++))
    fi
else
    echo -e "      ${RED}❌ .env file not found${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 4: App templates seeded
echo "🔍 [4/10] Checking app templates..."
if [ ! -z "$DATABASE_URL" ]; then
    TEMPLATE_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM app_templates" 2>/dev/null | tr -d ' ')

    if [ "$TEMPLATE_COUNT" -ge "10" ]; then
        echo -e "      ${GREEN}✅ $TEMPLATE_COUNT app templates seeded${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${YELLOW}⚠️  Only $TEMPLATE_COUNT templates (expected 10)${NC}"
        echo "         Fix: Run migration 081"
    fi
else
    echo -e "      ${RED}❌ Cannot check (no DATABASE_URL)${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 5: Test data exists
echo "🔍 [5/10] Checking test data..."
if [ ! -z "$DATABASE_URL" ]; then
    USER_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM \"user\"" 2>/dev/null | tr -d ' ')
    WORKSPACE_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM workspace" 2>/dev/null | tr -d ' ')

    if [ "$USER_COUNT" -gt "0" ] && [ "$WORKSPACE_COUNT" -gt "0" ]; then
        echo -e "      ${GREEN}✅ Test data exists ($USER_COUNT users, $WORKSPACE_COUNT workspaces)${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${YELLOW}⚠️  No test data (users: $USER_COUNT, workspaces: $WORKSPACE_COUNT)${NC}"
        echo "         Tip: Create test user with scripts/create_test_user.go"
    fi
else
    echo -e "      ${RED}❌ Cannot check${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 6: Groq API key
echo "🔍 [6/10] Checking Groq API key..."
if [ -f ".env" ]; then
    if grep -q "GROQ_API_KEY=gsk_" .env; then
        echo -e "      ${GREEN}✅ Groq API key configured${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${RED}❌ Groq API key missing or invalid${NC}"
        echo "         Fix: Set GROQ_API_KEY in .env"
        ((CHECKS_FAILED++))
    fi
else
    echo -e "      ${RED}❌ .env file not found${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 7: Google OAuth credentials
echo "🔍 [7/10] Checking Google OAuth..."
if [ -f ".env" ]; then
    if grep -q "GOOGLE_CLIENT_ID=" .env && grep -q "GOOGLE_CLIENT_SECRET=" .env; then
        echo -e "      ${GREEN}✅ Google OAuth credentials configured${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${RED}❌ Google OAuth credentials missing${NC}"
        echo "         Fix: Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET"
        ((CHECKS_FAILED++))
    fi
else
    echo -e "      ${RED}❌ .env file not found${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Check 8: Backend logs accessible
echo "🔍 [8/10] Checking backend logs..."
LOG_FILE="$TEMP/claude/C--Users-Pichau-Desktop-BusinessOS-main-dev/tasks/b1a7e05.output"
if [ -f "$LOG_FILE" ]; then
    RECENT_LOGS=$(tail -100 "$LOG_FILE" 2>/dev/null | wc -l)
    if [ "$RECENT_LOGS" -gt "0" ]; then
        echo -e "      ${GREEN}✅ Backend logs accessible ($RECENT_LOGS recent lines)${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${YELLOW}⚠️  Backend logs empty${NC}"
    fi
else
    echo -e "      ${YELLOW}⚠️  Backend log file not found${NC}"
    echo "         (This is OK if backend just started)"
fi
echo ""

# Check 9: Fallback materials ready
echo "🔍 [9/10] Checking fallback materials..."
DEMO_SCRIPT_EXISTS=false
SCREENSHOTS_EXIST=false

if [ -f "DEMO_SCRIPT.md" ]; then
    DEMO_SCRIPT_EXISTS=true
fi

if [ -d "docs/demo-screenshots" ]; then
    SCREENSHOT_COUNT=$(ls docs/demo-screenshots/*.png 2>/dev/null | wc -l)
    if [ "$SCREENSHOT_COUNT" -gt "0" ]; then
        SCREENSHOTS_EXIST=true
    fi
fi

if [ "$DEMO_SCRIPT_EXISTS" = true ]; then
    echo -e "      ${GREEN}✅ Demo script ready (DEMO_SCRIPT.md)${NC}"
    ((CHECKS_PASSED++))
else
    echo -e "      ${YELLOW}⚠️  Demo script not found${NC}"
fi

if [ "$SCREENSHOTS_EXIST" = true ]; then
    echo -e "      ${GREEN}✅ Screenshots ready ($SCREENSHOT_COUNT files)${NC}"
else
    echo -e "      ${YELLOW}⚠️  No screenshots found${NC}"
    echo "         Tip: Take screenshots during E2E testing"
fi
echo ""

# Check 10: System performance
echo "🔍 [10/10] Checking system performance..."
if [ ! -z "$DATABASE_URL" ]; then
    START_TIME=$(date +%s%3N)
    psql "$DATABASE_URL" -c "SELECT 1" > /dev/null 2>&1
    END_TIME=$(date +%s%3N)
    LATENCY=$((END_TIME - START_TIME))

    if [ "$LATENCY" -lt "100" ]; then
        echo -e "      ${GREEN}✅ Database latency: ${LATENCY}ms (excellent)${NC}"
        ((CHECKS_PASSED++))
    elif [ "$LATENCY" -lt "300" ]; then
        echo -e "      ${GREEN}✅ Database latency: ${LATENCY}ms (good)${NC}"
        ((CHECKS_PASSED++))
    else
        echo -e "      ${YELLOW}⚠️  Database latency: ${LATENCY}ms (slow)${NC}"
        echo "         Demo may feel sluggish"
    fi
else
    echo -e "      ${RED}❌ Cannot check${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

# Final summary
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                      CHECKLIST SUMMARY                           ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo -e "   ${GREEN}✅ Passed:  $CHECKS_PASSED${NC}"
echo -e "   ${RED}❌ Failed:  $CHECKS_FAILED${NC}"
echo ""

if [ "$CHECKS_FAILED" -eq "0" ]; then
    echo -e "${GREEN}🎉 ALL CHECKS PASSED - READY FOR DEMO!${NC}"
    echo ""
    echo "📋 Last-minute tips:"
    echo "   1. Open browser tabs in advance (localhost:5173/onboarding)"
    echo "   2. Have DEMO_SCRIPT.md open in editor"
    echo "   3. Close unnecessary applications (free up RAM)"
    echo "   4. Test microphone and screen sharing"
    echo "   5. Take a deep breath - you got this! 💪"
else
    echo -e "${RED}❌ SOME CHECKS FAILED - FIX BEFORE DEMO${NC}"
    echo ""
    echo "Review the failures above and fix them"
fi
echo ""
