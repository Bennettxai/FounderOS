# Backend Readiness

This page answers the practical question: what is actually built in the backend,
where does data live, and what still has to be finished before calling the
system production-ready.

## Current Verdict

Optimal Engine currently has a working backend spine.

It is ready for:

```text
local CLI use
workspace setup and initiation
source-first intake
workspace/node topology
source-backed Claims, Facts, and Memory Objects
governed raw asset preservation
typed multimodal extraction projections
Context Package assembly and refresh
Active Memory Pool records
workflow and Skill Package lifecycle records
tool/model/connector governance records
wiki/export projections
reality-check verification
```

It is not yet complete for:

```text
full enterprise deployment
full Postgres production runtime
rich human review UI
full FTS/vector/graph/temporal retrieval planner
real provider-specific connector downloads for every adapter
complete multimodal provider output parser coverage
workflow mining and executable skill runtime
dedicated A2A agent registry and delegation runtime
rebuild/recovery services
benchmark result dashboards
```

The right framing is:

```text
Backend spine: real.
Production product: not finished.
Enterprise hardening: still open.
```

## Storage Roles

The backend uses multiple storage roles. They should not be confused.

| Store role | Current state | Owns |
| --- | --- | --- |
| SQLite | Current local canonical runtime store. | Workspaces, Nodes, Sources, Claims, Facts, Memories, Context Packages, pools, workflows, skills, governance rows. |
| Postgres | Target production canonical runtime store. | Same lifecycle model as SQLite, but for team/enterprise deployment. |
| Raw artifact storage | Current local file-backed evidence path; target object storage for production. | Files, uploads, attachments, media, source payloads. |
| FTS/vector/chunk indexes | Rebuildable acceleration layer. | Search candidates, embeddings, chunks, summaries, rerank state. |
| ETS | In-memory graph/knowledge runtime. | Fast process-local graph state. |
| RocksDB | Optional graph/triple-store backend, not the main product database. | Persistent graph workload experiments when native support is installed. |
| Markdown/wiki/HTML/API | Projection surfaces. | Human-readable or app-readable views, not canonical truth. |

The rule is:

```text
database rows own governed runtime state
raw artifacts own source evidence
indexes/caches speed up retrieval
markdown/wiki/API/app views project state
```

## Backend Layer Status

| Layer | Status | Proof | Main gap |
| --- | --- | --- | --- |
| Workspace / Topology | Built spine. | `WorkspaceTopology`, topology tests, reality check. | Rich app flows and topology change review UX. |
| Source Intake | Built spine. | Source Package services, asset store, connector preservation tests. | More provider-specific sync/download implementations. |
| Signal Pipeline | Built compatibility path. | Parser/classifier/indexer tests. | Cleaner Signal-to-Claim extraction policy. |
| Memory Core | Built spine. | Claims, Facts, Memory Objects, ledger, supersession tests. | Stronger evidence thresholds and review policy configuration. |
| Multimodal Assets | Built governed storage/projection spine. | assets, adapter runs, extraction tables, parser tests. | More real output schemas and runtime availability packaging. |
| Retrieval / Context | Built structured Context Package path. | Retrieval coordinator and refresh tests. | Full FTS/vector/graph/temporal/workflow planner. |
| Active Memory Pools | Built record/lifecycle spine. | Active pool probes in reality check. | Membership enforcement and richer API/UI workflows. |
| Workflow / Skill | Built lifecycle records. | Workflow/skill reality probes. | Mining, clustering, runtime execution, rollback/exception handling. |
| Tool / Model Governance | Built governed call spine. | Tool/model governance probes and connector-run tests. | Richer schema validation, grants, and output normalization. |
| A2A Agent Collaboration | Documented architecture only. | Docs and protocol split. | Dedicated registry, Agent Card storage, delegation runs, and tests. |
| Wiki / Export | Built projection spine. | Wiki scheduler/service/integrity tests. | Full package/report/html rendering and rebuild policies. |
| Evaluation | Built executable dataset/run spine. | evaluation tests and reality check. | External judges, retrieval metrics, dashboards, exports. |

## What Must Be True For A Backend Change

Every backend feature must answer:

```text
Which layer owns the lifecycle?
Which table or artifact stores canonical state?
Is this durable truth or a projection?
Can it be rebuilt?
Which scope does it inherit: tenant, workspace, Node, pool?
Which actor or service account performed it?
What audit event proves it happened?
What test or probe proves it works?
```

If those answers are unclear, the feature is not ready.

## Verification Commands

Run these before calling a backend change safe:

```bash
mix compile
mix test
mix optimal.reality_check
scripts/public-audit.sh
```

For a faster backend spine check during development:

```bash
mix compile
mix optimal.reality_check
```

For public pushes, also run:

```bash
git diff --check
scripts/public-audit.sh
```

## Current Reality Check Coverage

`mix optimal.reality_check` currently verifies:

```text
store tables and row access
workspace topology and standard Node Types
workspace-scoped Node hierarchy
typed Node relationships and membership
projection drift and edit re-entry
Source Package -> Claim -> Fact -> Memory Object
Fact supersession
Context Package assembly, authorization filtering, and refresh
Active Memory Pool stale-context refresh
Workflow Traces, procedural memory, and Skill Packages
tool/model definitions and governed call records
connector registry and governed connector runs
connector payload asset preservation
evaluation records and JSONL dataset runner
wiki round trip and integrity checks
retrieval/RAG edge cases
compliance probes
```

This is the backend safety harness. It does not replace focused tests, but it
proves the main spine is still connected.

## Setup Order For Users

The backend setup order should be:

```text
install dependencies
  -> compile
  -> run reality check
  -> create or initiate workspace
  -> inspect topology
  -> add/import sources
  -> review Claims before Facts
  -> retrieve Context Packages
  -> use agents/tools through governed surfaces
  -> refresh projections
```

CLI shape:

```bash
mix deps.get
mix compile
mix optimal.reality_check
mix optimal.setup my-workspace --name "My Workspace"
mix optimal.topology --workspace default:my-workspace
```

For messy setup:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
```

That path now applies conservative Node candidates by default and writes the
workspace projections immediately. Use `--review-only` when topology proposals
must remain pending until approval.

## Next Backend Hardening Order

The next backend work should close gaps in this order:

1. Provider-specific connector/download implementations that preserve raw
   payloads and attachments.
2. Adapter-specific multimodal output parsers and command wrappers.
3. Review policy configuration for adapter-derived Claims.
4. Retrieval planner expansion across FTS, vector, graph, temporal, workflow,
   and permissions.
5. Active Memory Pool membership enforcement and API workflows.
6. Tool/model/connector schema validation and grant hardening.
7. A2A agent registry, Agent Card storage, delegated task runs, returned
   artifact preservation, and tests.
8. Workflow mining and executable Skill Package runtime.
9. Rebuild/recovery services for projections, indexes, context packages, and
   adapter outputs.
10. Production Postgres deployment path, backups, monitoring, secret management,
    and enterprise IAM/SSO integration.

This order keeps the backend aligned with the core product: people and agents
bring messy context in, the engine preserves evidence, structures it into
workspaces and Nodes, returns governed context, and learns from reviewed work.
