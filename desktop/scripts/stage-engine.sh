#!/usr/bin/env bash
# Stage the bundled OptimalEngine release into resources/engine/<platform>-<arch>/
# so electron-forge picks it up via extraResource. Every BusinessOS ships with a
# built-in engine; this copies the prod release next to the app.
#
# Usage: scripts/stage-engine.sh [arm64|x64]   (defaults to the host arch)
# The Elixir release with ERTS is architecture-locked, so build it on (or for)
# the target arch before staging. This script assumes the release already exists
# at ../optimal-engine/_build/prod/rel/optimal (run `MIX_ENV=prod mix release
# optimal --overwrite` in optimal-engine first).
set -euo pipefail

DESKTOP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_DIR="$(cd "$DESKTOP_DIR/.." && pwd)"
ENGINE_REL="$REPO_DIR/optimal-engine/_build/prod/rel/optimal"

ARCH="${1:-}"
if [ -z "$ARCH" ]; then
  case "$(uname -m)" in
    arm64) ARCH="arm64" ;;
    x86_64) ARCH="x64" ;;
    *) echo "unknown arch $(uname -m)"; exit 1 ;;
  esac
fi

PLATFORM="darwin"
DEST="$DESKTOP_DIR/resources/engine/${PLATFORM}-${ARCH}"

if [ ! -d "$ENGINE_REL" ]; then
  echo "ERROR: engine release not found at $ENGINE_REL"
  echo "Build it first: (cd $REPO_DIR/optimal-engine && MIX_ENV=prod mix release optimal --overwrite)"
  exit 1
fi

echo "Staging engine release -> $DEST"
rm -rf "$DEST"
mkdir -p "$DEST"
# Copy the full release (bin/, lib/, releases/, erts-*) but EXCLUDE runtime
# cruft: tmp/ can hold Erlang FIFO pipes + logs from a prior run, and forge
# cannot copy a FIFO ("Cannot copy a FIFO pipe"). rsync skips them cleanly.
rsync -a \
  --exclude='tmp/' --exclude='log/' --exclude='*.pipe.*' \
  --exclude='*.dSYM/' \
  --exclude='erl_crash.dump' --exclude='.optimal/' \
  "$ENGINE_REL"/ "$DEST"/
# Belt-and-suspenders: drop any FIFO/socket that slipped through.
find "$DEST" \( -type p -o -type s \) -delete 2>/dev/null || true
echo "Staged $(du -sh "$DEST" | awk '{print $1}') to resources/engine/${PLATFORM}-${ARCH}"
