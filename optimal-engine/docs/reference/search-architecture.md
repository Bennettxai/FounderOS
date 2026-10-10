# Search And Retrieval Architecture

Retrieval is not just semantic search. It is governed recall planning.

## Retrieval Flow

```text
query / task
  -> actor identity
  -> authorization envelope
  -> query intent
  -> subject/action/object/time/modality hints
  -> candidate generation
  -> structured filters
  -> lexical search
  -> vector search
  -> graph expansion
  -> temporal/freshness checks
  -> workflow/skill lookup
  -> evidence packaging
  -> Context Package
  -> audit
```

## Candidate Sources

| Source | Purpose |
| --- | --- |
| Facts | Accepted current or historical assertions. |
| Memory Objects | Contextual meaning around accepted knowledge. |
| Claims | Unaccepted assertions when the request explicitly asks for pending/disputed material. |
| Source Packages | Evidence inspection and citation. |
| Asset Extractions | Transcripts, OCR spans, visual observations, embedding refs. |
| Relationship Edges | Supports, contradicts, supersedes, depends_on, part_of, etc. |
| Workflow/Skill records | Process recall and reusable procedure lookup. |
| Wiki/export pages | Human-readable projections when fresh and authorized. |

## Ranking Signals

```text
permission fit
workspace/node scope
subject/action/object match
valid-time fit
transaction-time fit
freshness
confidence
precision
evidence strength
relationship distance
lexical score
vector score
workflow applicability
```

## Output

Retrieval returns Context Packages, not loose chunks:

```text
answer-ready summary
facts_used
memory_objects_used
source_links
asset_extractions_used
workflow_or_skill_links
confidence_summary
precision_summary
validity_state
filtered_object_summary
audit_event_id
```

This makes the agent's context inspectable, refreshable, and auditable.

