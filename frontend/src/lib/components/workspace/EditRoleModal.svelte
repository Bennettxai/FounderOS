<script lang="ts">
  import {
    updateWorkspaceRole,
    deleteWorkspaceRole,
    type WorkspaceRole,
  } from '$lib/api/workspaces';
  import {
    X,
    Shield,
    Save,
    Loader2,
    AlertCircle,
    Palette,
    Trash2,
    AlertTriangle,
  } from 'lucide-svelte';
  import PermissionsMatrixEditor from './PermissionsMatrixEditor.svelte';

  interface Props {
    workspaceId: string;
    role: WorkspaceRole;
    onsuccess?: () => void;
    ondeleted?: () => void;
    oncancel?: () => void;
  }

  let { workspaceId, role, onsuccess, ondeleted, oncancel }: Props = $props();

  // Form state (initialized from role)
  let displayName = $state(role.display_name);
  let description = $state(role.description || '');
  let color = $state(role.color || '#6366f1');
  let permissions = $state<Record<string, Record<string, boolean | string>>>(
    JSON.parse(JSON.stringify(role.permissions || {}))
  );

  // UI state
  let isSaving = $state(false);
  let isDeleting = $state(false);
  let error = $state<string | null>(null);
  let showDeleteConfirm = $state(false);
  let activeTab = $state<'general' | 'permissions'>('general');

  // Predefined colors
  const colorOptions = [
    { value: '#8b5cf6', label: 'Purple' },
    { value: '#3b82f6', label: 'Blue' },
    { value: '#10b981', label: 'Green' },
    { value: '#f59e0b', label: 'Amber' },
    { value: '#ef4444', label: 'Red' },
    { value: '#ec4899', label: 'Pink' },
    { value: '#06b6d4', label: 'Cyan' },
    { value: '#6366f1', label: 'Indigo' },
  ];

  // Check if role is system role (can't be deleted)
  const isSystemRole = $derived(role.is_system);

  // Validation
  const isValid = $derived(() => {
    if (!displayName.trim()) return false;
    return true;
  });

  const hasChanges = $derived(() => {
    if (displayName !== role.display_name) return true;
    if (description !== (role.description || '')) return true;
    if (color !== (role.color || '#6366f1')) return true;
    if (JSON.stringify(permissions) !== JSON.stringify(role.permissions || {})) return true;
    return false;
  });

  async function handleSave() {
    if (!isValid() || !hasChanges()) return;

    try {
      isSaving = true;
      error = null;

      await updateWorkspaceRole(workspaceId, role.id, {
        display_name: displayName.trim(),
        description: description.trim() || undefined,
        color,
        permissions,
      });

      onsuccess?.();
    } catch (err) {
      console.error('Failed to update role:', err);
      error = err instanceof Error ? err.message : 'Failed to update role';
    } finally {
      isSaving = false;
    }
  }

  async function handleDelete() {
    if (isSystemRole) return;

    try {
      isDeleting = true;
      error = null;

      await deleteWorkspaceRole(workspaceId, role.id);

      ondeleted?.();
    } catch (err) {
      console.error('Failed to delete role:', err);
      error = err instanceof Error ? err.message : 'Failed to delete role';
      showDeleteConfirm = false;
    } finally {
      isDeleting = false;
    }
  }

  function handleCancel() {
    oncancel?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showDeleteConfirm) {
        showDeleteConfirm = false;
      } else {
        handleCancel();
      }
    }
  }

  function handlePermissionsChange(newPermissions: Record<string, Record<string, boolean | string>>) {
    permissions = newPermissions;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-overlay" onclick={handleCancel}>
  <div class="modal-content" onclick={(e) => e.stopPropagation()}>
    <div class="modal-header">
      <div class="header-info">
        <div
          class="role-icon"
          style="background: {color}15; color: {color}"
        >
          <Shield class="w-5 h-5" />
        </div>
        <div>
          <h2>Edit Role: {role.display_name}</h2>
          <p>
            {#if isSystemRole}
              System role - some properties cannot be changed
            {:else}
              Customize this role's name and permissions
            {/if}
          </p>
        </div>
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

    <!-- Tabs -->
    <div class="modal-tabs">
      <button
        type="button"
        class="tab-button"
        class:active={activeTab === 'general'}
        onclick={() => activeTab = 'general'}
      >
        General
      </button>
      <button
        type="button"
        class="tab-button"
        class:active={activeTab === 'permissions'}
        onclick={() => activeTab = 'permissions'}
      >
        Permissions
      </button>
    </div>

    <div class="modal-body">
      {#if activeTab === 'general'}
        <!-- General Settings Tab -->
        <div class="form-group">
          <label for="displayName">
            <Shield class="w-4 h-4" />
            Display Name
          </label>
          <input
            id="displayName"
            type="text"
            bind:value={displayName}
            placeholder="e.g., Project Lead"
            disabled={isSaving}
            autofocus
          />
        </div>

        <div class="form-group">
          <label for="internalName">Internal Name</label>
          <input
            id="internalName"
            type="text"
            value={role.name}
            disabled
            class="readonly"
          />
          <p class="field-hint">Internal names cannot be changed after creation</p>
        </div>

        <div class="form-group">
          <label for="description">Description</label>
          <textarea
            id="description"
            bind:value={description}
            placeholder="What can this role do?"
            rows="3"
            disabled={isSaving}
          ></textarea>
        </div>

        <!-- Color Selection -->
        <div class="form-group">
          <label>
            <Palette class="w-4 h-4" />
            Role Color
          </label>
          <div class="color-options">
            {#each colorOptions as colorOpt}
              <button
                type="button"
                class="btn-pill btn-pill-ghost color-swatch"
                class:selected={color === colorOpt.value}
                style="background-color: {colorOpt.value}"
                onclick={() => color = colorOpt.value}
                title={colorOpt.label}
                disabled={isSaving}
              >
                {#if color === colorOpt.value}
                  <span class="color-check">✓</span>
                {/if}
              </button>
            {/each}
          </div>
        </div>

        <!-- Role Info -->
        <div class="role-info-card">
          <div class="info-row">
            <span class="info-label">Hierarchy Level</span>
            <span class="info-value">{role.hierarchy_level}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Type</span>
            <span class="info-value">
              {#if isSystemRole}
                <span class="badge system">System Role</span>
              {:else}
                <span class="badge custom">Custom Role</span>
              {/if}
            </span>
          </div>
          {#if role.is_default}
            <div class="info-row">
              <span class="info-label">Default</span>
              <span class="info-value">
                <span class="badge default">Default for new members</span>
              </span>
            </div>
          {/if}
        </div>

        <!-- Delete Section (only for custom roles) -->
        {#if !isSystemRole}
          <div class="danger-zone">
            <h3>Danger Zone</h3>
            <p>Deleting this role will remove it permanently. Members with this role will need to be reassigned.</p>
            <button
              type="button"
              class="btn-pill btn-pill-ghost delete-button"
              onclick={() => showDeleteConfirm = true}
              disabled={isSaving || isDeleting}
            >
              <Trash2 class="w-4 h-4" />
              Delete Role
            </button>
          </div>
        {/if}
      {:else}
        <!-- Permissions Tab -->
        <div class="permissions-tab">
          {#if isSystemRole}
            <div class="system-notice">
              <AlertTriangle class="w-4 h-4" />
              <span>System role permissions are managed by the system but can be customized</span>
            </div>
          {/if}
          <PermissionsMatrixEditor
            bind:permissions
            onchange={handlePermissionsChange}
          />
        </div>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn-pill btn-pill-ghost cancel-button" onclick={handleCancel} type="button" disabled={isSaving}>
        Cancel
      </button>
      <button
        class="btn-pill btn-pill-ghost save-button"
        onclick={handleSave}
        disabled={!isValid() || !hasChanges() || isSaving}
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

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
  <div class="confirm-overlay" onclick={() => showDeleteConfirm = false}>
    <div class="confirm-dialog" onclick={(e) => e.stopPropagation()}>
      <div class="confirm-icon">
        <AlertTriangle class="w-8 h-8" />
      </div>
      <h3>Delete Role?</h3>
      <p>
        Are you sure you want to delete the <strong>{role.display_name}</strong> role?
        This action cannot be undone. Members with this role will need to be reassigned.
      </p>
      <div class="confirm-actions">
        <button
          type="button"
          class="btn-pill btn-pill-ghost confirm-cancel"
          onclick={() => showDeleteConfirm = false}
          disabled={isDeleting}
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-pill btn-pill-ghost confirm-delete"
          onclick={handleDelete}
          disabled={isDeleting}
        >
          {#if isDeleting}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <Trash2 class="w-4 h-4" />
          {/if}
          Delete Role
        </button>
      </div>
    </div>
  </div>
{/if}

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
    max-width: 700px;
    max-height: 90vh;
    background: var(--dbg);
    border-radius: 0.75rem;
    box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 1.5rem;
    border-bottom: 1px solid var(--dbd);
    flex-shrink: 0;
  }

  .header-info {
    display: flex;
    gap: 1rem;
    align-items: flex-start;
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
  }

  .modal-tabs {
    display: flex;
    gap: 0;
    border-bottom: 1px solid var(--dbd);
    padding: 0 1.5rem;
    flex-shrink: 0;
  }

  .tab-button {
    padding: 0.75rem 1.5rem;
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--dt2);
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
    margin-bottom: -1px;
  }

  .tab-button:hover {
    color: var(--dt);
  }

  .tab-button.active {
    color: var(--bos-primary-color);
    border-bottom-color: var(--bos-primary-color);
  }

  .modal-body {
    padding: 1.5rem;
    overflow-y: auto;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
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

  .form-group input,
  .form-group textarea {
    padding: 0.625rem 0.875rem;
    border: 1px solid var(--dbd);
    border-radius: 0.375rem;
    font-size: 0.875rem;
    background: var(--dbg);
    color: var(--dt);
    transition: all 0.15s;
  }

  .form-group input:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: var(--bos-primary-color);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--bos-primary-color) 15%, transparent);
  }

  .form-group input.readonly {
    background: var(--dbg2);
    color: var(--dt2);
  }

  .form-group input:disabled,
  .form-group textarea:disabled {
    background: var(--dbg2);
    color: var(--dt3);
    cursor: not-allowed;
  }

  .field-hint {
    font-size: 0.75rem;
    color: var(--dt2);
    margin: 0;
  }

  .color-options {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .color-swatch {
    width: 2rem;
    height: 2rem;
    border: 2px solid transparent;
    border-radius: 0.375rem;
    cursor: pointer;
    transition: all 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .color-swatch:hover:not(:disabled) {
    transform: scale(1.1);
  }

  .color-swatch.selected {
    border-color: var(--dt);
    box-shadow: 0 0 0 2px var(--dbg), 0 0 0 4px currentColor;
  }

  .color-check {
    color: white;
    font-weight: bold;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  .role-info-card {
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 0.5rem;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .info-label {
    font-size: 0.875rem;
    color: var(--dt2);
  }

  .info-value {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--dt);
  }

  .badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 9999px;
  }

  .badge.system {
    background: color-mix(in srgb, var(--bos-status-info) 15%, var(--dbg));
    color: var(--bos-status-info);
  }

  .badge.custom {
    background: var(--bos-status-warning-bg);
    color: var(--bos-status-warning);
  }

  .badge.default {
    background: color-mix(in srgb, var(--bos-status-success) 15%, var(--dbg));
    color: var(--bos-status-success);
  }

  .danger-zone {
    margin-top: 1rem;
    padding: 1rem;
    background: var(--bos-background-error-color, #fef2f2);
    border: 1px solid color-mix(in srgb, var(--bos-error-color) 20%, var(--dbg));
    border-radius: 0.5rem;
  }

  .danger-zone h3 {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--bos-error-color);
    margin: 0 0 0.5rem 0;
  }

  .danger-zone p {
    font-size: 0.875rem;
    color: var(--bos-error-color);
    opacity: 0.8;
    margin: 0 0 1rem 0;
  }

  .delete-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: var(--bos-error-color);
    color: white;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .delete-button:hover:not(:disabled) {
    filter: brightness(0.9);
  }

  .delete-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .permissions-tab {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .system-notice {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: var(--bos-status-warning-bg);
    color: var(--bos-status-warning);
    font-size: 0.875rem;
    border-radius: 0.5rem;
  }

  .modal-footer {
    display: flex;
    gap: 0.75rem;
    padding: 1.5rem;
    border-top: 1px solid var(--dbd);
    flex-shrink: 0;
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

  /* Delete Confirmation Modal */
  .confirm-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1100;
    padding: 1rem;
  }

  .confirm-dialog {
    width: 100%;
    max-width: 400px;
    background: var(--dbg);
    border-radius: 0.75rem;
    padding: 1.5rem;
    text-align: center;
  }

  .confirm-icon {
    width: 4rem;
    height: 4rem;
    margin: 0 auto 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bos-background-error-color, #fef2f2);
    color: var(--bos-error-color);
    border-radius: 50%;
  }

  .confirm-dialog h3 {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--dt);
    margin: 0 0 0.5rem 0;
  }

  .confirm-dialog p {
    font-size: 0.875rem;
    color: var(--dt2);
    margin: 0 0 1.5rem 0;
  }

  .confirm-actions {
    display: flex;
    gap: 0.75rem;
  }

  .confirm-cancel,
  .confirm-delete {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.625rem 1rem;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .confirm-cancel {
    background: var(--dbg2);
    color: var(--dt2);
    border: 1px solid var(--dbd);
  }

  .confirm-cancel:hover:not(:disabled) {
    background: var(--dbg3);
  }

  .confirm-delete {
    background: var(--bos-error-color);
    color: white;
  }

  .confirm-delete:hover:not(:disabled) {
    filter: brightness(0.9);
  }

  .confirm-delete:disabled,
  .confirm-cancel:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
