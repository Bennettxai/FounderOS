# Optimal Engine — HTTP API Reference

Base URL: `http://localhost:4200` (configurable via `config :optimal_engine, :api, port: 4200`)

All responses are JSON (`Content-Type: application/json`). CORS is open (`*`) by default. All `POST`/`PATCH` bodies are JSON.

---

## Retrieval

### POST /api/rag

Governed retrieval envelope for agent consumption. The endpoint assembles
workspace-scoped context from Memory Core state, search/index projections, and
wiki/export projections where useful.

**Body**

| Field | Type | Default | Description |
|---|---|---|---|
| `query` | string | required | Natural-language question |
| `workspace` | string | `"default"` | Workspace slug or ID |
| `format` | string | `"markdown"` | `markdown` / `text` / `claude` / `openai` / `json` |
| `bandwidth` | string | `"medium"` | `l0` (~100 tok) / `medium` (~2K tok) / `full` |
| `audience` | string | `"default"` | Audience tag for wiki variant selection |

**Returns** — envelope with `answer`, `sources`, `wiki_hit` boolean, `citations`

**Use when** — agent asks an open question about organizational knowledge

```bash
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"current pricing strategy","workspace":"sales","format":"claude","audience":"sales"}'
```

---

### GET /api/search

Hybrid BM25 + vector + graph search. Returns context-level metadata (not chunk-level).

**Params**

| Param | Default | Description |
|---|---|---|
| `q` | required | Search terms |
| `workspace` | `"default"` | Workspace scope |
| `limit` | `10` | Max results |

**Returns** — `{query, results: [{id, title, slug, score, snippet, node}]}`

**Use when** — user wants to find documents / signals by keyword or semantic similarity

---

### GET /api/grep

Hybrid semantic + literal grep at chunk level. Returns chunk-level matches with full signal trace (slug, scale, intent, sn_ratio, modality, snippet, score).

**Params**

| Param | Default | Description |
|---|---|---|
| `q` | required | Search terms |
| `workspace` | `"default"` | Workspace scope |
| `intent` | — | Filter by intent atom (one of 10 canonical values) |
| `scale` | — | `document` / `section` / `paragraph` / `chunk` |
| `modality` | — | `text` / `image` / `audio` / `code` |
| `limit` | `25` | Max results |
| `literal` | `false` | Force FTS literal match, skip vector |
| `path` | — | Restrict to node slug prefix |

**Returns** — `{query, workspace_id, results: [{slug, scale, intent, sn_ratio, modality, snippet, score}]}`

**Use when** — debugging pipeline output, building filtered context, auditing intent distribution

---

### GET /api/profile

4-tier workspace profile snapshot: static facts + dynamic signals + curated wiki + recent activity. One call to rule them all.

**Params**

| Param | Default | Description |
|---|---|---|
| `workspace` | `"default"` | Workspace slug or ID |
| `audience` | `"default"` | Audience tag |
| `bandwidth` | `"l1"` | `l0` / `l1` / `full` |
| `node` | — | Restrict Tier 1/2 to one node slug |
| `tenant` | `"default"` | Tenant ID |

**Returns** — `{workspace_id, tenant_id, audience, static, dynamic, curated, activity, entities, generated_at}`

**Use when** — seeding an agent's system prompt with workspace context

---

### GET /api/l0

Returns the always-loaded L0 abstract (~100 tokens). The minimal context any agent should receive for every query.

**Returns** — `{l0: string}`

---

## Cued Recall

Five typed endpoints, one per memory-failure pattern. Each builds an intent-optimized query internally and routes through the same `/api/rag` pipeline. Returns the same envelope as `/api/rag` plus `recall_query`.

### GET /api/recall/actions

Past actions, decisions, and commitments.

| Param | Description |
|---|---|
| `actor` | Person who acted (optional) |
| `topic` | Subject matter (optional) |
| `since` | ISO-8601 date lower bound (optional) |
| `workspace` | Workspace scope |

### GET /api/recall/who

Contact / ownership lookup.

| Param | Description |
|---|---|
| `topic` | Subject or object to look up |
| `role` | `owner` (default), `lead`, `contact`, etc. |
| `workspace` | Workspace scope |

### GET /api/recall/when

Schedule and temporal lookup.

| Param | Description |
|---|---|
| `event` | Event name or description |
| `workspace` | Workspace scope |

### GET /api/recall/where

Object-location lookup — which node, file, or path.

| Param | Description |
|---|---|
| `thing` | Item to locate |
| `workspace` | Workspace scope |

### GET /api/recall/owns

Open-task and current-commitment lookup for an actor.

| Param | Description |
|---|---|
| `actor` | Person whose commitments to surface |
| `workspace` | Workspace scope |

---

## Wiki / Export Projection

### GET /api/wiki

List all curated wiki pages for a workspace.

**Params** — `tenant` (default: `"default"`), `workspace` (default: `"default"`)

**Returns** — `{tenant_id, workspace_id, pages: [{slug, audience, version, last_curated, curated_by, size_bytes, workspace_id}]}`

### GET /api/wiki/:slug

Render a single wiki page. Resolves `{{cite}}`, `{{include}}`, `{{expand}}` directives.

**Params** — `tenant`, `workspace`, `audience` (default: `"default"`), `format` (`markdown` / `text`)

**Returns** — `{slug, audience, version, workspace_id, body, warnings}`

### GET /api/wiki/contradictions

Active contradiction surfacing events from the last 90 days.

**Params** — `workspace` (default: `"default"`)

**Returns** — `{workspace_id, contradictions: [{page_slug, contradictions, entities, score, detected_at}], count}`

---

## Memory Primitive

Every memory carries: version chain (`parent_memory_id`, `root_memory_id`, `is_latest`), typed relations (5 types), soft-delete (`is_forgotten`, `forget_after`, `forget_reason`), audience tag, citation URI, and metadata.

### POST /api/memory

Create a new memory.

**Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `content` | string | yes | The memory text |
| `workspace` | string | no | Workspace scope (default: `"default"`) |
| `is_static` | boolean | no | Static (rarely changes) vs dynamic |
| `audience` | string | no | Audience tag |
| `citation_uri` | string | no | `optimal://` URI pointing to source |
| `source_chunk_id` | string | no | Chunk ID this memory derives from |
| `metadata` | object | no | Arbitrary key-value metadata |

**Returns** — `201` + full memory struct

### GET /api/memory

List memories for a workspace.

**Params** — `workspace`, `audience`, `include_forgotten` (bool), `include_old_versions` (bool), `limit` (default 50)

**Returns** — `{workspace_id, count, memories: [...]}`

### GET /api/memory/:id

Fetch one memory by ID.

**Returns** — full memory struct or `404`

### GET /api/memory/:id/versions

Full version chain in chronological order.

**Returns** — `{memory_id, root_id, versions: [...]}`

### GET /api/memory/:id/relations

Inbound and outbound typed relations.

**Returns** — `{memory_id, inbound: [...], outbound: [...]}`

### POST /api/memory/:id/update

Create a new version of a memory (relation type: `updates`).

**Body** — `{content, audience?, citation_uri?, metadata?}`

**Returns** — `201` + new memory struct

### POST /api/memory/:id/extend

Create a child memory (relation type: `extends`). Use for addenda, clarifications, or specializations.

**Body** — `{content, audience?, citation_uri?, metadata?}`

**Returns** — `201` + new memory struct

### POST /api/memory/:id/derive

Create a derived memory (relation type: `derives`). Use for summaries, conclusions, or inferences drawn from the parent.

**Body** — `{content, audience?, citation_uri?, metadata?}`

**Returns** — `201` + new memory struct

### POST /api/memory/:id/forget

Soft-delete a memory. Sets `is_forgotten: true`. Optionally schedule for future forgetting.

**Body** — `{reason?: string, forget_after?: ISO-8601 datetime}`

**Returns** — `204`

### DELETE /api/memory/:id

Hard delete. Irreversible.

**Returns** — `204`

---

## Workspaces

### GET /api/workspaces

List workspaces in an organization.

**Params** — `tenant` (default: `"default"`), `status` (`active` / `archived` / `all`)

**Returns** — `{tenant_id, workspaces: [{id, tenant_id, slug, name, description, status, created_at, archived_at, metadata}]}`

### POST /api/workspaces

Create a workspace.

**Body** — `{slug: string, name: string, description?: string, tenant?: string}`

**Returns** — `201` + workspace struct

### GET /api/workspaces/:id

Fetch one workspace.

### PATCH /api/workspaces/:id

Update name or description.

**Body** — `{name?: string, description?: string}`

### POST /api/workspaces/:id/archive

Soft-delete a workspace.

**Returns** — `204`

### GET /api/workspaces/:id/config

Read merged workspace config (defaults + on-disk `.optimal/config.yaml`).

**Returns** — `{workspace_id, config: {...}}`

### PATCH /api/workspaces/:id/config

Deep-merge body into on-disk config. Returns full merged config after write.

---

## Subscriptions + Proactive Surfacing

### GET /api/subscriptions

List subscriptions for a workspace.

**Params** — `workspace` (default: `"default"`)

**Returns** — `{workspace_id, subscriptions: [{id, scope, scope_value, categories, principal_id, ...}]}`

### POST /api/subscriptions

Create a subscription.

**Body**

| Field | Description |
|---|---|
| `workspace` | Workspace scope |
| `scope` | `workspace` / `topic` / `node` / `entity` |
| `scope_value` | Slug / name / ID matching `scope` |
| `categories` | Array of 14-category surfacing labels |
| `principal_id` | Who receives the pushes |

**Returns** — `201` + subscription struct

### POST /api/subscriptions/:id/pause

Pause delivery without deleting.

**Returns** — `204`

### POST /api/subscriptions/:id/resume

Resume a paused subscription.

**Returns** — `204`

### DELETE /api/subscriptions/:id

Delete a subscription.

**Returns** — `204`

### GET /api/surface/stream?subscription=:id

Server-Sent Events stream. Connect with `EventSource`. Pushes newline-delimited JSON envelopes when the Surfacer fires.

### POST /api/surface/test

Trigger a synthetic push to all listeners of a subscription. Useful for testing.

**Body** — `{subscription: id, slug: wiki_page_slug}`

**Returns** — `204`

---

## Workspace Explorer

### GET /api/workspace

Full node forest — flat list with `parent_id` links for client-side tree building.

**Params** — `workspace` (default: `"default"`)

### GET /api/signals/:id

Full signal granularity: 4-scale chunks, entities by type, classification, intent, cluster membership, wiki citations, architecture binding.

### GET /api/activity

Reverse-chronological events (audit log).

**Params** — `limit` (default 100, max 1000), `kind` (optional filter: `ingest`, `erasure`, `retention_action`, etc.)

---

## Graph Analysis

### GET /api/graph

Full graph payload — all edges + entity summary + node summary.

### GET /api/graph/hubs

Hub entities: degree > 2σ above mean.

### GET /api/graph/triangles

Open triangles — synthesis opportunities.

**Params** — `limit` (default 20)

### GET /api/graph/clusters

Connected components in the entity graph.

### GET /api/graph/reflect

Co-occurrence gaps — entity pairs that appear together but lack an explicit edge.

**Params** — `min` (default 2, minimum co-occurrence count)

### GET /api/node/:node_id

Subgraph for one node: contexts + edges.

---

## Architectures

### GET /api/architectures

All data architectures: built-ins + tenant-registered schemas. Includes processor registry summary.

**Returns** — `{architectures: [{id, name, version, description, modality_primary, granularity, field_count}], processors: [{id, modality, emits}]}`

### GET /api/architectures/:id

Field-level detail for one architecture.

---

## Organizations

### GET /api/organizations

List organizations (tenants) the caller can see. Single-tenant v0.1 returns the default org.

---

## Ops + Observability

### GET /api/status

Liveness + readiness report. Checks: store, migrations, credentials, embedder.

**Returns** — `{status, ok?, checks, degraded}`

### GET /api/metrics

Telemetry snapshot — counters, histograms, uptime_ms.

### GET /api/health

Knowledge-base diagnostic checks — orphaned chunks, embedding drift, duplicate signals.

---

## Desktop Graph Feed

These endpoints match the shape expected by the bundled SvelteKit desktop UI components.

### GET /api/optimal/graph

Entity graph for `OptimalGraphView.svelte`.

**Returns** — `{entities: [{name, type, connections}], edges: [{source, target, relation, weight}], stats}`

### GET /api/optimal/nodes

Node card grid for `NodeDrillDown`.

**Returns** — `{nodes: [{slug, name, type, signal_count}]}`

### GET /api/optimal/nodes/:slug/files

File tree drill-down for a node.

**Returns** — `{files: [{name, path, is_dir, size, children?}]}`
