# Sample Workspace

This sample shows the public, current workspace shape for Optimal Engine.

The files are not the whole database. They are the human-readable projection
and editing surface for a governed backend.

```text
Organization / Tenant
  -> Workspace
    -> Nodes
      -> entity-company
      -> operation-weekly-review
      -> product-customer-portal
      -> project-platform-launch
      -> learning-research-library
```

Projects are Nodes. A project is not a peer of a workspace. A workspace can
contain many project Nodes, product Nodes, person Nodes, operation Nodes,
learning Nodes, context Nodes, and other custom Node types.

## Directory Shape

```text
sample-workspace/
  workspace.yaml
  README.md
  .wiki/
    SCHEMA.md
    weekly-review.md
  assets/
    README.md
  architectures/
    customer_requirement.yaml
  nodes/
    entity-company/
    operation-weekly-review/
    product-customer-portal/
    project-platform-launch/
    learning-research-library/
```

Every Node folder follows the same projection shape:

```text
node.yaml       stable identity, type, owner, lifecycle, policy
context.md     durable context projection
signal.md      current operating state projection
signals/       source-backed event stream
packages/      outbound bundles sent to a person, team, client, partner, or channel
loops/         scheduled or repeatable agent/human loops
```

## Database Role

The database owns governed runtime state:

```text
workspaces
nodes
node_types
node_relationships
source_packages
claims
facts
memory_objects
relationship_edges
derivation_ledger
context_packages
active_memory_pools
workflow_traces
skill_packages
tool/model runs
audit events
```

Markdown is still important because humans and agents can inspect, edit, and
version it. When markdown changes, the engine should ingest it as source,
classify it as a Signal, route it, and decide what projections need rebuilding.

## Try It

```bash
mix optimal.ingest_workspace sample-workspace
mix optimal.search "launch blockers"
mix optimal.rag "what context should I know before the weekly review?"
mix optimal.wiki render-tree --workspace sample
```
