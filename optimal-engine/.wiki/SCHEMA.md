---
name: Optimal Engine Wiki Schema
version: 1
type: governance
---

# Wiki Projection Schema

Authoritative rules for the wiki/export projection surface — enforced by
`OptimalEngine.Wiki.Integrity.against_schema/2` when serialized as the
map below, and read by projection/rendering services when constructing
human-facing pages.

The wiki is not canonical truth. It is a projection over governed engine state:
workspace topology, Source Packages, Claims, Facts, Memory Objects, Context
Packages, workflow records, and audit state. Canonical truth remains in Memory
Core and its lineage records.

A page that violates the "error" rules cannot be committed. A page that
violates the "warning" rules still commits but surfaces in
`mix optimal.wiki verify`.

## Required sections (errors)

Every curated wiki page MUST contain these level-2 headings:

- `## Summary` — 2–4 paragraph overview with citations.
- `## Open threads` — active topics, decisions pending, outstanding asks.
- `## Related` — cross-links to other wiki pages via `[[slug]]`.

## Required frontmatter (errors)

Every page MUST carry this frontmatter:

- `slug` — kebab-case, unique within `(tenant_id, audience)`.
- `audience` — one of: `default`, `sales`, `engineering`, `exec-brief`, or a
  tenant-defined custom audience.
- `version` — monotonic integer, bumped by the curator on every write.
- `last_curated` — ISO-8601 timestamp of the most recent curator run.

## Citation and lineage rules (errors)

- Every factual claim in `## Summary` and `## Decisions` (if present) MUST
  carry at least one `{{cite: optimal://…}}` directive.
- Every `{{cite: uri}}` MUST resolve to governed evidence: a Source Package,
  Fact, Memory Object, asset extraction, or derivation record. Broken citations
  block the commit.
- Citations point from projection back to evidence or accepted memory. Cross-page
  links use `[[slug]]`, not `{{cite}}`.

## Size ceilings (warnings)

- `max_bytes: 50_000` — pages above this threshold should spawn child
  pages; leave a `## Related` pointer.

## Directive whitelist (errors)

Only these verbs are allowed inside `{{verb: argument [key=value…]}}`:

- `cite` — URI pointer (rendered as footnote)
- `include` — inline the referenced content, optionally at a tier
- `expand` — sub-query the wiki for a slug
- `search` — invoke hybrid retrieval
- `table` — fetch a structured row from a CSV/sheet
- `trace` — walk the knowledge graph from an entity
- `recent` — inject recent signals from a node

Anything else triggers a `:invalid_verb` error.

## Machine-readable subset

The integrity checker consumes this schema as a map with the fields:

```elixir
%{
  "required_sections" => ["Summary", "Open threads", "Related"],
  "required_frontmatter" => ["slug", "audience", "version", "last_curated"],
  "max_bytes" => 50_000
}
```

Add rules here; do not hardcode them in the checker. Schema evolution is a
governance action, not a hidden code change.
