# Optimal Engine System Map

Optimal Engine is a scoped memory and retrieval runtime.
It turns raw inputs into governed context that humans, agents, APIs, and apps can use.

## Core Flow

```text
Raw inputs
  -> Source Packages
  -> Signals
  -> Claims
  -> Review and policy
  -> Facts
  -> Memory Objects
  -> Retrieval and Context Packages
  -> Agent or app action
  -> Observations
  -> new Claims
```

## Ownership

| Layer | Owns |
| --- | --- |
| Topology | Tenants, organizations, workspaces, Nodes, relationships, aliases, and policies. |
| Intake | Raw source preservation, source metadata, and evidence boundaries. |
| Signal | Mode, genre, type, format, structure, routing, and classification. |
| Memory Core | Claims, Facts, Memory Objects, provenance, lifecycle, and active pools. |
| Retrieval | Search, graph recall, context assembly, ranking, and token budgets. |
| Workflow and Skill | Repeatable procedures, tool traces, skill packages, and run records. |
| Governance | Tool grants, API keys, policy checks, audit, and fail-closed access. |
| Export | Markdown, wiki, HTML, APIs, app views, and other projections. |

## Store Model

SQLite is the local durable store today.
Postgres is the production durable target.
RocksDB is the default local persistent graph backend when installed.
ETS and Mnesia are alternate graph backends.
Vectors, chunks, FTS rows, and caches are projections.
The engine exposes 12 logical stores through `GET /api/stores`.
Use `GET /api/stores/audit` to verify SQLite integrity, foreign keys, migrations, FTS parity, workspace isolation, vector validity, asset cataloging, verified backups, RLM readiness, and cache readiness.
The audit endpoint returns HTTP `503` when any required check fails.

Do not call RAG a store.
RAG is a retrieval and answer path over scoped context.

## Isolation Rule

Every read and write must carry explicit scope:

```text
tenant
organization
workspace
Node
policy
principal
```

If scope is missing or ambiguous, fail closed or ask for clarification.
Never leak one workspace into another.

## BusinessOS Boundary

BusinessOS is an app surface and operational system.
It owns desktop state, windows, modules, roles, users, sessions, and operational records.

Optimal Engine is the knowledge and memory system.
It owns evidence, retrieval, graph, RAG, claims, facts, memories, context packages, and source packages.
