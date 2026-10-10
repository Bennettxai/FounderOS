# Reality Audit

This audit compares the target Optimal Engine architecture with the current
codebase reality.

The architecture direction is correct, but the current implementation is still
transitional. The biggest risk is not that the model is too complex. The risk is
that old compatibility paths keep acting as truth after the new layer model says
they should only be projections.

## Short Verdict

The right target architecture is:

```text
Workspace / Topology
  -> Source Intake
  -> Signal Pipeline
  -> Memory Core
  -> Retrieval / Context
  -> Active Collaboration
  -> Tool / Model Governance
  -> Workflow / Skill Runtime
  -> Workspace Export / Projection
  -> Audit / Governance
```

The current codebase has pieces of this, but not the full lifecycle.

What is real now:

- Topology has `workspaces`, `workspace_members`, `nodes`, `node_members`, and
  skill-related tables.
- Intake creates Signals, writes workspace files, inserts compatibility Context
  rows, and records a Source Package plus Derivation Ledger entry.
- Memory Core has the schema spine for sources, claims, facts, memories,
  relationships, workflows, skills, active pools, context packages, tool
  definitions, and model-call operations.
- Source Package and Derivation Ledger writes are implemented.

What is not real yet:

- Claims and Facts are schema, not lifecycle.
- Memory Objects are schema, not lifecycle.
- Context Packages are schema, not governed retrieval output.
- Active Memory Pools are schema, not task-state lifecycle.
- Skill Packages are schema, not governed procedure runtime.
- Workspace exports still behave like a major source of user-facing truth.

## Code Reality Matrix

This is the current implementation status by layer.

| Layer | Code proves today | What the diagram must not imply yet |
| --- | --- | --- |
| Workspace | `OptimalEngine.Workspace` can create and list workspaces, persist workspace members, and provision workspace folders. | Workspace creation is not yet a complete topology/export transaction; filesystem provision failure can be non-fatal after the workspace row exists. |
| Topology / Nodes | `nodes` table exists, `workspace_id` exists in the migration, and `OptimalEngine.Topology.Node` supports tenant-scoped node CRUD and hierarchy. | Node ownership is not yet enforced through `workspace_id`; there is no first-class `node_types`, `node_relationships`, or topology change lifecycle. |
| Signal Pipeline | `Pipeline.Intake` classifies raw input into Signals, routes it, writes workspace files, and inserts compatibility Context rows. | Intake is not yet source-first. Source evidence is recorded after the old pipeline succeeds, not before interpretation. |
| Memory Core | `source_packages` and `derivation_ledger` writes exist through `MemoryCore.SourcePackageService` and `MemoryCore.Store`. Migration 032 creates the full schema spine. | Claims, Facts, Memory Objects, Details, Relationships, Pools, Context Packages, Workflows, Skills, model calls, and tool definitions are mostly schema-only. |
| Retrieval | Search, RAG/context assembly, wiki/export projection lookup, and principal post-filter tests exist. | This audit predates the full governed-recall buildout; verify current Retrieval Package / Context Package behavior against `docs/reference/backend-readiness.md` and `docs/reference/build-goal-alignment.md`. |
| Wiki / Export | `Wiki.Store` persists versioned pages and citations. `Wiki.Integrity` checks citations, directives, claim density, size, and contradictions. | The newer Workspace Export lifecycle is not built: export records, projection revisions, backlinks, broken-link records, projection drift, import lineage, and edit collision handling are not first-class yet. |
| Active Collaboration | `active_memory_pools` table exists. | There is no pool lifecycle service for opening pools, loading context, publishing observations, proposing Claims, or closing pools. |
| Workflow / Skill | Workflow and Skill tables exist in the schema spine. | There is no evidence-linked workflow extraction, generalized workflow validation, or governed Skill Package runtime yet. |
| Tool / Model Governance | Definition tables exist. Some connector/API code exists elsewhere. | Tool and model calls are not yet forced through a registered, permissioned, schema-validated dispatcher with run records and output validation. |
| Proof Layer | Focused tests exist for topology nodes, retrieval principal filtering, wiki integrity, and the Memory Core spine. | The tests do not yet prove the full target lifecycle from source-first intake through Claim/Fact/Memory, governed Context Packages, Active Pools, Skills, and projection freshness. |

## Critical Gap 1: Node Ownership Is Not Workspace-Enforced Yet

Target architecture says:

```text
Workspace
  -> Nodes
  -> Node Types
  -> Node Relationships
```

Current schema reality:

```text
nodes
  -> tenant_id
  -> workspace_id added by migration 026
  -> slug
  -> kind
  -> parent_id
```

Current interface reality:

```text
Topology.Node
  -> struct has tenant_id but no workspace_id
  -> natural key is tenant_id + slug
  -> deterministic id is tenant_id:slug
  -> reads filter by tenant_id, not workspace_id
```

So the database has a `workspace_id` column after migration, but the topology
module does not yet make Workspace the ownership interface for Nodes. That is
the biggest structural mismatch.

Why it matters:

- two workspaces in the same tenant still cannot safely have independent node
  maps through the current topology interface;
- routing can confuse workspace-local Nodes;
- permissions and membership become harder to reason about;
- exports cannot reliably prove which workspace owns a Node;
- retrieval scope cannot depend on a clean Workspace -> Node relationship.

Correction:

```text
Add workspace_id to the Node struct and Topology.Node interface.
Make the Node natural key workspace_id + slug.
Make the deterministic id workspace_id:slug or another workspace-scoped id.
Read/list/children/ancestors by workspace_id.
Add node_relationships or equivalent first-class relationship table.
Keep parent_id as a convenience hierarchy, not the whole relationship model.
```

Until this is fixed, the Topology Foundation gate is not complete.

## Critical Gap 2: Source Evidence Is Not Guaranteed First

Target architecture says:

```text
Raw input
  -> Source Package
  -> Signal
  -> route
  -> Context projection
  -> Claim / Fact / Memory
```

Current intake reality:

```text
raw_text
  -> build SourcePackage struct in memory
  -> classify
  -> route
  -> write files
  -> insert Context row
  -> record SourcePackage + DerivationLedger
```

The source package is built first, but persisted after the pipeline succeeds.
If classification rejects the input or the pipeline fails before provenance
write, the raw source may not be preserved.

Why it matters:

- rejected or quarantined inputs may disappear instead of becoming governed
  evidence;
- generated files or Context rows may exist without durable source provenance;
- provenance write failure currently logs a warning and continues;
- the source-to-signal/context write is not guaranteed atomic with the old
  compatibility writes.

Correction:

```text
Persist Source Package before classification.
Record intake attempt ledger entry even when classification rejects.
Treat Signal, Context row, file export, and ledger output as one lifecycle
operation where possible.
If full atomicity across files and DB is impossible, write an export record and
repair/invalidation path.
```

Until this is fixed, Evidence Intake is useful but not yet source-first enough.

## Critical Gap 3: Context Is Still Carrying Too Much Meaning

Target architecture says:

```text
Context row = compatibility/search projection
Context Package = governed actor/task recall package
Memory Object = durable interpreted memory
Fact = accepted truth
```

Current code reality:

- `OptimalEngine.Store` documentation still describes `contexts` as the
  canonical table.
- `Pipeline.Intake` indexes the signal into `contexts`.
- Retrieval mostly reads compatibility Context rows.
- Context Package tables exist, but the governed assembler is not built.

Why it matters:

- old code can keep treating Context as truth;
- search results can look like memory even when they are only projections;
- AI context can still become loose chunks instead of governed packages;
- it becomes unclear where accepted truth lives.

Correction:

```text
Keep contexts as compatibility/search projection.
Add explicit ContextStore or compatibility-store naming.
Build RetrievalPackage and ContextPackage structs/services.
Update Store docs so contexts are no longer described as canonical truth.
```

Until this is fixed, the old context engine can keep pulling the system
backward.

## Critical Gap 4: The Truth Lifecycle Is Schema-Only

Migration 032 creates:

```text
claims
facts
memory_objects
memory_detail_objects
relationship_edges
```

But the code does not yet have:

```text
ClaimExtractor
FactPromoter
MemoryObjectBuilder
RelationshipEdgeBuilder
TemporalSupersession
ScoringPolicy
```

Why it matters:

- the system can store the tables but not enforce the lifecycle;
- anything can write rows unless ownership is enforced;
- confidence and precision can exist as fields without real calibration;
- valid-time and transaction-time can exist as fields without real behavior.

Correction:

Build the smallest source-backed truth loop:

```text
Source Package
  -> Claim
  -> Fact
  -> Memory Object
  -> Relationship Edge
  -> Derivation Ledger
```

This should happen before the system claims to have real governed memory.

## Critical Gap 5: Retrieval Is Not Yet Governed Recall

Target architecture says:

```text
query
  -> authorization envelope
  -> retrieval plan
  -> candidate generation
  -> source/freshness/policy validation
  -> Retrieval Package
  -> Context Package
```

Current code reality:

- Retrieval has useful search, RAG, wiki/export projection lookup, intent, and context assembly
  modules.
- These are not yet the same as governed retrieval packages.
- Permission-aware candidate generation and graph expansion are not the central
  retrieval contract yet.

Why it matters:

- search can return useful results while still being unsafe or ungrounded;
- agents may receive snippets instead of governed Context Packages;
- freshness, validity, and filtered-object accounting may be missing;
- retrieval can leak through edges, summaries, or citations if authorization is
  applied too late.

Correction:

```text
Build RetrievalCoordinator.
Build RetrievalPackage.
Build ContextPackageAssembler.
Apply permissions before candidate expansion, not only after answer assembly.
Record retrieval audit.
```

## Critical Gap 6: Workspace Export Needs Export Records And Wiki Mechanics

Target architecture says:

```text
Engine state -> projection -> markdown / HTML / app view
Human edit -> source package or topology change request
```

Current reality:

- workspace files are still central to the operating experience;
- generated architecture HTML exists;
- node folders and signal files are written directly;
- export records are not yet the hard proof tying each projection to the engine
  object/version that produced it.

Why it matters:

- files and database can drift;
- a human can edit a projection and the engine may not know what semantic
  lifecycle should handle it;
- generated HTML can become a second truth if it is not tied to source objects.
- a weak wiki/export layer makes the system feel unusable even if the backend is
  correct;
- broken links, stale backlinks, rename drift, and concurrent edits can corrupt
  the human operating surface;
- import from existing Markdown/Obsidian-style folders can lose lineage if it is
  treated as a blind file copy instead of source intake.

Correction:

```text
Add export_records.
Record object links, object versions, source hashes, output path, export time,
export policy, and rebuild command.
Re-ingest human edits as Source Packages or topology change requests.
Add link health checks for markdown and HTML projections.
Track backlinks and broken links for Node pages.
Add revision records for user-facing projections.
Detect projection drift between files and engine object versions.
Handle edit collisions with optimistic locking or equivalent version checks.
Handle rename/move link refactor or create repair work.
Import Markdown folders as Source Packages with folder/tree lineage.
```

The export/interface layer should be treated as a real lifecycle, not a dump
directory:

```text
render projection
  -> record export
  -> check page tree
  -> check links/backlinks
  -> serve/edit/import
  -> capture edit as source or topology change request
  -> review/promote
  -> refresh projections
```

## Critical Gap 7: Active Memory Pools Are Not Runtime Yet

Target architecture says:

```text
Active Memory Pool
  -> loaded Context Package
  -> humans / agents / tools
  -> observations
  -> pending Claims
  -> promotion / rejection
```

Current reality:

- `active_memory_pools` table exists.
- There is not yet a complete pool lifecycle service.

Why it matters:

- agent work still risks living in chat/session state;
- tool results can be hard to promote safely;
- multi-agent or human-agent collaboration has no governed task memory.

Correction:

Build:

```text
ActiveMemoryPool.open/2
ActiveMemoryPool.load_context/2
ActiveMemoryPool.publish_observation/3
ActiveMemoryPool.propose_claim/3
ActiveMemoryPool.close/2
```

## Critical Gap 8: Tool And Model Calls Are Not Fully Governed

Target architecture says agents should call registered, permissioned,
schema-validated tools and models.

Current reality:

- connector and API code exists;
- target tables for tool definitions and model-call operations exist;
- the full governed dispatcher is not the central path yet.

Why it matters:

- agents can act without enough policy accounting;
- outputs can enter the system without source/observation classification;
- tool and model behavior cannot be replayed or audited consistently.

Correction:

```text
ToolRegistry
ToolDispatcher
ModelCallOperation
ModelCallRun
ToolCallRun
OutputValidator
ObservationWriter
```

## Critical Gap 9: The Proof Layer Exists, But It Proves The Old Shape

The codebase already has useful verification infrastructure:

```text
mix optimal.reality_check
mix optimal.search
mix optimal.rag
mix optimal.ingest_workspace
mix optimal.topology
wiki integrity tests
wiki contradiction tests
retrieval principal-filter tests
ACL / compliance / DSAR / retention tests
Memory Core spine tests
```

This is good. It means the engine is not starting from zero.

But most of this proof still validates the compatibility architecture:

```text
contexts
wiki_pages
citations
events
legacy node strings
RAG envelopes
post-filtered search
workspace ingest into Context rows
```

The new architecture needs proof for the Memory Core lifecycle:

```text
Source Package committed before classification
Signal linked to source and route
Claim extracted from source span
Fact promoted from accepted evidence
Memory Object built from Claims/Facts
Relationship Edge created with evidence
Retrieval Package assembled with authorization, freshness, and source links
Context Package stored with filtered-object accounting
Active Memory Pool records observations and pending Claims
Tool/model run creates output object, audit event, and optional Source Package
Export record proves markdown/HTML projection freshness
Wiki/export link health proves page trees, backlinks, broken links, revisions,
imports, and edit collisions are handled
```

Why it matters:

- the existing proof layer can pass while the new object lifecycle is still
  schema-only;
- retrieval can be useful while not yet returning governed packages;
- wiki citation checks can catch unsupported pages while Claims/Facts remain
  unimplemented;
- reality checks can count tables without verifying lifecycle ownership.

Correction:

```text
Extend mix optimal.reality_check with Memory Core probes.
Add mix optimal.doctor for install/runtime wiring.
Add mix optimal.benchmark or mix optimal.eval for retrieval/grounding scorecards.
Add source-first intake regression tests.
Add workspace-topology lifecycle tests.
Add projection drift tests for markdown/HTML exports.
Add wiki/export health tests for tree rendering, backlinks, broken links,
revision records, import lineage, rename/move drift, and edit collisions.
Add MCP/tool smoke tests once tool dispatch exists.
```

Until this is fixed, the architecture has partial proof, but not proof of the
new operating model.

## What Is Actually Missing

The missing piece is not another object name.

The missing piece is a **commit boundary** for each lifecycle.

Right now the architecture names the right objects, but the code still needs
clear transaction-like boundaries:

```text
Topology commit:
  workspace/node/relationship/member/export/audit

Intake commit:
  source/signal/context/file-export/ledger/audit

Truth commit:
  claim/fact/memory/edge/ledger/audit

Recall commit:
  retrieval-plan/context-package/filtering/audit

Agent commit:
  pool/tool-run/observation/pending-claim/audit

Export commit:
  generated file/export-record/source links/rebuild metadata/link health/edit safety
```

Each commit boundary should answer:

```text
What changed?
Who owns it?
What source/evidence supports it?
What policy allowed it?
What projections must refresh?
What happens if the operation partially fails?
```

Without commit boundaries, the layers can be named correctly but still behave
like a pile of writes.

## Build Order Correction

The next real build should not start with the app or the full agent runtime.

It should start with two foundation fixes:

1. **Topology Foundation correction**

```text
workspace-scoped Nodes
node relationships
export records
topology change requests
```

2. **Source-first Intake correction**

```text
persist Source Package before classification
record rejected/quarantined inputs
ledger every output
tie file exports to export records
```

Then build:

```text
ClaimExtractor
FactPromoter
MemoryObjectBuilder
RetrievalPackage
ContextPackageAssembler
ActiveMemoryPool lifecycle
```

## Final Reality Check

The architecture is right if we enforce the following:

```text
Workspace owns Nodes.
Store does not own meaning.
Context is projection.
Source Package comes before interpretation.
Claim is not Fact.
Agent observation is not Fact.
Context Package is not Memory Object.
Export is not truth.
Every generated object has lineage.
Every lifecycle has a commit boundary.
```

If any of those are false in code, the implementation has drifted from the
architecture.
