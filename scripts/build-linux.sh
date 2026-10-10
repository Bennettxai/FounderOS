#!/usr/bin/env bash
# Build the BusinessOS Linux desktop app ON an Ubuntu/Debian machine.
# Run this on your Ubuntu box (NOT on the Mac - a Linux app + its bundled engine
# must be built on Linux). It produces .deb / .rpm / AppImage installers in
# desktop/out/make/ and (optionally) uploads them to the download bucket.
#
# Usage:
#   ./scripts/build-linux.sh            # build only
#   ./scripts/build-linux.sh --upload   # build + upload (needs: gcloud already logged in)
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
UPLOAD="${1:-}"

echo "==> BusinessOS Linux build"
echo "==> repo: $REPO"

# 1. Toolchains (Ubuntu/Debian). Skips anything already installed.
echo "==> Installing build toolchains (erlang, elixir, node, rpm, fakeroot)..."
sudo apt-get update -y
sudo apt-get install -y erlang elixir nodejs npm rpm fakeroot dpkg rsync curl || {
  echo "apt install failed - if elixir/erlang are too old, use asdf or the Erlang Solutions repo."; exit 1;
}

# 2. Build + stage the native Linux OptimalEngine.
echo "==> Building the OptimalEngine (Linux release)..."
cd "$REPO/optimal-engine"
MIX_ENV=prod mix local.hex --force >/dev/null 2>&1 || true
MIX_ENV=prod mix local.rebar --force >/dev/null 2>&1 || true
MIX_ENV=prod mix deps.get
MIX_ENV=prod mix release optimal --overwrite

DEST="$REPO/desktop/resources/engine/linux-x64"
rm -rf "$DEST"; mkdir -p "$DEST"
rsync -a --exclude='tmp/' --exclude='log/' --exclude='*.pipe.*' \
  "$REPO/optimal-engine/_build/prod/rel/optimal/" "$DEST/"
echo "==> Engine staged to resources/engine/linux-x64"

# 3. Build the app + make the Linux installers.
echo "==> Building the desktop app + installers..."
cd "$REPO/desktop"
npm ci
npm run build:all
npx electron-forge make --arch=x64

echo "==> Done. Installers are in: $REPO/desktop/out/make/"
find "$REPO/desktop/out/make" -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.AppImage" \) -print

# 4. Optional upload to the download bucket (you must already be `gcloud auth login`'d).
if [ "$UPLOAD" = "--upload" ]; then
  echo "==> Uploading installers to gs://businessos-downloads ..."
  find "$REPO/desktop/out/make" -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.AppImage" \) -print0 \
    | xargs -0 -I{} gsutil cp "{}" gs://businessos-downloads/
  echo "==> Uploaded. They are now downloadable."
fi
