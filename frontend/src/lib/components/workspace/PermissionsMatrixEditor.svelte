<script lang="ts">
  import { Check, X, Lock } from 'lucide-svelte';

  interface Props {
    permissions: Record<string, Record<string, boolean | string>>;
    readonly?: boolean;
    compact?: boolean;
    onchange?: (permissions: Record<string, Record<string, boolean | string>>) => void;
  }

  let { permissions = $bindable({}), readonly = false, compact = false, onchange }: Props = $props();

  // Default permission categories with their actions
  const permissionSchema: Record<string, { label: string; actions: { key: string; label: string }[] }> = {
    projects: {
      label: 'Projects',
      actions: [
        { key: 'create', label: 'Create' },
        { key: 'read', label: 'View' },
        { key: 'update', label: 'Edit' },
        { key: 'delete', label: 'Delete' },
        { key: 'manage_members', label: 'Manage Members' },
      ],
    },
    tasks: {
      label: 'Tasks',
      actions: [
        { key: 'create', label: 'Create' },
        { key: 'read', label: 'View' },
        { key: 'update', label: 'Edit' },
        { key: 'delete', label: 'Delete' },
        { key: 'assign', label: 'Assign' },
      ],
    },
    contexts: {
      label: 'Contexts',
      actions: [
        { key: 'create', label: 'Create' },
        { key: 'read', label: 'View' },
        { key: 'update', label: 'Edit' },
        { key: 'delete', label: 'Delete' },
      ],
    },
    clients: {
      label: 'Clients',
      actions: [
        { key: 'create', label: 'Create' },
        { key: 'read', label: 'View' },
        { key: 'update', label: 'Edit' },
        { key: 'delete', label: 'Delete' },
      ],
    },
    artifacts: {
      label: 'Artifacts',
      actions: [
        { key: 'create', label: 'Create' },
        { key: 'read', label: 'View' },
        { key: 'update', label: 'Edit' },
        { key: 'delete', label: 'Delete' },
      ],
    },
    members: {
      label: 'Members',
      actions: [
        { key: 'view', label: 'View' },
        { key: 'invite', label: 'Invite' },
        { key: 'manage', label: 'Manage' },
      ],
    },
    roles: {
      label: 'Roles',
      actions: [
        { key: 'view', label: 'View' },
        { key: 'manage', label: 'Manage' },
      ],
    },
    workspace: {
      label: 'Workspace',
      actions: [
        { key: 'view', label: 'View Settings' },
        { key: 'manage', label: 'Manage Settings' },
      ],
    },
    agent: {
      label: 'AI Agent',
      actions: [
        { key: 'use_all_agents', label: 'Use All Agents' },
        { key: 'create_custom_agents', label: 'Create Custom Agents' },
        { key: 'access_workspace_memory', label: 'Access Workspace Memory' },
        { key: 'modify_workspace_memory', label: 'Modify Workspace Memory' },
      ],
    },
  };

  // Ensure all categories exist in permissions
  function ensurePermissionStructure(): void {
    let updated = false;
    for (const [category, schema] of Object.entries(permissionSchema)) {
      if (!permissions[category]) {
        permissions[category] = {};
        updated = true;
      }
      for (const action of schema.actions) {
        if (permissions[category][action.key] === undefined) {
          permissions[category][action.key] = false;
          updated = true;
        }
      }
    }
    if (updated && onchange) {
      onchange(permissions);
    }
  }

  // Initialize on mount
  $effect(() => {
    ensurePermissionStructure();
  });

  function togglePermission(category: string, action: string): void {
    if (readonly) return;
    
    const currentValue = permissions[category]?.[action];
    permissions[category] = {
      ...permissions[category],
      [action]: !currentValue,
    };
    
    if (onchange) {
      onchange(permissions);
    }
  }

  function toggleAllInCategory(category: string, value: boolean): void {
    if (readonly) return;
    
    const schema = permissionSchema[category];
    if (!schema) return;
    
    const newCategoryPerms: Record<string, boolean | string> = {};
    for (const action of schema.actions) {
      newCategoryPerms[action.key] = value;
    }
    permissions[category] = newCategoryPerms;
    
    if (onchange) {
      onchange(permissions);
    }
  }

  function getCategoryStatus(category: string): 'all' | 'some' | 'none' {
    const schema = permissionSchema[category];
    if (!schema) return 'none';
    
    const categoryPerms = permissions[category] || {};
    const values = schema.actions.map(a => !!categoryPerms[a.key]);
    
    if (values.every(v => v)) return 'all';
    if (values.some(v => v)) return 'some';
    return 'none';
  }

  function getPermissionValue(category: string, action: string): boolean {
    return !!permissions[category]?.[action];
  }
</script>

<div class="permissions-matrix" class:compact class:readonly>
  {#each Object.entries(permissionSchema) as [category, schema]}
    <div class="permission-category">
      <div class="category-header">
        <div class="category-title">
          <span class="category-name">{schema.label}</span>
          {#if !readonly}
            {@const status = getCategoryStatus(category)}
            <button
              type="button"
              class="toggle-all-btn"
              class:active={status === 'all'}
              class:partial={status === 'some'}
              onclick={() => toggleAllInCategory(category, status !== 'all')}
              title={status === 'all' ? 'Disable all' : 'Enable all'}
            >
              {#if status === 'all'}
                <Check class="w-3 h-3" />
              {:else if status === 'some'}
                <span class="partial-indicator"></span>
              {/if}
            </button>
          {/if}
        </div>
      </div>
      
      <div class="permission-actions">
        {#each schema.actions as action}
          {@const isEnabled = getPermissionValue(category, action.key)}
          <button
            type="button"
            class="btn-pill btn-pill-ghost permission-toggle"
            class:enabled={isEnabled}
            class:readonly
            disabled={readonly}
            onclick={() => togglePermission(category, action.key)}
            title={action.label}
          >
            <span class="toggle-indicator">
              {#if isEnabled}
                <Check class="w-3 h-3" />
              {:else}
                <X class="w-3 h-3" />
              {/if}
            </span>
            <span class="toggle-label">{action.label}</span>
          </button>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .permissions-matrix {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .permissions-matrix.compact {
    gap: 0.75rem;
  }

  .permission-category {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
    overflow: hidden;
  }

  .category-header {
    padding: 0.75rem 1rem;
    background: white;
    border-bottom: 1px solid #e5e7eb;
  }

  .category-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .category-name {
    font-size: 0.875rem;
    font-weight: 600;
    color: #374151;
  }

  .toggle-all-btn {
    width: 1.25rem;
    height: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: white;
    border: 2px solid #d1d5db;
    border-radius: 0.25rem;
    cursor: pointer;
    transition: all 0.15s;
    color: white;
  }

  .toggle-all-btn:hover {
    border-color: #3b82f6;
  }

  .toggle-all-btn.active {
    background: #3b82f6;
    border-color: #3b82f6;
  }

  .toggle-all-btn.partial {
    border-color: #3b82f6;
  }

  .partial-indicator {
    width: 0.5rem;
    height: 0.5rem;
    background: #3b82f6;
    border-radius: 0.125rem;
  }

  .permission-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
  }

  .permission-toggle {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.625rem;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 0.375rem;
    font-size: 0.75rem;
    color: #6b7280;
    cursor: pointer;
    transition: all 0.15s;
  }

  .permission-toggle:hover:not(:disabled) {
    border-color: #d1d5db;
    background: #f9fafb;
  }

  .permission-toggle.enabled {
    background: #eff6ff;
    border-color: #3b82f6;
    color: #1e40af;
  }

  .permission-toggle.readonly {
    cursor: default;
  }

  .permission-toggle:disabled {
    opacity: 0.7;
  }

  .toggle-indicator {
    width: 1rem;
    height: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.25rem;
  }

  .permission-toggle.enabled .toggle-indicator {
    background: #3b82f6;
    color: white;
  }

  .permission-toggle:not(.enabled) .toggle-indicator {
    background: #e5e7eb;
    color: #9ca3af;
  }

  .toggle-label {
    font-weight: 500;
  }

  /* Compact mode */
  .compact .category-header {
    padding: 0.5rem 0.75rem;
  }

  .compact .permission-actions {
    padding: 0.5rem 0.75rem;
    gap: 0.375rem;
  }

  .compact .permission-toggle {
    padding: 0.25rem 0.5rem;
    font-size: 0.6875rem;
  }

  /* Dark mode */
  :global(.dark) .permission-category {
    background: #1f2937;
    border-color: #374151;
  }

  :global(.dark) .category-header {
    background: #111827;
    border-bottom-color: #374151;
  }

  :global(.dark) .category-name {
    color: #f3f4f6;
  }

  :global(.dark) .toggle-all-btn {
    background: #1f2937;
    border-color: #4b5563;
  }

  :global(.dark) .permission-toggle {
    background: #1f2937;
    border-color: #374151;
    color: #9ca3af;
  }

  :global(.dark) .permission-toggle:hover:not(:disabled) {
    background: #111827;
    border-color: #4b5563;
  }

  :global(.dark) .permission-toggle.enabled {
    background: #1e3a8a;
    border-color: #3b82f6;
    color: #93c5fd;
  }

  :global(.dark) .permission-toggle:not(.enabled) .toggle-indicator {
    background: #374151;
    color: #6b7280;
  }
</style>
