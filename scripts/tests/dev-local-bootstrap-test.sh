#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/businessos-bootstrap-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

FIXTURE_ROOT="$TMP_ROOT/repo"
RUN_DIR="$FIXTURE_ROOT/.run"
FAKE_BIN="$TMP_ROOT/bin"
FAKE_PGDATA="$RUN_DIR/pgdata"
CALLS="$TMP_ROOT/calls"

mkdir -p \
    "$FIXTURE_ROOT/desktop/backend-go" \
    "$FIXTURE_ROOT/frontend" \
    "$FIXTURE_ROOT/optimal-engine" \
    "$RUN_DIR" \
    "$FAKE_BIN"
cp "$ROOT/desktop/backend-go/.env.example" "$FIXTURE_ROOT/desktop/backend-go/.env.example"

cat > "$FIXTURE_ROOT/.env.dev" <<'EOF'
BACKEND_PORT=8801
FRONTEND_PORT=5273
POSTGRES_PORT=25432
REDIS_PORT=26379
ENGINE_PORT=4200
EOF

fake_command() {
    local name="$1"
    shift
    cat > "$FAKE_BIN/$name" <<EOF
#!/usr/bin/env bash
$*
EOF
    chmod +x "$FAKE_BIN/$name"
}

fake_command pg_config 'echo "$FAKE_BIN"'
fake_command initdb '
set -e
while [ "$#" -gt 0 ]; do
    if [ "$1" = "-D" ]; then shift; data_dir="$1"; fi
    shift
done
mkdir -p "$data_dir"
echo 16 > "$data_dir/PG_VERSION"
echo "initdb" >> "$CALLS"
'
fake_command pg_ctl '
echo "pg_ctl $*" >> "$CALLS"
if [ "${1:-}" = "--version" ]; then echo "pg_ctl (PostgreSQL) 16.1"; fi
'
fake_command pg_isready 'exit 0'
fake_command psql '
echo "psql $*" >> "$CALLS"
if printf "%s" "$*" | grep -q "SELECT 1 FROM pg_database"; then
    [ -f "$FAKE_DATABASE_EXISTS" ] && echo 1
fi
'
fake_command createdb 'touch "$FAKE_DATABASE_EXISTS"; echo "createdb $*" >> "$CALLS"'
fake_command redis-server 'touch "$FAKE_REDIS_READY"; echo "redis-server" >> "$CALLS"'
fake_command redis-cli '
[ -f "$FAKE_REDIS_READY" ] && echo PONG
'

export BUSINESSOS_ROOT="$FIXTURE_ROOT"
export BUSINESSOS_RUN_DIR="$RUN_DIR"
export BUSINESSOS_PG_BIN="$FAKE_BIN"
export PATH="$FAKE_BIN:$PATH"
export FAKE_BIN CALLS
export FAKE_DATABASE_EXISTS="$TMP_ROOT/database-exists"
export FAKE_REDIS_READY="$TMP_ROOT/redis-ready"
export REDIS_CLI="$FAKE_BIN/redis-cli"

# shellcheck disable=SC1091
. "$ROOT/scripts/dev-local.sh"

ensure_backend_env
[ -f "$FIXTURE_ROOT/desktop/backend-go/.env" ]
grep -q '^DATABASE_URL=postgres://postgres@127.0.0.1:25432/businessos_dev?sslmode=disable$' \
    "$FIXTURE_ROOT/desktop/backend-go/.env"
grep -q '^REDIS_URL=redis://127.0.0.1:26379/0$' \
    "$FIXTURE_ROOT/desktop/backend-go/.env"
token_encryption_key=$(grep '^TOKEN_ENCRYPTION_KEY=' "$FIXTURE_ROOT/desktop/backend-go/.env" | cut -d= -f2-)
[ "$(printf '%s' "$token_encryption_key" | openssl base64 -d -A | wc -c | tr -d ' ')" -eq 32 ]

original_checksum=$(cksum "$FIXTURE_ROOT/desktop/backend-go/.env")
ensure_backend_env
[ "$original_checksum" = "$(cksum "$FIXTURE_ROOT/desktop/backend-go/.env")" ]

ensure_local_postgres
[ -f "$FAKE_PGDATA/PG_VERSION" ]
[ -f "$FAKE_DATABASE_EXISTS" ]
grep -q '^initdb$' "$CALLS"
grep -q '^createdb .*businessos_dev$' "$CALLS"

# Credentials embedded in DATABASE_URL must survive database probes. This is
# required when a Docker-managed local PostgreSQL instance owns the dev port.
sed -i.bak \
    's#^DATABASE_URL=.*#DATABASE_URL=postgres://postgres:local-secret@127.0.0.1:25432/businessos_dev?sslmode=disable#' \
    "$FIXTURE_ROOT/desktop/backend-go/.env"
rm -f "$FIXTURE_ROOT/desktop/backend-go/.env.bak"
touch "$FAKE_DATABASE_EXISTS"
ensure_local_postgres
grep -q 'psql postgres://postgres:local-secret@127.0.0.1:25432/postgres?sslmode=disable' "$CALLS"

ensure_local_redis
[ -f "$FAKE_REDIS_READY" ]
grep -q '^redis-server$' "$CALLS"

echo "dev-local bootstrap tests passed"
