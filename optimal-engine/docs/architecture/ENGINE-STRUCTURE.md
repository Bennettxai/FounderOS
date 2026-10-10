# Engine Structure

This page defines the high-level structure of Optimal Engine.

The purpose is to make the system easy to reason about:

```text
Who owns the workspace shape?
Where does data enter?
Where is truth stored?
Which stores are canonical vs rebuildable?
How do humans and agents use the same state?
Which surfaces display the state?
```

## Product Definition

Optimal Engine is a self-hosted second brain and operating engine for human and
AI workspaces.

It lets a person, team, or company:

- define their work structure;
- preserve source evidence;
- classify noisy input;
- extract Claims;
- promote reviewed Facts;
- build source-backed Memory Objects;
- retrieve governed Context Packages;
- operate with agents and tools;
- learn workflows;
- project state into markdown, wiki, app, API, and reports.

## Hierarchy

The hierarchy is:

```text
Tenant / Organization
  -> Workspace
    -> Node graph
      -> Node
        -> attached sources, signals, claims, facts, memories, workflows, skills
```

Definitions:

| Object | Meaning |
| --- | --- |
| Tenant / Organization | Governance boundary for one person, team, or company. |
| Workspace | A bounded operating area with its own nodes, policies, members, sources, and projections. |
| Node | A governed unit of context, purpose, relationships, state, and activity. |
| Node Type | The kind of Node: project, person, product, operational, context, learning, entity, team, department, etc. |
| Node Relationship | Typed relationship between Nodes: parent/child, depends_on, blocks, owns, supports, participates_in, references. |
| Alias | Scoped user-facing name that resolves to a stable canonical object. |

Projects are Nodes inside a Workspace:

```text
Workspace
  -> Project Node
  -> Person Node
  -> Product Node
  -> Operational Node
  -> Context Node
  -> Learning Node
```

A folder may represent a Node, but a folder is only a projection. The Node is
the governed topology object.

Users can call the same canonical type different things. A Project Node may be
called an initiative, campaign, engagement, deal, case, program, or account in
one workspace. The engine should preserve those words as aliases/display labels
while routing durable writes through stable IDs.

For the concrete filesystem projection, scope switching rules, and naming
discipline, see:

- [`../guides/workspace-filesystem.md`](../guides/workspace-filesystem.md)
- [`../guides/scope-switching.md`](../guides/scope-switching.md)
- [`../guides/naming-and-aliases.md`](../guides/naming-and-aliases.md)

## Layer Stack

```text
User / Agent / App / Connector
  -> Command and Query Gateway
  -> Workspace / Topology
  -> Source Intake
  -> Signal Pipeline
  -> Memory Core
  -> Retrieval / Context
  -> Active Memory Pools
  -> Workflow / Skill Runtime
  -> Tool / Model Governance
  -> Wiki / Export
  -> Evaluation / Recovery
```

## Layer Responsibilities

| Layer | Owns | Does not own |
| --- | --- | --- |
| Workspace / Topology | workspaces, nodes, node types, relationships, members, routing rules, pending topology changes | Claims, Facts, model outputs |
| Source Intake | raw evidence preservation, source package creation, hashes, source metadata | truth decisions |
| Signal Pipeline | classification, parsing, modalities, compatibility search rows | accepted knowledge |
| Memory Core | Claims, Facts, Memory Objects, evidence edges, lineage, temporal validity, supersession | UI layout, raw tool execution |
| Retrieval / Context | query planning, authorization envelope, Context Package assembly, stale package refresh | source preservation or truth promotion |
| Active Memory Pools | task-scoped working memory, loaded context, observations, pending Claims from work | permanent workspace topology |
| Workflow / Skill Runtime | workflow traces, generalized workflows, procedural memory, skill packages | arbitrary ungoverned prompts |
| Tool / Model Governance | registered tools, schemas, grants, model/tool call records, audit | final truth |
| Wiki / Export | markdown/wiki/HTML/report/API projections and projection revisions | canonical truth unless re-ingested |
| Evaluation / Recovery | benchmark runs, cases, reality checks, rebuild checks | production user workflow decisions |

## Data Lifecycle

The normal knowledge lifecycle is:

```text
Input
  -> Source Package
  -> Signal
  -> Claim
  -> Fact
  -> Memory Object
  -> Context Package
  -> Active Memory Pool
  -> Observation
  -> Pending Claim
```

Promotion rules:

```text
Source Package preserves what arrived.
Signal classifies what kind of thing it is.
Claim records what a source appears to say.
Fact records what review/policy accepts as true.
Memory Object records why that accepted truth matters.
Context Package packages authorized context for a task.
Active Memory Pool holds temporary working context.
Observation records what happened during work.
Pending Claim waits for review before becoming truth.
```

## Storage Model

One physical database can hold many logical layers. That does not mean every
module can write every table.

```text
Same physical store is allowed.
Lifecycle ownership is separate.
Only the owning layer writes durable lifecycle state.
Other layers read through interfaces or projections.
```

### General Stores

| Store | Use |
| --- | --- |
| SQLite | Local default canonical runtime store. |
| Postgres | Production target canonical runtime store. |
| Raw artifact storage | Preserved files, uploads, attachments, media, and raw source evidence. |

### Specific Stores And Projections

| Store/projection | Use |
| --- | --- |
| FTS tables | Lexical search acceleration. |
| Vector/chunk tables | Semantic retrieval and chunk-level context. |
| Cache directories | Parse/model/index acceleration; rebuildable. |
| ETS | Fast in-memory graph/knowledge backend. |
| RocksDB | Optional persistent graph/triple-store backend for specialized workloads. |
| Mnesia/Riak-style backends | Optional distributed graph/knowledge experiments. |
| Markdown files | Human-operable projection/editing surface. |
| Wiki/HTML/API/app views | Display/control projections over engine state. |

## Surface Model

Humans and agents should not have separate memory systems. They use different
surfaces over the same backend state.

| Surface | Human use | Agent/tool use |
| --- | --- | --- |
| Markdown | inspect and edit workspace files directly | read/write projection files when allowed |
| CLI | setup, initiate, inspect, search, render, verify | deterministic local control surface |
| API | dashboards, apps, services | programmatic access |
| MCP/tools | agent-safe commands | governed tool calls |
| A2A agents | rarely direct; configured as collaborators | delegated agent work and returned artifacts |
| Wiki/export | browse and share readable state | load curated/cited pages |
| Context Packages | rarely inspected directly | primary task context for agents |
| Active Memory Pools | task collaboration view | task-local working memory |

## Setup Flow

There are two setup paths.

Use `optimal.initiate` when the user starts with messy context:

```text
messy dump
  -> Source Package
  -> setup Claim
  -> conservative Node candidates
  -> applied workspace Nodes by default
  -> proposed integrations remain disabled
  -> optional review-only topology requests
```

Use `optimal.setup` when the user already knows the structure:

```text
workspace slug/name
  -> standard Node Types
  -> starter Nodes or provided Nodes
  -> rhythm files
  -> AGENTS.md projection
  -> workspace export records
```

## Integration Flow

External systems do not become trusted memory automatically.

```text
MCP / connector / API / script / cron / tool call / A2A delegation
  -> registered tool or connector definition
  -> credentials/scopes/partition policy
  -> governed execution
  -> raw output preserved as Source Package or observation
  -> pending Claim
  -> review/policy promotion
```

The integration surface can be general or specific:

| Integration type | General or specific? | Example use |
| --- | --- | --- |
| MCP server | General tool protocol | calendar, docs, repo, task manager access |
| Connector | Product-specific sync adapter | mail, tickets, CRM, source control |
| API adapter | Specific integration | a custom internal service |
| A2A agent | Agent collaboration | specialist review, partner/supplier agent, delegated long-running task |
| Script/cron | Specific local automation | daily import, report generation, backup |
| Model call | Governed model operation | classification, extraction, summarization, judging |

## Self-Learning Loop

The engine learns through governed loops, not by silently mutating truth.

```text
work happens
  -> observation
  -> pending Claim
  -> reviewed Fact
  -> Memory Object update
  -> repeated Episodes / Details
  -> Workflow Trace
  -> Generalized Workflow
  -> Procedural Memory Object
  -> Skill Package
```

This lets agents improve the workspace without bypassing human or policy review.

## What Good Looks Like

A well-structured Optimal Engine workspace has:

- clear tenant/organization boundary;
- one or more workspaces;
- Nodes for meaningful operating areas;
- typed Node relationships;
- preserved raw sources;
- Claims separated from Facts;
- source-backed Memory Objects;
- Context Packages for agents;
- Active Memory Pools for tasks;
- registered tools/connectors;
- workflow traces and draft Skill Packages;
- markdown/wiki/app/API projections generated from engine state;
- verification through `mix optimal.reality_check`.
