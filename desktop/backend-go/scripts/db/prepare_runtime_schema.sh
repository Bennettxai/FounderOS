#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

echo "Applying canonical schema..."
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$ROOT/internal/database/schema.sql"

echo "Applying active module migrations..."
find "$ROOT/internal/database/migrations" -maxdepth 1 -type f -name '*.sql' | sort | while read -r migration; do
  base="$(basename "$migration")"
  number="${base%%_*}"
  if [[ "$number" =~ ^[0-9]+$ ]] && [ "$number" -ge 110 ]; then
    echo "Applying $base..."
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
  fi
done

echo "Verifying runtime schema contract..."
(cd "$ROOT" && go run ./scripts/db/verify_schema_contract.go)
