# Build Goal Alignment

This page maps the current implementation to the intended Optimal Engine goals.
It is deliberately concrete: each goal names the layer, the code that exists now,
the proof we have, and the gap that still remains.

## Alignment Verdict

The current build is aligned with the target architecture as a backend spine.
It is not yet the full product.

What is real now:

```text
Workspace/Topology
  -> Source evidence
  -> Signal/Search compatibility
  -> Claim/Fact/Memory lifecycle
  -> Governed multimodal assets
  -> Adapter run records
  -> Asset extraction projection records
  -> Context Packages
  -> Active Memory Pools
  -> Workflow/Skill records
  -> Tool/Model governance
  -> Workspace export projections
  -> Reality-check probes
```

What is still not complete:

```text
full app UI
full benchmark harness
rich review queue UI/API
more adapter-specific output parsers
automatic adapter-output-to-Fact promotion
full structured + FTS + vector + graph + temporal retrieval planning
workflow mining and runtime execution
recovery/rebuild services
production deployment hardening
```

## Layer Rule

Optimal Engine is organized by lifecycle ownership. The same physical database
can hold many objects, but every object needs one owning layer.

```text
Layer = owns lifecycle decisions
Store = executes persistence
Object = durable truth or governed projection
Projection = rebuildable view for humans, agents, apps, or indexes
```

The core rule still holds:

```text
Same physical store is allowed.
Table ownership is separate.
Only the owning layer writes lifecycle state.
Other layers call through the owner.
```

## Hierarchy Rule

Projects are not peers of workspaces. Projects are Nodes inside a Workspace.

```text
Tenant
  -> Workspace
    -> Node graph
      -> Project Node
      -> Person Node
      -> Product Node
      -> Operational Node
      -> Context Node
```

A workspace is the governed boundary. A Node is a unit of organized context,
purpose, relationships, and activity inside that boundary. A folder is only a
workspace export of a Node.

## Goal Map

| Goal | Layer owner | Built now | Proof | Remaining gap |
| --- | --- | --- | --- | --- |
| Human defines the world as workspaces and Nodes. | Workspace / Topology | `WorkspaceTopology` creates workspaces, standard Node Types, Nodes, relationships, and node memberships. `optimal.initiate` preserves a messy setup dump, creates an unreviewed setup Claim, applies conservative Node candidates by default, and supports `--review-only` for governed approval. `optimal.topology approve/reject` applies or rejects pending topology changes. | `test/topology/workspace_topology_test.exs`, `test/topology/node_test.exs`, `test/topology/node_member_test.exs`, `test/topology/workspace_surface_spine_test.exs`, `test/workspace_initiation_test.exs`, `test/mix_tasks/optimal_initiate_test.exs`, `test/mix_tasks/optimal_topology_test.exs`, `mix optimal.reality_check`. | Rich UI flows for creating and editing topology, deeper policy controls, and non-CLI review workflows. |
| Projects live inside workspaces as Nodes. | Workspace / Topology | README and topology code treat `project` as a standard Node Type, not as a workspace peer. | Node type setup in `WorkspaceTopology.ensure_standard_node_types/1`; topology tests. | App navigation and docs should keep reinforcing this so the product does not drift back to project-as-root. |
| Raw evidence is preserved before interpretation. | Memory Core | Text inputs become Source Packages. File-backed assets become Source Packages plus workspace-scoped `assets` rows. `POST /api/assets` gives API callers a governed JSON upload path for local files or base64 content. `Connectors.preserve_payload_assets/4` gives connector adapters a governed attachment/file preservation path, and `Connectors.Runner` now preserves attachments/files from raw sync payloads automatically. | `MemoryCore.source_package_from_text/2`, `MemoryCore.store_asset_file/2`, `test/memory_core/asset_store_test.exs`, `test/api/router_test.exs`, `test/connectors/asset_ingest_test.exs`, `test/connectors/runner_test.exs`. | Real connector sync implementations still need provider-specific download code that returns raw payloads with attachment/file data. |
| Signal classification remains separate from truth. | Signal Pipeline | Signal modules classify and validate signals. Compatibility rows can still exist for search. | `test/signal/dispatcher_test.exs`, `test/pipeline/classify_store_test.exs`, parser/classifier tests. | The Signal-to-Claim bridge needs richer extraction policies and clearer routing hooks. |
| Extracted assertions do not become truth automatically. | Memory Core | Claims are created first; promotion creates Facts through `ClaimReview` and `FactPromoter`. The review queue service/API returns counts and filterable Claim rows for app and agent review workflows. Promotion blocks source-less/stale Claims by default and blocks contradictions unless the reviewer explicitly supersedes a current Fact. | `test/memory_core/spine_test.exs`, `test/memory_core/claim_review_test.exs`, `test/api/router_test.exs`, reality-check evidence lifecycle probes. | Adapter-specific review policies and stronger evidence thresholds for extracted Claims. |
| Accepted knowledge becomes source-backed memory. | Memory Core | Facts can become Memory Objects with links to evidence, Claims, and derivation. Fact supersession marks the replaced Fact and its linked Memory Objects as superseded, marks affected Context Packages stale, writes a `supersedes` Relationship Edge, and records the replacement in the Derivation Ledger. Stale Context Packages can be refreshed individually, in batches, through the public API route, by a CLI/cron task, or through a supervised scheduler. | `MemoryCore.build_memory_object/2`, `MemoryCore.promote_claim/2`, `MemoryCore.refresh_context_package/2`, `MemoryCore.refresh_stale_context_packages/1`, `ContextRefreshScheduler.run_once/1`, `POST /api/memory-core/context-packages/refresh-stale`, `mix optimal.context.refresh_stale`, `test/memory_core/context_refresh_scheduler_test.exs`, `test/memory_core/spine_test.exs`, `test/memory_core/claim_review_test.exs`, `test/api/router_test.exs`, reality-check memory/recall probes. | Per-workspace scheduler policy, metrics, and backoff controls. |
| Multimodal inputs are first-class evidence. | Memory Core / Pipeline | Parser-produced assets are preserved through `AssetStore`; chunks can carry `asset_ref`; indexer passes workspace scope; API asset upload can preserve a raw file and optionally run a multimodal adapter in the same request; connector attachment payloads can be preserved directly or automatically from raw sync payloads through the runner. | `test/pipeline/pipeline_asset_store_test.exs`, `test/pipeline/indexer_asset_store_test.exs`, `test/api/router_test.exs`, `test/connectors/asset_ingest_test.exs`, `test/connectors/runner_test.exs`, asset store tests. | Real connector adapters need attachment download implementations plus adapter-specific extraction runs. |
| Open-source multimodal adapters are planned and governed. | Pipeline / Model Governance | `MultimodalToolRegistry` catalogs local-first adapter targets for documents, OCR, audio, video, visual reasoning, visual retrieval, and cross-modal embeddings. Each adapter now has a profile for primary role, output formats, claimability, default runtime arguments, and install profile. | `test/pipeline/multimodal_tool_registry_test.exs`, `docs/reference/multimodal-open-source-stack.md`. | Runtime availability checks and deployment packaging per adapter. |
| Adapter execution is recorded instead of invisible. | Memory Core / Pipeline | `MultimodalAdapterRunner` executes configured local commands and records completed, failed, or unavailable runs in `asset_adapter_runs`. Supported completed runs auto-project output into typed extraction tables unless disabled. Runtime defaults now come from adapter profiles instead of hardcoded runner branches. | `test/pipeline/multimodal_adapter_runner_test.exs`, migration 036, `MemoryCore.run_asset_adapter/3`. | More real-tool output parsers and provider-specific command wrappers. |
| Adapter outputs have typed projection storage. | Memory Core | `MemoryCore.record_asset_extraction/2` writes `asset_extractions` plus typed transcript, OCR span, visual observation, and embedding-ref projection rows linked to assets, adapter runs, scopes, hashes, and derivation ledger. `MultimodalExtractionParser` now splits structured transcript JSON into segment rows, document JSON into page/element/table OCR rows, and frame JSON into visual observations and object detections. | migration 037, `test/memory_core/asset_store_test.exs`, `test/pipeline/multimodal_adapter_runner_test.exs`, `mix optimal.reality_check` table probes. | More real provider output schemas and adapter-specific command builders. |
| Completed adapter output can become reviewable knowledge. | Memory Core | `MemoryCore.claim_from_asset_adapter_run/2` and `MemoryCore.claim_from_asset_extraction/2` turn text-bearing completed outputs into derived Source Packages and pending Claims. Failed/unavailable adapter runs and reference-only extractions cannot become Claims. | `test/memory_core/asset_store_test.exs`. | Review policies that decide when adapter-derived Claims become Facts. |
| Humans and agents receive Context Packages, not random chunks. | Retrieval / Context | `RetrievalCoordinator` returns and stores `context_packages` with Facts, Memory Objects, asset extraction projections, evidence links, confidence/precision summaries, authorization envelope, and explicit retrieval-plan metadata. It now applies structured subject/action/object filters for Facts and Memory Objects plus modality/extraction-type filters for asset projections. Stale packages can be refreshed from their original query, authorization envelope, actor, time mode, detail depth, and limit; batch/API/CLI/scheduler refresh marks old packages `refreshed` after replacement so stale queues do not repeat the same work. | `test/memory_core/context_refresh_scheduler_test.exs`, `test/memory_core/spine_test.exs`, `test/memory_core/claim_review_test.exs`, `test/api/router_test.exs`, reality-check retrieval/context probes. | Full planner across FTS, vector search, graph traversal, temporal validity, workflows, and deeper permissions. |
| Agents work in task-scoped pools. | Active Collaboration | `ActiveMemoryPool` opens task pools, loads Context Packages, refreshes stale loaded Context Packages through the Retrieval Coordinator, publishes observations as Source Packages and pending Claims, and closes pools. | Reality-check active pool probes, `test/memory_core/spine_test.exs`, and MemoryCore delegates. | Pool membership enforcement, scheduled refresh workflows, and UI/API surfaces. |
| Repeated work can become workflows and skills. | Workflow / Skill Runtime | First lifecycle records workflow traces, generalized workflows, procedural memory objects, and skill packages. | `MemoryCore.capture_workflow_trace/2`, `generalize_workflow/2`, `create_procedural_memory/2`, `package_skill/2`, reality-check workflow probes. | Real workflow mining, clustering, review gates, skill execution runtime, and exception/rollback handling. |
| Tool and model calls are governed. | Model / Tool Governance | Tool/model definitions and calls can enforce privileges, partitions, required inputs/outputs, and audit links. Connector runs use governed execution by default, while raw sync requires an explicit `governed: false` bypass. | `test/connectors/runner_test.exs`, reality-check governance probes. | Richer schema validation/output normalization and policy-specific grants. |
| Markdown/files are projections, not the only truth. | Workspace Export | Workspace export records projections and can re-ingest edits as evidence. | `test/workspace_export_test.exs`, README storage model. | Full app/page rendering, HTML/report generation, and projection invalidation/rebuild policies. |
| Benchmarks are inspectable, not just screenshots. | Evaluation / Audit | `Evaluation.run_benchmark/2` creates executable benchmark runs over governed retrieval, assembles Context Packages per question, records retrieved object links and Context Package IDs, applies deterministic expected-answer judging by default, accepts external answerer/judge callbacks, persists each case, and updates aggregate run scores. `Evaluation.load_dataset/2`, `Evaluation.run_dataset/2`, and `mix optimal.eval.run` load JSON/JSONL benchmark cases from disk and execute them through the same path. | `OptimalEngine.Evaluation`, `Mix.Tasks.Optimal.Eval.Run`, `test/evaluation_test.exs`, migration 038, and `mix optimal.reality_check` currently report 125 OK probes including dataset-runner execution. | External judge/model execution adapters, retrieval metrics beyond deterministic scoring, scoped result exports, and result dashboards. |
| Public repo stays clean. | Governance | Public README/docs avoid non-public project material and local progress artifacts stay untracked. | Current public worktree status and committed file scope. | Continue auditing generated docs before public pushes. |

## Current Data Flow

The current backend flow is:

```text
Human/app/CLI/agent/connector input
  -> workspace and Node scope
  -> Source Package
  -> Signal/Search compatibility path
  -> Claim candidate
  -> reviewed Fact
  -> Memory Object
  -> Relationship Edge / Derivation Ledger
  -> Retrieval Coordinator
  -> Context Package
  -> Active Memory Pool
  -> observation or action result
  -> pending Claim
  -> reviewed memory update
```

File-backed multimodal flow:

```text
file
  -> parser asset
  -> API asset upload when entering through HTTP
  -> connector runner preserves raw sync payload attachments/files
  -> MemoryCore.AssetStore
  -> Source Package + assets row + derivation ledger
  -> optional adapter run
  -> asset_adapter_runs
  -> asset_extractions
  -> typed transcript / OCR span / table / visual observation / object detection / embedding ref projection
  -> retrieval as source-linked Context Package evidence
  -> text-bearing extraction becomes derived Source Package
  -> pending Claim
  -> reviewed Fact only after review or policy acceptance
```

## What This Means For Build Order

The next work should not add random features. It should close the gaps in the
same lifecycle order:

```text
1. Keep preserving sources and assets correctly across parser, API, connector runner, and provider-specific sync surfaces.
2. Expand provider-specific adapter parsers and command wrappers that populate richer extraction projection rows.
3. Improve adapter-specific review policy for adapter-output Claims.
4. Expand Context Package scheduler policy, metrics, and backoff now that automatic
   refresh exists.
5. Expand retrieval planning beyond current structured SQL-like filters into FTS,
   vector, graph, temporal, workflow, and deeper permissions.
6. Strengthen Active Memory Pools and agent/tool governance.
7. Add external judge/model execution adapters, richer retrieval metrics, result
   exports, and dashboards on top of the executable evaluation runner and dataset
   loader.
8. Build app/HTML/report surfaces from governed state.
9. Add rebuild/recovery services for derived projections.
```

This keeps complexity meaningful. The engine should be complex around evidence,
ownership, validity, permissions, workflows, and audit. It should stay simple
around pass-through wrappers, duplicate truth, and unowned state.
