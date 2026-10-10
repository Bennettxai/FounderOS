# Architecture Coverage Matrix

This matrix checks whether the important ideas we studied have a home in
Optimal Engine.

The answer is: yes, the architecture now covers the major concepts, but many of
them are still target lifecycle services rather than implemented code.

## Coverage Verdict

| Area | Covered in target architecture? | Built in code today? | Build gate |
| --- | --- | --- | --- |
| Workspace and Node topology | Yes | Partial | Topology And Workspace Surface Foundation |
| Markdown/HTML/wiki surface | Yes | Partial | Topology And Workspace Surface Foundation |
| Source Package / raw evidence | Yes | Partial | Evidence Intake |
| Multimodal data anatomy | Yes | Partial | Evidence Intake |
| Signal classification and routing | Yes | Built legacy path | Evidence Intake |
| Claims vs Facts | Yes | Schema only | Truth Lifecycle |
| Memory Objects / institutional meaning | Yes | Schema only | Truth Lifecycle |
| Episode and Memory Detail objects | Yes | Schema only | Workflow And Skill Promotion |
| Derivation Ledger / provenance | Yes | Partial | Evidence Intake and later gates |
| Relationship Edges | Yes | Schema only | Truth Lifecycle |
| Temporal validity and supersession | Yes | Fields/schema, not behavior | Truth Lifecycle |
| Confidence and precision | Yes | Fields/schema, not policy | Truth Lifecycle |
| Security-aware retrieval | Yes | Partial/post-filter path | Governed Recall |
| Retrieval Package / Context Package | Yes | Schema only / compatibility context exists | Governed Recall |
| Active Memory Pools | Yes | Schema only | Agent Work Loop |
| Tool/model governance | Yes | Schema only / connectors exist elsewhere | Agent Work Loop |
| Workflow Trace to Skill Package | Yes | Schema only | Workflow And Skill Promotion |
| Export records and projection health | Yes | Target | Topology And Workspace Surface Foundation |
| Benchmarks, evals, doctor commands | Yes | Partial/target | Evaluation And Recovery |
| Backup/rebuild of derived artifacts | Yes | Target | Evaluation And Recovery |

## Source Paper Concepts

| Concept | Optimal Engine placement | Notes |
| --- | --- | --- |
| Source evidence preservation | Source Package, Raw Artifact, Asset Store target. | Must be committed before interpretation. |
| Claim / Fact separation | ClaimExtractor and FactPromoter target services. | Prevents raw text, model output, or agent observation from becoming truth directly. |
| Memory Object | Memory Core truth lifecycle. | Stores meaning around Claims, Facts, sources, Nodes, edges, and time. |
| Episode Object | Event memory / workflow input. | Needed for incidents, calls, meetings, deployments, tasks, and repeated work. |
| Memory Detail Object | Recursive detail for steps, commands, checks, exceptions. | Needed before Workflow Trace and Skill Package runtime. |
| Derivation Ledger | Memory Core / Audit. | Every generated object needs lineage: source, processor, parser, model, actor, policy. |
| Relationship Edge | Memory Core / graph layer. | Supports, contradicts, supersedes, depends_on, derived_from, part_of, etc. |
| Bitemporal validity | Truth Lifecycle. | Track when true in the world and when accepted/recorded by the engine. |
| Confidence and precision | ScoringPolicy target. | Keep confidence separate from specificity/precision. |
| Security-aware retrieval | Retrieval Coordinator. | Permissions must apply before and during candidate expansion, not only after answer assembly. |
| Context Package | Governed recall output. | What agents/apps receive instead of loose chunks. |
| Active Memory Pool | Task working memory. | Shared human-agent task state with observations and pending Claims. |
| Workflow and Skill promotion | Workflow Trace -> Generalized Workflow -> Procedural Memory -> Skill Package. | Procedure is not a prompt; it needs sources, checks, permissions, exceptions, audit. |
| Tool/model governance | Tool/Model registry and dispatcher. | Registered tools, schema validation, grants, run records, output validation. |
| Rebuildability | Source Packages + Derivation Ledger + processor versions. | Derived outputs should be rebuildable when models/indexes change. |

## Reference-System Concepts

| Reference lesson | Adopted as | Gate |
| --- | --- | --- |
| Source-backed curated knowledge | Source Package, Claim citation checks, wiki/export integrity. | Evidence Intake / Truth Lifecycle |
| Shared team memory and living briefs | Node briefs, Context Packages, Active Memory Pools. | Governed Recall / Agent Work Loop |
| CLI/MCP brain access and eval proof | Doctor commands, MCP smoke tests, benchmark harness. | Evaluation And Recovery |
| Schema/topology packs | Workspace templates, Node Types, project/person/product/operation Nodes. | Topology And Workspace Surface Foundation |
| Graph + hybrid retrieval + gap analysis | Retrieval Coordinator and Retrieval Package. | Governed Recall |
| Markdown wiki usability | Page tree, backlinks, broken links, edit locks, revisions, import/export. | Topology And Workspace Surface Foundation |

## What Is Missing Before Build Starts

The architecture is ready enough to start implementation, but the first gate
must not skip these:

```text
workspace-scoped Node identity
node_types
node_relationships
export_records
projection_revisions
link_health_records
source-first intake commit
quarantine/rejected-source recording
raw_artifacts / asset_store
```

If those are skipped, later memory/retrieval/agent work will attach to unstable
structure.

## Build Readiness

Ready to build:

```text
Topology And Workspace Surface Foundation
Evidence Intake
```

Not ready to build first:

```text
Full app UI
Full agent automation
Skill runtime
Workflow mining
Public benchmark claims
```

Those depend on the first gates.
