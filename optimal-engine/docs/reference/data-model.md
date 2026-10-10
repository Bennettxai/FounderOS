# Data Model

The data model is organized by lifecycle ownership. One physical database can
hold many table groups, but each table group has one owning domain layer.

## Hierarchy

```text
Tenant / Organization
  -> Workspace
    -> Node graph
      -> Node
        -> Sources, Signals, Claims, Facts, Memories, Workflows, Skills
```

`Tenant` and `Organization` are the outer governance boundary. A single person
can run one organization for themselves. A company can run many Workspaces under
one organization. Projects, deals, engagements, campaigns, cases, accounts, and
initiatives are usually Node types or Node labels inside a Workspace, not peers
of Workspace.

## Core Lifecycle Objects

| Object | Meaning | Owner |
| --- | --- | --- |
| Source Package | Preserved raw input and metadata. | Source Intake / Memory Core |
| Asset | File, attachment, media, or raw artifact linked to a Source Package. | Memory Core |
| Signal | Classified interpretation of a source. | Signal Pipeline |
| Claim | Unaccepted assertion extracted from a source or observation. | Memory Core |
| Fact | Reviewed/policy-accepted assertion with validity and evidence. | Memory Core |
| Memory Object | Source-backed institutional meaning around Facts, Claims, and context. | Memory Core |
| Relationship Edge | Typed link among objects or nodes. | Memory Core / Topology |
| Derivation Ledger Entry | Lineage record for generated or transformed objects. | Memory Core / Audit |
| Context Package | Authorized context assembled for a query/task. | Retrieval / Context |
| Active Memory Pool | Task-scoped working state for humans and agents. | Active Collaboration |
| Workflow Trace | Evidence-linked record of work that happened. | Workflow |
| Skill Package | Governed reusable procedure with policy, tools, checks, and audit. | Workflow / Skill Runtime |

## Workspace Tables

```text
tenants / organizations
workspaces
nodes
node_types
node_relationships
node_members
routing_rules
topology_change_requests
```

## Memory Tables

```text
source_packages
assets
asset_adapter_runs
asset_extractions
asset_transcripts
asset_ocr_spans
asset_visual_observations
asset_embedding_refs
claims
facts
memory_objects
relationship_edges
derivation_ledger
```

## Retrieval And Work Tables

```text
contexts
chunks
FTS/search projections
context_packages
retrieval_audit
active_memory_pools
pool_observations
workflow_traces
generalized_workflows
procedural_memory_objects
skill_packages
```

## Governance Tables

```text
model_call_operations
mcp_tool_definitions
tool_call_runs
model_call_runs
policies
audit_events
evaluation_runs
evaluation_cases
export_records
projection_revisions
```

## Required Fields For Durable Objects

Durable governed objects should carry these fields or links when applicable:

```text
id
tenant_id
workspace_id
node_id / scope
lifecycle_state
valid_time_start / valid_time_end
transaction_time_start / transaction_time_end
confidence
precision
security_labels
partition_ids
source_package_links
derivation_ledger_links
created_by
policy_version
audit_event_links
```

## Storage Rule

```text
SQLite is the local canonical store today.
Postgres is the production canonical store target.
Raw artifact storage preserves evidence.
Indexes/caches/projections are rebuildable.
Markdown/wiki/app/API views are surfaces, not separate truth.
```

## Enterprise Rule

Multi-user use does not change the model. It makes scope mandatory:

```text
every durable write has tenant_id
every workspace object has workspace_id
every Node-scoped object has a stable node_id or explicit scope
every external action has actor_id and audit_event_links
every generated object has derivation_ledger_links
```

If a feature cannot answer which tenant, workspace, Node, actor, policy, and
source evidence own it, it is not ready for shared organizational use.
