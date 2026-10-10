#!/usr/bin/env bash
# Cloudflare Pages build script for BusinessOS.
# Set as build command: bash frontend/build-cloudflare.sh
# Set build output: frontend/build
# Set root directory: (empty)
set -euo pipefail

echo "==> Building @optimal-engine/client SDK"
cd optimal-engine/sdks/typescript
npm install --no-audit --no-fund
npm run build

echo "==> Building @miosa/optimal-engine SDK"
cd ../miosa
npm install --no-audit --no-fund
npm run build

echo "==> Installing frontend deps"
cd ../../../frontend
pnpm install --frozen-lockfile

echo "==> Building SvelteKit (Cloudflare static adapter)"
CLOUDFLARE_BUILD=true NODE_OPTIONS="--max-old-space-size=4096" pnpm run build

echo "==> Done - output in frontend/build/"
