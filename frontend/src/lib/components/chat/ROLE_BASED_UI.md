# Role-Based UI Components

This document describes the role-based UI components for BusinessOS that integrate with the workspace role context system.

## Components

### 1. RoleContextBadge

A visual indicator showing the current user's role, permissions, and workspace context.

**Location:** `frontend/src/lib/components/chat/RoleContextBadge.svelte`

**Features:**
- Displays user's role with color-coded badge
- Shows workspace name
- Interactive tooltip with detailed permissions
- Responsive sizing options

**Props:**
```typescript
interface Props {
  showLabel?: boolean;     // Show role name text (default: true)
  size?: 'sm' | 'md';      // Badge size (default: 'sm')
  showTooltip?: boolean;   // Enable interactive tooltip (default: true)
}
```

**Usage:**
```svelte
<script>
  import { RoleContextBadge } from '$lib/components/chat';
</script>

<!-- Basic usage -->
<RoleContextBadge />

<!-- Small badge with label -->
<RoleContextBadge size="sm" showLabel={true} />

<!-- Icon only, no label -->
<RoleContextBadge showLabel={false} />

<!-- Medium size with tooltip -->
<RoleContextBadge size="md" showTooltip={true} />
```

**Role Colors:**
- `owner` - Purple (👑)
- `admin` - Blue (⚡)
- `editor` - Green (✏️)
- `member` - Yellow (👤)
- `viewer` - Gray (👁️)
- `guest` - Light Gray (🔒)

**Tooltip Content:**
- Role display name
- Hierarchy level (0-5)
- User title and department
- Key permissions list
- Expertise areas

---

### 2. PermissionGate

A wrapper component that conditionally renders children based on user permissions and role hierarchy.

**Location:** `frontend/src/lib/components/chat/PermissionGate.svelte`

**Features:**
- Resource-based permission checking
- Hierarchy level validation
- Optional fallback UI
- Type-safe permission gates

**Props:**
```typescript
interface Props {
  resource?: string;           // Permission resource (e.g., "agents", "projects")
  permission?: string;         // Permission action (e.g., "create", "edit", "delete")
  minLevel?: number;           // Minimum hierarchy level (0-5, lower = higher authority)
  maxLevel?: number;           // Maximum hierarchy level (0-5)
  showFallback?: boolean;      // Show fallback UI when denied (default: false)
  fallbackMessage?: string;    // Custom fallback message
  children?: Snippet;          // Content to render when permitted
  fallback?: Snippet;          // Custom fallback content
}
```

**Usage Examples:**

#### Check specific permission
```svelte
<PermissionGate resource="agents" permission="create">
  <button>Create Custom Agent</button>
</PermissionGate>
```

#### Check hierarchy level (only owners and admins, levels 0-1)
```svelte
<PermissionGate minLevel={1}>
  <button>Workspace Settings</button>
</PermissionGate>
```

#### Show fallback message
```svelte
<PermissionGate
  resource="projects"
  permission="delete"
  showFallback={true}
  fallbackMessage="Only project owners can delete projects."
>
  <button class="btn-danger">Delete Project</button>
</PermissionGate>
```

#### Custom fallback UI
```svelte
<PermissionGate resource="agents" permission="manage">
  <button>Manage Agents</button>
  {#snippet fallback()}
    <div class="text-gray-500">
      Upgrade to Pro to manage custom agents
    </div>
  {/snippet}
</PermissionGate>
```

#### Multiple conditions (permission AND level)
```svelte
<PermissionGate
  resource="contexts"
  permission="edit"
  minLevel={2}
>
  <button>Edit Knowledge Base</button>
</PermissionGate>
```

#### Range-based level check
```svelte
<!-- Only show to members (not owners/admins, not viewers/guests) -->
<PermissionGate minLevel={2} maxLevel={3}>
  <div>Member-only content</div>
</PermissionGate>
```

---

## Permission Structure

Permissions are stored in the `UserRoleContext` with this structure:

```typescript
interface UserRoleContext {
  user_id: string;
  workspace_id: string;
  role_name: string;              // "owner", "admin", "editor", etc.
  role_display_name: string;      // "Owner", "Administrator", etc.
  hierarchy_level: number;        // 0 (highest) to 5 (lowest)
  permissions: {
    [resource: string]: {
      [action: string]: boolean | string;
    }
  };
  title: string | null;
  department: string | null;
  expertise_areas: string[] | null;
}
```

**Example permissions object:**
```json
{
  "agents": {
    "view": true,
    "create": true,
    "edit": true,
    "delete": false
  },
  "projects": {
    "view": true,
    "create": true,
    "edit": true,
    "delete": true,
    "invite": true
  },
  "contexts": {
    "view": true,
    "create": true,
    "edit": true,
    "delete": false
  }
}
```

---

## Hierarchy Levels

Role hierarchy (lower number = higher authority):

| Level | Role | Typical Permissions |
|-------|------|---------------------|
| 0 | Owner | Full control, billing, delete workspace |
| 1 | Admin | Manage members, settings, all resources |
| 2 | Editor | Create/edit content, limited member management |
| 3 | Member | Create/edit own content, view all |
| 4 | Viewer | Read-only access to most resources |
| 5 | Guest | Very limited read-only access |

**Using hierarchy checks:**
```svelte
<!-- Only owners (level 0) -->
<PermissionGate minLevel={0} maxLevel={0}>
  <button>Delete Workspace</button>
</PermissionGate>

<!-- Owners and admins (levels 0-1) -->
<PermissionGate minLevel={1}>
  <button>Workspace Settings</button>
</PermissionGate>

<!-- Everyone except guests (levels 0-4) -->
<PermissionGate maxLevel={4}>
  <button>Create Project</button>
</PermissionGate>
```

---

## Common Patterns

### 1. Conditional Buttons
```svelte
<div class="flex gap-2">
  <!-- Always visible -->
  <button>View Details</button>

  <!-- Only if can edit -->
  <PermissionGate resource="projects" permission="edit">
    <button>Edit</button>
  </PermissionGate>

  <!-- Only if can delete -->
  <PermissionGate resource="projects" permission="delete">
    <button class="btn-danger">Delete</button>
  </PermissionGate>
</div>
```

### 2. Feature Sections
```svelte
<PermissionGate resource="agents" permission="view">
  <section>
    <h2>Custom Agents</h2>

    <div class="agents-list">
      {#each agents as agent}
        <AgentCard {agent} />
      {/each}
    </div>

    <PermissionGate resource="agents" permission="create">
      <button>Create New Agent</button>
    </PermissionGate>
  </section>
</PermissionGate>
```

### 3. Settings Panels
```svelte
<PermissionGate
  minLevel={1}
  showFallback={true}
  fallbackMessage="Only workspace administrators can access settings."
>
  <SettingsPanel />
</PermissionGate>
```

### 4. Navigation Items
```svelte
<nav>
  <a href="/chat">Chat</a>
  <a href="/projects">Projects</a>

  <PermissionGate minLevel={1}>
    <a href="/workspace/settings">Settings</a>
  </PermissionGate>

  <PermissionGate resource="team" permission="invite">
    <a href="/team/invite">Invite Members</a>
  </PermissionGate>
</nav>
```

### 5. Form Fields
```svelte
<form>
  <input type="text" bind:value={name} />

  <PermissionGate resource="projects" permission="edit">
    <select bind:value={status}>
      <option>Active</option>
      <option>Archived</option>
    </select>
  </PermissionGate>

  <PermissionGate minLevel={1}>
    <input type="checkbox" bind:checked={isPublic} />
    <label>Make public</label>
  </PermissionGate>
</form>
```

---

## Integration with Chat

The role badge is integrated into the chat header:

**File:** `frontend/src/routes/(app)/chat/+page.svelte`

```svelte
<script>
  import RoleContextBadge from '$lib/components/chat/RoleContextBadge.svelte';
</script>

<div class="chat-header">
  <!-- Left: Menu and model selector -->
  <div class="left-group">...</div>

  <!-- Center: Role badge -->
  <div class="center-group">
    <RoleContextBadge size="sm" showLabel={true} showTooltip={true} />
  </div>

  <!-- Right: Project, Node, Panel -->
  <div class="right-group">...</div>
</div>
```

The badge automatically shows:
- User's current role in the workspace
- Workspace name
- Interactive tooltip with permissions on hover/click

---

## Backend Integration

The role context is automatically loaded when switching workspaces:

**Store:** `frontend/src/lib/stores/workspaces.ts`

```typescript
// Current user's role context
export const currentUserRoleContext = writable<UserRoleContext | null>(null);

// Helper to check permissions
export const hasPermission = derived(
  currentUserRoleContext,
  ($context) => (resource: string, permission: string): boolean => {
    if (!$context) return false;
    return !!$context.permissions?.[resource]?.[permission];
  }
);

// Helper to check hierarchy level
export const isAtLeastLevel = derived(
  currentUserRoleContext,
  ($context) => (level: number): boolean => {
    if (!$context) return false;
    return $context.hierarchy_level <= level; // Lower = higher authority
  }
);
```

**Backend Endpoint:**
```
GET /api/workspaces/:workspaceId/role-context
```

Returns user's role context including permissions, which is automatically stored in `currentUserRoleContext`.

---

## Testing

To test with different role levels:

1. **Create test users** with different roles in your workspace
2. **Switch between users** in the frontend
3. **Verify UI elements** show/hide based on permissions
4. **Check tooltip content** shows correct permissions
5. **Test fallback messages** when permissions are denied

**Test Scenarios:**
- Owner (level 0) - Should see all features
- Admin (level 1) - Should see most features
- Editor (level 2) - Should see editing features
- Member (level 3) - Limited to own content
- Viewer (level 4) - Read-only
- Guest (level 5) - Very limited access

---

## Styling

The components use Tailwind CSS and match the existing BusinessOS design system.

**Badge Colors:**
- Purple: Owner
- Blue: Admin
- Green: Editor
- Yellow: Member
- Gray: Viewer/Guest

**Tooltip:**
- Dark background (`bg-gray-900`)
- White text
- Rounded corners (`rounded-lg`)
- Drop shadow (`shadow-lg`)

---

## Future Enhancements

Potential improvements:

1. **Permission Groups** - Group related permissions for cleaner checks
2. **Role Inheritance** - Define role hierarchies in config
3. **Dynamic Permissions** - Load permissions from backend dynamically
4. **Permission Explanations** - Add "Why can't I do this?" tooltips
5. **Temporary Permissions** - Support time-limited permission grants
6. **Audit Logging** - Log permission checks for security audits

---

## Resources

- **Store:** `frontend/src/lib/stores/workspaces.ts`
- **Types:** `frontend/src/lib/api/workspaces/types.ts`
- **Components:** `frontend/src/lib/components/chat/`
- **Backend Status:** See `FRONTEND_INTEGRATION_STATUS.md`
