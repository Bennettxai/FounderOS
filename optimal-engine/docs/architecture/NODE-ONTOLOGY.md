# Node Ontology

This document defines the canonical meaning of a Node in Optimal Engine.

A Node is not a folder. A folder can be an export of a Node, but the Node itself
is a governed topology object inside a workspace.

## Definition

A Node is a discrete unit of organized context, purpose, relationship, and
activity inside a workspace.

A Node represents:

- a bounded context with a clear purpose
- a place where related sources, signals, claims, facts, decisions, people,
  projects, workflows, and skill packages can attach
- a point in a larger topology graph
- a unit that can be tracked, reviewed, measured, and evolved

A Node is not:

- just a folder
- just a category
- just static documentation
- just a task list
- isolated information

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

Not every Node type uses every state. A project can complete. A department,
team, product, or operating function may stay active indefinitely and later be
archived or retired.

Every lifecycle transition should record:

```text
actor
reason
transaction_time
valid_time_start / valid_time_end when applicable
policy_version
audit event
```

## Universal Node Fields

Every Node should have these fields:

| Field | Meaning |
| --- | --- |
| `node_id` | Stable unique ID. |
| `workspace_id` | Workspace that owns the Node. |
| `tenant_id` | Tenant boundary. |
| `name` | Human-readable name. |
| `node_type` | Type such as entity, department, team, project, operational, learning, person, product, partnership, context. |
| `purpose` | Why the Node exists. |
| `status` | Lifecycle state. |
| `owner_id` | Primary responsible human, agent, team, or service account. |
| `parent_node_id` | Optional containing Node. |
| `health` | Red/yellow/green or policy-specific health state. |
| `progress` | Optional progress score or milestone state. |
| `review_cadence` | Expected review schedule. |
| `created_at` | Creation timestamp. |
| `updated_at` | Last update timestamp. |
| `valid_time_start` | When the Node definition became valid. |
| `valid_time_end` | When the Node definition stopped being valid. |
| `transaction_time_start` | When the store recorded this version. |
| `transaction_time_end` | When the store closed this version. |
| `metadata` | Custom workspace-specific fields. |

## Node Anatomy

Every Node has the same core sections, even if some are empty:

```text
Node
  -> Identity
  -> Relationships
  -> State
  -> Focus
  -> Context
  -> Decisions
  -> Tracking
  -> Meta
```

### Identity

```text
name
type
purpose
owner
status
```

### Relationships

```text
parent node
child nodes
lateral connections
dependencies
people involved
systems involved
workflows involved
skill packages involved
```

### State

```text
current status
health
progress
key metrics
risks
blockers
```

### Focus

```text
current priorities
active projects
this-week focus
open blockers
next review items
```

### Context

```text
source packages
signals
memory objects
context packages
documents
conversation history
notes
references
```

### Decisions

```text
pending decision queue
recent decisions
open questions
accepted facts
contradictions
supersessions
```

### Tracking

```text
progress log
metrics history
milestones
workflow traces
review records
```

### Meta

```text
created date
last updated date
update frequency
review schedule
retention policy
access policy
```

## Node Types

Node types are standardized enough for the engine to reason about them, but a
workspace can add custom fields or custom subtypes.

Projects are Nodes. They are not a mandatory hierarchy level between Workspace
and Node. A project usually appears as:

```text
Workspace
  -> Node(node_type = project)
```

Other Nodes can link to the project Node through relationships such as
`participates_in`, `depends_on`, `supports`, `owns`, or `part_of`.

## Universal Node Modeling Rule

The same rule applies to every major life, business, and work concept:

```text
Workspace is the container.
Node is the universal organized unit.
Node Type describes what kind of thing it is.
Relationships describe how Nodes connect.
Type-specific profiles exist only when extra structure is justified.
```

Do not create a new top-level table or hierarchy level for every concept just
because the concept has a name. Start with a Node and a Node Type.

Examples:

| Concept | Default model | Optional extension only if needed |
| --- | --- | --- |
| Project | `Node(node_type = project)` | `project_profiles` for milestones, budget, dates, deliverables. |
| Person | `Node(node_type = person)` | `person_profiles` for contact, role, contract, compensation. |
| Company / Entity | `Node(node_type = entity)` | `entity_profiles` for legal structure, ownership, finance. |
| Product | `Node(node_type = product)` | `product_profiles` for version, roadmap, pricing, users. |
| Operation | `Node(node_type = operational)` | `operation_profiles` for cadence, SOPs, inputs, outputs. |
| Money / Metrics | `Node(node_type = metric_area)` or `Node(node_type = finance)` | `metric_series` or finance-specific records. |
| Rhythm | `Node(node_type = rhythm)` | `rhythm_entries` for daily/weekly/monthly state. |
| Research | `Node(node_type = learning)` or `Node(node_type = context)` | source-backed research collections and claim maps. |
| Client / Customer | `Node(node_type = entity)` or custom subtype. | customer profile fields, contracts, account state. |

Use a profile table when:

- the fields are stable enough to deserve schema;
- many Nodes of that type need the same fields;
- the engine needs to query or validate those fields directly;
- the profile has lifecycle, permissions, audit, or reporting needs that generic
  metadata cannot handle cleanly.

Keep the data in `node.metadata` when:

- the fields are workspace-specific;
- the shape is still changing;
- only one or a few Nodes need the fields;
- the fields are mostly display/context, not operational logic.

| Type | Meaning | Typical contents |
| --- | --- | --- |
| `entity` | Business, organization, institution, or operating entity. | departments, teams, projects, products, financial context. |
| `department` | Functional area inside an entity. | teams, processes, KPIs, tools, budget. |
| `team` | Group of people or agents working together. | people, projects, processes, responsibilities. |
| `project` | Bounded initiative with start/end or target outcome. | tasks, milestones, deliverables, decisions, blockers. |
| `operational` | Ongoing process or business function. | SOPs, workflows, cadence, metrics, inputs, outputs. |
| `learning` | Knowledge acquisition or capability development. | resources, notes, research, frameworks. |
| `person` | Individual human, agent, partner, customer, or stakeholder. | profile, role, skills, history, relationships. |
| `product` | Product, platform, system, or offer. | features, roadmap, users, version, pricing, technical state. |
| `partnership` | Collaboration, joint venture, or external working relationship. | terms, people, projects, commitments. |
| `context` | Reference context that deserves organization but not operational ownership. | documents, research, market analysis, reference material. |

## Type-Specific Extensions

Type-specific fields should extend the universal Node fields. They should not
replace them.

### Entity Node

```text
legal_structure
ownership
revenue_model
departments
products
financials
```

### Department Node

```text
function
team_members
processes
kpis
tools
budget
```

### Project Node

```text
scope
timeline_start
timeline_end
milestones
deliverables
tasks
budget_or_value
client_or_sponsor
```

### Operational Node

```text
process_flow
sops
frequency
inputs
outputs
tools
metrics
```

### Product Node

```text
product_type
features
roadmap
users
tech_stack
version
pricing
```

### Person Node

```text
role
team_or_entity
skills
contact_profile
compensation_or_contract
history
relationships
```

## Relationship Types

Node relationships are topology edges. They are separate from Memory Core
Relationship Edges, but the two can link together.

| Relationship | Meaning |
| --- | --- |
| `parent_child` | Hierarchical containment. |
| `depends_on` | One Node depends on another. |
| `blocks` | One Node blocks another. |
| `references` | Informational connection. |
| `collaborates_with` | Nodes work together. |
| `sequence_before` | Ordered flow or process sequence. |
| `instance_of` | Specific instance of a template or product family. |
| `owns` | Ownership or responsibility. |
| `participates_in` | Person/team/entity participates in another Node. |
| `supports` | One Node provides support to another. |

Each relationship should carry:

```text
relationship_id
source_node_id
target_node_id
relationship_type
direction
strength
valid_time_start
valid_time_end
lifecycle_state
metadata
created_by
```

## Node Information Flow

Nodes participate in several flow types:

```text
upstream nodes
  -> current node
  -> downstream nodes
```

| Flow | Meaning |
| --- | --- |
| data flow | Information passes between Nodes. |
| decision flow | Decisions cascade into dependent Nodes. |
| status flow | Health/status changes propagate. |
| dependency flow | Blockers affect connected Nodes. |
| workflow flow | Work moves through operational Nodes. |
| context flow | Context Packages and Memory Objects are scoped by Node. |

## When To Create A Node

Create a Node when the thing:

- has a distinct purpose
- needs tracking over time
- has decisions made about it
- has multiple people, agents, tools, or systems interacting with it
- connects to other Nodes
- has context worth preserving
- needs access policy, review cadence, metrics, or lifecycle state

Do not create a Node when the thing:

- is a one-time task inside an existing Node
- is simple reference data that belongs in an existing Node
- has no meaningful relationships
- is temporary/disposable
- does not need lifecycle or ownership

## Node Naming

A display name should be human-readable:

```text
Project: Platform Launch
Product: Customer Portal
Operational: Hiring Pipeline
Person: Support Lead
Context: Competitor Research
```

The stable ID should be machine-readable and workspace-scoped:

```text
node_project_platform_launch
node_product_customer_portal
node_operational_hiring_pipeline
```

Display names can change. Stable IDs should not.

## Storage Model

The minimum storage model:

```text
nodes
node_types
node_relationships
node_members
node_metrics
node_reviews
node_decisions
node_focus_items
topology_change_requests
```

The workspace topology layer owns these tables.

The filesystem export can create:

```text
nodes/{node-folder}/context.md
nodes/{node-folder}/signals/
nodes/{node-folder}/decisions/
nodes/{node-folder}/workflows/
```

Those files are projections unless re-ingested as Source Packages.

## Relationship To Memory Core

Node topology tells the Memory Core where knowledge belongs.

```text
Node
  -> scopes Source Packages
  -> scopes Signals
  -> scopes Claims and Facts
  -> anchors Memory Objects
  -> constrains Retrieval Packages
  -> scopes Active Memory Pools
  -> determines Skill Package applicability
```

Memory Core Relationship Edges can reference Nodes, but they should not replace
topology relationships. Topology relationships define workspace structure.
Memory Core Relationship Edges define evidentiary, semantic, temporal, causal,
and procedural relationships among memory objects.

## Human And AI Topology Changes

Humans can create and edit Nodes directly through UI, CLI, API, or workspace
setup flows.

AI agents can propose topology changes, but durable topology changes should flow
through review:

```text
agent proposes node or relationship
  -> topology_change_request
  -> human/policy review
  -> accepted topology version
  -> route/context/export invalidation
```

This keeps the topology dynamic without letting agents silently rewrite the
workspace structure.
