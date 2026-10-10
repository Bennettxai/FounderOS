# Optimal Engine Options

This file lists the setup options that matter for users, agents, and products embedding Optimal Engine.

## Command Surface

Use the repo wrapper for normal usage:

```bash
bin/optimal <command>
```

Use direct Mix tasks when you are developing the engine internals:

```bash
mix optimal.<task>
```

Recommended agent commands:

```bash
bin/optimal boot
bin/optimal doctor
bin/optimal find "query" --workspace default:my-workspace
bin/optimal capture "raw signal" --workspace default:my-workspace
bin/optimal aware "durable correction" --workspace default:my-workspace
bin/optimal close "what changed and how verified"
```

## Runtime Environment

| Variable | Default | Purpose |
| --- | --- | --- |
| `OPTIMAL_ENGINE_API_URL` | `http://localhost:4200` | CLI API target. |
| `OPTIMAL_ENGINE_API_KEY` | empty | Optional bearer token for HTTP calls. |
| `OPTIMAL_ENGINE_DB` | `.optimal/index.db` | Local SQLite store path. |
| `OPTIMAL_ENGINE_CACHE` | `.optimal/cache` | Local cache path. |
| `OPTIMAL_ENGINE_ROOT` | repo root | Engine root for source-checkout usage. |
| `OPTIMAL_KNOWLEDGE_BACKEND` | `rocksdb` when available | Graph backend selection. |
| `CONNECTOR_KEY` | generated into `.optimal/connector_key` | Local connector secret. |

Keep private values in environment variables, local secret stores, or deployment secret managers.
Do not commit them.

## Knowledge Graph Backend

Use RocksDB for normal local persistence:

```bash
OPTIMAL_KNOWLEDGE_BACKEND=rocksdb
```

Use ETS for fast in-memory testing:

```bash
OPTIMAL_KNOWLEDGE_BACKEND=ets
```

Use Mnesia only when intentionally testing distributed graph behavior:

```bash
OPTIMAL_KNOWLEDGE_BACKEND=mnesia
```

SQLite remains the durable local source of truth.
The graph backend supports graph runtime behavior and retrieval.

## Product Embedding

Products such as BusinessOS should connect over HTTP:

```text
BusinessOS
  -> scoped engine URL
  -> scoped API key or local connector grant
  -> tenant, organization, workspace, Node, and policy scope
  -> Optimal Engine API
```

Downloaded users should get their own local bundled engine and local store.
They should not connect to Roberto's private engine unless explicitly configured as Roberto's machine.

## Setup Profiles

| Profile | Use when | Expected behavior |
| --- | --- | --- |
| Local source checkout | Developer or local agent work | `bin/optimal dev` runs the engine on `localhost:4200`. |
| Bundled desktop engine | BusinessOS desktop user | App launches a local release with fresh local runtime data. |
| Production service | Hosted org or workspace | Service uses production secrets, scoped keys, and Postgres target when configured. |
| Test mode | CI and unit tests | Use isolated temp stores and avoid private `.optimal/` data. |

## Safe Defaults

Prefer repo-relative defaults.
Prefer environment-driven config.
Fail closed when scope or credentials are missing.
Never silently share memory across tenants, organizations, or workspaces.
