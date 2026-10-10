<script lang="ts">
  import { type WorkspaceRole } from '$lib/api/workspaces';
  import { Shield, Lock, Plus, Pencil, Eye } from 'lucide-svelte';
  import CreateRoleModal from './CreateRoleModal.svelte';
  import EditRoleModal from './EditRoleModal.svelte';

  interface Props {
    workspaceId: string;
    roles: WorkspaceRole[];
    canManage: boolean;
    onupdated?: () => void;
  }

  let { workspaceId, roles, canManage, onupdated }: Props = $props();

  // Modal state
  let showCreateModal = $state(false);
  let showEditModal = $state(false);
  let selectedRole = $state<WorkspaceRole | null>(null);

  function getRoleColor(role: WorkspaceRole): string {
    if (role.color) return role.color;

    const defaultColors: Record<string, string> = {
      owner: '#8b5cf6',
      admin: '#3b82f6',
      manager: '#10b981',
      member: '#6b7280',
      viewer: '#f59e0b',
      guest: '#9ca3af',
    };

    return defaultColors[role.name] || '#6b7280';
  }

  function getPermissionsCount(permissions: Record<string, Record<string, boolean | string>>): number {
    let count = 0;
    Object.values(permissions).forEach((category) => {
      Object.values(category).forEach((value) => {
        if (value === true || (typeof value === 'string' && value !== 'none')) {
          count++;
        }
      });
    });
    return count;
  }

  function handleCreateRole() {
    showCreateModal = true;
  }

  function handleEditRole(role: WorkspaceRole) {
    selectedRole = role;
    showEditModal = true;
  }

  function handleCreateSuccess() {
    showCreateModal = false;
    onupdated?.();
  }

  function handleEditSuccess() {
    showEditModal = false;
    selectedRole = null;
    onupdated?.();
  }

  function handleRoleDeleted() {
    showEditModal = false;
    selectedRole = null;
    onupdated?.();
  }
</script>

<div class="roles-list">
  <div class="list-header">
    <div>
      <h2>Roles</h2>
      <p>Define and manage workspace roles and permissions</p>
    </div>
    {#if canManage}
      <button class="btn-pill btn-pill-ghost create-button" onclick={handleCreateRole}>
        <Plus class="w-4 h-4" />
        Create Role
      </button>
    {/if}
  </div>

  <div class="roles-grid">
    {#each roles as role (role.id)}
      <div class="role-card" class:system={role.is_system}>
        <div class="role-header">
          <div
            class="role-icon"
            style="background: {getRoleColor(role)}15; color: {getRoleColor(role)}"
          >
            {#if role.is_system}
              <Lock class="w-5 h-5" />
            {:else}
              <Shield class="w-5 h-5" />
            {/if}
          </div>
          <div class="role-info">
            <div class="role-name">{role.display_name}</div>
            <div class="role-type">
              {#if role.is_system}
                <span class="badge system">System Role</span>
              {:else}
                <span class="badge custom">Custom Role</span>
              {/if}
              {#if role.is_default}
                <span class="badge default">Default</span>
              {/if}
            </div>
          </div>
        </div>

        {#if role.description}
          <p class="role-description">{role.description}</p>
        {/if}

        <div class="role-meta">
          <div class="meta-item">
            <span class="meta-label">Hierarchy Level</span>
            <span class="meta-value">{role.hierarchy_level}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">Permissions</span>
            <span class="meta-value">{getPermissionsCount(role.permissions)}</span>
          </div>
        </div>

        <div class="role-actions">
          {#if canManage}
            <button class="btn-pill btn-pill-ghost action-button" onclick={() => handleEditRole(role)}>
              <Pencil class="w-3.5 h-3.5" />
              Edit
            </button>
          {:else}
            <button class="btn-pill btn-pill-ghost action-button" onclick={() => handleEditRole(role)}>
              <Eye class="w-3.5 h-3.5" />
              View
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  {#if roles.length === 0}
    <div class="empty-state">
      <Shield class="w-12 h-12" />
      <p>No roles defined</p>
      {#if canManage}
        <button class="btn-pill btn-pill-ghost empty-create-button" onclick={handleCreateRole}>
          <Plus class="w-4 h-4" />
          Create your first role
        </button>
      {/if}
    </div>
  {/if}
</div>

<!-- Create Role Modal -->
{#if showCreateModal}
  <CreateRoleModal
    {workspaceId}
    existingRoles={roles}
    onsuccess={handleCreateSuccess}
    oncancel={() => showCreateModal = false}
  />
{/if}

<!-- Edit Role Modal -->
{#if showEditModal && selectedRole}
  <EditRoleModal
    {workspaceId}
    role={selectedRole}
    onsuccess={handleEditSuccess}
    ondeleted={handleRoleDeleted}
    oncancel={() => {
      showEditModal = false;
      selectedRole = null;
    }}
  />
{/if}

<style>
  .roles-list {
    padding: 2rem;
  }

  .list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1.5rem;
  }

  .list-header h2 {
    font-size: 1.25rem;
    font-weight: 600;
    color: #111827;
    margin: 0 0 0.5rem 0;
  }

  .list-header p {
    color: #6b7280;
    font-size: 0.875rem;
    margin: 0;
  }

  .create-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1.25rem;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
    position: relative;
  }

  .create-button:hover:not(:disabled) {
    background: #2563eb;
  }

  .create-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .roles-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1rem;
  }

  .role-card {
    padding: 1.5rem;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
    transition: all 0.15s;
  }

  .role-card:hover {
    border-color: #d1d5db;
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  }

  .role-card.system {
    background: #f9fafb;
  }

  .role-header {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .role-icon {
    width: 3rem;
    height: 3rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.5rem;
    flex-shrink: 0;
  }

  .role-info {
    flex: 1;
    min-width: 0;
  }

  .role-name {
    font-size: 1rem;
    font-weight: 600;
    color: #111827;
    margin-bottom: 0.25rem;
  }

  .role-type {
    display: flex;
    gap: 0.375rem;
    flex-wrap: wrap;
  }

  .badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 9999px;
  }

  .badge.system {
    background: #dbeafe;
    color: #1e40af;
  }

  .badge.custom {
    background: #fef3c7;
    color: #92400e;
  }

  .badge.default {
    background: #dcfce7;
    color: #166534;
  }

  .role-description {
    font-size: 0.875rem;
    color: #6b7280;
    line-height: 1.5;
    margin: 0 0 1rem 0;
  }

  .role-meta {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.75rem;
    padding: 0.75rem;
    background: #f9fafb;
    border-radius: 0.375rem;
    margin-bottom: 1rem;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .meta-label {
    font-size: 0.75rem;
    color: #6b7280;
    text-transform: uppercase;
    font-weight: 500;
  }

  .meta-value {
    font-size: 0.875rem;
    color: #111827;
    font-weight: 600;
  }

  .role-actions {
    display: flex;
    gap: 0.5rem;
    padding-top: 1rem;
    border-top: 1px solid #e5e7eb;
  }

  .action-button {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    padding: 0.5rem;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    color: #374151;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .action-button:hover:not(:disabled) {
    background: #f9fafb;
    border-color: #3b82f6;
    color: #3b82f6;
  }

  .action-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    color: #9ca3af;
    gap: 1rem;
  }

  .empty-state p {
    font-size: 0.875rem;
    margin: 0;
  }

  .empty-create-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1.25rem;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
    margin-top: 0.5rem;
  }

  .empty-create-button:hover {
    background: #2563eb;
  }

  :global(.dark) .list-header h2 {
    color: #f9fafb;
  }

  :global(.dark) .list-header p {
    color: #9ca3af;
  }

  :global(.dark) .role-card {
    background: #1f2937;
    border-color: #374151;
  }

  :global(.dark) .role-card.system {
    background: #111827;
  }

  :global(.dark) .role-name {
    color: #f9fafb;
  }

  :global(.dark) .role-meta {
    background: #111827;
  }

  :global(.dark) .meta-value {
    color: #f9fafb;
  }

  :global(.dark) .role-actions {
    border-top-color: #374151;
  }

  :global(.dark) .action-button {
    background: #111827;
    border-color: #374151;
    color: #d1d5db;
  }

  :global(.dark) .action-button:hover:not(:disabled) {
    background: #0f172a;
  }

  :global(.dark) .empty-create-button {
    background: #2563eb;
  }

  :global(.dark) .empty-create-button:hover {
    background: #1d4ed8;
  }
</style>
