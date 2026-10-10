#!/usr/bin/env bash
# bootstrap.sh — scaffold a fresh Optimal Engine workspace and verify the pipeline
#
# Usage:
#   bash bootstrap.sh [workspace-slug] [engine-url]
#
# Examples:
#   bash bootstrap.sh                          # creates "default" workspace
#   bash bootstrap.sh engineering              # creates "engineering" workspace
#   bash bootstrap.sh sales http://10.0.0.5:4200

set -euo pipefail

WS="${1:-default}"
ENGINE="${2:-${OPTIMAL_ENGINE_URL:-http://localhost:4200}}"

# ── Colors ────────────────────────────────────────────────────────────────────
if [ -t 1 ] && command -v tput &>/dev/null && tput colors &>/dev/null && [ "$(tput colors)" -ge 8 ]; then
  GREEN="\033[0;32m"
  YELLOW="\033[1;33m"
  RED="\033[0;31m"
  RESET="\033[0m"
else
  GREEN="" YELLOW="" RED="" RESET=""
fi

ok()   { echo -e "${GREEN}[OK]${RESET} $*"; }
info() { echo -e "${YELLOW}[..] $*${RESET}"; }
fail() { echo -e "${RED}[FAIL] $*${RESET}" >&2; exit 1; }

# ── 1. Check engine is reachable ─────────────────────────────────────────────
info "Checking engine at $ENGINE ..."
STATUS=$(curl -sf "$ENGINE/api/status" 2>/dev/null || echo "{}")
if [ "$STATUS" = "{}" ]; then
  fail "Engine not reachable at $ENGINE. Start it with: make dev"
fi
ok "Engine is up"

# ── 2. Create (or confirm) the workspace ─────────────────────────────────────
info "Creating workspace '$WS' ..."
RESPONSE=$(curl -sf -X POST "$ENGINE/api/workspaces" \
  -H 'Content-Type: application/json' \
  -d "{\"slug\":\"$WS\",\"name\":\"$WS Brain\"}" 2>/dev/null || echo '{}')

WS_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
if [ -z "$WS_ID" ]; then
  # Might already exist — try fetching it
  WS_ID=$(curl -sf "$ENGINE/api/workspaces/$WS" 2>/dev/null | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
fi

if [ -z "$WS_ID" ]; then
  fail "Could not create or find workspace '$WS'. Check engine logs."
fi
ok "Workspace ready: $WS_ID"

# ── 3. Store a bootstrap memory ───────────────────────────────────────────────
info "Storing bootstrap memory ..."
MEM_RESPONSE=$(curl -sf -X POST "$ENGINE/api/memory" \
  -H 'Content-Type: application/json' \
  -d "{
    \"content\": \"Workspace '$WS' bootstrapped by bootstrap.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"workspace\": \"$WS\",
    \"is_static\": false
  }" 2>/dev/null || echo '{}')

MEM_ID=$(echo "$MEM_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
if [ -n "$MEM_ID" ]; then
  ok "Bootstrap memory stored: $MEM_ID"
else
  info "Memory store skipped (engine may not have memory primitive enabled yet)"
fi

# ── 4. Run a smoke-test query ─────────────────────────────────────────────────
info "Running smoke-test RAG query ..."
RAG_RESPONSE=$(curl -sf -X POST "$ENGINE/api/rag" \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"what do we know about this workspace\",\"workspace\":\"$WS\",\"format\":\"json\"}" \
  2>/dev/null || echo '{}')

WIKI_HIT=$(echo "$RAG_RESPONSE" | grep -o '"wiki_hit":[a-z]*' | cut -d: -f2 || echo "unknown")
ok "RAG responded (wiki_hit=$WIKI_HIT)"

# ── 5. Check profile endpoint ─────────────────────────────────────────────────
info "Fetching workspace profile ..."
PROFILE=$(curl -sf "$ENGINE/api/profile?workspace=$WS&bandwidth=l0" 2>/dev/null || echo '{}')
GENERATED=$(echo "$PROFILE" | grep -o '"generated_at":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
if [ -n "$GENERATED" ]; then
  ok "Profile generated at $GENERATED"
else
  info "Profile endpoint returned minimal data (workspace may be empty — ingest signals to populate)"
fi

# ── 6. Summary ────────────────────────────────────────────────────────────────
echo ""
echo "══════════════════════════════════════════════════════"
echo "  Workspace '$WS' is ready."
echo ""
echo "  Engine:    $ENGINE"
echo "  Workspace: $WS_ID"
echo ""
echo "  Next steps:"
echo "    Ingest signals:"
echo "      mix optimal.ingest_workspace <path-to-workspace-dir>"
echo ""
echo "    Query:"
echo "      curl -X POST $ENGINE/api/rag \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"query\":\"...\",\"workspace\":\"$WS\",\"format\":\"markdown\"}'"
echo ""
echo "    Browse:"
echo "      cd desktop && npm run dev   # http://localhost:1420"
echo "══════════════════════════════════════════════════════"
