#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MISSING=()
MIN_NODE_MAJOR=22

require_command() {
    local command_name="$1" install_hint="$2"
    if ! command -v "$command_name" >/dev/null 2>&1; then
        MISSING+=("$command_name - $install_hint")
    fi
}

require_command node "install Node.js 22 or newer"
require_command npm "installed with Node.js"
require_command go "install the Go version declared in desktop/backend-go/go.mod"
require_command pg_config "macOS: brew install postgresql@16"
require_command redis-server "macOS: brew install redis"
require_command redis-cli "installed with Redis"
require_command mix "macOS: brew install elixir"
require_command python3 "install Python 3"
require_command curl "install curl"
require_command openssl "install OpenSSL"

if [ "${#MISSING[@]}" -gt 0 ]; then
    printf 'BusinessOS cannot start because these prerequisites are missing:\n\n' >&2
    printf '  - %s\n' "${MISSING[@]}" >&2
    printf '\nInstall them, then rerun: make onboard\n' >&2
    exit 1
fi

node_major="$(node --version | sed 's/^v//' | cut -d. -f1)"
if ! [[ "$node_major" =~ ^[0-9]+$ ]] || [ "$node_major" -lt "$MIN_NODE_MAJOR" ]; then
    printf 'BusinessOS requires Node.js %s or newer. Found: %s\n' \
        "$MIN_NODE_MAJOR" "$(node --version)" >&2
    exit 1
fi

if ! command -v pnpm >/dev/null 2>&1; then
    if command -v corepack >/dev/null 2>&1; then
        printf 'Activating the repository-pinned pnpm version through Corepack.\n'
        corepack enable
        corepack prepare pnpm@9.15.4 --activate
    else
        printf 'pnpm is missing and Corepack is unavailable.\n' >&2
        printf 'Install pnpm 9.15.4, then rerun: make onboard\n' >&2
        exit 1
    fi
fi

on_error() {
    local exit_code=$?
    printf '\nBusinessOS onboarding stopped before verification completed.\n' >&2
    printf 'Status: make dev-local-status\n' >&2
    printf 'Logs:   %s/.run/*.log\n' "$ROOT" >&2
    exit "$exit_code"
}
trap on_error ERR

printf 'Starting BusinessOS first-run setup.\n'
printf 'Existing environment files and local data will be preserved.\n\n'

bash "$ROOT/scripts/dev-local.sh" start
bash "$ROOT/scripts/dev-local.sh" verify

trap - ERR
printf '\nBusinessOS is ready.\n'
frontend_port="$(grep '^FRONTEND_PORT=' "$ROOT/.env.dev" | cut -d= -f2)"
printf 'Open:       http://localhost:%s/register\n' "$frontend_port"
printf 'Next:       Create a local account. No OAuth or cloud account is required.\n'
printf 'Workspace:  Your first workspace starts empty with the foundational modules ready to shape.\n'
printf 'Run again: make dev-local\n'
printf 'Stop:      make dev-local-stop\n'
