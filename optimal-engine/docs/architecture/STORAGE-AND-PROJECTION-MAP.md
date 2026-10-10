# Storage And Projection Map

Optimal Engine separates three concerns that are easy to accidentally mix:

```text
storage substrate = where bytes and records live
domain layer      = who owns lifecycle and meaning
projection surface = how humans, agents, and apps see or control it
```

The same information can appear in a database row, a markdown file, an API
response, a wiki page, a dashboard card, and an agent context package. Only one
layer should own the lifecycle.

## The Rule

```text
Store data once as governed state.
Project it many times for different users, agents, tools, and interfaces.
Accept edits back as new evidence or reviewed topology changes.
```

Markdown, HTML, dashboards, wiki pages, reports, and agent prompts are not
separate truth systems. They are projections or control surfaces over the engine.

## Storage Substrates

| Substrate | Role | Canonical? | Rebuildable? | Notes |
| --- | --- | --- | --- | --- |
| SQLite | Local canonical runtime store today. | Yes, for local/dev/personal runs. | No for durable rows; yes for derived rows. | Owns current DB-backed runtime tables. |
| Postgres | Target production canonical runtime store. | Yes, target for multi-user production. | No for durable rows; yes for derived rows. | Keeps the same layer ownership model as SQLite. |
| Raw artifact storage | Preserved source files, uploads, attachments, media, exported payloads. | Yes. | No unless external source still exists. | Source evidence must survive parser/model/index changes. |
| FTS/vector/chunk indexes | Retrieval acceleration and ranked candidate generation. | No. | Yes. | Rebuilt from sources, chunks, memories, and extraction rows. |
| Cache directories | Parse, embedding, context, and local runtime caches. | No. | Yes. | Never trust caches as source of truth. |
| ETS | In-memory graph/knowledge backend. | No for durable truth. | Yes. | Fast local process state. |
| RocksDB | Optional persistent graph/triple-store backend. | Optional for graph workload, not main workspace DB. | Depends on configured use. | Requires native RocksDB runtime support. |
| Mnesia/Riak backends | Optional distributed graph/knowledge backend experiments. | Optional. | Depends on configured use. | Not the primary product storage path today. |
| S3-compatible object storage | Shared artifacts, media, backups, and multi-device evidence. | Yes for artifacts assigned to it. | No unless replicated elsewhere. | Garage provides an open-source self-hosted option. |
| NATS JetStream | Replayable event, job, connector, and replication transport. | No. | Yes from the governed mutation and job ledgers. | Transport never replaces canonical state. |
| Valkey | Distributed cache, rate limits, locks, and coordination. | No. | Yes. | Ephemeral acceleration only. |
| Qdrant | High-scale semantic retrieval projection. | No. | Yes. | Vector results remain candidates, not truth. |
| DuckDB and Parquet | Large analytical and historical projections. | No for transactional state. | Yes. | Analytics remain isolated from operational writes. |
| OpenBao | Secret, key, PKI, and credential lifecycle. | Yes for managed secrets. | Recovery follows secret-manager policy. | Workspace policies contain references, not secret values. |
| Fractal Computing | Enterprise synchronized digital-twin substrate. | Governed by the enterprise integration boundary. | Depends on the enterprise contract. | Cloud-only partner capability with controlled, auditable promotion to systems of record. |
| Markdown workspace | Human-operable projection and editing surface. | No by default. | Yes from engine state. | Edits re-enter as Source Packages or topology change requests. |
| HTML/wiki/dashboard/API | Display and control projections. | No. | Yes. | Should be regenerated from engine state. |

## Store Decision Matrix

Use this table when adding a new feature, connector, agent action, parser, or
projection. The question is not "where can I put this fastest?" The question is
"which layer owns this lifecycle?"

| Data | Goes in | Owner | Why |
| --- | --- | --- | --- |
| Organization, tenant, workspace, Node, Node type, membership, relationship, alias, routing rule | SQLite locally, Postgres target for production | Workspace / Topology | This is durable structure. It decides scope, permissions, routing, and projections. |
| Raw text dump, message, note, event, API payload, connector payload, tool result | `source_packages` plus raw artifact storage when file-backed | Source Intake / Memory Core | Preserve evidence before interpretation. |
| File, attachment, image, audio, video, PDF, archive | raw artifact storage plus `assets` row | Memory Core | The file is evidence; parser/model output is derived. |
| Signal classification | signal metadata and compatibility context/search rows | Signal Pipeline | Classification routes and parses input; it is not accepted truth. |
| Claim candidate | `claims` | Memory Core | A Claim records what a source appears to say before acceptance. |
| Reviewed truth | `facts` | Memory Core | Facts require evidence, validity, confidence, precision, and review/policy state. |
| Institutional meaning | `memory_objects` | Memory Core | Memory explains why Facts/Claims matter in context. |
| Evidence, semantic, temporal, procedural links | `relationship_edges` | Memory Core / Topology depending on edge type | Graph links need ownership and permissions, not loose references. |
| Parser/model/tool lineage | `derivation_ledger`, model/tool run records | Memory Core / Governance / Audit | Every generated artifact needs provenance and replay/rebuild information. |
| Transcript/OCR/visual observation/embedding reference | `asset_extractions` and typed projection tables | Memory Core / Pipeline | These are governed derived artifacts linked to the raw asset. |
| Vector, FTS, chunk, rerank, summary acceleration | index/cache/projection tables | Retrieval / Pipeline | Rebuild from canonical sources and derived extraction rows. |
| Query result for a human or agent | `context_packages` | Retrieval / Context | Context is an authorized package, not random chunks. |
| Task-local working state | `active_memory_pools`, observations, loaded context links | Active Memory Pools | Humans and agents need scoped RAM without corrupting durable truth. |
| Repeated execution evidence | `workflow_traces`, execution records | Workflow / Skill Runtime | Workflows should be learned from evidence-linked work. |
| Reusable procedure | `procedural_memory_objects`, `skill_packages` | Workflow / Skill Runtime | A Skill Package is validated operational knowledge, not a prompt file. |
| MCP/API/script/connector/model definitions and calls | governance tables and audit events | Tool / Model Governance | External action must be registered, permissioned, schema-checked, and logged. |
| Markdown node page, HTML page, wiki page, dashboard card, report, API response | projection/export records and generated files | Wiki / Export | These are views over engine state unless re-ingested as evidence. |
| Delivery bundle/zip for a receiver/channel | Node-local `packages/` or workspace-level package when cross-node | Wiki / Export plus owning Node | Packages are receiver/channel bundles with source links and review state. |
| Secrets/API keys | secret manager or deployment environment, referenced by connector/tool config | Governance / Deployment | Secrets do not belong in markdown, Source Packages, context, or package manifests. |

## Multi-User And Enterprise Scope

The hierarchy stays the same for a single person and for an organization:

```text
Tenant / Organization
  -> Workspace
    -> Node graph
      -> Sources, Signals, Claims, Facts, Memories, Workflows, Skills
```

In a team or enterprise deployment, every durable row should carry or inherit:

```text
tenant_id
workspace_id
node_id / scope where applicable
security_labels
partition_ids
created_by / actor_id
policy_version
audit_event_links
```

The purpose is not bureaucracy. It prevents the three common failures:

```text
one user's context leaking into another workspace
agent/tool calls acting outside their granted scope
project/package/workflow names resolving to the wrong Node
```

SQLite can exercise this model locally.

Multi-user deployments can activate PostgreSQL or an equivalent managed relational store when concurrent writes, availability, or scale require it, while preserving the same ownership rules.

The workspace policy and replication ledger make that transition explicit instead of silently changing the source of truth.

## Domain Ownership

The storage substrate does not decide meaning. Domain layers do.

| Domain layer | Owns durable lifecycle for | Writes |
| --- | --- | --- |
| Workspace / Topology | organizations, tenants, workspaces, nodes, node types, node aliases, node relationships, memberships, routing rules | topology tables and topology change requests |
| Source Intake | preserved input, source package creation, raw artifact capture | `source_packages`, raw artifact records |
| Signal Pipeline | signal classification, multimodal parsing, compatibility search rows | signal metadata, contexts, chunks, classification records |
| Memory Core | claims, facts, memory objects, evidence edges, derivation ledger, temporal state | memory core tables |
| Retrieval / Context | query plans, authorization envelope, context package assembly, stale package refresh | context packages, retrieval audit, projection links |
| Active Memory Pools | task-scoped working state for humans and agents | pools, observations, loaded context links |
| Workflow / Skill Runtime | traces, generalized workflows, procedures, skill packages, execution records | workflow and skill tables |
| Tool / Model Governance | registered tools, MCP/API/script/model calls, schemas, grants, call audit | tool/model definitions and call runs |
| Agent Collaboration | remote agent definitions, Agent Cards, delegated task runs, returned artifacts | A2A registry/projection records and audit, backed by the governance spine first |
| Wiki / Export | markdown/wiki/html/report/package projections | export records, wiki pages, projection revisions |
| Evaluation / Recovery | benchmark runs, cases, rebuild checks, health probes | evaluation and recovery records |

## Projection Surfaces

| Surface | What it shows or controls | Backing state |
| --- | --- | --- |
| Markdown workspace | node pages, rhythm, status, playbooks, package outputs | topology, memory, workflow, export records |
| Wiki pages | curated human-readable node/workspace pages with citations | wiki pages, facts, memories, source links |
| CLI | setup, initiation, topology inspection, retrieval, rendering, verification | all domain layer interfaces |
| API | app/dashboard/service access | same engine state as CLI |
| MCP/tools | governed agent access to retrieval, memory, workspace, and tools | tool governance, context packages, active pools |
| A2A agents | delegation to other agents and returned artifact capture | agent collaboration records, context packages, active pools, audit |
| Desktop/app UI | workspace switcher, node pages, memory views, runs, dashboards | API projections over engine state |
| Browser/Raycast extensions | quick capture, search, ask, add memory | API, source intake, retrieval |
| SDKs | typed integration layer for apps and agents | API contracts |
| Reports/packages | receiver-specific exported bundles | export records and source-linked projection state |

## Data Flow Across The Separation

```text
Human / agent / connector produces input
  -> Source Intake preserves evidence
  -> Signal Pipeline classifies and parses
  -> Workspace Topology scopes it to a workspace/node
  -> Memory Core extracts Claims and promotes reviewed Facts
  -> Retrieval assembles Context Packages
  -> Active Pools give task-local working state
  -> Tools/models/workflows act under governance
  -> observations re-enter as pending Claims
  -> Wiki/Export projects state to markdown/html/app/API
```

## Edit Flow

Edits from human-facing surfaces must re-enter the engine correctly.

```text
markdown edit
  -> Source Package if it changes knowledge
  -> topology_change_request if it changes workspace structure
  -> export revision if it only changes projection metadata
```

```text
app form save
  -> owning domain service
  -> canonical table write
  -> audit event
  -> projection invalidation
```

```text
agent tool result
  -> governed tool call run
  -> Source Package or observation
  -> pending Claim
  -> review/policy promotion
```

## What Must Not Happen

```text
Markdown file directly becomes truth without source/provenance.
Agent writes Facts directly instead of Claims/observations.
Vector search becomes the final authority.
Tool call output bypasses governance.
App UI invents a second data model.
Cache/index rows are treated as canonical memory.
Workspace folders become the only identity for Nodes.
Loose names route durable writes without stable ID resolution.
```

## Production Shape

For a production deployment, the intended shape is:

```text
Postgres
  -> canonical runtime tables

object/file storage
  -> preserved raw artifacts and attachments

index/cache layer
  -> FTS, vectors, chunks, parse cache, model outputs where rebuildable

optional knowledge backend
  -> RocksDB/ETS/Mnesia/Riak style graph acceleration or triple-store workloads

projection layer
  -> markdown, wiki, HTML, app views, API responses, reports, agent context
```

The important point is not which physical store is used first. The important
point is that each table, file, cache, and projection has an owner, lifecycle,
rebuild policy, and audit path.
