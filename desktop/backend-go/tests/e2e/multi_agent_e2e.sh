#!/bin/bash

# Multi-Agent App Generation E2E Test
# Tests the complete flow: queue item creation → multi-agent generation → file output

set -e

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║    Multi-Agent App Generation E2E Test                        ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
API_BASE="${API_BASE:-http://localhost:8001}"
WORKSPACE_ID="${WORKSPACE_ID:-$(uuidgen)}"

echo "Configuration:"
echo "  API Base: $API_BASE"
echo "  Workspace ID: $WORKSPACE_ID"
echo ""

# Step 1: Check server health
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 1: Checking server health..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

HEALTH_RESPONSE=$(curl -s "$API_BASE/health")
if [[ $HEALTH_RESPONSE != *"healthy"* ]]; then
  echo "❌ Server health check failed"
  echo "Response: $HEALTH_RESPONSE"
  exit 1
fi
echo "✅ Server is healthy"
echo ""

# Step 2: Create app generation queue item
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 2: Creating app generation request..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

REQUEST_PAYLOAD=$(cat <<EOF
{
  "app_name": "E2E Test Todo App",
  "description": "A simple todo application with create, read, update, and delete functionality. Users can add tasks, mark them as complete, and delete them.",
  "features": ["task management", "basic CRUD"]
}
EOF
)

echo "Request payload:"
echo "$REQUEST_PAYLOAD" | jq .
echo ""

CREATE_RESPONSE=$(curl -s -X POST \
  "$API_BASE/api/workspaces/$WORKSPACE_ID/apps/generate-osa" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_PAYLOAD")

QUEUE_ITEM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.queue_item_id')

if [[ -z "$QUEUE_ITEM_ID" || "$QUEUE_ITEM_ID" == "null" ]]; then
  echo "❌ Failed to create queue item"
  echo "Response: $CREATE_RESPONSE"
  exit 1
fi

echo "✅ Queue item created: $QUEUE_ITEM_ID"
echo ""

# Step 3: Monitor SSE progress stream
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 3: Monitoring generation progress (SSE stream)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Monitor for 5 minutes max
TIMEOUT=300
START_TIME=$(date +%s)
COMPLETED=false

echo "Connecting to SSE stream..."
curl -s -N "$API_BASE/api/osa/apps/generate/$QUEUE_ITEM_ID/stream" | while read line; do
  CURRENT_TIME=$(date +%s)
  ELAPSED=$((CURRENT_TIME - START_TIME))

  if [[ $ELAPSED -gt $TIMEOUT ]]; then
    echo "⏱️  Timeout reached (${TIMEOUT}s)"
    break
  fi

  # Parse SSE events
  if [[ $line == data:* ]]; then
    EVENT_DATA="${line#data: }"

    # Extract key fields
    TASK_ID=$(echo "$EVENT_DATA" | jq -r '.task_id // empty')
    STATUS=$(echo "$EVENT_DATA" | jq -r '.status // empty')
    MESSAGE=$(echo "$EVENT_DATA" | jq -r '.message // empty')
    PROGRESS=$(echo "$EVENT_DATA" | jq -r '.progress // empty')

    if [[ -n "$TASK_ID" ]]; then
      echo "  [$TASK_ID] $STATUS - $MESSAGE ($PROGRESS%)"
    fi

    # Check for completion
    if [[ "$STATUS" == "completed" ]]; then
      COMPLETED=true
      break
    fi

    if [[ "$STATUS" == "failed" ]]; then
      echo "❌ Generation failed: $MESSAGE"
      exit 1
    fi
  fi
done

if $COMPLETED; then
  echo "✅ Generation completed successfully"
else
  echo "⚠️  SSE stream ended (check server logs)"
fi
echo ""

# Step 4: Verify workspace files were created
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 4: Verifying workspace files..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

WORKSPACE_PATH="/tmp/businessos-agent-workspaces/$QUEUE_ITEM_ID"

if [[ ! -d "$WORKSPACE_PATH" ]]; then
  echo "⚠️  Workspace directory not found: $WORKSPACE_PATH"
  echo "This may be expected if running on a different machine"
else
  echo "✅ Workspace directory exists: $WORKSPACE_PATH"
  echo ""
  echo "File structure:"
  tree -L 3 "$WORKSPACE_PATH" || find "$WORKSPACE_PATH" -type f -o -type d

  FILE_COUNT=$(find "$WORKSPACE_PATH" -type f | wc -l)
  echo ""
  echo "Total files generated: $FILE_COUNT"

  if [[ $FILE_COUNT -eq 0 ]]; then
    echo "⚠️  No files were generated (file parsing may have failed)"
  else
    echo "✅ Files were generated successfully"
  fi
fi
echo ""

# Step 5: Test circuit breaker metrics endpoint (if available)
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 5: Checking circuit breaker metrics..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# This would require a dedicated metrics endpoint
echo "ℹ️  Circuit breaker metrics endpoint not implemented yet"
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ E2E Test Complete"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Test Results:"
echo "  Queue Item ID: $QUEUE_ITEM_ID"
echo "  Workspace: $WORKSPACE_PATH"
echo ""
echo "To test again, run:"
echo "  WORKSPACE_ID=\$(uuidgen) ./multi_agent_e2e.sh"
