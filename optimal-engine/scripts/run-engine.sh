#!/usr/bin/env bash
# Always-on launcher for the Optimal Engine (launchd KeepAlive).
set -euo pipefail

export PATH="/opt/homebrew/opt/erlang/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
export OPTIMAL_API_ENABLED=true
export MIX_ENV=dev
export OPTIMAL_KNOWLEDGE_BACKEND="${OPTIMAL_KNOWLEDGE_BACKEND:-rocksdb}"
ENGINE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPTIMALOS_DIR="$(cd "$ENGINE_DIR/.." && pwd)"

export OPTIMAL_ENGINE_ROOT="${OPTIMAL_ENGINE_ROOT:-$OPTIMALOS_DIR/workspaces}"
export OPTIMAL_ENGINE_DB="${OPTIMAL_ENGINE_DB:-$ENGINE_DIR/.optimal/index.db}"
export OPTIMAL_ENGINE_CACHE="${OPTIMAL_ENGINE_CACHE:-$ENGINE_DIR/.optimal/cache}"

cd "$ENGINE_DIR"

mkdir -p .optimal

if [ -z "${CONNECTOR_KEY:-}" ]; then
  key_file=".optimal/connector_key"

  if [ ! -f "$key_file" ]; then
    umask 077
    openssl rand -hex 32 > "$key_file"
  fi

  chmod 600 "$key_file"
  CONNECTOR_KEY="$(tr -d '[:space:]' < "$key_file")"
  export CONNECTOR_KEY
fi

exec mix run --no-halt
