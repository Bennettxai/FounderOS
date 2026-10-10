#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE_SLUG="${WORKSPACE_SLUG:-agency-miosa}"
WORKSPACES_ROOT="${KNOWLEDGE_WORKSPACES_ROOT:-$ROOT/workspaces}"
WORKSPACE_PATH="${AGENCY_MIOSA_WORKSPACE_PATH:-$WORKSPACES_ROOT/$WORKSPACE_SLUG}"
ENGINE_DIR="${OPTIMAL_ENGINE_DIR:-$ROOT/optimal-engine}"
RESET_ENGINE_WORKSPACE="${RESET_ENGINE_WORKSPACE:-0}"
DRY_RUN="${DRY_RUN:-0}"

if [ ! -d "$WORKSPACE_PATH" ]; then
  echo "Missing workspace path: $WORKSPACE_PATH" >&2
  exit 1
fi

if [ ! -d "$ENGINE_DIR" ]; then
  echo "Missing Optimal Engine directory: $ENGINE_DIR" >&2
  exit 1
fi

export KNOWLEDGE_WORKSPACES_ROOT="$WORKSPACES_ROOT"
export OPTIMAL_ENGINE_ROOT="${OPTIMAL_ENGINE_ROOT:-$WORKSPACE_PATH}"
export OPTIMAL_ENGINE_DB="${OPTIMAL_ENGINE_DB:-$WORKSPACE_PATH/.optimal/index.db}"
export OPTIMAL_ENGINE_CACHE="${OPTIMAL_ENGINE_CACHE:-$ROOT/.run/optimal-engine-cache}"

mkdir -p "$(dirname "$OPTIMAL_ENGINE_DB")" "$OPTIMAL_ENGINE_CACHE"

echo "Agency MIOSA bootstrap"
echo "  repo:                 $ROOT"
echo "  workspace slug:       $WORKSPACE_SLUG"
echo "  workspaces root:      $KNOWLEDGE_WORKSPACES_ROOT"
echo "  workspace path:       $WORKSPACE_PATH"
echo "  optimal engine dir:   $ENGINE_DIR"
echo "  optimal engine db:    $OPTIMAL_ENGINE_DB"
echo

mapfile -t DOC_FILES < <(
  find "$WORKSPACE_PATH/operating-system" "$ROOT/packages" \
    -type f \( -name '*.md' -o -name '*.txt' \) 2>/dev/null | sort
)

echo "Files ready for import:"
echo "  operating/package docs: ${#DOC_FILES[@]}"
echo

if [ "$DRY_RUN" = "1" ]; then
  printf '%s\n' "${DOC_FILES[@]}"
  exit 0
fi

if ! command -v mix >/dev/null 2>&1; then
  echo "Elixir mix is required to ingest into Optimal Engine." >&2
  exit 1
fi

cd "$ENGINE_DIR"

if [ ! -d deps ]; then
  MIX_ENV=prod mix deps.get
fi

INGEST_WORKSPACE_ARGS=("$WORKSPACE_PATH" "--workspace" "$WORKSPACE_SLUG")
if [ "$RESET_ENGINE_WORKSPACE" = "1" ]; then
  INGEST_WORKSPACE_ARGS+=("--reset")
fi

echo "Ingesting workspace topology, nodes, signals, and wiki pages..."
MIX_ENV=prod OPTIMAL_API_ENABLED=false mix optimal.ingest_workspace "${INGEST_WORKSPACE_ARGS[@]}"

echo "Ingesting operating-system and package docs as workspace resources..."
for file in "${DOC_FILES[@]}"; do
  rel="${file#$ROOT/}"
  echo "  + $rel"
  MIX_ENV=prod OPTIMAL_API_ENABLED=false mix optimal.ingest --file "$file" --workspace "$WORKSPACE_SLUG" --type resource --title "$rel" >/dev/null
done

echo
echo "Agency MIOSA workspace import complete."
echo "BusinessOS should use KNOWLEDGE_WORKSPACES_ROOT=$KNOWLEDGE_WORKSPACES_ROOT"
echo "Optimal Engine should use OPTIMAL_ENGINE_DB=$OPTIMAL_ENGINE_DB"
