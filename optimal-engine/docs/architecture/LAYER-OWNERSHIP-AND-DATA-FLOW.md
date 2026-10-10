# Layer Ownership and Data Flow

Optimal Engine layers are responsibility boundaries. They are not necessarily
separate databases, services, or folders.

For the concrete markdown-operable workspace example, see
[`CURRENT-WORKSPACE-USAGE-MAP.md`](CURRENT-WORKSPACE-USAGE-MAP.md).

For the pressure-test of failure modes and guardrails, see
[`DEVILS-ADVOCATE-REVIEW.md`](DEVILS-ADVOCATE-REVIEW.md).

For the dependency-driven build sequence, see
[`OPTIMIZED-BUILD-PHASES.md`](OPTIMIZED-BUILD-PHASES.md).

For the comparison against nearby public systems, see
[`REFERENCE-SYSTEM-COMPARISON.md`](REFERENCE-SYSTEM-COMPARISON.md).

For the build plan that absorbs reference-system patterns into Optimal Engine,
see [`REFERENCE-ADOPTION-BUILD-PLAN.md`](REFERENCE-ADOPTION-BUILD-PLAN.md).

For the source/data object anatomy and multimodal contract, see
[`DATA-ANATOMY-AND-MULTIMODALITY.md`](DATA-ANATOMY-AND-MULTIMODALITY.md).

For the coverage check against studied reference systems and source-paper
concepts, see [`ARCHITECTURE-COVERAGE-MATRIX.md`](ARCHITECTURE-COVERAGE-MATRIX.md).

The clean model is:

```text
Layer = owns a lifecycle
Object = durable thing or projection the layer manages
Store = physical persistence mechanism
Interface = how other layers ask for behavior
Projection = derived view another layer can consume
```

The same physical database can hold many layers' data. What matters is that each
table has a clear owner and that other layers do not bypass that owner when they
write.

## Clean Mental Model

Optimal Engine is the whole operating engine.

Inside it:

| Subsystem | Meaning |
| --- | --- |
| Workspace Topology | Defines the shape of the human/business world. |
| Signal Pipeline | Turns raw input into classified Signals. |
| Memory Core | Stores source-backed truth and institutional memory. |
| Retrieval / Context | Gives humans and AI the right context at the right time. |
| Active Memory Pools | Shared task workspace for humans and agents. |
| Workflow / Skill Runtime | Turns repeated work into reusable procedures. |
| Tool / Model Governance | Controls what agents, models, and tools are allowed to do. |
| Workspace Export | Turns governed state into files and folders humans can browse. |

The full engine can be read as four connected loops:

```text
Human defines the world:
  Workspace -> Nodes -> Relationships -> Policies

Data enters the world:
  Scope Envelope -> Source Package -> Signal -> Route -> Claim -> Fact -> Memory

Humans and AI use the world:
  Query/Task -> Retrieval -> Context Package -> Active Pool -> Action

Work improves the world:
  Observation -> Claim -> Fact -> Memory -> Workflow -> Skill
```

## Canonical Flow

```text
Human / agent / connector / file / app command
  -> Command Gateway
  -> Scope Envelope
       workspace_id
       actor_id
       node hints
       source type
       permissions
       operation class
  -> Source Package when the operation contains new evidence
  -> Signal
  -> Route / Node assignment
  -> Compatibility Context row
  -> Claim
  -> Fact
  -> Memory Object / Episode Object / Memory Detail Object
  -> Relationship Edge
  -> Retrieval Package
  -> Context Package
  -> Active Memory Pool
  -> Workflow Trace
  -> Generalized Workflow
  -> Procedural Memory Object
  -> Skill Package
  -> Tool / Model Execution
  -> Observation / Pending Claim
  -> Promotion back into Memory Core
```

This flow is not always linear. Some inputs stop at Signal. Some Claims never
become Facts. Some Memory Objects never become workflows. The point is that each
promotion step has an owning layer.

The Scope Envelope may be incomplete at first. Unknown scope is not a reason to
skip evidence preservation. If the engine does not yet know the right Node, the
source is still stored and routed to inbox/quarantine or a pending topology
decision.

## Flow Correctness Rules

These rules decide whether a flow is architecturally valid.

| Rule | Meaning |
| --- | --- |
| Scope before side effects | The engine identifies actor, workspace, operation class, permissions, and best-known Node hints before writing derived state. |
| Source before interpretation | Raw evidence is committed or quarantined before classification, summarization, extraction, routing side effects, or generated output. |
| Route before projection | Files, wiki pages, HTML, and app views are generated only after the object has workspace/Node scope or explicit quarantine state. |
| Claim before Fact | Extracted assertions, tool output, generated summaries, and human edits do not become accepted truth directly. |
| Package before prompt | Agents receive Context Packages or Skill Packages, not loose search chunks as the final governed context. |
| Observation before promotion | Agent/tool work writes observations or pending Claims first; promotion creates Facts, Memory Objects, Workflows, or Skills later. |
| Export as projection | Markdown, wiki, HTML, dashboards, and app responses are projections with export records, versions, source links, and drift checks. |
| Store as adapter | `Store` executes persistence, but layer interfaces decide object meaning and lifecycle. |
| Audit every governed transition | Every promotion, retrieval package, export, tool/model call, permission decision, and topology change has an audit or ledger record. |

## Layer Map

| Layer | What it owns | Receives | Produces | Durable state | Derived state |
| --- | --- | --- | --- | --- | --- |
| Source Intake | Preserved inputs entering the engine. | Raw text, files, connector payloads, tool outputs. | Source Packages. | `source_packages` through Memory Core. | ingestion run metadata. |
| Signal Pipeline | Classification and signal quality. | Source Package or raw input. | Signal, quality action, Signal dimensions. | compatibility `contexts` row for signal search. | L0/L1 summaries, embeddings. |
| Workspace / Topology | Workspaces, nodes, node types, custom fields, and relationships. A project is usually a Node whose `node_type` is `project`. | Human setup, imported organization structure, connector metadata, templates. | Workspace scope, node definitions, project Nodes, allowed relationship types. | `workspaces`, `nodes`, `node_types`, `node_relationships`, custom metadata. | filesystem layout, routing hints, partition map. |
| Routing | Where signals and memory objects belong inside the topology. | Signal, entities, workspace topology, routing hints. | Node assignment, cross-reference destinations, partition hints. | routing rules/config, route audit. | workspace export paths. |
| Memory Core | Source-backed institutional memory. | Source Packages, Signals, Claims, evidence, review decisions. | Claims, Facts, Memory Objects, Episodes, Details, Relationship Edges. | `claims`, `facts`, `memory_objects`, `relationship_edges`, `derivation_ledger`. | memory summaries, derived indexes. |
| Retrieval | Governed recall planning. | Query, actor, task, time mode, hints, authorization envelope. | Retrieval Package, Context Package. | retrieval audit, `context_packages`. | ranked hits, snippets, reranked candidates. |
| Active Collaboration | Task-scoped working memory. | Context Package, pool scope, members, agents, observations. | Active Memory Pool, pending Claims, refresh state. | `active_memory_pools`, pool observations, membership records. | loaded task context. |
| Workflow | Repeated work and process traces. | Episode Objects, Memory Details, tool runs, source evidence. | Workflow Traces, candidate Generalized Workflows. | `workflow_traces`, `generalized_workflows`. | clustered variants, exception summaries. |
| Skill Runtime | Governed procedural reuse. | Generalized Workflows, Procedural Memory Objects, policies, tools. | Skill Packages, execution records, validation outcomes. | `procedural_memory_objects`, `skill_packages`, execution records. | generated prompts, runtime plans. |
| Model Governance | Approved model invocation. | Model-call operation request, input object links, policy. | Model output object, extraction, summary, embedding, rerank, validation. | `model_call_operations`, model call runs. | embeddings, summaries, scores. |
| Tool Governance | Approved tool/API/MCP invocation. | Tool request, actor, pool, schema, partition scope. | Tool result, observation, pending Claim, audit event. | `mcp_tool_definitions`, tool call runs. | normalized tool output. |
| Workspace Export / Wiki Surface | Human-readable filesystem, HTML, and wiki projection. | Topology, Signals, Memory Objects, Facts, workflows, context packages, human edits, imports. | Markdown/files/folders, HTML pages, page tree, backlinks, link status, revisions, import records. | export records, projection revisions, link health records, import records, sync metadata. | node folders, `context.md`, generated docs, dashboards, app/API views. |
| Audit / Governance | Proof of what happened and who could access it. | Every governed read/write/promotion/execution. | Audit events, policy decisions, invalidation notices. | audit/event tables, policy tables. | reports, public/private audit output. |

## Storage Rule

Optimal Engine can start with one physical database:

```text
SQLite or Postgres
```

But code should treat it as many layer-owned table groups:

```text
Memory Core tables
Signal/search compatibility tables
Topology tables
Retrieval projection tables
Workflow/Skill tables
Model/tool governance tables
Audit tables
Workspace export/wiki records
```

The physical store is shared. Ownership is not.

## Storage Classification Rule

Every stored thing must be classified by what it means, not only by where its
bytes live.

```text
Canonical truth
Task-local working state
Execution record
Projection
Cache / index
Policy / audit record
```

The same SQLite or Postgres database can hold all of these classes, but they are
not interchangeable.

| Storage class | Examples | Owner | Meaning | Rebuildable? |
| --- | --- | --- | --- | --- |
| Topology truth | `workspaces`, `workspace_members`, `nodes`, `node_members`, current `nodes.kind`, target `node_types`, target `node_relationships` | Workspace / Topology | Defines the user's world and operating boundaries. | No; corrections need audit/version records. |
| Evidence truth | `source_packages`, content hashes, source URI, raw text, trust/retention/security metadata | Memory Core / Source Intake | Preserved source material before interpretation. | No, unless the external source still exists. |
| Interpretation truth | `claims`, `facts`, `memory_objects`, `memory_detail_objects`, `relationship_edges` | Memory Core | What was asserted, what was accepted, what it means, and how objects relate. | Partly; candidates can be regenerated, accepted history remains. |
| Lineage/audit truth | `derivation_ledger`, audit events, policy versions, access decisions, validation records | Memory Core / Audit / Governance | Proof of how an object was created and why it was allowed. | No. |
| Working state | `active_memory_pools`, pool observations, loaded context, promotion candidates | Active Collaboration | Task-scoped state while work is happening. | Partly; loaded context can be rebuilt, old sessions remain historical. |
| Recall projection | `context_packages`, retrieval plans, authorization envelopes, returned/redacted links | Retrieval / Context | Actor/task-specific context assembled from governed state. | Yes, but old packages can remain useful for audit. |
| Search/index projection | `contexts`, `contexts_fts`, `chunks`, `chunk_embeddings`, vectors, summaries, caches | Signal/Search/Retrieval | Compatibility and acceleration state. | Yes. |
| Workflow/skill truth | `workflow_traces`, `generalized_workflows`, `procedural_memory_objects`, `skill_packages` | Workflow / Skill Runtime | Evidence-backed repeated work and validated procedure. | Partly; candidates can be regenerated, reviewed versions remain. |
| Execution record | tool call runs, model call runs, connector sync runs, normalized outputs | Tool / Model Governance | Proof of what the system or agent did during execution. | No; re-running creates a new record. |
| Human-facing projection | markdown files, node folders, generated HTML, dashboards, packages, app/API views | Workspace Export / Wiki Surface | Readable/editable surfaces generated from engine state. | Yes, unless a human edit becomes a new source or topology change. |
| Projection health | page tree, backlinks, broken-link status, export freshness, projection diffs, edit locks, revision history, import lineage | Workspace Export / Wiki Surface / Audit | Proof that the human surface matches governed state and can be repaired. | Partly; checks can rerun, history remains. |

Non-negotiable storage rules:

```text
Do not store institutional truth only in contexts.
Do not let vectors or summaries become facts.
Do not let agent observations become facts without promotion.
Do not let exported markdown silently mutate canonical state.
Do not create generated objects without source links and ledger/audit records.
Do not let any layer bypass the owning lifecycle because it can reach Store.
Do not ship a wiki/export surface without link health, revision, import, and
projection drift checks.
```

## Topology Modeling Rule

Workspace Topology should not create a separate peer table for every domain
concept by default.

The default model is:

```text
workspace
  -> nodes
  -> node_types
  -> node_relationships
  -> optional type-specific profile tables
```

That means projects, people, entities, products, operations, rhythm areas,
finance areas, research areas, clients, and inboxes usually start as Nodes.

```text
project        -> Node(node_type = project)
person         -> Node(node_type = person)
company/entity -> Node(node_type = entity)
product        -> Node(node_type = product)
operation      -> Node(node_type = operational)
rhythm         -> Node(node_type = rhythm)
finance area   -> Node(node_type = finance)
research area  -> Node(node_type = learning/context)
```

Add a profile table only when the type needs stable, queryable, governed fields:

```text
project_profiles
person_profiles
entity_profiles
product_profiles
operation_profiles
rhythm_entries
metric_series
```

Do not make this mistake:

```text
workspace
  -> projects
  -> people
  -> products
  -> operations
  -> nodes
```

That creates parallel hierarchies and makes routing, permissions, export,
retrieval, and agent context harder. The stable hierarchy is Workspace -> Nodes.
Relationships and profiles supply the shape underneath.

## Table Ownership

| Table or artifact | Owner layer | Truth or projection? | Rebuildable? |
| --- | --- | --- | --- |
| `source_packages` | Memory Core | Durable truth | No, unless source still exists externally. |
| `derivation_ledger` | Memory Core / Audit | Durable truth | No. |
| `claims` | Memory Core | Durable truth | Can be re-extracted, but prior claim history is truth. |
| `facts` | Memory Core | Durable truth | No; corrections create versions/supersession. |
| `memory_objects` | Memory Core | Durable truth with derivation links | Can be rebuilt, but prior versions remain historical truth. |
| `memory_detail_objects` | Memory Core / Workflow | Durable truth when validated | Partly. |
| `relationship_edges` | Memory Core / Graph | Durable truth or derived edge depending on edge type | Derived edges can be rebuilt. |
| `contexts` | Signal/Search compatibility | Projection/compatibility row | Yes, if source and metadata remain. |
| `contexts_fts` | Retrieval/Search | Projection | Yes. |
| `vectors` / `chunk_embeddings` | Retrieval/Search | Projection | Yes. |
| `workspaces` | Workspace / Topology | Durable configuration | No. |
| `nodes` | Workspace / Topology | Durable configuration | No, except from topology config. |
| `node_types` | Workspace / Topology | Durable configuration | No. |
| `node_relationships` | Workspace / Topology | Durable topology relationship | No. |
| `node_members` | Workspace / Topology / Identity | Durable governance | No. |
| `project_profiles` | Workspace / Topology | Optional extension for project-specific fields keyed by project Node. | No. |
| `person_profiles`, `entity_profiles`, `product_profiles`, etc. | Workspace / Topology | Optional type-specific profile tables keyed by Node. | No. |
| `routing_rules` | Routing | Durable routing policy | No. |
| `context_packages` | Retrieval | Projection with audit value | Yes, but old packages remain useful for audit. |
| `active_memory_pools` | Active Collaboration | Durable task state | No. |
| pool observations | Active Collaboration | Task-local truth | No; promotion creates Claims. |
| `workflow_traces` | Workflow | Durable evidence-linked trace | No. |
| `generalized_workflows` | Workflow | Derived candidate / reviewed object | Yes from traces, but versions remain historical. |
| `procedural_memory_objects` | Skill Runtime / Workflow | Durable reviewed procedure | Partly. |
| `skill_packages` | Skill Runtime | Durable executable/guided package | Partly. |
| `model_call_operations` | Model Governance | Durable definition | No. |
| model call runs | Model Governance / Audit | Durable execution record | No. |
| `mcp_tool_definitions` | Tool Governance | Durable definition | No. |
| tool call runs | Tool Governance / Audit | Durable execution record | No. |
| workspace markdown files | Workspace Export / Wiki Surface | Projection | Yes from core state, unless user edits are accepted as new sources. |
| generated HTML pages | Workspace Export / Wiki Surface | Projection | Yes from core state. |
| `export_records` | Workspace Export / Wiki Surface / Audit | Projection proof | No; each render gets its own record. |
| projection revisions | Workspace Export / Wiki Surface | User-facing projection history | No. |
| link health records | Workspace Export / Wiki Surface | Rebuildable validation result | Yes. |
| import records | Workspace Export / Source Intake | Import lineage and source boundary | No. |

## Write Ownership Rule

Every write path should answer:

```text
Who owns this object lifecycle?
Is this durable truth or a projection?
Can this be rebuilt?
What invalidates it?
What audit event proves it happened?
```

Examples:

```text
Pipeline.Intake may create a Source Package through MemoryCore.
Pipeline.Intake may create a compatibility Context row.
Pipeline.Intake must not create a Fact directly.
```

```text
Workspace.Topology may define nodes, project Nodes, node types, and node relationships.
Projects are usually Nodes with `node_type = project`, not a mandatory layer above all Nodes.
Routing may use those definitions.
Routing must not invent durable topology without going through Workspace.Topology.
```

```text
Retrieval.Search may read Context rows and indexes.
Retrieval.Search must not decide what is true.
MemoryCore.FactPromoter decides what is true.
```

```text
Workspace Export may write files.
Workspace Export files are not truth unless re-ingested as Source Packages.
Workspace Export must record what object versions produced each file.
Workspace Export must detect stale projections, broken links, and edit collisions.
```

```text
Active Memory Pool observations are task-local.
They become durable knowledge only through Claim promotion.
```

## Layer Interfaces

New code should depend on layer interfaces, not table names.

| Caller needs | Call this kind of module | Avoid |
| --- | --- | --- |
| Store raw evidence | `MemoryCore.SourcePackageService` | direct SQL into `source_packages` |
| Define a workspace/node/project Node | `WorkspaceTopology` | hardcoded node maps in feature code |
| Persist Memory Core row | `MemoryCore.Store` | `OptimalEngine.Store.raw_execute` from business modules |
| Promote assertion to truth | `MemoryCore.FactPromoter` | setting `facts` rows directly |
| Retrieve governed context | `MemoryCore.RetrievalCoordinator` or `Retrieval` facade | raw vector search as final answer |
| Create task context | `MemoryCore.ActiveMemoryPool` | free-floating session memory |
| Run a reusable procedure | `SkillPackageService` | arbitrary prompt/tool call |
| Export files/pages | `WorkspaceExport` or `WikiSurface` | treating node files as source of truth |
| Check projection health | `WorkspaceExport.Health` or `WikiSurface.Health` | manual eyeballing of generated pages |

## Practical Rule For The Codebase

The codebase should not be organized around "where the bytes are stored." It
should be organized around "who owns the lifecycle."

Good:

```text
MemoryCore.FactPromoter.promote/2
MemoryCore.ActiveMemoryPool.publish_observation/3
Retrieval.Search.search/2
WorkspaceExport.write_node_files/2
```

Bad:

```text
Store.create_fact/2
Store.publish_observation/3
Store.run_skill/2
Context.make_truth/1
```

`Store` can physically execute writes. It should not own domain meaning.

## Current Transitional State

Today, the codebase has three generations of naming:

1. Signal-first engine names: `Signal`, `Pipeline`, `Routing`, `NodeMap`.
2. Compatibility context names: `Context`, `Store`, `Retrieval.Search`.
3. New Memory Core names: `SourcePackage`, `DerivationLedgerEntry`, future
   `Claim`, `Fact`, `MemoryObject`, `ContextPackage`, `ActiveMemoryPool`.

That is acceptable during migration if the rule is clear:

```text
Old names may remain for compatibility.
New durable-memory behavior should use Memory Core names.
```

## Next Cleanup Target

After the Memory Core namespace is established, the next structural cleanup
should be:

```text
OptimalEngine.Store
  -> keep SQLite connection ownership and low-level execution

OptimalEngine.ContextStore
  -> compatibility Context CRUD

OptimalEngine.MemoryCore.Store
  -> Memory Core table persistence adapter

OptimalEngine.Retrieval.Store
  -> retrieval/index projection access
```

This keeps one physical database while separating lifecycle ownership.

## Workspace Topology Lifecycle

Humans do not only upload data. They define the shape of the work:

```text
workspace
  -> nodes
  -> node types
  -> project Nodes when needed
  -> node relationships
  -> members / agents / tools
  -> allowed source types
  -> routing rules
  -> export layout
```

That data is dynamic. Some parts are standardized across every workspace, while
other parts are custom to one organization, project, team, or operating model.

### Standardized Topology Objects

These should have stable engine-level meaning:

| Object | Meaning |
| --- | --- |
| Workspace | A governed knowledge and work boundary. |
| Project Node | A Node whose type is `project`; it represents a scoped initiative, customer, product, incident, or operational effort. |
| Node | A meaningful place knowledge, work, decisions, and relationships can belong. |
| Node Type | Category of node, such as person, team, system, customer, project, inbox, domain, program, workflow. |
| Node Relationship | Typed link between nodes, such as owns, participates_in, depends_on, reports_to, supports, customer_of, part_of. |
| Member | Human, agent, service account, team, or tool allowed to participate. |
| Routing Rule | Rule or hint for assigning incoming Signals to nodes, project Nodes, or partitions. |

### Custom Topology Data

Workspaces also need custom fields because every organization models itself
differently:

```text
node.custom_metadata
project.custom_metadata
relationship.custom_metadata
routing_rule.conditions
workspace.policy_overrides
```

Custom does not mean ungoverned. The owning layer still records:

```text
who defined it
when it became valid
which workspace it belongs to
which policy controls it
which exports/routes/context packages depend on it
```

### Human Setup Flow

```text
Human creates workspace
  -> defines nodes
  -> chooses node types
  -> marks some nodes as project Nodes when needed
  -> links nodes with relationship types
  -> grants humans/agents/tool access
  -> sets routing rules and default partitions
  -> engine validates topology
  -> workspace export layout is generated
  -> intake/retrieval/workflows can use the topology
```

### AI Agent Setup Flow

An AI agent can help create or update topology, but it should not silently
change durable workspace structure.

```text
Agent suggests node/project/relationship
  -> creates pending topology change
  -> human or policy review accepts it
  -> topology version is updated
  -> routes/exports/context packages are invalidated or refreshed
```

### Storage Shape

The first implementation can be simple:

```text
workspaces
nodes
node_types
node_relationships
node_members
routing_rules
topology_change_requests
project_profiles when project-specific fields are needed
```

Every row should include:

```text
tenant_id
workspace_id
lifecycle_state
valid_time_start / valid_time_end
transaction_time_start / transaction_time_end
metadata JSON
created_by
policy_version
```

### Why This Matters

Topology is upstream of almost everything:

```text
Intake uses it to route Signals.
Memory Core uses it for partitions and Relationship Edges.
Retrieval uses it for scoped recall.
Active Memory Pools use it for membership and task scope.
Workspace Export uses it to generate folders/files.
Skill Packages use it to decide applicability.
```

So topology is not decoration. It is governed operational data.

For the canonical Node object model, see `NODE-ONTOLOGY.md`.
