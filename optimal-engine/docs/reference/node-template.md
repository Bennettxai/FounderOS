# Node Template

A Node is a governed topology object. A markdown folder or page is an export of
that Node.

## Node Fields

```yaml
node_id: node-project-launch
workspace_id: default:workspace
name: Project Launch
node_type: project
purpose: Coordinate launch work and preserve decisions.
owner_id: person-owner
status: active
health: yellow
review_cadence: weekly
parent_node_id:
metadata: {}
```

## Markdown Projection

`context.md` should represent slow-changing node context:

```markdown
# Project Launch

## Identity

| Field | Value |
| --- | --- |
| Type | project |
| Owner | Product lead |
| Status | active |
| Health | yellow |
| Review | weekly |

## Purpose

Coordinate launch work and preserve source-backed decisions.

## Relationships

| Node | Relationship | Meaning |
| --- | --- | --- |
| Product | parent | Launch belongs to the product. |
| Sales | depends_on | Launch needs sales enablement. |

## Decisions

| Date | Decision | Evidence |
| --- | --- | --- |
| 2026-06-11 | Example decision | Source Package link |
```

`signal.md` should represent current operating state:

```markdown
# Project Launch Signal

## Focus

1. Confirm release checklist.
2. Resolve launch blocker.
3. Refresh customer-facing docs.

## Blockers

| Blocker | Owner | Since |
| --- | --- | --- |
| Example blocker | Product lead | 2026-06-11 |
```

Edits to these files should re-enter as source evidence or topology changes.

