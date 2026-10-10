#!/bin/bash
# =============================================================================
# BusinessOS — Production Deploy Script
# =============================================================================
# Deploys:
#   Backend  → GCP Cloud Run (api.businessos.dev)
#   Frontend → Cloudflare Pages (app.businessos.dev)
#
# Prerequisites:
#   - gcloud CLI authenticated: gcloud auth login
#   - GCP project set: gcloud config set project YOUR_PROJECT_ID
#   - wrangler authenticated: npx wrangler login
#   - Secrets already loaded in GCP Secret Manager (run once: ./desktop/backend-go/deploy.sh secrets)
#
# Usage:
#   ./scripts/deploy-production.sh              # deploy both
#   ./scripts/deploy-production.sh backend      # backend only
#   ./scripts/deploy-production.sh frontend     # frontend only

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/desktop/backend-go"
FRONTEND_DIR="$REPO_ROOT/frontend"

GCP_REGION="${GCP_REGION:-us-central1}"
CF_PROJECT_NAME="${CF_PROJECT_NAME:-businessos}"

TARGET="${1:-all}"

log()  { echo "[$(date '+%H:%M:%S')] $*"; }
fail() { echo "[ERROR] $*" >&2; exit 1; }

# =============================================================================
# Backend: GCP Cloud Run via Cloud Build
# =============================================================================
deploy_backend() {
  log "=== Deploying Go backend to Cloud Run ==="

  if ! command -v gcloud &>/dev/null; then
    fail "gcloud CLI not found. Install: https://cloud.google.com/sdk/docs/install"
  fi

  PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
  if [ -z "$PROJECT_ID" ] || [ "$PROJECT_ID" = "(unset)" ]; then
    fail "GCP project not set. Run: gcloud config set project YOUR_PROJECT_ID"
  fi

  log "GCP project: $PROJECT_ID | Region: $GCP_REGION"
  log "Submitting Cloud Build job..."

  # Cloud Build runs from the repo root; cloudbuild.yaml specifies dir: backend-go
  cd "$REPO_ROOT"
  gcloud builds submit \
    --config="$BACKEND_DIR/cloudbuild.yaml" \
    --substitutions="_REGION=$GCP_REGION" \
    .

  SERVICE_URL=$(gcloud run services describe businessos-api \
    --platform=managed \
    --region="$GCP_REGION" \
    --format='value(status.url)' 2>/dev/null || echo "unknown")

  log "Backend deployed: $SERVICE_URL"
  log "Custom domain: https://api.businessos.dev (configure in Cloud Run console)"
}

# =============================================================================
# Frontend: Cloudflare Pages via wrangler
# =============================================================================
deploy_frontend() {
  log "=== Deploying SvelteKit frontend to Cloudflare Pages ==="

  if ! command -v node &>/dev/null; then
    fail "Node.js not found."
  fi

  cd "$FRONTEND_DIR"

  log "Installing dependencies..."
  npm ci

  log "Building for Cloudflare Pages (static SPA)..."
  CLOUDFLARE_BUILD=true \
  VITE_API_URL="https://api.businessos.dev/api/v1" \
  VITE_BACKEND_URL="https://api.businessos.dev" \
  npm run build

  log "Deploying to Cloudflare Pages (project: $CF_PROJECT_NAME)..."
  npx wrangler pages deploy build \
    --project-name="$CF_PROJECT_NAME" \
    --branch=main

  log "Frontend deployed: https://app.businessos.dev"
}

# =============================================================================
# Dispatch
# =============================================================================
case "$TARGET" in
  backend)
    deploy_backend
    ;;
  frontend)
    deploy_frontend
    ;;
  all)
    deploy_backend
    deploy_frontend
    log "=== All deployments complete ==="
    log "  API:     https://api.businessos.dev"
    log "  App:     https://app.businessos.dev"
    ;;
  *)
    echo "Usage: $0 [all|backend|frontend]"
    exit 1
    ;;
esac
