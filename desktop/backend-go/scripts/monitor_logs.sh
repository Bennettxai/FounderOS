#!/bin/bash
# Monitor backend logs in real-time during E2E testing
# Usage: ./scripts/monitor_logs.sh [filter]
# Example: ./scripts/monitor_logs.sh onboarding
# Example: ./scripts/monitor_logs.sh groq

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get backend output file
OUTPUT_FILE="$TEMP/claude/C--Users-Pichau-Desktop-BusinessOS-main-dev/tasks/b1a7e05.output"

# Check if file exists
if [ ! -f "$OUTPUT_FILE" ]; then
    echo -e "${RED}❌ Backend output file not found${NC}"
    echo "Expected: $OUTPUT_FILE"
    echo ""
    echo "Make sure backend server is running:"
    echo "  cd desktop/backend-go && go run cmd/server/main.go"
    exit 1
fi

# Filter keyword (optional)
FILTER="$1"

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              BUSINESSOS BACKEND LOG MONITOR                      ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

if [ -z "$FILTER" ]; then
    echo "📊 Monitoring ALL logs (no filter)"
    echo "   Tip: Use './scripts/monitor_logs.sh onboarding' to filter"
else
    echo "🔍 Filtering logs for: $FILTER"
fi

echo ""
echo "Press Ctrl+C to stop monitoring"
echo "─────────────────────────────────────────────────────────────────"
echo ""

# Tail and colorize
if [ -z "$FILTER" ]; then
    tail -f "$OUTPUT_FILE" | while read line; do
        # Colorize based on log level
        if echo "$line" | grep -q "ERROR"; then
            echo -e "${RED}$line${NC}"
        elif echo "$line" | grep -q "WARN"; then
            echo -e "${YELLOW}$line${NC}"
        elif echo "$line" | grep -q "INFO.*onboarding\|INFO.*analysis\|INFO.*generation"; then
            echo -e "${GREEN}$line${NC}"
        elif echo "$line" | grep -q "DEBUG"; then
            echo -e "${BLUE}$line${NC}"
        else
            echo "$line"
        fi
    done
else
    tail -f "$OUTPUT_FILE" | grep -i "$FILTER" | while read line; do
        if echo "$line" | grep -q "ERROR"; then
            echo -e "${RED}$line${NC}"
        elif echo "$line" | grep -q "WARN"; then
            echo -e "${YELLOW}$line${NC}"
        elif echo "$line" | grep -q "INFO"; then
            echo -e "${GREEN}$line${NC}"
        else
            echo "$line"
        fi
    done
fi
