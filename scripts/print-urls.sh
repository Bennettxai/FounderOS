#!/usr/bin/env bash
# =============================================================================
# print-urls.sh — Print service URLs after stack is up.
# =============================================================================

# Load .env if present (non-fatal)
if [[ -f "$(dirname "$0")/../.env" ]]; then
  # shellcheck disable=SC1091
  set -a
  source "$(dirname "$0")/../.env" 2>/dev/null || true
  set +a
fi

FRONTEND_PORT="${FRONTEND_PORT:-3000}"
BACKEND_PORT="${BACKEND_PORT:-8001}"

BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RESET='\033[0m'

printf "\n"
printf "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
printf "${BOLD}  BusinessOS is running${RESET}\n"
printf "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
printf "\n"
printf "  ${GREEN}Frontend${RESET}   ${CYAN}http://localhost:${FRONTEND_PORT}${RESET}\n"
printf "  ${GREEN}Backend${RESET}    ${CYAN}http://localhost:${BACKEND_PORT}${RESET}\n"
printf "  ${GREEN}Health${RESET}     ${CYAN}http://localhost:${BACKEND_PORT}/healthz${RESET}\n"
printf "  ${GREEN}Readiness${RESET}  ${CYAN}http://localhost:${BACKEND_PORT}/readyz${RESET}\n"
printf "\n"
printf "  Logs:       ${BOLD}docker compose logs -f${RESET}\n"
printf "  Stop:       ${BOLD}docker compose down${RESET}\n"
printf "  Teardown:   ${BOLD}make clean${RESET}  (destroys all data)\n"
printf "\n"
