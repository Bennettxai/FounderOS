<script lang="ts">
  import {
    updateWorkspaceMemberRole,
    type WorkspaceMember,
    type WorkspaceRole,
  } from '$lib/api/workspaces';
  import { X, Shield, Save, Loader2, AlertCircle } from 'lucide-svelte';

  interface Props {
    workspaceId: string;
    member: WorkspaceMember;
    roles: WorkspaceRole[];
    currentUserRole: string;
    onsuccess?: () => void;
    oncancel?: () => void;
  }

  let { workspaceId, member, roles, currentUserRole, onsuccess, oncancel }: Props = $props();

  let selectedRole = $state(member.role);
  let isSaving = $state(false);
  let error = $state<string | null>(null);

  // Filter roles based on hierarchy - users can only assign roles below their level
  const roleHierarchy: Record<string, number> = {
    owner: 100,
    admin: 80,
    manager: 60,
    member: 40,
    viewer: 20,
    guest: 10,
  };

  const currentUserLevel = roleHierarchy[currentUserRole] || 0;
  const assignableRoles = roles.filter((role) => {
    const roleLevel = roleHierarchy[role.name] || 0;
    return roleLevel < currentUserLevel && role.name !== 'owner';
  });

  async function handleSave() {
    if (selectedRole === member.role) {
      oncancel?.();
      return;
    }

    try {
      isSaving = true;
      error = null;

      await updateWorkspaceMemberRole(workspaceId, member.id, {
        role: selectedRole,
      });

      onsuccess?.();
    } catch (err) {
      console.error('Failed to update member role:', err);
      error = err instanceof Error ? err.message : 'Failed to update member role';
    } finally {
      isSaving = false;
    }
  }

  function handleCancel() {
    oncancel?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleCancel();
    } else if (e.key === 'Enter' && selectedRole && !isSaving) {
      handleSave();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-overlay" onclick={handleCancel}>
  <div class="modal-content" onclick={(e) => e.stopPropagation()}>
    <div class="modal-header">
      <div>
        <h2>Change Member Role</h2>
        <p>Update the role for this member</p>
      </div>
      <button class="btn-pill btn-pill-ghost close-button" onclick={handleCancel} type="button">
        <X class="w-5 h-5" />
      </button>
    </div>

    {#if error}
      <div class="error-message">
        <AlertCircle class="w-4 h-4" />
        <span>{error}</span>
      </div>
    {/if}

    <div class="modal-body">
      <div class="member-info">
        <div class="member-avatar">
          {member.user_id.charAt(0).toUpperCase()}
        </div>
        <div>
          <div class="member-id">User {member.user_id.slice(0, 8)}...</div>
          <div class="current-role">Current role: {member.role}</div>
        </div>
      </div>

      <div class="form-group">
        <label for="role">
          <Shield class="w-4 h-4" />
          New Role
        </label>
        {#if assignableRoles.length > 0}
          <select id="role" bind:value={selectedRole} disabled={isSaving} autofocus>
            {#each assignableRoles as role (role.id)}
              <option value={role.name}>
                {role.display_name}
                {#if role.description}
                  - {role.description}
                {/if}
              </option>
            {/each}
          </select>
          <p class="field-hint">Select the new role for this member</p>
        {:else}
          <div class="no-roles-message">
            <AlertCircle class="w-4 h-4" />
            <span>You don't have permission to assign any roles to this member</span>
          </div>
        {/if}
      </div>
    </div>

    <div class="modal-footer">
      <button class="btn-pill btn-pill-ghost cancel-button" onclick={handleCancel} type="button" disabled={isSaving}>
        Cancel
      </button>
      <button
        class="btn-pill btn-pill-ghost save-button"
        onclick={handleSave}
        disabled={!selectedRole || selectedRole === member.role || isSaving || assignableRoles.length === 0}
        type="button"
      >
        {#if isSaving}
          <Loader2 class="w-4 h-4 animate-spin" />
        {:else}
          <Save class="w-4 h-4" />
        {/if}
        Save Changes
      </button>
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1rem;
  }

  .modal-content {
    width: 100%;
    max-width: 500px;
    background: var(--dbg);
    border-radius: 0.75rem;
    box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 1.5rem;
    border-bottom: 1px solid var(--dbd);
  }

  .modal-header h2 {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--dt);
    margin: 0 0 0.25rem 0;
  }

  .modal-header p {
    color: var(--dt2);
    font-size: 0.875rem;
    margin: 0;
  }

  .close-button {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.25rem;
    background: transparent;
    border: none;
    color: var(--dt3);
    cursor: pointer;
    border-radius: 0.25rem;
    transition: all 0.15s;
  }

  .close-button:hover {
    background: var(--dbg2);
    color: var(--dt);
  }

  .error-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1.5rem;
    background: var(--bos-background-error-color, #fef2f2);
    color: var(--bos-error-color);
    font-size: 0.875rem;
    border-bottom: 1px solid color-mix(in srgb, var(--bos-error-color) 20%, var(--dbg));
  }

  .modal-body {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .member-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem;
    background: var(--dbg2);
    border-radius: 0.5rem;
  }

  .member-avatar {
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: var(--bos-surface-on-color);
    font-weight: 600;
    font-size: 0.875rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .member-id {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--dt);
  }

  .current-role {
    font-size: 0.75rem;
    color: var(--dt2);
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-group label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--dt2);
  }

  .form-group select {
    padding: 0.625rem 0.875rem;
    border: 1px solid var(--dbd);
    border-radius: 0.375rem;
    font-size: 0.875rem;
    background: var(--dbg);
    color: var(--dt);
    transition: all 0.15s;
  }

  .form-group select:focus {
    outline: none;
    border-color: var(--bos-primary-color);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--bos-primary-color) 15%, transparent);
  }

  .form-group select:disabled {
    background: var(--dbg2);
    color: var(--dt3);
    cursor: not-allowed;
  }

  .field-hint {
    font-size: 0.75rem;
    color: var(--dt2);
    margin: 0;
  }

  .no-roles-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem;
    background: var(--bos-status-warning-bg);
    color: var(--bos-status-warning);
    font-size: 0.875rem;
    border-radius: 0.375rem;
  }

  .modal-footer {
    display: flex;
    gap: 0.75rem;
    padding: 1.5rem;
    border-top: 1px solid var(--dbd);
  }

  .cancel-button,
  .save-button {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.625rem 1.25rem;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .cancel-button {
    background: var(--dbg2);
    color: var(--dt2);
    border: 1px solid var(--dbd);
  }

  .cancel-button:hover:not(:disabled) {
    background: var(--dbg3);
  }

  .save-button {
    background: var(--bos-primary-color);
    color: white;
  }

  .save-button:hover:not(:disabled) {
    filter: brightness(0.9);
  }

  .save-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
