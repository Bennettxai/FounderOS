# Optimal Engine Roadmap

This roadmap explains what is being built and where each piece belongs.

Optimal Engine is a self-hosted runtime for a company, team, or operator's
second brain. It lets people and agents organize work into Nodes, preserve raw
sources, classify Signals, promote source-backed truth, assemble context,
operate tools, learn workflows, and project the same state into markdown, APIs,
apps, reports, and packages.

## Product Direction

```text
Human defines the world:
  Organization / tenant
    -> workspaces
      -> nodes
      -> relationships
      -> policies

Data enters the world:
  source package
    -> signal
    -> route
    -> claim
    -> fact
    -> memory object

Humans and agents use the world:
  query or task
    -> retrieval
    -> context package
    -> active memory pool
    -> action

Work improves the world:
  observation
    -> pending claim
    -> reviewed fact
    -> memory
    -> workflow
    -> skill package
```

## Naming Rules

- A Workspace is the governed boundary where work happens.
- A Project is a Node inside a Workspace.
- A Package is an outbound bundle sent to a person, team, client, partner, or
  channel.
- A Signal is classified input or event data.
- A Claim is what a source appears to assert.
- A Fact is reviewed or policy-accepted truth.
- A Memory Object is the source-backed meaning the engine should remember.
- Markdown is a projection and editing surface, not the only owner of truth.

## Build Gates

The gates are ordered by dependency. Later gates can exist as a spine before
earlier gates are complete, but production use depends on the earlier gates
being reliable.

| Gate | Goal | Current status |
| --- | --- | --- |
| 1. Workspace initiation | Let a user dump context and produce reviewed organization/workspace/node topology. | Built spine, needs more guided templates and import recipes. |
| 2. Topology ownership | Workspaces, Nodes, Node types, relationships, members, policies, and project-as-Node modeling. | Built spine with tests. |
| 3. Source-first intake | Preserve raw sources before classification or interpretation. | Built spine through Memory Core Source Packages. |
| 4. Signal classification | Parse, decompose, classify mode/genre/type/format/structure, route, and index. | Text strongest; multimodal paths exist and need deeper coverage. |
| 5. Claim/fact/memory lifecycle | Separate source assertions from reviewed truth and durable memory. | Schema and service spine built; promotion UX/process still expanding. |
| 6. Retrieval/context packages | Return governed context, not loose chunks. | Wiki-first RAG and Context Package spine built; graph/vector/temporal expansion ongoing. |
| 7. Active memory pools | Task-scoped shared workspaces for humans and agents. | Schema/spine built; more CLI/API workflow needed. |
| 8. Workflow/skill learning | Turn repeated episodes into validated procedures and skill packages. | Schema/spine built; runtime execution still expanding. |
| 9. Tool/model governance | Register tools/models, validate inputs/outputs, enforce permissions, audit calls. | Built spine with dispatcher coverage. |
| 10. Projection/export | Render markdown/wiki/API/package views from governed state. | First slice built; backlinks/import/rebuild/link health need hardening. |
| 11. Connectors and imports | Support files, existing wikis, chat, calendar, docs, repos, CRM, tickets, transcripts, and custom APIs/CLIs/MCPs. | Adapter spine exists; production connectors need auth and sync polish. |
| 12. Self-learning loops | Let users define scheduled agent loops with validation and promotion gates. | Loop docs and sample exist; scheduler/runtime needs product hardening. |

## Backend Storage Roles

```text
SQLite now:
  local/dev/default runtime store

Postgres target:
  multi-user, enterprise, hosted or team deployment

RocksDB/fast indexes:
  optional graph/cache/index acceleration, not canonical truth

Vector/embedding stores:
  rebuildable retrieval projections

Filesystem/markdown:
  human-operable projection and source editing surface

Object storage:
  large raw artifacts, audio, video, PDFs, exports
```

The rule is simple: every table or artifact must have an owner, rebuild policy,
and lifecycle state.

## User Setup Flow

```text
1. User installs engine.
2. User creates a workspace.
3. User dumps messy context.
4. Engine preserves the dump as source evidence.
5. Engine proposes Nodes, relationships, policies, and routing hints.
6. User or policy approves topology changes.
7. Engine ingests files/connectors/tools into source packages.
8. Engine classifies Signals and extracts Claims.
9. Claims are reviewed/promoted into Facts and Memory Objects.
10. Agents receive Context Packages and work inside Active Memory Pools.
11. Useful observations become pending Claims.
12. Repeated work becomes Workflow Traces and Skill Packages.
```

## What Is Not Required Now

- A dedicated frontend. External apps can use the API/CLI/MCP/tool surfaces.
- A mandatory Docker setup. Docker is optional for backend packaging.
- A single universal workflow. Each organization customizes Nodes, packages,
  loops, tools, and integrations.

## Current Production Priority

1. Keep public docs and sample workspace aligned with the backend.
2. Make CLI setup and workspace initiation usable without internal knowledge.
3. Harden source-first intake and topology review.
4. Expand connector/import documentation and examples.
5. Improve multimodal extraction and storage docs.
6. Keep audits strict so stale private/example material does not return.
