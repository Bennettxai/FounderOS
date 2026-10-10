# Project Access Control UI - Usage Guide

## Overview

This module provides a complete UI for managing project members and their permissions based on the role-based access control system.

## Components

### 1. ProjectMembersPanel (Main Component)

The main component that displays all project members and handles member management.

**Props:**
- `projectId: string` - The project ID
- `workspaceId: string` - The workspace ID
- `currentUserId: string` - Current user's ID
- `userRole?: ProjectRole` - Current user's role (default: 'viewer')
- `canInvite?: boolean` - Whether user can invite members (default: false)

**Features:**
- Lists all project members with their roles
- Search and filter members
- Add new members (if permission granted)
- Update member roles (if permission granted)
- Remove members (if permission granted)
- Role distribution statistics
- Permission-based UI (shows actions based on user role)

**Example:**
```svelte
<script>
  import { ProjectMembersPanel } from '$lib/components/projects';

  let projectId = 'proj_123';
  let workspaceId = 'ws_456';
  let currentUserId = 'user_789';
  let userRole = 'lead'; // or 'contributor', 'reviewer', 'viewer'
  let canInvite = true;
</script>

<ProjectMembersPanel
  {projectId}
  {workspaceId}
  {currentUserId}
  {userRole}
  {canInvite}
/>
```

### 2. MemberCard

Displays individual member information with role and permissions.

**Props:**
- `member: ProjectMember` - Member data
- `canEdit?: boolean` - Can edit member role
- `canRemove?: boolean` - Can remove member
- `currentUserId?: string` - Current user ID (to highlight "You")
- `onRoleChange?: (memberId, newRole) => void` - Role change callback
- `onRemove?: (memberId) => void` - Remove member callback

**Example:**
```svelte
<script>
  import { MemberCard } from '$lib/components/projects';

  let member = {
    id: 'mem_123',
    user_id: 'user_456',
    user_name: 'John Doe',
    user_email: 'john@example.com',
    role: 'contributor',
    can_edit: true,
    can_delete: false,
    can_invite: false,
    // ... other fields
  };

  function handleRoleChange(memberId, newRole) {
    console.log('Role changed:', memberId, newRole);
  }

  function handleRemove(memberId) {
    console.log('Remove member:', memberId);
  }
</script>

<MemberCard
  {member}
  canEdit={true}
  canRemove={true}
  currentUserId="user_789"
  onRoleChange={handleRoleChange}
  onRemove={handleRemove}
/>
```

### 3. AddMemberModal

Modal dialog for adding new members to the project.

**Props:**
- `open?: boolean` - Modal open state (bindable)
- `workspaceId: string` - Workspace ID
- `onClose?: () => void` - Close callback
- `onAdd?: (data) => void` - Add member callback

**Example:**
```svelte
<script>
  import { AddMemberModal } from '$lib/components/projects';

  let modalOpen = false;
  let workspaceId = 'ws_123';

  async function handleAddMember(data) {
    console.log('Adding member:', data);
    // data = { user_id, role, workspace_id }
    // Call API to add member
  }
</script>

<button on:click={() => modalOpen = true}>
  Add Member
</button>

<AddMemberModal
  bind:open={modalOpen}
  {workspaceId}
  onAdd={handleAddMember}
/>
```

### 4. RoleSelector

Dropdown selector for project roles.

**Props:**
- `value: ProjectRole` - Selected role (bindable)
- `disabled?: boolean` - Disable selector
- `onChange?: (role) => void` - Change callback

**Example:**
```svelte
<script>
  import { RoleSelector } from '$lib/components/projects';

  let selectedRole = 'viewer';

  function handleRoleChange(role) {
    console.log('New role:', role);
  }
</script>

<RoleSelector
  bind:value={selectedRole}
  onChange={handleRoleChange}
/>
```

## API Functions

All API functions are available in `$lib/api/projects/members.ts`:

```typescript
import {
  listProjectMembers,
  addProjectMember,
  updateProjectMemberRole,
  removeProjectMember,
  checkProjectAccess
} from '$lib/api/projects/members';

// List all members
const members = await listProjectMembers(projectId);

// Add a member
const newMember = await addProjectMember(projectId, {
  user_id: 'user_123',
  role: 'contributor',
  workspace_id: 'ws_456'
});

// Update member role
const updated = await updateProjectMemberRole(projectId, memberId, {
  role: 'lead'
});

// Remove member
await removeProjectMember(projectId, memberId);

// Check access
const access = await checkProjectAccess(projectId, userId);
// Returns: { has_access, role, can_edit, can_delete, can_invite }
```

## Project Roles

### Lead
- **Permissions**: Full control
- **Can do**: Edit, Delete, Invite members
- **Color**: Purple
- **Icon**: Shield

### Contributor
- **Permissions**: Edit only
- **Can do**: Edit project content
- **Color**: Blue
- **Icon**: Edit3

### Reviewer
- **Permissions**: Review and comment
- **Can do**: Review and comment
- **Color**: Green
- **Icon**: Users

### Viewer
- **Permissions**: Read-only
- **Can do**: View project
- **Color**: Gray
- **Icon**: Eye

## Integration Example

Here's a complete example of integrating the ProjectMembersPanel into a project detail page:

```svelte
<!-- src/routes/(app)/projects/[id]/+page.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  import { ProjectMembersPanel } from '$lib/components/projects';
  import { currentWorkspace, currentUserRole } from '$lib/stores/workspaces';
  import { checkProjectAccess } from '$lib/api/projects/members';
  import { onMount } from 'svelte';

  let projectId = $page.params.id;
  let userAccess = $state(null);
  let loading = $state(true);

  onMount(async () => {
    try {
      // Get current user's access level
      userAccess = await checkProjectAccess(projectId, $currentUser.id);
    } catch (error) {
      console.error('Failed to check access:', error);
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="flex items-center justify-center p-8">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>
{:else if !userAccess?.has_access}
  <div class="p-8 text-center">
    <p class="text-red-600">You don't have access to this project</p>
  </div>
{:else}
  <div class="container mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">Project Details</h1>

    <!-- Project info here -->

    <!-- Members Panel -->
    <div class="mt-8">
      <ProjectMembersPanel
        {projectId}
        workspaceId={$currentWorkspace.id}
        currentUserId={$currentUser.id}
        userRole={userAccess.role}
        canInvite={userAccess.can_invite}
      />
    </div>
  </div>
{/if}
```

## Styling

All components use Tailwind CSS and follow the existing BusinessOS design patterns:

- **Primary color**: Blue (#2563eb)
- **Borders**: Gray-200
- **Hover states**: Gray-50/100
- **Rounded corners**: 0.5rem (rounded-lg) to 1rem (rounded-xl)
- **Icons**: Lucide Svelte icons
- **Dialogs**: Bits UI components

## Error Handling

All API calls include proper error handling:

```typescript
try {
  await addProjectMember(projectId, data);
  // Success handling
} catch (error) {
  // Error is displayed in the component
  console.error('Failed to add member:', error);
}
```

Errors are displayed inline using AlertCircle icons and red backgrounds.

## Permission Checks

The UI automatically adjusts based on user permissions:

- Only users with `can_invite` permission see the "Add Member" button
- Only users with `can_edit` (leads/contributors) can change roles
- Only project leads can remove members
- Users cannot modify their own role or remove themselves

## Backend Endpoints

The components integrate with these backend endpoints:

- `GET /api/projects/:id/members` - List members
- `POST /api/projects/:id/members` - Add member
- `PUT /api/projects/:id/members/:memberId/role` - Update role
- `DELETE /api/projects/:id/members/:memberId` - Remove member
- `GET /api/projects/:id/access/:userId` - Check access

## Types

All TypeScript types are defined in `$lib/api/projects/types.ts`:

```typescript
export type ProjectRole = 'lead' | 'contributor' | 'reviewer' | 'viewer';
export type MemberStatus = 'active' | 'inactive' | 'removed';

export interface ProjectMember {
  id: string;
  project_id: string;
  user_id: string;
  workspace_id: string;
  role: ProjectRole;
  can_edit: boolean;
  can_delete: boolean;
  can_invite: boolean;
  // ... other fields
}
```

## Migration

The backend uses migration `029_project_members.sql` which creates:
- `project_members` table
- `project_role_definitions` table
- Helper functions for access control
- Audit logging triggers

## Best Practices

1. **Always check permissions** before showing UI elements
2. **Use the ProjectMembersPanel** for complete member management
3. **Handle errors gracefully** and show user-friendly messages
4. **Reload data after mutations** to ensure consistency
5. **Use optimistic updates** with rollback on error
6. **Validate user input** before API calls
7. **Show loading states** during async operations

## Testing

Test the components by:

1. Creating a project with multiple members
2. Testing each role's permissions
3. Adding/removing members
4. Updating roles
5. Searching and filtering
6. Testing permission boundaries

## Support

For issues or questions:
1. Check the backend migration file: `029_project_members.sql`
2. Review API endpoint implementations
3. Test with different user roles
4. Verify workspace context is properly set
