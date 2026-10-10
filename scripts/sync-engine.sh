#!/usr/bin/env bash
# Sync the bundled OptimalEngine to the latest canonical public source.
#
# BusinessOS ships a VENDORED copy of the engine at optimal-engine/ (not a git
# submodule). The canonical source is the OptimalOS engine checkout
# (github.com/Miosa-osa/OptimalEngine), checked out by default next to this repo
# at ../OptimalOS/engine.
#
# Canonical config/runtime.exs is env-driven and contains no personal paths.
# Private checkouts provide their paths through environment variables. Keeping
# runtime.exs in the normal sync is required so packaged releases receive storage,
# model, graph, and security fixes.
#
# Usage: scripts/sync-engine.sh [CANONICAL_ENGINE_DIR]
#   CANONICAL_ENGINE_DIR defaults to $OPTIMAL_ENGINE_CANONICAL_DIR or
#   ../OptimalOS/engine
#
# Idempotent and safe: it rsyncs source, restores the preserved runtime.exs,
# fetches deps, and builds the prod release. It does NOT start the engine (no
# `:4200` squatting on Roberto's real engine).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VENDORED_DIR="$REPO_DIR/optimal-engine"

CANONICAL_DIR="${1:-${OPTIMAL_ENGINE_CANONICAL_DIR:-$REPO_DIR/../OptimalOS/engine}}"
# Normalize to an absolute path with a trailing slash for rsync source semantics.
CANONICAL_DIR="$(cd "$CANONICAL_DIR" && pwd)"

if [ ! -f "$CANONICAL_DIR/mix.exs" ]; then
  echo "ERROR: canonical engine not found at $CANONICAL_DIR (no mix.exs)"
  echo "Pass the canonical engine dir as arg1, e.g.:"
  echo "  scripts/sync-engine.sh ../OptimalOS/engine"
  exit 1
fi

if [ ! -d "$VENDORED_DIR" ]; then
  echo "ERROR: vendored engine dir not found at $VENDORED_DIR"
  exit 1
fi

echo "==> Syncing engine"
echo "    canonical: $CANONICAL_DIR"
echo "    vendored:  $VENDORED_DIR"

# 1. Rsync canonical source into the vendored copy. --delete keeps the vendored
#    tree an exact mirror of canonical (minus the excludes). Excludes drop build
#    artifacts, deps, VCS, per-user data, and databases so nothing developer- or
#    machine-specific rides along.
echo "==> rsync canonical -> vendored"
rsync -a --delete \
  --exclude='_build' \
  --exclude='deps' \
  --exclude='.git' \
  --exclude='node_modules' \
  --exclude='*.db' \
  --exclude='.optimal' \
  --exclude='tmp/' \
  "$CANONICAL_DIR"/ "$VENDORED_DIR"/

# 2. Fetch deps + build the prod release for staging. Does NOT start the engine.
echo "==> mix deps.get + release (MIX_ENV=prod)"
(
  cd "$VENDORED_DIR"
  MIX_ENV=prod mix deps.get
  rm -rf _build/prod/rel/optimal
  MIX_ENV=prod mix release optimal --overwrite
)

# 3. Report the resulting version.
VERSION="$(grep -m1 -E '@version\s+"' "$VENDORED_DIR/mix.exs" | sed -E 's/.*"([^"]+)".*/\1/')"
echo "==> Engine synced. Bundled version: ${VERSION:-unknown}"
echo "    Next: desktop/scripts/stage-engine.sh to stage this release, then rebuild the DMG."
