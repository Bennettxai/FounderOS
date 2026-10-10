#!/usr/bin/env bash
# =============================================================================
# wait-healthy.sh — Block until all named compose services are healthy.
# Usage: bash scripts/wait-healthy.sh <service> [<service> ...]
# =============================================================================
set -euo pipefail

MAX_WAIT=120   # seconds before giving up per service
POLL=3         # polling interval in seconds

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
RESET='\033[0m'

# Compose can be `docker compose` (plugin) or `docker-compose` (standalone)
if docker compose version &>/dev/null 2>&1; then
  COMPOSE="docker compose"
else
  COMPOSE="docker-compose"
fi

wait_for_service() {
  local service="$1"
  local elapsed=0

  printf "  Waiting for ${YELLOW}%s${RESET} to be healthy" "$service"

  while true; do
    local health
    health=$($COMPOSE ps --format json "$service" 2>/dev/null \
      | python3 -c "import sys,json; data=sys.stdin.read().strip(); \
          rows=[json.loads(l) for l in data.splitlines() if l.strip()]; \
          print(rows[0].get('Health','') if rows else '')" 2>/dev/null || true)

    # Fallback: check via docker inspect when the compose JSON output is empty
    if [[ -z "$health" ]]; then
      local container
      container=$($COMPOSE ps -q "$service" 2>/dev/null | head -1 || true)
      if [[ -n "$container" ]]; then
        health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || true)
      fi
    fi

    case "$health" in
      healthy)
        printf "\n  ${GREEN}✓${RESET} %s is healthy\n" "$service"
        return 0
        ;;
      unhealthy)
        printf "\n  ${RED}✗${RESET} %s is unhealthy\n" "$service"
        $COMPOSE logs --tail=20 "$service" || true
        exit 1
        ;;
      *)
        printf "."
        ;;
    esac

    if (( elapsed >= MAX_WAIT )); then
      printf "\n  ${RED}✗${RESET} Timed out waiting for %s (${MAX_WAIT}s)\n" "$service"
      $COMPOSE logs --tail=30 "$service" || true
      exit 1
    fi

    sleep "$POLL"
    (( elapsed += POLL ))
  done
}

for svc in "$@"; do
  wait_for_service "$svc"
done
