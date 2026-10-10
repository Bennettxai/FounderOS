# ContentOS Architecture

## Status

ContentOS is the workspace-scoped content operations module in BusinessOS.
It manages the lifecycle from an idea through production, approval, publication, measurement, and learning.
It must support an owned creator brand, an agency's client brands, and an internal company workspace without embedding any one person's names, brands, workflow, or device-local preferences into product code.

The current implementation is a useful first pipeline.
It has a workspace-scoped `content_items` record, a board projection, a calendar projection, and basic editing.
It is not yet the complete domain model required for reliable agency operations.

## Non-Negotiable Rules

1. A workspace owns all durable ContentOS records.
2. A content record is canonical.
   Boards, calendars, dashboards, and analytics views are projections of canonical records.
3. A client profile, workstream, theme, campaign, workflow, assignment, schedule, asset, approval, and metric observation are distinct concepts.
   They cannot be represented only by unvalidated free-text fields on a content record.
4. A browser may store cosmetic per-user preferences such as panel width or a temporary filter.
   Workspace configuration, board ordering, themes, profiles, topics, workflow definitions, and color rules must be stored on the server and shared through the workspace.
5. Google Calendar is an integration target, not the ContentOS source of truth.
   A content schedule can be exported or synchronized to Google only through an explicit, traceable connection.
6. Delete actions must not silently destroy production history.
   Content records are archived or soft-deleted first and remain recoverable by an authorized workspace member.
7. Every endpoint must resolve the active workspace and verify membership before reading or writing data.
   Related records must belong to the same workspace.

## Domain Model

### Content Item

`content_items` remains the primary record for a single piece of content.

It owns the immutable identity, title, format, brief, copy or script, lifecycle state, creator, timestamps, and workspace ID.

It may reference a profile, campaign, workflow, and current version.

It must not hold an unbounded collection of unrelated text labels simply because a view needs them.

### Content Profile

A content profile is the publishing identity that owns the audience and external channels.

Examples are an agency's owned brand, a creator, or a specific client brand.

`content_profiles` owns `workspace_id`, `name`, `kind`, optional CRM client reference, default timezone, brand metadata, and active or archived state.

This replaces hard-coded frontend lists such as specific people or client names.

### Workstream And Theme

A workstream is an operational lane such as organic content, paid creative, newsletter, or long-form video.

A theme is a strategic editorial lane inside a profile or workstream.

`content_workstreams` and `content_themes` are workspace configuration records with names, ordering, color tokens, and optional descriptions.

They may be seeded by templates, but templates never become product constants that force every workspace into one agency's structure.

### Campaign

A campaign is a time-bounded commercial or editorial initiative.

`content_campaigns` stores its workspace, optional profile, name, objective, start and end dates, and status.

Content items join a campaign by foreign key.

### Workflow And Lifecycle

A workflow defines the permitted lifecycle states and transitions for a workspace or profile.

`content_workflows` and `content_workflow_states` make stages configurable while preserving a stable default template.

The default production template is `idea -> scripting -> production -> editing -> review -> approved -> scheduled -> published -> archived`.

Client approval is a review policy or state in a workflow, not a globally hard-coded rule that disappears for owned content.

Every state transition is recorded in `content_activity` with actor, timestamp, prior state, next state, and optional note.

### Assignments, Assets, And Revisions

Assignments use IDs, never display-name strings.

`content_assignments` represents a member's role such as owner, writer, editor, reviewer, or approver.

`content_assets` represents source footage, design files, drafts, review links, and live URLs.

`content_versions` and `content_reviews` preserve approval and revision history.

The current editor can remain a single screen, but it must write to these entities as the feature grows.

### Schedules And Publication

`content_schedules` represents a planned publication or production event.

It owns `content_item_id`, `workspace_id`, purpose, channel or account, start and end time with timezone, and status.

One content item can have multiple schedules.

This supports a filming appointment, review deadline, and multiple channel publications without pretending all dates are one string field.

`publication_targets` and `publication_attempts` track a connected publishing destination, external IDs, sync result, error details, and retry state.

### Analytics

Metrics are observations, not mutable columns on the content item.

`content_metric_snapshots` stores `content_item_id`, `publication_target_id`, provider, external post ID, observed at, metric name, numeric value, and raw provider payload reference.

The dashboard can derive current views, reach, engagement, retention, and trend deltas from snapshots.

This retains provenance and supports different metric definitions across Instagram, LinkedIn, YouTube, and future providers.

## Relationship Diagram

```text
workspace
  |- content_profiles
  |    |- content_workstreams
  |    |    |- content_themes
  |    |- content_campaigns
  |    |- content_workflows
  |
  |- content_items
       |- content_assignments -> workspace members
       |- content_assets
       |- content_versions
       |- content_reviews
       |- content_activity
       |- content_schedules
       |    |- publication_targets
       |    |- publication_attempts
       |- content_metric_snapshots
```

## View Contract

### Overview

The overview answers what needs attention now.

It shows counts by lifecycle state, blocked review work, upcoming schedules, overdue work, and selected performance summaries.

### Board

The board groups canonical content items by a server-defined workflow state.

The user can filter by profile, workstream, theme, campaign, assignee, channel, and date range.

Drag and drop issues a state-transition command and receives the persisted result.

It does not infer workstreams or profiles from the title text.

### Calendar

The Content Calendar renders `content_schedules`.

Moving an event changes the corresponding schedule record through a schedule command.

Calendar events are never deduplicated using title and date because two distinct content records may legitimately share both.

The item ID and schedule ID remain the identity throughout the projection.

### Content Detail

The content detail screen is the operational source for brief, copy, assignments, assets, review history, schedules, publication state, and performance.

The interface uses tabs or a compact side panel instead of forcing all fields into one unstructured form.

### Settings

ContentOS settings configure profiles, workstreams, themes, lifecycle templates, defaults, publishing integrations, and permissions.

They are workspace-scoped server records with audit history.

## Integration Contract

Google Calendar receives a schedule only after an explicit action or an enabled workspace policy.

The integration persists the external calendar ID and event ID on the publication or schedule mapping.

Updates and deletion are idempotent.

A failed external request leaves the canonical content schedule intact and records a visible sync failure for retry.

Inbound Google events remain calendar events unless an operator explicitly links or imports them into ContentOS.

No generic calendar CRUD path may guess that a calendar event is a content record from a title prefix or a synthetic user ID.

## Current-Code Constraints

The following current patterns are temporary compatibility behavior and must not grow:

- `client`, `campaign`, `owner`, `editor`, and `category` are free-text fields on `content_items`.
- `due_date` and `publish_date` are date strings without schedule identity or timezone.
- `draft`, `scheduled`, and `published` coexist with the newer pipeline stages.
- Content calendar events are synthetic frontend objects rather than persisted schedules.
- Browser local storage controls workspace themes, topics, and board order.

## Required Work Before Expanding Robert's PR

1. Do not merge the hard-coded profile, client, workstream, or theme constants.
2. Do not merge the port changes in the PR.
   BusinessOS local development remains backend `8801`, frontend `5273`, and external Optimal Engine `4200`.
3. Replace client and assignment display strings with workspace-scoped references.
4. Introduce `content_profiles`, `content_workstreams`, `content_themes`, and `content_campaigns` before shipping configurable board customization.
5. Introduce `content_schedules` before treating the content calendar as a production planning surface.
6. Make browser preferences strictly cosmetic and per-user.
   Persist shared configuration through versioned APIs.
7. Replace direct metric columns with metric snapshots before automated analytics sync is implemented.
8. Add lifecycle transition validation, an activity log, archive and restore behavior, and authorization tests.
9. Add end-to-end tests for workspace isolation, board movement, schedule movement, publish sync failure, and recovery from a failed integration request.

## Rollout Sequence

### Phase 1 - Stabilize The Existing Pipeline

Keep the current `content_items` CRUD API compatible.

Remove title-based calendar classification and duplicate suppression.

Add archive and restore semantics.

Consolidate the lifecycle states to the default workflow mapping.

### Phase 2 - Add Durable Configuration

Add profiles, workstreams, themes, campaigns, workflow definitions, and assignments.

Migrate existing labels using an explicit mapping review per workspace.

The migration must never silently create a profile, client, or person from arbitrary free text.

### Phase 3 - Scheduling And Review

Add schedules, assets, review decisions, revisions, and activity history.

Move the Content Calendar to schedules.

Add explicit Google Calendar export and sync controls with idempotent external mappings.

### Phase 4 - Publishing And Measurement

Add publishing targets, attempts, provider adapters, metric snapshots, dashboards, and AI recommendations.

AI may propose content or summaries, but it cannot bypass workspace permissions, workflow transitions, approval requirements, or publication policies.

## Acceptance Criteria

- Creating content in one workspace can never appear in another workspace.
- A new workspace can configure its own profiles, stages, themes, and colors without a code deployment.
- A person can be renamed without losing assignments.
- Two records with the same title and date both appear separately in the calendar.
- A schedule can be moved without mutating an unrelated field or losing timezone information.
- A failed Google operation is visible, retryable, and cannot delete the canonical content record.
- Every state transition, approval, revision, publish attempt, and metric observation is traceable to an actor or integration and timestamp.
- Deleting a content item is recoverable until an authorized retention policy permanently removes it.
