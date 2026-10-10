#!/usr/bin/env bash
# Compatibility entrypoint. The former launcher referenced a retired checkout
# and applied obsolete schema.sql directly to a live database.
# All local starts now use the canonical, readiness-checked launcher.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$ROOT/scripts/dev-local.sh" "$@"

# Legacy implementation retained below temporarily for forensic reference.
# It is unreachable because the canonical launcher above replaces it.
if [ "$(uname -s)" != "Darwin" ]; then
  echo "✗ run-local.sh is the macOS launcher (Postgres.app toolchain, DYLD paths)." >&2
  echo "  On Linux use: 'make dev' (Docker) or 'make dev-local' (host processes)." >&2
  exit 1
fi
TC="$HOME/.local-toolchain"; RUN="$TC/.run"; REPO="$HOME/Desktop/agency-miosa-businessos"
export PATH="$TC/go/bin:$PATH"
mkdir -p "$RUN"

# Postgres (self-contained Postgres.app binaries in the toolchain; port 25432)
PGBIN="$(cat "$TC/.pgbin" 2>/dev/null || true)"
PGDATA="$TC/pgdata"
pg() { "$PGBIN/$1" "${@:2}"; }

start() {
  # Postgres
  if [ -n "$PGBIN" ] && [ -x "$PGBIN/pg_ctl" ]; then
    export DYLD_LIBRARY_PATH="$(dirname "$PGBIN")/lib:${DYLD_LIBRARY_PATH:-}"
    if ! pg pg_isready -h localhost -p 25432 -U postgres >/dev/null 2>&1; then
      [ -f "$PGDATA/PG_VERSION" ] || pg initdb -D "$PGDATA" -U postgres --auth=trust --encoding=UTF8 >/dev/null 2>&1
      pg pg_ctl -D "$PGDATA" -o "-p 25432 -k /tmp" -l "$RUN/postgres.log" start >/dev/null 2>&1 || true
      sleep 2
      pg psql -h localhost -p 25432 -U postgres -d postgres -tc \
        "SELECT 1 FROM pg_database WHERE datname='businessos_dev'" 2>/dev/null | grep -q 1 || \
        pg psql -h localhost -p 25432 -U postgres -d postgres -c "CREATE DATABASE businessos_dev;" >/dev/null 2>&1 || true
      echo "✔ postgres started on :25432"
    fi
    # Apply the canonical schema on EVERY start. schema.sql is idempotent
    # (CREATE ... IF NOT EXISTS / guarded types+constraints / ON CONFLICT), so
    # this is a safe no-op reconciliation that heals any drift — the dev DB can
    # never fall behind schema.sql the way it did before (missing workspace_id /
    # org tables -> 500s). ON_ERROR_STOP makes a broken schema fail loudly.
    if pg pg_isready -h localhost -p 25432 -U postgres >/dev/null 2>&1; then
      if pg psql -h localhost -p 25432 -U postgres -d businessos_dev \
           -q -v ON_ERROR_STOP=1 -f "$REPO/desktop/backend-go/internal/database/schema.sql" \
           > "$RUN/schema-apply.log" 2>&1; then
        echo "✔ schema applied (idempotent, no drift)"
      else
        echo "✗ schema apply FAILED — see $RUN/schema-apply.log"; tail -3 "$RUN/schema-apply.log"
      fi
    fi
  else
    echo "⚠ postgres binaries not found (expected path in $TC/.pgbin) — backend will fail to start"
  fi
  # Redis
  "$TC/redis-stable/src/redis-cli" -p 6379 ping >/dev/null 2>&1 || \
    ("$TC/redis-stable/src/redis-server" --port 6379 --dir "$TC/redis-data" --daemonize yes && echo "✔ redis started")
  # Backend (rebuild if missing) — port + DB come from desktop/backend-go/.env (SERVER_PORT=8801, DB :25432)
  [ -x "$RUN/businessos-backend" ] || (cd "$REPO/desktop/backend-go" && go build -o "$RUN/businessos-backend" ./cmd/server)
  if ! curl -s -m2 http://127.0.0.1:8801/health >/dev/null 2>&1; then
    (cd "$REPO/desktop/backend-go"; set -a; . ./.env; set +a; nohup "$RUN/businessos-backend" > "$RUN/backend.log" 2>&1 &)
    echo "✔ backend starting on :8801"
  fi
  # Frontend (binds :5273 via frontend/.env FRONTEND_PORT)
  if ! curl -s -m2 http://localhost:5273/ >/dev/null 2>&1; then
    (cd "$REPO/frontend"; nohup pnpm dev > "$RUN/frontend.log" 2>&1 &)
    echo "✔ frontend starting on :5273"
  fi
  sleep 6
  echo "→ Desktop app: cd $REPO/desktop && npm start"
  echo "→ Or browser:  http://localhost:5273   (login: len@lunivate.com / BusinessOS123!)"
}
stop() {
  pkill -f businessos-backend 2>/dev/null || true
  lsof -ti:5273 | xargs kill -9 2>/dev/null || true
  "$TC/redis-stable/src/redis-cli" -p 6379 shutdown nosave 2>/dev/null || true
  [ -n "$PGBIN" ] && pg pg_ctl -D "$PGDATA" stop >/dev/null 2>&1 || true
  echo "stopped"
}
status() {
  if [ -n "$PGBIN" ]; then
    export DYLD_LIBRARY_PATH="$(dirname "$PGBIN")/lib:${DYLD_LIBRARY_PATH:-}"
    pg pg_isready -h localhost -p 25432 -U postgres 2>/dev/null | sed 's/^/postgres :25432 -> /' || echo "postgres down"
  fi
  curl -s -m2 -o /dev/null -w "backend  :8801 -> %{http_code}\n" http://127.0.0.1:8801/health || echo "backend down"
  curl -s -m2 -o /dev/null -w "frontend :5273 -> %{http_code}\n" http://localhost:5273/ || echo "frontend down"
  "$TC/redis-stable/src/redis-cli" -p 6379 ping 2>/dev/null | sed 's/^/redis    :6379 -> /'
}
case "${1:-start}" in start) start;; stop) stop;; status) status;; *) echo "usage: $0 start|stop|status";; esac
