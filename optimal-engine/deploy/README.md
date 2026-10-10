# Optimal Engine Deployment

Docker is optional. The normal local path is still:

```bash
make install
make bootstrap
make dev
curl http://localhost:4200/api/health
```

Local runtime state is created under `.optimal/`.
That directory contains the SQLite store, caches, WAL files, and local connector key.
It is ignored by git and should not be committed or copied into public docs.

Use this directory when you want a packaged backend service with a persistent
data volume and HTTP API.

## What Runs By Default

```bash
cd deploy
cp env.example .env
docker compose up --build
```

Default service:

| Service | URL | Purpose |
| --- | --- | --- |
| `engine` | `http://localhost:4200` | Elixir/OTP API, SQLite store, indexes, workspace ingestion, retrieval, wiki/export services. |

Optional app/docs/site surfaces are not part of the backend contract. Start them
only when you explicitly need them:

```bash
docker compose --profile surfaces up --build
```

## Storage Layout

The engine container uses `/data` as its runtime root:

```text
/data/
  .optimal/
    index.db
    index.db-wal
    index.db-shm
    cache/
  sample-workspace/
  <mounted workspaces>/
```

The database is the governed runtime state. Markdown workspaces are projections
and editing surfaces. Indexes and caches are rebuildable acceleration layers.

## Optional Storage Profile

The default deployment remains local-first and requires no external database.

Use the optional overlay when a team workload needs shared relational state, object storage, replayable events, distributed cache, high-scale vectors, or centralized secrets.

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.storage.yml \
  --profile storage up -d
```

The overlay provisions PostgreSQL, Garage, NATS JetStream, Valkey, Qdrant, and OpenBao with development defaults.

Replace every development credential and configure production security before exposing these services.

Fractal Computing is configured separately as an enterprise partner substrate and is not started by Compose.

Read [`../docs/architecture/STORAGE-CAPABILITIES-AND-WORKSPACE-FLOW.md`](../docs/architecture/STORAGE-CAPABILITIES-AND-WORKSPACE-FLOW.md) before activating or migrating a provider.

## Environment

Copy `env.example` to `.env` for local Docker or `.env.prod` for the production
overlay.

Important variables:

| Variable | Meaning |
| --- | --- |
| `OPTIMAL_API_PORT` | Host/API port, default `4200`. |
| `OPTIMAL_AUTH_REQUIRED` | Require API auth. Set `true` outside local development. |
| `OPTIMAL_ENGINE_ROOT` | Runtime root inside the container, default `/data`. |
| `OPTIMAL_ENGINE_DB` | SQLite database path, default `/data/.optimal/index.db`. |
| `OPTIMAL_ENGINE_CACHE` | Cache/index path, default `/data/.optimal/cache`. |
| `OPTIMAL_OLLAMA_URL` | Optional local model endpoint for embeddings/generation. |

## Backend Production

Production should expose the engine through a TLS reverse proxy and require auth:

```bash
cd deploy
cp env.example .env.prod
```

Set:

```dotenv
OPTIMAL_AUTH_REQUIRED=true
DOMAIN=engine.example.com
LETSENCRYPT_EMAIL=admin@example.com
```

Then run:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

The production overlay starts Caddy for TLS and keeps the engine port private
behind the proxy.

## Backups

Back up the Docker data volume:

```bash
docker run --rm \
  -v optimal-engine_optimal_data:/data:ro \
  -v "$(pwd)":/backup \
  alpine \
  tar czf /backup/optimal_data_$(date +%Y%m%d_%H%M%S).tar.gz /data
```

Restore:

```bash
docker compose down
docker run --rm \
  -v optimal-engine_optimal_data:/data \
  -v "$(pwd)":/backup \
  alpine \
  sh -c "cd / && tar xzf /backup/optimal_data_<timestamp>.tar.gz"
docker compose up -d
```

## When Not To Use Docker

Do not use Docker just to organize a local workspace or run the CLI. For that,
run the local setup commands and let agents use the `./optimal` wrapper, Mix
tasks, or the HTTP API directly.
