#!/bin/bash

# Build the SvelteKit frontend for Electron (static build)
# This script builds the frontend and copies it to the renderer directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(dirname "$SCRIPT_DIR")"
FRONTEND_DIR="$(dirname "$DESKTOP_DIR")/frontend"
RENDERER_DIR="$DESKTOP_DIR/src/renderer"

echo "Building SvelteKit frontend for Electron..."
echo "Frontend directory: $FRONTEND_DIR"
echo "Renderer directory: $RENDERER_DIR"

# Navigate to frontend directory
cd "$FRONTEND_DIR"

# Set environment for static build
export ELECTRON_BUILD=true

# Install dependencies if needed (pnpm — the frontend uses link: workspace deps
# which npm does not support)
if [ ! -d "node_modules" ]; then
  echo "Installing frontend dependencies..."
  pnpm install
fi

# Build the static version
echo "Building static SvelteKit app..."
pnpm run build

# Clear existing renderer content (except index.html placeholder)
echo "Preparing renderer directory..."
rm -rf "$RENDERER_DIR"/*

# Copy the build output
echo "Copying build to renderer..."
cp -r build/* "$RENDERER_DIR/"

# Fix absolute paths for Electron file:// protocol
echo "Fixing paths for Electron..."
# sed -i syntax differs between GNU (Linux) and BSD (macOS)
if sed --version >/dev/null 2>&1; then SEDI=(sed -i); else SEDI=(sed -i ''); fi
# Replace /_app/ with ./_app/ in index.html
"${SEDI[@]}" 's|"/_app/|"./_app/|g' "$RENDERER_DIR/index.html"
"${SEDI[@]}" "s|'/_app/|'./_app/|g" "$RENDERER_DIR/index.html"
# Also fix any absolute paths to assets
"${SEDI[@]}" 's|href="/_|href="./_|g' "$RENDERER_DIR/index.html"
"${SEDI[@]}" 's|src="/_|src="./_|g' "$RENDERER_DIR/index.html"

echo "Frontend build complete!"
echo "Output: $RENDERER_DIR"
