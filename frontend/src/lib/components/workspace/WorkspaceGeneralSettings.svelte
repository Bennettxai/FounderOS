<script lang="ts">
  import { goto } from '$app/navigation';
  import {
    updateWorkspace,
    deleteWorkspace,
    type Workspace,
  } from '$lib/api/workspaces';
  import {
    Save,
    Trash2,
    Copy,
    Check,
    Loader2,
    AlertCircle,
    Upload,
  } from 'lucide-svelte';
  import ConfirmDeleteModal from './ConfirmDeleteModal.svelte';

  interface Props {
    workspace: Workspace;
    canManage: boolean;
    isOwner: boolean;
    onupdated?: (workspace: Workspace) => void;
  }

  let { workspace, canManage, isOwner, onupdated }: Props = $props();

  // Form state
  let name = $state(workspace.name);
  let description = $state(workspace.description || '');
  let logoUrl = $state(workspace.logo_url || '');

  // UI state
  let isSaving = $state(false);
  let saveMessage = $state('');
  let slugCopied = $state(false);
  let showDeleteModal = $state(false);
  let error = $state<string | null>(null);

  async function handleSave() {
    if (!canManage) return;

    try {
      isSaving = true;
      error = null;

      const updated = await updateWorkspace(workspace.id, {
        name: name.trim(),
        description: description.trim() || undefined,
        logo_url: logoUrl.trim() || undefined,
      });

      onupdated?.(updated);
      saveMessage = 'Settings saved successfully!';
      setTimeout(() => (saveMessage = ''), 3000);
    } catch (err) {
      console.error('Failed to update workspace:', err);
      error = err instanceof Error ? err.message : 'Failed to update workspace';
    } finally {
      isSaving = false;
    }
  }

  async function copySlug() {
    try {
      await navigator.clipboard.writeText(workspace.slug);
      slugCopied = true;
      setTimeout(() => (slugCopied = false), 2000);
    } catch (err) {
      console.error('Failed to copy slug:', err);
    }
  }

  async function handleDelete() {
    if (!isOwner) return;

    try {
      await deleteWorkspace(workspace.id);
      // Redirect to workspace selection or home
      goto('/dashboard');
    } catch (err) {
      console.error('Failed to delete workspace:', err);
      error = err instanceof Error ? err.message : 'Failed to delete workspace';
    }
  }

  function handleLogoUpload(event: Event) {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) return;

    // In a real app, you would upload to your storage service
    // For now, we'll just use a placeholder
    const reader = new FileReader();
    reader.onload = (e) => {
      logoUrl = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  }

  $effect(() => {
    // Reset form when workspace changes
    name = workspace.name;
    description = workspace.description || '';
    logoUrl = workspace.logo_url || '';
  });
</script>

<div class="general-settings">
  <div class="section-header">
    <h2>General Settings</h2>
    <p>Manage your workspace's basic information and settings</p>
  </div>

  {#if error}
    <div class="error-message">
      <AlertCircle class="w-4 h-4" />
      <span>{error}</span>
    </div>
  {/if}

  {#if saveMessage}
    <div class="success-message">
      <Check class="w-4 h-4" />
      <span>{saveMessage}</span>
    </div>
  {/if}

  <div class="settings-form">
    <!-- Logo -->
    <div class="form-group">
      <label for="logo">Workspace Logo</label>
      <div class="logo-upload">
        {#if logoUrl}
          <img src={logoUrl} alt="Workspace logo" class="logo-preview" />
        {:else}
          <div class="logo-placeholder">
            <Upload class="w-8 h-8" />
          </div>
        {/if}
        {#if canManage}
          <div class="logo-actions">
            <label for="logo-input" class="upload-button">
              <Upload class="w-4 h-4" />
              Upload Logo
            </label>
            <input
              id="logo-input"
              type="file"
              accept="image/*"
              onchange={handleLogoUpload}
              disabled={!canManage}
              class="hidden"
            />
            {#if logoUrl}
              <button
                type="button"
                class="btn-pill btn-pill-ghost remove-button"
                onclick={() => (logoUrl = '')}
                disabled={!canManage}
              >
                <Trash2 class="w-4 h-4" />
                Remove
              </button>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    <!-- Workspace Name -->
    <div class="form-group">
      <label for="name">Workspace Name</label>
      <input
        id="name"
        type="text"
        bind:value={name}
        disabled={!canManage}
        placeholder="Enter workspace name"
        required
      />
    </div>

    <!-- Description -->
    <div class="form-group">
      <label for="description">Description</label>
      <textarea
        id="description"
        bind:value={description}
        disabled={!canManage}
        placeholder="Describe your workspace"
        rows="3"
      ></textarea>
    </div>

    <!-- Slug (Read-only) -->
    <div class="form-group">
      <label for="slug">Workspace Slug</label>
      <div class="slug-field">
        <input id="slug" type="text" value={workspace.slug} disabled />
        <button type="button" class="btn-pill btn-pill-ghost copy-button" onclick={copySlug}>
          {#if slugCopied}
            <Check class="w-4 h-4" />
          {:else}
            <Copy class="w-4 h-4" />
          {/if}
        </button>
      </div>
      <p class="field-hint">This is your workspace's unique identifier</p>
    </div>

    <!-- Plan Info -->
    <div class="form-group">
      <label>Plan & Limits</label>
      <div class="plan-info">
        <div class="plan-detail">
          <span class="plan-label">Plan Type</span>
          <span class="plan-value">{workspace.plan_type}</span>
        </div>
        <div class="plan-detail">
          <span class="plan-label">Max Members</span>
          <span class="plan-value">{workspace.max_members}</span>
        </div>
        <div class="plan-detail">
          <span class="plan-label">Max Projects</span>
          <span class="plan-value">{workspace.max_projects}</span>
        </div>
        <div class="plan-detail">
          <span class="plan-label">Storage</span>
          <span class="plan-value">{workspace.max_storage_gb} GB</span>
        </div>
      </div>
    </div>

    <!-- Actions -->
    {#if canManage}
      <div class="form-actions">
        <button
          type="button"
          class="btn-pill btn-pill-ghost save-button"
          onclick={handleSave}
          disabled={isSaving || !name.trim()}
        >
          {#if isSaving}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <Save class="w-4 h-4" />
          {/if}
          Save Changes
        </button>
      </div>
    {/if}

    <!-- Danger Zone -->
    {#if isOwner}
      <div class="danger-zone">
        <div class="danger-zone-header">
          <h3>Danger Zone</h3>
          <p>Irreversible actions that affect your workspace</p>
        </div>
        <button
          type="button"
          class="btn-pill btn-pill-ghost delete-button"
          onclick={() => (showDeleteModal = true)}
        >
          <Trash2 class="w-4 h-4" />
          Delete Workspace
        </button>
      </div>
    {/if}
  </div>
</div>

{#if showDeleteModal}
  <ConfirmDeleteModal
    title="Delete Workspace"
    message="Are you sure you want to delete this workspace? This action cannot be undone. All projects, members, and data will be permanently deleted."
    confirmText="Delete Workspace"
    onconfirm={handleDelete}
    oncancel={() => (showDeleteModal = false)}
  />
{/if}

<style>
  .general-settings {
    padding: 2rem;
  }

  .section-header {
    margin-bottom: 2rem;
  }

  .section-header h2 {
    font-size: 1.25rem;
    font-weight: 600;
    color: #111827;
    margin: 0 0 0.5rem 0;
  }

  .section-header p {
    color: #6b7280;
    font-size: 0.875rem;
    margin: 0;
  }

  .error-message,
  .success-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    margin-bottom: 1.5rem;
  }

  .error-message {
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fee2e2;
  }

  .success-message {
    background: #f0fdf4;
    color: #16a34a;
    border: 1px solid #dcfce7;
  }

  .settings-form {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    max-width: 600px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-group label {
    font-size: 0.875rem;
    font-weight: 500;
    color: #374151;
  }

  .form-group input,
  .form-group textarea {
    padding: 0.625rem 0.875rem;
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    transition: all 0.15s;
  }

  .form-group input:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .form-group input:disabled,
  .form-group textarea:disabled {
    background: #f9fafb;
    color: #9ca3af;
    cursor: not-allowed;
  }

  .field-hint {
    font-size: 0.75rem;
    color: #6b7280;
    margin: 0;
  }

  .logo-upload {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .logo-preview {
    width: 80px;
    height: 80px;
    object-fit: cover;
    border-radius: 0.5rem;
    border: 1px solid #e5e7eb;
  }

  .logo-placeholder {
    width: 80px;
    height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f3f4f6;
    border: 2px dashed #d1d5db;
    border-radius: 0.5rem;
    color: #9ca3af;
  }

  .logo-actions {
    display: flex;
    gap: 0.5rem;
  }

  .upload-button,
  .remove-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.875rem;
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    background: white;
    color: #374151;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .upload-button:hover,
  .remove-button:hover {
    background: #f9fafb;
  }

  .remove-button {
    color: #dc2626;
  }

  .hidden {
    display: none;
  }

  .slug-field {
    display: flex;
    gap: 0.5rem;
  }

  .slug-field input {
    flex: 1;
  }

  .copy-button {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.625rem;
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    background: white;
    color: #6b7280;
    cursor: pointer;
    transition: all 0.15s;
  }

  .copy-button:hover {
    background: #f9fafb;
    color: #111827;
  }

  .plan-info {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
    padding: 1rem;
    background: #f9fafb;
    border-radius: 0.375rem;
    border: 1px solid #e5e7eb;
  }

  .plan-detail {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .plan-label {
    font-size: 0.75rem;
    color: #6b7280;
    text-transform: uppercase;
    font-weight: 500;
  }

  .plan-value {
    font-size: 0.875rem;
    color: #111827;
    font-weight: 600;
  }

  .form-actions {
    padding-top: 1rem;
  }

  .save-button {
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
  }

  .save-button:hover:not(:disabled) {
    background: #2563eb;
  }

  .save-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .danger-zone {
    margin-top: 2rem;
    padding: 1.5rem;
    border: 1px solid #fee2e2;
    border-radius: 0.5rem;
    background: #fef2f2;
  }

  .danger-zone-header h3 {
    font-size: 1rem;
    font-weight: 600;
    color: #dc2626;
    margin: 0 0 0.5rem 0;
  }

  .danger-zone-header p {
    font-size: 0.875rem;
    color: #991b1b;
    margin: 0 0 1rem 0;
  }

  .delete-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1.25rem;
    background: #dc2626;
    color: white;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .delete-button:hover {
    background: #b91c1c;
  }

  :global(.dark) .section-header h2 {
    color: #f9fafb;
  }

  :global(.dark) .section-header p {
    color: #9ca3af;
  }

  :global(.dark) .form-group label {
    color: #d1d5db;
  }

  :global(.dark) .form-group input,
  :global(.dark) .form-group textarea {
    background: #111827;
    border-color: #374151;
    color: #f9fafb;
  }

  :global(.dark) .form-group input:disabled,
  :global(.dark) .form-group textarea:disabled {
    background: #1f2937;
    color: #6b7280;
  }

  :global(.dark) .plan-info {
    background: #111827;
    border-color: #374151;
  }

  :global(.dark) .plan-value {
    color: #f9fafb;
  }
</style>
