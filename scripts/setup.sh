#!/usr/bin/env bash
# =============================================================================
# BusinessOS — First-time setup script
# Usage: bash scripts/setup.sh
# =============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

ok()   { printf "  ${GREEN}✓${RESET} %s\n" "$*"; }
warn() { printf "  ${YELLOW}!${RESET} %s\n" "$*"; }
err()  { printf "  ${RED}✗${RESET} %s\n" "$*" >&2; }
step() { printf "\n${BOLD}%s${RESET}\n" "$*"; }

# ── Prerequisite checks ───────────────────────────────────────────────────────
step "Checking prerequisites..."

if ! command -v docker &>/dev/null; then
  err "Docker is not installed. Install it from https://docs.docker.com/get-docker/"
  exit 1
fi
ok "Docker $(docker --version | awk '{print $3}' | tr -d ',')"

if ! docker info &>/dev/null 2>&1; then
  err "Docker daemon is not running. Start Docker and try again."
  exit 1
fi
ok "Docker daemon is running"

# Compose can be `docker compose` (plugin) or `docker-compose` (standalone)
if docker compose version &>/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose &>/dev/null; then
  COMPOSE="docker-compose"
else
  err "Docker Compose not found. Install the Compose plugin: https://docs.docker.com/compose/install/"
  exit 1
fi
ok "Compose: $($COMPOSE version --short 2>/dev/null || $COMPOSE version)"

# ── Environment files ─────────────────────────────────────────────────────────
step "Setting up environment files..."

cd "$ROOT"

if [[ ! -f .env ]]; then
  cp .env.example .env
  ok "Created .env from .env.example"

  # Auto-generate required secrets
  step "Generating security keys..."

  gen_key() {
    # Generate a URL-safe random key (no special chars that break sed/awk)
    if command -v openssl &>/dev/null; then
      openssl rand -hex "$1" 2>/dev/null
    else
      head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c $(( $1 * 2 ))
    fi
  }

  set_env_key() {
    local key="$1" value="$2" tmpfile
    # Use awk for exact key match (sed regex can match substrings)
    tmpfile="$(mktemp)"
    awk -v k="$key" -v v="$value" 'BEGIN{FS=OFS="="} $1==k {$0=k "=" v} 1' .env > "$tmpfile"
    mv "$tmpfile" .env
  }

  # Generate in reverse-specificity order to avoid substring collisions
  set_env_key "REDIS_KEY_HMAC_SECRET" "$(gen_key 32)"
  ok "Generated REDIS_KEY_HMAC_SECRET"

  # Backend requires base64 of exactly 32 raw bytes (AES-256), not hex —
  # see desktop/backend-go/internal/security/encryption.go
  set_env_key "TOKEN_ENCRYPTION_KEY" "$(openssl rand -base64 32)"
  ok "Generated TOKEN_ENCRYPTION_KEY"

  set_env_key "SECRET_KEY" "$(gen_key 64)"
  ok "Generated SECRET_KEY"

  set_env_key "REDIS_PASSWORD" "$(gen_key 32)"
  ok "Generated REDIS_PASSWORD"

  set_env_key "POSTGRES_PASSWORD" "$(gen_key 32)"
  ok "Generated POSTGRES_PASSWORD"
else
  ok ".env already exists"
fi

if [[ ! -f desktop/backend-go/.env ]]; then
  cp desktop/backend-go/.env.example desktop/backend-go/.env
  ok "Created desktop/backend-go/.env"
else
  ok "desktop/backend-go/.env already exists"
fi

# ── Validate critical keys ────────────────────────────────────────────────────
step "Validating required keys in .env..."

# shellcheck disable=SC1091
source .env 2>/dev/null || true

MISSING=()
[[ -z "${POSTGRES_PASSWORD:-}" ]] && MISSING+=("POSTGRES_PASSWORD")
[[ -z "${REDIS_PASSWORD:-}" ]]    && MISSING+=("REDIS_PASSWORD")
[[ -z "${SECRET_KEY:-}" ]]        && MISSING+=("SECRET_KEY")
[[ -z "${TOKEN_ENCRYPTION_KEY:-}" ]] && MISSING+=("TOKEN_ENCRYPTION_KEY")
[[ -z "${REDIS_KEY_HMAC_SECRET:-}" ]] && MISSING+=("REDIS_KEY_HMAC_SECRET")

if [[ ${#MISSING[@]} -gt 0 ]]; then
  err "The following required keys are still empty in .env:"
  for key in "${MISSING[@]}"; do
    err "  - $key"
  done
  err "This shouldn't happen with auto-generation. Check .env manually."
  exit 1
fi
ok "All required keys are set"

# ── Pull base images ──────────────────────────────────────────────────────────
step "Pulling base images..."
$COMPOSE pull postgres redis

# ── Start infrastructure ──────────────────────────────────────────────────────
step "Starting postgres and redis..."
$COMPOSE up -d postgres redis

bash "$ROOT/scripts/wait-healthy.sh" postgres redis

# ── Build and start application ───────────────────────────────────────────────
step "Building and starting backend..."
$COMPOSE up -d --build backend

bash "$ROOT/scripts/wait-healthy.sh" backend

step "Building and starting frontend..."
$COMPOSE up -d --build frontend

# ── Print URLs ────────────────────────────────────────────────────────────────
bash "$ROOT/scripts/print-urls.sh"
