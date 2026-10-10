<script lang="ts">
  import { createWorkspaceInvite, type WorkspaceRole } from '$lib/api/workspaces';
  import { X, Mail, Send, Loader2, AlertCircle, CheckCircle } from 'lucide-svelte';

  interface Props {
    workspaceId: string;
    roles: WorkspaceRole[];
    onsuccess?: () => void;
    oncancel?: () => void;
  }

  let { workspaceId, roles, onsuccess, oncancel }: Props = $props();

  let email = $state('');
  let selectedRole = $state('member');
  let isSending = $state(false);
  let error = $state<string | null>(null);
  let success = $state(false);

  // Filter out system roles that shouldn't be assigned via invite
  const assignableRoles = $derived(roles.filter((r) => r.name !== 'owner'));

  async function handleSend() {
    if (!email.trim() || !selectedRole) return;

    try {
      isSending = true;
      error = null;

      // In dev mode with mock workspace, simulate success
      if (import.meta.env.DEV && workspaceId.startsWith('mock-')) {
        if (import.meta.env.DEV) console.log('[InviteMemberModal] Mock mode - simulating invite to:', email.trim());
        await new Promise(resolve => setTimeout(resolve, 800)); // Fake delay
        success = true;
        setTimeout(() => onsuccess?.(), 1500);
        return;
      }

      await createWorkspaceInvite(workspaceId, {
        email: email.trim(),
        role: selectedRole,
      });

      onsuccess?.();
    } catch (err) {
      console.error('Failed to send invitation:', err);
      error = err instanceof Error ? err.message : 'Failed to send invitation';
    } finally {
      isSending = false;
    }
  }

  function handleCancel() {
    oncancel?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleCancel();
    } else if (e.key === 'Enter' && email.trim() && selectedRole && !isSending) {
      handleSend();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-overlay" onclick={handleCancel}>
  <div class="modal-content" onclick={(e) => e.stopPropagation()}>
    <div class="modal-header">
      <div>
        <h2>Invite Member</h2>
        <p>Send an invitation to join this workspace</p>
      </div>
      <button class="btn-pill btn-pill-ghost close-button" onclick={handleCancel} type="button">
        <X class="w-5 h-5" />
      </button>
    </div>

    {#if success}
      <div class="success-message">
        <CheckCircle class="w-5 h-5" />
        <span>Invitation sent to <strong>{email}</strong>!</span>
      </div>
    {:else}
      {#if error}
        <div class="error-message">
          <AlertCircle class="w-4 h-4" />
          <span>{error}</span>
        </div>
      {/if}

      <div class="modal-body">
        <div class="form-group">
          <label for="email">
            <Mail class="w-4 h-4" />
            Email Address
          </label>
          <input
            id="email"
            type="email"
            bind:value={email}
            placeholder="member@example.com"
            disabled={isSending}
            autofocus
          />
        </div>

        <div class="form-group">
          <label for="role">Role</label>
          <select id="role" bind:value={selectedRole} disabled={isSending}>
            {#each assignableRoles as role (role.id)}
              <option value={role.name}>
                {role.display_name}
                {#if role.description}
                  - {role.description}
                {/if}
              </option>
            {/each}
          </select>
          <p class="field-hint">The role determines what permissions the member will have</p>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn-pill btn-pill-ghost cancel-button" onclick={handleCancel} type="button" disabled={isSending}>
          Cancel
        </button>
        <button
          class="btn-pill btn-pill-ghost send-button"
          onclick={handleSend}
          disabled={!email.trim() || !selectedRole || isSending}
          type="button"
        >
          {#if isSending}
          <Loader2 class="w-4 h-4 animate-spin" />
        {:else}
          <Send class="w-4 h-4" />
        {/if}
        Send Invitation
      </button>
    </div>
    {/if}
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
    padding: var(--space-4);
  }

  .modal-content {
    width: 100%;
    max-width: 500px;
    background: var(--dbg);
    border: 1px solid var(--dbd);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-xl);
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: var(--space-6);
    border-bottom: 1px solid var(--dbd);
  }

  .modal-header h2 {
    font-size: var(--text-xl);
    font-weight: var(--font-semibold);
    color: var(--dt);
    margin: 0 0 var(--space-1) 0;
  }

  .modal-header p {
    color: var(--dt2);
    font-size: var(--text-sm);
    margin: 0;
  }

  .close-button {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-1);
    background: transparent;
    border: none;
    color: var(--dt3);
    cursor: pointer;
    border-radius: var(--radius-xs);
    transition: all 150ms ease;
  }

  .close-button:hover {
    background: var(--dbg2);
    color: var(--dt);
  }

  .error-message {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-6);
    background: color-mix(in srgb, var(--color-error) 10%, var(--dbg));
    color: var(--color-error);
    font-size: var(--text-sm);
    border-bottom: 1px solid color-mix(in srgb, var(--color-error) 20%, var(--dbg));
  }

  .success-message {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    padding: 3rem var(--space-6);
    background: color-mix(in srgb, var(--color-success) 10%, var(--dbg));
    color: var(--color-success);
    font-size: var(--text-base);
    text-align: center;
  }

  .success-message strong {
    color: var(--color-success);
  }

  .modal-body {
    padding: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .form-group label {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    font-weight: var(--font-medium);
    color: var(--dt2);
  }

  .form-group input,
  .form-group select {
    padding: 0.625rem 0.875rem;
    border: 1px solid var(--dbd);
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    background: var(--dbg);
    color: var(--dt);
    transition: all 150ms ease;
  }

  .form-group input:focus,
  .form-group select:focus {
    outline: none;
    border-color: var(--accent-blue);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent-blue) 15%, transparent);
  }

  .form-group input:disabled,
  .form-group select:disabled {
    background: var(--dbg2);
    color: var(--dt4);
    cursor: not-allowed;
  }

  .field-hint {
    font-size: var(--text-xs);
    color: var(--dt3);
    margin: 0;
  }

  .modal-footer {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-6);
    border-top: 1px solid var(--dbd);
  }

  .cancel-button,
  .send-button {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: 0.625rem var(--space-5);
    border: none;
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    font-weight: var(--font-medium);
    cursor: pointer;
    transition: all 150ms ease;
  }

  .cancel-button {
    background: var(--dbg2);
    color: var(--dt2);
    border: 1px solid var(--dbd);
  }

  .cancel-button:hover:not(:disabled) {
    background: var(--dbg3);
    color: var(--dt);
  }

  .send-button {
    background: var(--accent-blue);
    color: #fff;
  }

  .send-button:hover:not(:disabled) {
    filter: brightness(1.1);
  }

  .send-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
