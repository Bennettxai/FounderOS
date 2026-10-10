# Workspace Pattern

Workspaces are isolated brains. Each workspace has its own node tree, signal files, derivative index, and wiki. Data in workspace A is never visible to queries in workspace B unless explicitly cross-linked.

---

## Why Multi-Workspace

**Problem with a single namespace:** Sales context bleeds into Engineering answers. M&A-sensitive material leaks into general queries. Personal workspaces intermix with company-wide knowledge.

**Solution:** One workspace per isolated brain. Common configurations:

| Workspace | Who uses it | What it holds |
|---|---|---|
| `default` | Everyone | Cross-cutting company knowledge |
| `engineering` | Engineers | Architecture decisions, code context, post-mortems |
| `sales` | Sales team | Deals, negotiations, customer signals |
| `legal` | Legal + leadership | Contracts, regulatory analysis |
| `ma` | Leadership only | M&A targets, due-diligence |
| `alice` | Alice | Personal working notes, private context |

Audience tags (`audience=sales`, `audience=legal`) let one workspace serve multiple receiver types. Workspaces are the isolation boundary; audience is the presentation layer.

---

## Creating a Workspace

### Via HTTP API

```bash
curl -X POST http://localhost:4200/api/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"slug": "engineering", "name": "Engineering Brain", "tenant": "default"}'
```

Response (201):
```json
{
  "id": "ws:engineering",
  "tenant_id": "default",
  "slug": "engineering",
  "name": "Engineering Brain",
  "status": "active",
  "created_at": "2026-04-28T00:00:00Z"
}
```

### Via Mix Task

```bash
mix optimal.init ~/company-brain/engineering
```

This scaffolds the full on-disk structure:

```
engineering/
├── nodes/
│   ├── 01-team/
│   │   ├── context.md
│   │   └── signals/
│   └── product-customer-portal/
│       ├── context.md
│       └── signals/
├── .wiki/
│   └── SCHEMA.md
├── .optimal/           # created by engine on first ingest
│   └── config.yaml
└── assets/
```

---

## On-Disk Convention

### Node Structure

```
nodes/
└── <slug>/             e.g. entity-company, product-customer-portal, 03-sales
    ├── context.md      Persistent facts — edit in place as ground truth changes
    ├── signal.md       Rolling weekly status — overwritten each cycle
    └── signals/        Append-only dated signals
        └── YYYY-MM-DD-slug.md
```

### Signal File Frontmatter

Every signal file has YAML frontmatter + body:

```markdown
---
title: Customer pricing call — Q4
genre: transcript
mode: linguistic
node: 03-sales
sn_ratio: 0.75
entities:
  - { name: "Alice", type: person }
  - { name: "Customer Portal Product", type: product }
authored_at: 2026-04-28T14:00:00Z
---

## Summary
One-sentence abstract. Engine pulls this for L0 (~100 tokens).

## Key points
- Each bullet is an atomic claim. Engine pulls this for L1 (~2K tokens).

## Detail
Full content (L2 / full). Prose is fine; the decomposer splits automatically.
```

| Field | Required | Used for |
|---|---|---|
| `title` | yes | Display + search ranking |
| `genre` | yes | Classification + retrieval filter |
| `mode` | no | `linguistic` / `visual` / `code` / `data` / `mixed` |
| `node` | inferred | Routed from directory; override if needed |
| `sn_ratio` | no | Signal/noise boost; defaults to 0.5 |
| `entities` | no | Pre-extracted (engine can derive them) |
| `authored_at` | no | ISO-8601; defaults to file mtime |

---

## Workspace Config Schema

`.optimal/config.yaml` — read by `GET /api/workspaces/:id/config` and used by the engine to govern ingestion and retrieval behavior for this workspace.

```yaml
# .optimal/config.yaml

workspace:
  slug: engineering
  name: Engineering Brain
  tenant: default

ingestion:
  # Formats to ingest (default: all supported)
  formats: [md, pdf, code, yaml, json]
  # Skip files matching these globs
  exclude_globs: ["*.lock", "node_modules/**", ".git/**"]
  # Minimum sn_ratio to ingest (0.0–1.0, default: 0.2)
  min_sn_ratio: 0.3

retrieval:
  # Default audience when none supplied
  default_audience: engineering
  # Default bandwidth
  default_bandwidth: l1
  # Enable graph boost in hybrid retrieval
  graph_boost: true

wiki:
  # How often the curator re-checks each page (in seconds)
  curation_interval: 3600
  # Audience variants the curator should maintain
  audiences: [default, engineering, leadership]
  # Max age of a citation before it's flagged as stale (days)
  citation_ttl_days: 90

surfacing:
  # Categories this workspace monitors
  categories:
    - recent_actions
    - blockers
    - contradictions
    - specifications
    - metrics
```

### Reading and Writing Config

```bash
# Read
curl http://localhost:4200/api/workspaces/engineering/config | jq '.config'

# Write (deep-merge)
curl -X PATCH http://localhost:4200/api/workspaces/engineering/config \
  -H 'Content-Type: application/json' \
  -d '{"ingestion": {"min_sn_ratio": 0.4}}'
```

---

## Isolation Guarantees

1. **Filesystem isolation** — each workspace resolves to its own directory tree. The engine resolves `workspace_id` → directory path before any file I/O.
2. **Query isolation** — all SQL queries include `workspace_id = ?` predicates. Cross-workspace joins don't exist.
3. **Wiki isolation** — `.wiki/` is per-workspace. A page in `engineering` is never served in response to a `sales` query.
4. **Memory isolation** — `POST /api/memory` scopes to `workspace`. `GET /api/memory` returns only memories matching the requested workspace.
5. **Subscription isolation** — surfacing events are scoped to `workspace_id`. A subscription in `sales` never receives events from `engineering`.

---

## Ingesting a Workspace

```bash
# Full workspace ingest (walk the node tree, process all signals)
mix optimal.ingest_workspace ~/company-brain/engineering

# Ingest a single signal
mix optimal.ingest --file ~/company-brain/engineering/nodes/product-customer-portal/signals/2026-04-28-api-design.md

# Or via HTTP (POST body can be text or a file upload)
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query": "verify ingest worked", "workspace": "engineering"}'
```

---

## Archiving a Workspace

```bash
# Soft delete (data preserved, queries return 404)
curl -X POST http://localhost:4200/api/workspaces/engineering/archive

# List all workspaces including archived
curl 'http://localhost:4200/api/workspaces?status=all'
```
