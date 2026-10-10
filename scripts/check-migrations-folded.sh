#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Migration-coverage check (gap #2 guard).
#
# schema.sql is the single source the dev DB and fresh installs are built from.
# If someone adds a numbered migration under supabase/migrations/ that CREATEs a
# table but forgets to fold that table into schema.sql, the live DB never gets it
# and the feature 500s — exactly the organizations/teams/sync_policies drift that
# broke the module pages.
#
# This is a STATIC check (no database): it extracts every `CREATE TABLE <name>`
# from the numbered migrations and asserts each table also appears in schema.sql.
# It is intentionally table-level and advisory — run it to see coverage gaps.
#
# Usage:  ./scripts/check-migrations-folded.sh          # report only (rc=0)
#         ./scripts/check-migrations-folded.sh --strict  # rc=1 if any gap
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="$REPO/desktop/backend-go/internal/database/schema.sql"
MIGRATIONS="$REPO/desktop/backend-go/supabase/migrations"
STRICT="${1:-}"

[ -f "$SCHEMA" ]     || { echo "✗ schema not found: $SCHEMA"; exit 2; }
[ -d "$MIGRATIONS" ] || { echo "✗ migrations dir not found: $MIGRATIONS"; exit 2; }

# Extract normalized (lowercase, unquoted) table names from one or more SQL files.
extract_tables() {
  grep -ihoE 'CREATE TABLE (IF NOT EXISTS )?"?[a-zA-Z_][a-zA-Z0-9_]*"?' "$@" 2>/dev/null \
    | sed -E 's/CREATE TABLE (IF NOT EXISTS )?//I; s/"//g' \
    | tr 'A-Z' 'a-z' | sort -u
}

# schema.sql's tables (the source of truth).
schema_tables="$(extract_tables "$SCHEMA")"

# Tables created by any numbered migration (skip rollback_* — they DROP).
mig_files="$(find "$MIGRATIONS" -name '*.sql' ! -name 'rollback_*' 2>/dev/null)"
mig_tables="$(extract_tables $mig_files)"

# Anything a migration creates that schema.sql lacks = candidate drift, minus
# tables that legitimately live outside schema.sql:
#   *_backup_*        one-off backup snapshots
#   *_yNNNNmNN        time partitions (created dynamically)
#   account/session/verification  better-auth tables (managed by the auth layer,
#                                 not schema.sql)
IGNORE='(_backup_|_y[0-9]{4}m[0-9]{2}$|^account$|^session$|^verification$)'
missing="$(comm -23 <(echo "$mig_tables") <(echo "$schema_tables") | sed '/^$/d' | grep -vE "$IGNORE" || true)"

if [ -z "$missing" ]; then
  echo "✅ migration coverage OK — every table created by a numbered migration exists in schema.sql"
  exit 0
fi

echo "⚠ tables created by a numbered migration but MISSING from schema.sql:"
echo "$missing" | sed 's/^/    - /'
echo ""
echo "  Fold these into desktop/backend-go/internal/database/schema.sql (idempotent,"
echo "  in dependency order) so the dev DB and fresh installs actually get them."

if [ "$STRICT" = "--strict" ]; then exit 1; fi
exit 0
