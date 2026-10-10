#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Schema drift guard.
#
# Builds desktop/backend-go/internal/database/schema.sql on a THROWAWAY database
# and asserts:
#   1. it applies cleanly from empty (ON_ERROR_STOP=1, rc=0) — catches missing
#      tables/columns, statement-ordering bugs, and migrations that were never
#      folded in (all three drift classes that broke the module pages);
#   2. it is IDEMPOTENT — applying a second time to the populated DB is also a
#      clean no-op, so run-local.sh can safely re-apply it on every start.
#
# Run it before committing a schema change, and in CI. Exit non-zero = drift.
#
# Usage:  ./scripts/check-schema.sh
# Env:    PGPORT (default 25432), PGHOST (default localhost), PGUSER (default postgres)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="$REPO/desktop/backend-go/internal/database/schema.sql"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-25432}"
PGUSER="${PGUSER:-postgres}"
CHECKDB="schema_drift_check_$$"

# Prefer the local no-Docker toolchain's psql; fall back to psql on PATH.
TC="$HOME/.local-toolchain"
PGBIN="$(cat "$TC/.pgbin" 2>/dev/null || true)"
if [ -n "$PGBIN" ] && [ -x "$PGBIN/psql" ]; then
  if [ "$(uname -s)" = "Darwin" ]; then
    DYLD_LIBRARY_PATH="$(dirname "$PGBIN")/lib:${DYLD_LIBRARY_PATH:-}"
    export DYLD_LIBRARY_PATH
  else
    LD_LIBRARY_PATH="$(dirname "$PGBIN")/lib:${LD_LIBRARY_PATH:-}"
    export LD_LIBRARY_PATH
  fi
  PSQL="$PGBIN/psql"; CREATEDB="$PGBIN/createdb"; DROPDB="$PGBIN/dropdb"
else
  PSQL="psql"; CREATEDB="createdb"; DROPDB="dropdb"
fi
CONN=(-h "$PGHOST" -p "$PGPORT" -U "$PGUSER")

[ -f "$SCHEMA" ] || { echo "✗ schema not found: $SCHEMA"; exit 2; }

cleanup() { "$DROPDB" "${CONN[@]}" --if-exists "$CHECKDB" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "→ building schema on throwaway DB '$CHECKDB' ($PGHOST:$PGPORT)"
"$CREATEDB" "${CONN[@]}" "$CHECKDB"

run() { "$PSQL" "${CONN[@]}" -d "$CHECKDB" -q -v ON_ERROR_STOP=1 -f "$SCHEMA" 2>&1; }

if ! out1="$(run)"; then
  echo "✗ FRESH build failed — schema.sql does not build from empty:"
  echo "$out1" | grep -iE 'ERROR|FATAL' | head
  exit 1
fi
echo "✔ fresh build clean"

if ! out2="$(run)"; then
  echo "✗ RE-APPLY failed — schema.sql is not idempotent (cannot self-heal a live DB):"
  echo "$out2" | grep -iE 'ERROR|FATAL' | head
  exit 1
fi
echo "✔ idempotent (second apply clean)"

echo "✅ schema OK — builds clean and re-applies safely"
