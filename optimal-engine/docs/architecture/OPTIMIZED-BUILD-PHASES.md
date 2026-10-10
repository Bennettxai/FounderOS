# Dependency Build Sequence

This document defines the build order for Optimal Engine.

For how outside reference systems are absorbed into this build sequence, see
[`REFERENCE-ADOPTION-BUILD-PLAN.md`](REFERENCE-ADOPTION-BUILD-PLAN.md).

The important point is not the number of phases. The build should have exactly
as many gates as the dependency structure requires. Complexity is acceptable
when it has a job. The risk is not complexity; the risk is building surfaces,
tables, or agents before the lifecycle underneath them exists.

## Build Principle

Use dependency gates instead of an arbitrary phase count.

```text
Build the smallest complete loop that proves the next dependency.
Do not split meaningful lifecycle steps just to make the plan look smaller.
Do not merge lifecycle steps when merging hides a risk.
```

Each gate must prove:

```text
human action
agent action or agent-readable interface
stored record
retrieval or export result
audit or validation record
failure/repair behavior
```

If a gate cannot prove those, it is not functional yet.

## Layer vs Gate

Do not confuse layers with gates.

```text
Layer = owns a lifecycle and decides what an object means.
Gate = proves a working dependency across layers.
Surface = how a human or agent uses the engine.
Projection = rebuildable view of governed state.
```

Examples:

| Term | Meaning | Example |
| --- | --- | --- |
| Layer | Responsibility owner. | Workspace / Topology owns Nodes. |
| Gate | Build proof. | Topology Foundation proves create, link, export, re-ingest, and audit. |
| Surface | Control or viewing interface. | Markdown, HTML, CLI, app, API, agent, tool call. |
| Projection | Generated view. | Node markdown, HTML page, dashboard card, context snippet. |

## What We Build First

The first gate is **Topology And Workspace Surface Foundation**.

This comes first because every later subsystem needs a place to attach:

```text
Sources need Node scope.
Signals need routing targets.
Claims and Facts need subject anchors.
Retrieval needs workspace and permission boundaries.
Context Packages need task and Node scope.
Active Memory Pools need members and allowed tools.
Workflows and Skills need applicability conditions.
Exports need a canonical object/version behind each file.
Humans need a visible workspace surface they can operate.
```

So the first proof is:

```text
Human creates workspace.
Human creates Nodes.
Human marks one Node as a project Node.
Human links Nodes.
Engine exports a markdown or HTML Node page.
Engine records the export object versions.
Engine checks page tree, links, backlinks, and broken links.
Human edits the projection.
Engine re-ingests the edit as a Source Package or governed topology change.
Engine records audit.
```

What we do not build first:

```text
not the full app
not all skills
not all domain-specific tables
not unreviewed agent truth
not a replacement for markdown
```

The goal is to make the object boundary real before memory, retrieval, workflow,
and agent automation depend on it.

## Gate: Topology Foundation

Build:

```text
Workspace
  -> Node
  -> Node Type
  -> Node Relationship
  -> Node Member
  -> Export Record
  -> Page Tree / Link Health
  -> Projection Revision
```

This proves that the user can define the world and see it as markdown or HTML
without letting the file surface become separate truth.

Completion proof:

```text
Create a workspace.
Create Nodes.
Mark one Node as a project Node.
Link Nodes.
Export a markdown or HTML page.
Record the object versions behind the export.
Check backlinks and broken links.
Detect stale projection drift.
Re-ingest an edit as a source or governed topology change.
Preserve a projection revision.
```

Do not wait until the end to build this. The workspace surface is not visual
polish. It is how humans operate the system, so it must be proven early.

## Gate: Evidence Intake

Build:

```text
Command or raw input
  -> Scope Envelope
       actor
       workspace
       operation class
       node hints
       permissions
  -> Source Package
  -> Signal
  -> route to Node
  -> compatibility Context row
  -> Derivation Ledger entry
```

This proves that the engine preserves raw evidence and keeps Optimal Engine's
signal-first routing advantage.

Completion proof:

```text
Add a note or file.
Resolve the best-known workspace/actor scope.
Store it as a Source Package.
Classify it as a Signal.
Route it to a Node.
Create the compatibility search row.
Record derivation.
Reject or quarantine a low-quality input without losing the Source Package.
```

## Gate: Truth Lifecycle

Build:

```text
Signal / Source Package
  -> Claim
  -> evidence review
  -> Fact
  -> Memory Object
  -> temporal state
  -> confidence and precision
```

This proves that the engine separates what was said from what is accepted as
true and useful.

Completion proof:

```text
Extract a Claim from a source.
Promote one Fact through policy or review.
Create one Memory Object with source links.
Record temporal validity, confidence, precision, and ledger entries.
```

## Gate: Governed Recall

Build:

```text
Retrieval Coordinator
  -> authorization envelope
  -> retrieval plan
  -> Retrieval Package
  -> Context Package
```

This proves that humans and agents can receive scoped, permitted, source-backed
context.

Completion proof:

```text
Ask a Node/project question.
Return a Context Package with Facts, Memory Objects, source links, freshness,
permissions, and filtered-object summary.
```

## Gate: Agent Work Loop

Build:

```text
Active Memory Pool
  -> loaded Context Package
  -> humans / agents / tools
  -> observations
  -> pending Claims
  -> minimal tool/model registry
```

This proves that agents can work inside a governed task context without silently
writing truth.

Completion proof:

```text
Agent loads context.
Agent calls a registered tool.
Tool output becomes an observation.
Observation becomes a pending Claim.
Human or policy promotes or rejects it.
Audit records the run.
```

## Gate: Workflow And Skill Promotion

Build:

```text
Episode / Memory Detail
  -> Workflow Trace
  -> Generalized Workflow
  -> Procedural Memory Object
  -> Skill Package
```

This proves that repeated work can become reusable procedure.

Completion proof:

```text
One repeated task becomes a Workflow Trace.
The trace becomes a reviewed Skill Package.
The Skill Package has source links, validation checks, permissions, exceptions,
and audit.
```

## Gate: Product Interfaces And Evaluation

Build:

```text
markdown projection
HTML projection
dashboard / app view
workflow runner
benchmark suite
security and recovery checks
link health checks
projection drift checks
import/export lineage checks
edit collision checks
```

This proves that product surfaces can scale around real engine objects instead
of creating a second data model.

Completion proof:

```text
The same Node, Memory Object, Context Package, and Skill Package appear in
markdown, HTML, app/API output, and agent context.
The markdown/HTML surface has page tree, backlinks, broken-link status,
revision history, import lineage, and projection freshness.
Benchmarks check retrieval, grounding, temporal behavior, security, workflow,
and recovery.
```

## Capability Preservation Matrix

Nothing in this dependency sequence is allowed to remove critical capability.
The sequence is wrong if any row below disappears.

| Capability | Why it is critical | Build gate | Do not cut |
| --- | --- | --- | --- |
| Workspace boundary | Controls scope, permissions, exports, and operating context. | Topology Foundation | Do not let Nodes exist without a Workspace. |
| Universal Node model | Prevents parallel hierarchies for projects, people, products, operations, and rhythm. | Topology Foundation | Do not create peer top-level tables before trying Node Types and relationships. |
| Node relationships | Gives the system structure without forcing everything into a tree. | Topology Foundation | Do not model relationships only as markdown links. |
| Workspace export | Keeps the system markdown-operable and inspectable. | Topology Foundation and Product Interfaces | Do not make the database invisible to humans. |
| Wiki/export mechanics | Makes the human surface reliable: tree, backlinks, broken links, revisions, import, and edit safety. | Topology Foundation and Product Interfaces | Do not treat generated markdown or HTML as a dumb dump directory. |
| Source Package | Preserves raw evidence for every derived object. | Evidence Intake | Do not allow Claim, Fact, Memory, or Skill creation without source lineage. |
| Signal classification | Keeps Optimal Engine's signal-first advantage. | Evidence Intake | Do not reduce intake to generic document ingestion. |
| Claim / Fact split | Prevents extracted text or agent output from becoming truth too early. | Truth Lifecycle | Do not let summaries, search hits, or observations become Facts automatically. |
| Derivation Ledger | Makes generated state rebuildable and auditable. | Evidence Intake and later gates | Do not create derived objects without recording how they were produced. |
| Memory Object | Captures institutional meaning, not just atomic facts. | Truth Lifecycle | Do not leave memory as loose Context rows only. |
| Temporal validity | Distinguishes historical truth from current usability. | Truth Lifecycle and Governed Recall | Do not return stale operational guidance as current context. |
| Confidence and precision | Separates reliability from specificity. | Truth Lifecycle and Governed Recall | Do not collapse quality into one vague score. |
| Permission-aware retrieval | Prevents leaks through search, graph traversal, summaries, and citations. | Governed Recall | Do not apply authorization only after candidate retrieval. |
| Retrieval Package | Keeps recall inspectable and source-linked. | Governed Recall | Do not return bare ranked chunks as the final answer. |
| Context Package | Gives humans and agents task-ready context without hiding policy state. | Governed Recall | Do not send loose snippets directly into agents as final context. |
| Active Memory Pool | Provides shared task working memory for humans and agents. | Agent Work Loop | Do not use untracked chat/session state as the work context. |
| Observation / pending Claim loop | Lets agents learn without poisoning durable truth. | Agent Work Loop | Do not let tool output become Fact without promotion. |
| Tool / model governance | Controls what agents and models can do. | Agent Work Loop and Product Interfaces | Do not give agents an ungoverned tool universe. |
| Workflow Trace | Captures repeated work from real episodes. | Workflow And Skill Promotion | Do not invent workflows from prompts alone. |
| Skill Package | Turns validated procedure into reusable operational capability. | Workflow And Skill Promotion | Do not treat a skill as only a prompt file. |
| App / dashboard / HTML projections | Makes the engine usable through interfaces. | Product Interfaces | Do not let generated pages become separate truth. |
| Benchmarks and audit | Proves the system works and remains safe. | Governed Recall and Product Interfaces | Do not rely on demos without measured recall, grounding, security, and workflow checks. |

## First Vertical Slice

The first real proof should be:

```text
Human creates a project Node.
Engine exports Node page with export record and link health.
Human adds meeting notes.
Engine stores Source Package.
Engine classifies Signal.
Engine extracts Claim.
Human promotes Fact.
Engine creates Memory Object.
Retrieval returns Context Package.
Engine generates markdown/HTML projection.
Engine proves the projection is fresh and source-linked.
Agent loads Context Package and writes an observation.
Observation becomes pending Claim.
```

That proves the core without assuming a fixed number of phases.
