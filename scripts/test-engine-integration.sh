#!/usr/bin/env bash
# Integration smoke test for the BusinessOS ↔ OptimalEngine data flow.
#
# Exercises the chain that powers the Pages → Knowledge Graph view:
#   1. Elixir engine listens on :4200
#   2. BusinessOS proxy on :8801 forwards /api/optimal/* to :4200
#   3. Both processes share the same SQLite (.run/optimal-engine-data/.optimal/index.db)
#   4. Writing a markdown file under nodes/ + reindex makes it visible
#      to both
#
# Catches future drift from env-var mismatch, db path changes, or proxy
# misconfiguration.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${ENGINE_URL:-http://localhost:4200}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8801}"
TEST_ROOT="$ROOT/.run/optimal-engine-data"
TEST_NODE_DIR="$TEST_ROOT/nodes/19-integration-test"
EXPECTED_DB="$TEST_ROOT/.optimal/index.db"

color() { printf '\033[%sm%s\033[0m' "$1" "$2"; }
pass()  { echo "  $(color '32' '✔') $*"; }
fail()  { echo "  $(color '31' '✗') $*" >&2; FAILED=1; }
info()  { echo "$(color '36' '==>') $*"; }

FAILED=0
# We deliberately do NOT remove the test node on exit. The engine's
# indexer is async — it returns 200 from /api/reindex immediately and
# walks the filesystem in the background. Deleting the test files
# before the indexer finishes would race the indexer and produce false
# negatives. Operators can clean up with:
#   rm -rf ".run/optimal-engine-data/nodes/19-integration-test"
# (or run the test again — it overwrites in place).

# ----- 1: engine reachability -----------------------------------------------
info "1. Elixir engine reachable at $ENGINE_URL"
if curl -sf -m 5 "$ENGINE_URL/api/health" >/dev/null; then
    pass "engine /api/health responds"
else
    fail "engine unreachable — start with: make optimal-engine"
    exit 1
fi

# ----- 2: db alignment -------------------------------------------------------
info "2. Engine + BusinessOS share the same SQLite file"
ENGINE_PID=$(lsof -ti tcp:4200 | head -1)
if [ -n "$ENGINE_PID" ]; then
    ACTUAL_DB=$(lsof -p "$ENGINE_PID" 2>/dev/null | awk '/\.db$/ {print $NF; exit}')
    if [ "$ACTUAL_DB" = "$EXPECTED_DB" ]; then
        pass "engine reads $ACTUAL_DB"
    else
        fail "engine reads $ACTUAL_DB but expected $EXPECTED_DB"
        fail "  → restart engine with: make optimal-engine-stop && make optimal-engine"
    fi
fi

# ----- 3: Pages → engine flow ------------------------------------------------
info "3. Page-style markdown drop becomes a visible node"
mkdir -p "$TEST_NODE_DIR"
TEST_ID="integration-test-$(date +%s)"
TEST_FILE="$TEST_NODE_DIR/$TEST_ID.md"
cat > "$TEST_NODE_DIR/context.md" <<'EOF'
---
node: 19-integration-test
kind: workspace
style: internal
source: businessos
---

# Integration Test
EOF
cat > "$TEST_FILE" <<EOF
---
node: 19-integration-test
title: Integration test signal
genre: note
mode: linguistic
id: $TEST_ID
source: businessos
authored_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
---

# Integration test signal
Written by scripts/test-engine-integration.sh to verify the BO→engine flow.
EOF
pass "wrote $TEST_FILE"

# Trigger the Elixir engine's own reindex via its HTTP endpoint. The
# engine knows its own contexts_fts schema (which differs from the Go
# port's), so letting the engine own its writes avoids the "no such
# column" errors a Go-side reindex would produce against an Elixir-
# created db. 202 (already_running) is fine — the next pass picks up
# the new files.
RC=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H 'Content-Type: application/json' \
    --max-time 30 -d '{}' "$ENGINE_URL/api/reindex")
if [ "$RC" = "200" ] || [ "$RC" = "202" ]; then
    pass "engine accepted reindex (HTTP $RC)"
else
    fail "engine reindex returned HTTP $RC"
fi

# The indexer is async. Poll the nodes endpoint until 19-integration-test
# appears or we time out. Walking the seed corpus + embedding takes
# ~30-60s on first reindex; subsequent passes are quick.
echo -n "  waiting for indexer to commit"
for i in $(seq 1 90); do
    if curl -sf -m 3 "$ENGINE_URL/api/optimal/nodes" 2>/dev/null \
        | grep -q '"19-integration-test"'; then
        echo " (took ${i}s)"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# ----- 4: engine sees the node directly -------------------------------------
info "4. Engine /api/optimal/nodes lists the new node"
NODES_JSON=$(curl -sf -m 5 "$ENGINE_URL/api/optimal/nodes")
if echo "$NODES_JSON" | grep -q '"19-integration-test"'; then
    pass "engine sees 19-integration-test directly"
else
    fail "engine doesn't see 19-integration-test"
fi

# ----- 5: BO proxy returns identical data -----------------------------------
info "5. BusinessOS proxy forwards correctly"
if curl -sf -m 5 "$BACKEND_URL/health" >/dev/null 2>&1; then
    PROXY_JSON=$(curl -sf -m 5 "$BACKEND_URL/api/optimal/nodes")
    if echo "$PROXY_JSON" | grep -q '"19-integration-test"'; then
        pass "proxy returns 19-integration-test"
    else
        fail "proxy didn't return the new node — check OPTIMAL_ENGINE_URL in .env"
    fi
else
    info "  (BO server not running on $BACKEND_URL; skipping proxy check)"
fi

if [ "$FAILED" -eq 0 ]; then
    echo
    echo "$(color '32' '✔ all integration checks passed')"
    exit 0
else
    echo
    echo "$(color '31' '✗ integration checks failed')" >&2
    exit 1
fi
