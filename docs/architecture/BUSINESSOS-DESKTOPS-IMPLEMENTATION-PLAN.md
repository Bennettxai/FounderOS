# BusinessOS Desktops Implementation Plan

## Goal

BusinessOS desktops should become shared, workspace-backed collaborative surfaces.
The target experience is an infinite desktop canvas with zoom and pan like Miro, Figma, or n8n.
Users should place modules, apps, browser shortcuts, folders, notes, and windows on a large grid instead of only inside the current viewport.
Teams should be able to open the same workspace or team desktop, see each other's cursors, and eventually edit the same canvas in real time.
The plan keeps the current desktop product moving while creating a path to the future infinite canvas model.

This document is architecture and implementation planning only.
It does not require frontend or backend code changes in the current active-tool-presence slice.
It does not change the Agents module.

## Current Foundation

The current shared desktop foundation is `workspace_desktop_spaces`.
Migration `138_workspace_desktop_spaces.sql` adds `id`, `workspace_id`, `name`, `kind`, `config`, `created_by`, `created_at`, and `updated_at`.
`kind` already represents `personal`, `team`, and `workspace` desktop spaces.
`config` stores the current frontend `DesktopSpace` shape as JSON.

The current API surface is:

```text
GET    /api/v1/workspaces/:id/desktop-spaces
PUT    /api/v1/workspaces/:id/desktop-spaces
DELETE /api/v1/workspaces/:id/desktop-spaces/:desktopSpaceId
WS     /api/v1/workspaces/:id/desktop-spaces/:desktopSpaceId/presence/ws
```

The current presence WebSocket is roomed by `workspace_id` and `desktop_space_id`.
It supports `join`, `leave`, `heartbeat`, and throttled `cursor` events.
It verifies active workspace membership before admitting a socket.
It does not yet enforce per-desktop access levels because `workspace_desktop_spaces` does not yet have an access table.

The current frontend state still centers on `windowStore`, `desktopPersistence`, `desktopTypes`, `desktopThemeStore`, and `desktopPresenceStore`.
The local store shape already has desktop spaces, icons, folders, windows, dock pins, selected icons, and active desktop ID.
The active canvas work should evolve these stores rather than replacing them in one pass.

## Product Shape

A desktop space is the durable shared surface.
A personal desktop is private to one workspace member by default.
A workspace desktop is visible to the workspace according to role and access policy.
A team desktop is visible to one or more workspace teams.
A desktop surface can have a finite viewport but stores object coordinates in canvas space.
Canvas coordinates must remain stable across zoom level, pan offset, screen size, and browser tab.

The product should support:

- Large grid or infinite canvas navigation with zoom, pan, minimap, and reset view.
- Modules, apps, windows, folders, notes, and shortcuts placed as canvas nodes.
- Configurable backgrounds including color, image, gradient, grid density, and grid visibility.
- Shared workspace and team desktops with access levels.
- Collaborative cursors and presence avatars.
- Future live editing of icon, node, window, folder, dock, and background state.
- Local fallback for offline or failed sync, without making localStorage the source of truth.

## Data Model

Keep `workspace_desktop_spaces` as the compatibility table for the next slice.
Do not introduce a parallel `desktops` table until there is a deliberate migration away from the existing name.
The table should evolve from a raw config store into the canonical desktop-space metadata record.

### `workspace_desktop_spaces`

Add columns in small migrations as the product needs them:

```sql
ALTER TABLE workspace_desktop_spaces
  ADD COLUMN IF NOT EXISTS owner_user_id VARCHAR(255),
  ADD COLUMN IF NOT EXISTS description TEXT,
  ADD COLUMN IF NOT EXISTS visibility VARCHAR(40) NOT NULL DEFAULT 'private',
  ADD COLUMN IF NOT EXISTS status VARCHAR(40) NOT NULL DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS schema_version VARCHAR(40) NOT NULL DEFAULT '1.1.0',
  ADD COLUMN IF NOT EXISTS last_saved_by VARCHAR(255),
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
```

Recommended checks:

```sql
CHECK (visibility IN ('private', 'invited', 'team', 'workspace'));
CHECK (status IN ('active', 'archived', 'deleted'));
```

`config` remains the current snapshot for hydration.
`version` gives full-snapshot saves optimistic concurrency.
`schema_version` separates local desktop config versions from future infinite-canvas schema versions.
`owner_user_id` makes personal desktop ownership explicit.

### `workspace_desktop_space_access`

Add a separate access table before shipping real team and shared workspace desktops:

```sql
CREATE TABLE workspace_desktop_space_access (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  desktop_space_id UUID NOT NULL REFERENCES workspace_desktop_spaces(id) ON DELETE CASCADE,
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  subject_type VARCHAR(40) NOT NULL CHECK (subject_type IN ('user', 'team', 'workspace')),
  subject_id VARCHAR(255) NOT NULL,
  access_level VARCHAR(40) NOT NULL CHECK (access_level IN ('viewer', 'editor', 'manager', 'owner')),
  granted_by VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (desktop_space_id, subject_type, subject_id)
);
```

Workspace-wide access uses `subject_type = 'workspace'` and `subject_id = workspace_id::text`.
Team access uses one row per team.
Direct sharing uses one row per user.

### `workspace_desktop_space_revisions`

Add revisions when conflict resolution, restore, or audit history becomes user-visible:

```sql
CREATE TABLE workspace_desktop_space_revisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  desktop_space_id UUID NOT NULL REFERENCES workspace_desktop_spaces(id) ON DELETE CASCADE,
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  config JSONB NOT NULL,
  change_summary TEXT,
  created_by VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (desktop_space_id, version)
);
```

Create revisions on explicit saves, restores, imports, and large background or layout changes.
Do not create revisions for every cursor event, drag tick, or hover.

### Future Canvas Snapshot

The future infinite canvas snapshot should be an evolution of `config`, not a second source of truth:

```json
{
  "schema_version": "2.0.0",
  "canvas": {
    "viewport": { "x": 0, "y": 0, "zoom": 1 },
    "grid": { "size": 64, "visible": true, "snap": true },
    "bounds": { "min_x": -50000, "min_y": -50000, "max_x": 50000, "max_y": 50000 }
  },
  "background": {
    "type": "color",
    "value": "#0a0a0e",
    "asset_id": null,
    "fit": "cover",
    "blur": 0,
    "dim": 0
  },
  "nodes": [],
  "icons": [],
  "folders": [],
  "windows": [],
  "dock_pinned_items": [],
  "preferences": {
    "show_presence": true,
    "show_remote_cursors": true
  }
}
```

`nodes` should become the long-term canvas primitive.
Legacy `icons`, `folders`, and `windows` can be projected into nodes during hydration until the UI is fully canvas-native.

## Frontend Modules

Keep `windowStore` as the existing reducer while the product migrates.
Add narrow modules around it instead of expanding it into persistence, collaboration, and canvas navigation.

Recommended module boundaries:

```text
frontend/src/lib/api/desktop-spaces.ts
frontend/src/lib/stores/desktopSessionStore.ts
frontend/src/lib/stores/desktopCanvasStore.ts
frontend/src/lib/stores/desktopStateSync.ts
frontend/src/lib/stores/desktopPresenceStore.ts
frontend/src/lib/components/desktop/canvas/InfiniteDesktopCanvas.svelte
frontend/src/lib/components/desktop/canvas/CanvasViewportControls.svelte
frontend/src/lib/components/desktop/canvas/CanvasMinimap.svelte
frontend/src/lib/components/desktop/canvas/RemoteCursorLayer.svelte
```

`desktopSessionStore` owns the active workspace ID, active desktop space ID, desktop metadata, access level, loaded version, save status, dirty state, and conflict state.
`desktopCanvasStore` owns pan offset, zoom, viewport bounds, grid settings, selection rectangle, and coordinate transforms.
`desktopStateSync` bridges `windowStore` and canvas state to backend saves after hydration.
`desktopPresenceStore` remains responsible for local tab presence, remote WebSocket presence, cursor publication, and remote cursor cleanup.
`InfiniteDesktopCanvas.svelte` should own pointer capture, wheel zoom, drag pan, canvas-space conversion, and node layer rendering.
`RemoteCursorLayer.svelte` should render remote cursors in canvas space so cursors remain correct while users zoom and pan independently.

The first canvas implementation can render current desktop icons and windows at canvas coordinates.
It should avoid changing module launch behavior and should keep existing `?embed=true` window routes working.

## Backend Responsibilities

The backend owns desktop space authorization, versioned saves, and room admission.
The frontend may optimistically move objects, but the backend decides whether the change is allowed and what version is accepted.

REST responsibilities:

- List accessible desktop spaces for a workspace.
- Create personal, team, and workspace desktop spaces.
- Save full `config` snapshots with `base_version`.
- Reject stale saves with `409 Conflict`.
- Validate config size, schema version, known arrays, valid window bounds, and safe URLs.
- Resolve effective access from workspace role, ownership, and `workspace_desktop_space_access`.
- Archive and delete desktop spaces without leaking another workspace's IDs.
- Return presence counts for switchers once the WebSocket hub can expose them.

WebSocket responsibilities:

- Authenticate the user through the existing session path.
- Verify active workspace membership.
- Verify desktop-space access before joining the room.
- Use room key `desktop-space:{workspace_id}:{desktop_space_id}`.
- Broadcast presence, cursors, selected nodes, and focused windows.
- Throttle and bound high-volume messages.
- Reject edit operations from viewer connections.
- Assign server sequence numbers to accepted collaboration operations.
- Expire presence on socket close, missed heartbeat, or access revocation.

Initial WebSocket events should stay presence-only:

```text
presence.joined
presence.left
presence.heartbeat
cursor.moved
selection.changed
window.focused
```

Future collaborative edit events should be added after versioned snapshots and access checks are solid:

```text
node.moved
node.created
node.updated
node.deleted
window.moved
window.resized
window.opened
window.closed
folder.updated
dock.updated
background.updated
state.saved
state.conflict
access.changed
desktop.archived
```

## Implementation Phases

### Phase 1: Stabilize Shared Desktop Spaces

Treat `workspace_desktop_spaces` as the canonical shared desktop record.
Add explicit `version`, `schema_version`, `owner_user_id`, `visibility`, and `status`.
Update saves to require `base_version` and return `409 Conflict` on stale saves.
Keep the current config envelope compatible with existing frontend `DesktopSpace`.
Keep localStorage as offline cache and recovery only.

### Phase 2: Desktop Access And Switchers

Add `workspace_desktop_space_access`.
Define effective access for owner, workspace admin, workspace member, direct user share, team share, and workspace share.
Add backend tests for workspace isolation, viewer rejection, editor save, team access, and deleted access.
Expose personal, team, and workspace desktop spaces in the nav and desktop switchers.
Show access level and presence count when available.

### Phase 3: Infinite Canvas Foundation

Add `desktopCanvasStore` and an infinite-canvas component that wraps the current desktop icon and window layers.
Store icon and window positions in canvas coordinates.
Add zoom, pan, reset view, grid size, snap, and configurable background.
Persist canvas viewport and grid preferences in `config.canvas`.
Clamp user input to large bounded coordinates even if the product calls it infinite.

### Phase 4: Figma-like Presence

Keep the existing presence WebSocket route and extend its payloads to canvas-space coordinates.
Render remote cursors in the canvas layer instead of viewport pixels.
Add remote selection, focused window, connection state, and heartbeat expiry UI.
Require desktop-space access checks on every room join.
Keep active tool presence work local to its own slice and avoid mixing it with canvas persistence changes.

### Phase 5: Collaborative Operations

Add server-accepted operation messages for moving nodes, opening windows, resizing windows, updating folders, changing backgrounds, and updating dock pins.
Persist on operation acceptance or on idle debounce depending on event type.
Use last accepted operation wins for the same object in the first release.
Keep full snapshot saves as the recovery path.
Move toward CRDT or operation log only if multi-user edits become frequent enough to justify it.

### Phase 6: Revisions, Restore, And Invites

Add `workspace_desktop_space_revisions`.
Create revisions on explicit saves, imports, restores, and large bulk edits.
Add restore UI for managers and owners.
Add desktop-specific invites only after workspace membership and desktop access semantics are clear.
Acceptance should grant desktop access, not bypass workspace membership requirements.

## Migration Risks

The biggest risk is creating two desktop models that drift.
Use `workspace_desktop_spaces` as the compatibility base until a later rename is worth the migration cost.

The second risk is treating viewport pixels as durable coordinates.
Persist canvas coordinates and convert to screen coordinates only at render time.

The third risk is allowing shared desktops before access checks are explicit.
Do not ship team or workspace editing without `workspace_desktop_space_access` or equivalent effective access checks.

The fourth risk is high-volume realtime messages overwhelming the backend.
Cursor and drag messages need throttling, size limits, room fanout bounds, and no database writes per tick.

The fifth risk is stale localStorage overwriting shared state.
Backend-backed desktop spaces must hydrate first, then local state can be used only as migration input, offline cache, or conflict recovery.

The sixth risk is snapshot conflicts causing silent data loss.
Every backend save should include `base_version`, and stale saves should return the latest config plus conflict metadata.

The seventh risk is large backgrounds or unsafe app URLs entering shared desktop state.
Backgrounds should reference managed assets where possible, and backend validation should reject unsafe URLs and oversize payloads.

## Testing

Backend tests should cover workspace isolation, personal default creation, workspace desktop access, team desktop access, viewer mutation rejection, stale save conflicts, archive behavior, and revision restore.
WebSocket tests should cover join authorization, cursor broadcast, heartbeat, leave, room isolation, access revocation behavior, and malformed message rejection.
Frontend unit tests should cover backend hydration, local fallback, canvas coordinate transforms, zoom bounds, save debounce, conflict state, and desktop switch dirty prompts.
End-to-end tests should cover two users opening the same workspace desktop, seeing remote cursors, moving an icon, refreshing, and seeing the saved position restored.

## First Vertical Slice

The next shippable slice should not attempt the whole infinite canvas.
It should make the current shared desktop reliable first.

1. Add versioned saves to `workspace_desktop_spaces`.
2. Add access rows for shared and team desktops.
3. Keep current desktop config shape and local fallback.
4. Extend the existing presence WebSocket with access checks.
5. Add canvas-space cursor payloads without changing collaborative edit semantics.
6. Add the canvas wrapper, zoom, pan, grid, and configurable background after persistence is version-safe.
