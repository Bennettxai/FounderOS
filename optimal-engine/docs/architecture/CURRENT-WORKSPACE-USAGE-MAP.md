# Current Workspace Usage Map

This page maps the way a current markdown workspace is actually used to the
target Optimal Engine architecture.

The important conclusion is simple: the workspace already behaves like an
operating system. The filesystem is the human-readable control surface. Optimal
Engine should become the governed runtime that owns the topology, memory,
retrieval, workflow, agent, and export lifecycles behind that filesystem.

## Primary Use Case

The main example is a personal, company, or team operating system that a human
can use through markdown, an app, a CLI, or an AI agent.

The user does not start by creating database records. The user starts by
organizing real life and work:

```text
companies
  -> departments / domains
  -> teams
  -> projects
  -> people
  -> products
  -> operations
  -> money / metrics
  -> research
  -> rhythm / daily work
```

Each area becomes a Node. A project is one kind of Node, not a mandatory parent
of every other Node. The user can inspect any Node as markdown, talk to an agent
about it, open it in an app, export it as a package, or connect outside tools to
it through APIs and protocol adapters.

That is the user-facing product shape:

```text
Use markdown when you want direct control.
Use the app when you want dashboards and navigation.
Use the CLI when you want fast local commands.
Use an agent when you want work performed against the system.
Use the database when you need governed memory, search, history, and automation.
```

Optimal Engine is the backing runtime that keeps those surfaces consistent.

## Two Sides Of The System

Optimal Engine has two sides that must stay connected:

| Side | Purpose | Examples |
| --- | --- | --- |
| System side | Stores, governs, validates, indexes, retrieves, and audits state. | database tables, Memory Core, routing, permissions, ledger, workflows, skills. |
| Interface side | Lets humans and agents see, edit, ask, act, and export. | markdown files, HTML pages, app dashboards, CLI, APIs, agent tools, reports. |

The system side is responsible for correctness. The interface side is
responsible for usability.

The same object can appear in multiple interfaces:

```text
Node Object
  -> markdown node page
  -> dashboard card
  -> graph view
  -> agent context package
  -> exported report
```

The interface is not a second truth. It is a projection or editing surface over
the governed system state.

## What Exists Today

A current workspace has several distinct operating zones:

| Current artifact | What it is today | Target engine meaning |
| --- | --- | --- |
| `topology.yaml` | Entity, domain, operation, unit, endpoint, and agent map. | Workspace topology seed. |
| `.system/config.yaml` | Path map, node map, tier budgets, temporal decay, and routing rules. | Runtime configuration and topology import source. |
| `nodes/` | Knowledge areas for companies, projects, people, operations, money, team, research, and inbox state. | Workspace export of governed Nodes. |
| `nodes/*/context.md` | Stable node profile and durable context. | Node context projection from Memory Core and topology state. |
| `nodes/*/signal.md` | Current node status and recent operating signal. | Node state projection from Signals, Facts, and active work. |
| `nodes/*/signals/` | Time-based updates, calls, notes, decisions, and operating events. | Source Package and Signal event stream. |
| Node subfolders | Playbooks, deliverables, strategy, projects, assets, research, and local records. | Node-specific projections of workflows, Skill Packages, artifacts, and source links. |
| `rhythm/` | Daily, weekly, and monthly operating cadence. | Human operating layer and Focus/Tracking state. |
| `agent-dispatch/` | Agent dispatches, activation prompts, reports, and validation records. | Active Memory Pools, workflow runs, task assignments, and agent execution logs. |
| local agent skills | Local agent skills and operating procedures. | Skill Package and tool surface projections. |
| `packages/` | Exported bundles for a receiver, client, team, or internal use case. | Workspace Export artifacts. |
| `engine/` | Search, classification, routing, graph, health, simulation, memory, and API code. | Optimal Engine runtime. |

## What This Means

The current workspace is not just a document vault. It is a file-backed
operating surface.

The user creates and maintains bounded areas of life and work. Those areas may
be companies, projects, people, operations, products, money, team, research,
content, or temporary inboxes. Each area has identity, purpose, relationships,
current state, historical context, decisions, and outputs.

That is exactly the Node model:

```text
Workspace
  -> Node
  -> Node type
  -> Node context
  -> Node state
  -> Node signals
  -> Node relationships
  -> Node artifacts
```

Today, the node is represented mostly by a folder and markdown files. In the
target architecture, the node becomes a governed topology object. The folder is
still useful, but it becomes an export and editing surface rather than the only
place the truth can live.

## Markdown-First, Database-Backed

The operating model is:

```text
Markdown at the top
Database underneath
Agents and apps on the sides
```

Markdown gives the human a direct, portable, inspectable way to control the
system. The database gives the engine durable identity, search, permissions,
history, provenance, workflow state, tool state, and rebuildable indexes.

The engine should support both directions:

```text
Markdown -> ingestion -> database state
Database state -> projection -> markdown
```

This avoids two bad extremes:

- database-only systems that hide the workspace from the user;
- markdown-only systems that cannot govern memory, agents, permissions,
  workflows, or long-term retrieval.

The user should be able to keep working even if the app is not open. The app,
agent, CLI, files, and database should all understand the same workspace rather
than creating separate versions of reality.

## Current Human Workflow

The current human workflow looks like this:

```text
Human has work, context, decision pressure, or new information
  -> talks to an agent, writes a note, edits a file, or runs a command
  -> agent reads operating protocol, node context, rhythm state, and engine docs
  -> agent classifies the input as a Signal
  -> engine routes the Signal to one or more nodes
  -> node context, signal files, docs, code, or packages are updated
  -> engine indexes the new material for future recall
```

This is already a practical human-AI operating loop. The missing hardening is
that the engine should preserve source evidence, derivation, permissions,
temporal validity, and promotion state instead of relying only on markdown
placement.

## Current AI Workflow

When an agent works inside the current workspace, it effectively does this:

```text
Load operating instructions
  -> inspect node topology and current rhythm
  -> search or read relevant context
  -> infer the user's intent
  -> perform code, document, design, or planning work
  -> write artifacts back to the workspace
  -> run verification where possible
  -> report what changed
```

In target Optimal Engine terms, that is:

```text
Agent session
  -> Active Memory Pool
  -> Context Package
  -> retrieval from Memory Core and Workspace Topology
  -> Tool / Model execution
  -> observations and artifacts
  -> pending Claims or workflow records
  -> promotion into durable memory when accepted
```

The agent should not need to know every storage detail. It should call engine
interfaces that return the right Context Package, node state, workflow, tool, or
Skill Package for the task.

## From Markdown To HTML And Apps

Generated HTML pages, dashboards, and app views should come from the same
workspace state.

```text
markdown node file
  -> Source Package
  -> Signal
  -> Node state / Memory Core state
  -> projection builder
  -> HTML page / dashboard / report / API response
```

The agent can help build those interfaces:

```text
Human asks for a page or dashboard
  -> agent retrieves the Node and Context Package
  -> agent generates or updates HTML/components
  -> engine stores the generated artifact as an export
  -> source links and audit show what the page was built from
```

That makes generated pages useful without making them untracked one-off files.
The page is a view of the system, not the system itself.

## ROM, RAM, And Projections

The current workspace already separates slow and fast context:

| Current concept | Meaning | Target object |
| --- | --- | --- |
| `context.md` | Slow-changing node memory. | Node Context projection from Facts, Memory Objects, and topology metadata. |
| `signal.md` | Current state for the node. | Node State projection from recent Signals, Focus items, blockers, and metrics. |
| `signals/` | Time-based event stream. | Source Packages, Signals, Claims, and Episode Objects. |
| `rhythm/` | Daily and weekly operating state. | Focus, Tracking, Review Cadence, and Active Memory Pool state. |
| playbooks / procedures | Reusable ways of doing work. | Procedural Memory Objects and Skill Packages. |
| dispatch reports | Agent run history and validation. | Workflow Trace, tool-call audit, and execution records. |

The target engine should keep this separation, but it should make the lifecycle
explicit:

```text
Stable memory is not the same as live state.
Live state is not the same as source evidence.
Source evidence is not the same as accepted truth.
Accepted truth is not the same as executable procedure.
```

## Data Flow For The Current Workspace

The clean technical flow is:

```text
Conversation / note / file / tool result
  -> Scope Envelope
       actor
       workspace
       node hints
       operation class
       permissions
  -> Source Package
  -> Signal
  -> route to workspace and node
  -> optionally attach to a project Node
  -> write compatibility context row for existing search
  -> extract Claim
  -> verify or review into Fact
  -> compose Memory Object, Episode Object, or Memory Detail Object
  -> connect Relationship Edges
  -> update node context/state projections
  -> assemble Context Packages for agents and apps
  -> create or refresh Active Memory Pools during work
  -> capture repeated work as Workflow Traces
  -> promote validated procedures into Skill Packages
  -> export markdown, packages, UI views, or API responses
```

This flow explains where everything belongs. The same data can appear in
markdown, a UI, an API response, and an agent context window, but each view is a
projection from governed engine state.

## Storage Roles

The system needs storage roles, not competing truths.

| Storage surface | Role | Example ownership |
| --- | --- | --- |
| Markdown workspace | Human-operable projection and editing surface. | Node pages, current status, daily rhythm, playbooks, package outputs. |
| Database | Governed canonical runtime state. | Node identity, source packages, signals, claims, facts, memory objects, relationships, workflow traces, skills, permissions, audit. |
| Indexes and caches | Rebuildable acceleration layer. | Full-text search, vectors, summaries, routing hints, L0/L1 bundles. |
| App/API responses | Interactive projection. | Dashboards, node pages, agent runs, workflow views, package views. |
| Generated HTML | Published or local visual projection. | Node pages, architecture maps, reports, status views, operating dashboards. |

The engine decides which writes are accepted as new source evidence, which edits
update topology directly, which generated files are only projections, and which
derived artifacts can be rebuilt.

## What The Engine Should Own

Optimal Engine should own:

- Workspace topology: workspaces, nodes, project Nodes, node types, relationships,
  members, agents, tools, and routing rules.
- Memory state: Source Packages, Claims, Facts, Memory Objects, Episodes,
  Details, Relationship Edges, and the Derivation Ledger.
- Retrieval state: indexes, query plans, Context Packages, authorization
  envelopes, and retrieval audit.
- Active work state: Active Memory Pools, observations, pending Claims, focus
  state, task scope, and refresh state.
- Workflow state: Workflow Traces, Generalized Workflows, Procedural Memory
  Objects, Skill Packages, validation status, and execution records.
- Governance state: policies, permissions, tool definitions, model-call
  definitions, audit events, rebuild records, and promotion decisions.

The filesystem should remain important, but it should not be the only owner of
truth.

## What The Filesystem Should Become

The filesystem should become a projection and editing interface:

```text
Engine state -> workspace markdown export
Human edit -> Source Package -> Signal -> review/promotion -> Engine state
Agent write -> artifact or observation -> review/promotion -> Engine state
```

This gives the user the same practical experience as today while making the
system safer for apps, agents, search, permissions, and long-term memory.

## Application Interface Meaning

An application built on Optimal Engine should not invent a second data model. It
should display and operate the same objects:

| App surface | Engine object behind it |
| --- | --- |
| Workspace rail | Workspaces and top-level Nodes. |
| Node page | Node profile, state, relationships, context projection, Signals, decisions, artifacts. |
| Rhythm page | Focus items, daily state, review cadence, mode transitions, active pools. |
| Agent page | Agent accounts, skills, tool permissions, active runs, audit logs. |
| Project page | Project Nodes, milestones, blockers, linked people, linked workflows. |
| Memory page | Source-backed Facts, Memory Objects, Episodes, Details, and Relationship Edges. |
| Workflow page | Workflow Traces, procedures, Skill Packages, validations, exceptions. |
| Package page | Export jobs, receiver-specific bundles, generated artifacts, source links. |
| HTML/report page | Generated projection backed by source links, context package, and export record. |

The app is therefore a control surface for the same system the agent uses
through CLI, APIs, tool calls, or local filesystem access.

## Competitive Positioning

Many agent platforms are issue-first: create an issue, assign an agent, stream
the run, then review the result. That is useful, but it treats the workspace as
execution infrastructure.

Many knowledge systems are document-first: store pages, search pages, and maybe
summarize them. That is useful, but it does not naturally model current rhythm,
node state, people, operating cadence, decisions, workflow reuse, or agent
execution.

Many database-backed apps are app-first: the database is canonical, but the user
cannot directly operate the workspace outside the app.

Many markdown workspaces are file-first: the human can inspect everything, but
the system lacks governed memory, permissions, provenance, workflow state,
active agent context, and durable execution audit.

Optimal Engine should combine the useful parts:

| Pattern | Keep | Fix |
| --- | --- | --- |
| Issue-first agent platforms | Agent runs, assignment, budgets, live execution, review. | Ground the work in Nodes, rhythm, memory, and workspace context instead of standalone issues only. |
| Document knowledge bases | Portable pages, browsing, search, references. | Add source-backed memory, temporal validity, relationships, workflows, and active agent context. |
| Database-backed operations apps | Strong identity, permissions, state, and reporting. | Preserve direct workspace control through markdown, CLI, and agent surfaces. |
| Markdown operating systems | Human-readable control, portability, local-first operation. | Add canonical topology, database-backed memory, audit, retrieval, and app/API projections. |

The product category is therefore not just an agent task runner or knowledge
base. It is a workspace runtime for human and AI operation.

## Why This Is Stronger Than A Simple Knowledge Base

A knowledge base stores pages. The current workspace already does more than
that: it stores operating state, role-specific context, project relationships,
decision history, cadence, agent work, and deliverables.

Optimal Engine should preserve that behavior while adding:

- canonical node identity instead of folder-only identity;
- source-backed memory instead of unsupported summaries;
- explicit current state instead of scattered status notes;
- task-scoped Active Memory Pools instead of implicit chat context;
- validated workflows instead of loose playbooks;
- Skill Packages instead of untracked prompts;
- governed tool/model calls instead of untracked agent actions;
- rebuildable projections instead of one-off generated files.

That is the bridge from the current personal/company operating workspace to the
full Optimal Engine runtime.
