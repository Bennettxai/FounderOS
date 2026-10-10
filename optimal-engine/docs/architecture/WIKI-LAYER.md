# Wiki And Export Layer

The wiki/export layer is a projection and control surface over governed engine
state. It makes the workspace readable and operable for humans and agents, but
it does not own truth.

## Rule

```text
Engine state -> markdown / wiki / HTML / app / API projection
Human edit   -> Source Package or topology change request
Agent write  -> observation, artifact, or pending Claim
```

## Tiers

```text
Tier 1: Source Packages and preserved raw artifacts.
Tier 2: Rebuildable indexes, chunks, summaries, embeddings, graph projections.
Tier 3: Human-facing wiki/export pages and app/API views.
```

Tier 3 is where people browse and work. Tier 1 and governed runtime tables are
where evidence and canonical lifecycle state live.

## Export Lifecycle

```text
object versions selected
  -> render markdown / HTML / API response / report
  -> record export metadata
  -> check links and citations
  -> mark freshness
  -> serve projection
  -> capture edits
  -> re-enter engine as evidence or topology change
```

Every export record should keep:

```text
workspace_id
node_id
output_path
output_kind
input_object_links
input_object_versions
source_hashes
rendered_at
rendered_by
policy_version
link_health_status
projection_freshness_status
rebuild_command
audit_event_id
```

## What The Layer Must Provide

```text
node tree rendering
markdown page rendering
HTML/report rendering
backlinks
broken-link checks
rename/move repair
revision history
freshness checks
source citations
import from existing markdown folders
safe edit handling
```

## What Must Not Happen

```text
Wiki page silently becomes accepted truth.
Agent edits final Facts directly in markdown.
Generated page loses source links.
Projection drift is hidden from retrieval.
User edits are overwritten without an export/edit record.
```

The wiki is important because it is the human-readable operating surface. It is
safe because every durable knowledge change goes back through Source Packages,
Claims, Facts, Memory Objects, and audit.

