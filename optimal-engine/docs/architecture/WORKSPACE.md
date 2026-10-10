# Workspace Architecture

The workspace layer owns the shape of a person, team, or company's operating
world. It is not a folder convention. Folders are projections of workspace state.

## Hierarchy

```text
Tenant / Organization
  -> Workspace
    -> Node graph
      -> Node
        -> sources, signals, claims, facts, memories, workflows, skills
```

Projects are optional Nodes inside a Workspace. They are not peers of
Workspace, and a workspace does not need Project Nodes if the user's world is
better organized around operations, products, people, customers, rhythm, or
domains.

```text
Workspace
  -> Entity / Company Node
  -> Team / Department Node
  -> Person Node
  -> Product Node
  -> Operational Node
  -> Project Node, when there is a bounded initiative
  -> Context Node
  -> Learning Node
```

Use a Project Node for bounded work with scope, timeline, deliverables,
milestones, blockers, and review cadence. Use another Node type for ongoing
structure or context:

| Real-world thing | Typical Node type |
| --- | --- |
| Company, client, vendor, partner | `entity` |
| Department or team | `department` / `team` |
| Launch, implementation, migration, campaign | `project` |
| Recurring process or cadence | `operation` |
| Product, platform, service, offer | `product` |
| Person, agent, stakeholder | `person` |
| Research, reference, wiki area | `learning` / `context` |

For the recommended folder projection, see
[`../guides/workspace-filesystem.md`](../guides/workspace-filesystem.md).

For scope switching rules, see
[`../guides/scope-switching.md`](../guides/scope-switching.md).

For canonical naming and aliases, see
[`../guides/naming-and-aliases.md`](../guides/naming-and-aliases.md).

## Core Objects

| Object | Meaning | Owned by |
| --- | --- | --- |
| Tenant / Organization | Governance and account boundary. | Workspace / Topology |
| Workspace | Bounded operating area with members, nodes, policies, and projections. | Workspace / Topology |
| Node | Governed unit of context, purpose, relationships, state, and activity. | Workspace / Topology |
| Node Type | Type of node such as project, person, product, operational, context, learning, entity, team, department. | Workspace / Topology |
| Node Relationship | Typed relationship between nodes. | Workspace / Topology |
| Node Member | Human, agent, service account, or team with access/role in a node. | Workspace / Topology |
| Routing Rule | Rule or hint for placing new Signals into nodes and partitions. | Workspace / Topology |
| Alias | Scoped user-facing name for a canonical object. | Workspace / Topology |

## Node Lifecycle

```text
draft
  -> active
  -> paused
  -> blocked
  -> completed
  -> archived
  -> retired
```

Every transition should record actor, reason, valid time, transaction time,
policy version, and audit event.

## Node Anatomy

Every Node has the same universal sections, even when some are empty:

```text
identity
relationships
state
focus
context
decisions
tracking
meta
```

| Section | Contains |
| --- | --- |
| Identity | name, type, purpose, owner, lifecycle state |
| Relationships | parent, children, dependencies, collaborators, people, systems |
| State | health, progress, metrics, blockers, risks |
| Focus | current priorities, active work, review items |
| Context | source packages, signals, memory objects, documents, notes |
| Decisions | pending decisions, accepted Facts, contradictions, supersessions |
| Tracking | progress logs, milestones, workflow traces, review records |
| Meta | retention policy, access policy, timestamps, custom metadata |

## Storage

Minimum topology tables:

```text
tenants / organizations
workspaces
nodes
node_types
node_aliases
node_relationships
node_members
routing_rules
topology_change_requests
```

The current implementation uses local SQLite as the default runtime store. The
target production store is Postgres. The layer model does not change when the
physical database changes.

## Write Rules

```text
Workspace / Topology may create or update workspace structure.
Routing may read topology to place Signals.
Memory Core may link knowledge to Nodes.
Retrieval may scope recall by Nodes.
Export may render Node folders and pages.
No layer should invent durable Nodes without the topology layer.
```

## Scope Switching

Scope moves from broad to narrow:

```text
Organization
  -> Workspace
      -> Node
          -> Active Memory Pool
```

Switch organization when governance, ownership, credentials, tenant, or policy
changes. Switch workspace when the operating area, node graph, rhythm, routing,
or retrieval boundary changes. Switch Node when the focus changes inside a
workspace. Open an Active Memory Pool when a bounded task starts.

Each switch changes the authorization envelope, visible objects, routing rules,
retrieval search space, tool grants, export paths, and audit scope.

## Naming And Aliases

Workspaces can use different language for the same canonical object type.

```text
node_type: project
display_label: Campaign
display_name: Product Announcement
aliases:
  - launch
  - announcement push
```

Aliases are scoped to organization, workspace, or Node. Stable IDs do not
change. Display names and aliases can change. If a loose name resolves to more
than one active object in the current scope, the engine should ask for
clarification before writing durable state.

## Human And Agent Setup

Humans can create workspaces and nodes through CLI, API, app UI, or markdown
import. Agents can propose topology, but durable topology changes should flow
through review:

```text
agent proposes node or relationship
  -> topology_change_request
  -> human/policy review
  -> accepted topology version
  -> route/context/export invalidation
```

This keeps the system customizable without letting agents silently rewrite the
workspace.
