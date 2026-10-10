# Workspace Creation - Quick Start Guide

## For Developers

### 1. Component Location
```
frontend/src/lib/components/workspace/
├── WorkspaceCreateModal.svelte   (NEW - 625 lines)
├── WorkspaceSwitcher.svelte      (UPDATED - 410 lines)
└── index.ts                      (UPDATED - exports new component)
```

### 2. How It Works

```typescript
// WorkspaceSwitcher.svelte
import WorkspaceCreateModal from './WorkspaceCreateModal.svelte';

let showCreateModal = false;

function openCreateModal() {
  showCreateModal = true;
  isOpen = false; // Close dropdown
}

// In template:
<WorkspaceCreateModal show={showCreateModal} onClose={closeCreateModal} />
```

### 3. Form Fields

```typescript
interface CreateWorkspaceData {
  name: string;           // Required, 3-50 chars
  description?: string;   // Optional, max 500 chars
  plan_type?: 'free' | 'starter' | 'professional' | 'enterprise';
}
```

### 4. API Flow

```typescript
async function handleSubmit() {
  // 1. Create workspace
  const workspace = await createWorkspace({
    name: name.trim(),
    description: description.trim() || undefined,
    plan_type: planType,
  });

  // 2. Update with logo if provided
  if (logoUrl) {
    await updateWorkspace(workspace.id, {
      logo_url: logoUrl.trim(),
    });
  }

  // 3. Refresh workspace list
  await initializeWorkspaces();

  // 4. Switch to new workspace
  await switchWorkspace(workspace.id);

  // 5. Close modal
  onClose();
}
```

### 5. Validation Rules

```typescript
// Name
if (!name || name.trim().length === 0) {
  errors.name = 'Workspace name is required';
} else if (name.trim().length < 3) {
  errors.name = 'Workspace name must be at least 3 characters';
} else if (name.trim().length > 50) {
  errors.name = 'Workspace name must be less than 50 characters';
}

// Description
if (description && description.length > 500) {
  errors.description = 'Description must be less than 500 characters';
}

// Logo URL
if (logoUrl && logoUrl.trim().length > 0) {
  try {
    new URL(logoUrl);
  } catch {
    errors.logoUrl = 'Please enter a valid URL';
  }
}
```

## For Users

### How to Create a Workspace

1. **Open Workspace Switcher**
   - Click the workspace button in the top navigation
   - Current workspace name is displayed

2. **Click "Create Workspace"**
   - Look for the button at the bottom of the dropdown
   - Has a dashed border and plus (+) icon

3. **Fill Out the Form**
   - **Workspace Name** (required): Enter a name for your workspace
   - **Description** (optional): Add a brief description
   - **Logo URL** (optional): Enter a URL to a logo image
   - **Plan Type** (required): Select your plan (defaults to Free)

4. **Submit**
   - Click "Create Workspace" button
   - Wait for the loading indicator
   - You'll automatically switch to your new workspace

5. **Success**
   - Modal closes
   - New workspace is selected
   - Workspace appears in the switcher dropdown

### Plan Types

- **Free**: Basic features for getting started
- **Starter**: More members and storage
- **Professional**: Advanced features for teams
- **Enterprise**: Unlimited access and support

### Tips

- Choose a descriptive name that reflects your workspace purpose
- Add a description to help team members understand the workspace
- Logo URL should be a public image URL (e.g., `https://example.com/logo.png`)
- You can always edit these details later in workspace settings

## Keyboard Shortcuts

- **ESC**: Close the create workspace modal
- **Enter**: Submit the form (when focused on an input)
- **Tab**: Navigate between form fields

## Error Messages

### Common Errors

1. **"Workspace name is required"**
   - You forgot to enter a name
   - Solution: Add a workspace name

2. **"Workspace name must be at least 3 characters"**
   - Name is too short
   - Solution: Use a longer name

3. **"Please enter a valid URL"**
   - Logo URL format is invalid
   - Solution: Use a complete URL like `https://example.com/image.png`

4. **"Description must be less than 500 characters"**
   - Description is too long
   - Solution: Shorten your description

### API Errors

If you see an error message like "Failed to create workspace", it could be:
- Network connection issue
- Server error
- Permission issue

**Solution**: Try again, or contact support if the issue persists.

## Visual Guide

### Workspace Switcher Dropdown

```
┌─────────────────────────────────┐
│ 🏢  My Workspace          ▼    │  ← Trigger Button
└─────────────────────────────────┘
         ↓ (when clicked)
┌─────────────────────────────────┐
│  Workspace 1  [workspace-1] ✓  │  ← Current workspace
│  Workspace 2  [workspace-2]    │
│  Workspace 3  [workspace-3]    │
├─────────────────────────────────┤  ← Separator
│  ┌───────────────────────────┐ │
│  │  +  Create Workspace      │ │  ← NEW BUTTON
│  └───────────────────────────┘ │
└─────────────────────────────────┘
```

### Create Workspace Modal

```
┌─────────────────────────────────────────┐
│  Create Workspace                    ✕  │  ← Header
├─────────────────────────────────────────┤
│                                          │
│  WORKSPACE NAME *                        │
│  ┌────────────────────────────────────┐ │
│  │ My Workspace                       │ │
│  └────────────────────────────────────┘ │
│                                          │
│  DESCRIPTION                             │
│  ┌────────────────────────────────────┐ │
│  │                                    │ │
│  │ A brief description...             │ │
│  │                                    │ │
│  └────────────────────────────────────┘ │
│  0/500                                   │
│                                          │
│  LOGO URL                                │
│  ┌────────────────────────────────────┐ │
│  │ https://example.com/logo.png       │ │
│  └────────────────────────────────────┘ │
│                                          │
│  PLAN TYPE                               │
│  ┌────────────┐  ┌────────────┐        │
│  │ Free    [✓]│  │ Starter    │        │
│  └────────────┘  └────────────┘        │
│  ┌────────────┐  ┌────────────┐        │
│  │Professional│  │ Enterprise │        │
│  └────────────┘  └────────────┘        │
│                                          │
├─────────────────────────────────────────┤
│                    [Cancel] [Create]    │  ← Footer
└─────────────────────────────────────────┘
```

## Code Examples

### Using the Component

```svelte
<script lang="ts">
  import { WorkspaceSwitcher } from '$lib/components/workspace';
</script>

<!-- Everything is included automatically -->
<WorkspaceSwitcher />
```

### Customizing (if needed)

```svelte
<script lang="ts">
  import { WorkspaceCreateModal } from '$lib/components/workspace';

  let showModal = false;
</script>

<button on:click={() => showModal = true}>
  Create Workspace
</button>

<WorkspaceCreateModal
  show={showModal}
  onClose={() => showModal = false}
/>
```

### Accessing Created Workspace

```svelte
<script lang="ts">
  import { currentWorkspace } from '$lib/stores/workspaces';
</script>

{#if $currentWorkspace}
  <p>Current workspace: {$currentWorkspace.name}</p>
  <p>Plan: {$currentWorkspace.plan_type}</p>
{/if}
```

## Troubleshooting

### Modal Won't Open
- Check that WorkspaceSwitcher is imported correctly
- Verify the Plus icon is visible in the dropdown
- Check browser console for errors

### Form Won't Submit
- Ensure workspace name is filled and valid
- Check that all validation passes
- Look for error messages below fields
- Check network tab for API errors

### Workspace Not Appearing
- Workspace list should refresh automatically
- Try manually refreshing the page
- Check that the API call succeeded
- Verify you have permission to create workspaces

### Not Auto-Switching
- The switchWorkspace function should be called automatically
- Check browser console for errors
- Verify the workspace was created successfully

## Support

For issues or questions:
1. Check this guide first
2. Review the detailed implementation docs
3. Check browser console for errors
4. Contact the development team

## Related Documentation

- `WORKSPACE_CREATE_IMPLEMENTATION.md` - Detailed technical documentation
- `WORKSPACE_CREATE_SUMMARY.md` - Implementation summary
- API Documentation - Workspace endpoints
- Store Documentation - Workspace state management
