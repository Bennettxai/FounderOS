<script lang="ts">
  import {
    revokeWorkspaceInvite,
    type WorkspaceInvite,
    type WorkspaceRole,
  } from '$lib/api/workspaces';
  import {
    Mail,
    Clock,
    UserPlus,
    Trash2,
    Copy,
    Check,
    XCircle,
    CheckCircle,
    Loader2,
  } from 'lucide-svelte';
  import InviteMemberModal from './InviteMemberModal.svelte';

  interface Props {
    workspaceId: string;
    invites: WorkspaceInvite[];
    roles: WorkspaceRole[];
    canManage: boolean;
    canInvite: boolean;
    onupdated?: () => void;
  }

  let { workspaceId, invites, roles, canManage, canInvite, onupdated }: Props = $props();


  // UI state
  let showInviteModal = $state(false);
  let copiedId = $state<string | null>(null);
  let revokingId = $state<string | null>(null);

  function getStatusBadge(status: string): { label: string; color: string; icon: any } {
    const badges: Record<string, { label: string; color: string; icon: any }> = {
      pending: { label: 'Pending', color: '#f59e0b', icon: Clock },
      accepted: { label: 'Accepted', color: '#10b981', icon: CheckCircle },
      expired: { label: 'Expired', color: '#6b7280', icon: XCircle },
      revoked: { label: 'Revoked', color: '#dc2626', icon: XCircle },
    };
    return badges[status] || badges.pending;
  }

  function isExpired(invite: WorkspaceInvite): boolean {
    return new Date(invite.expires_at) < new Date();
  }

  async function copyInviteLink(invite: WorkspaceInvite) {
    try {
      const inviteUrl = `${window.location.origin}/invite/${invite.token}`;
      await navigator.clipboard.writeText(inviteUrl);
      copiedId = invite.id;
      setTimeout(() => (copiedId = null), 2000);
    } catch (err) {
      console.error('Failed to copy invite link:', err);
    }
  }

  async function handleRevoke(invite: WorkspaceInvite) {
    if (!canManage) return;

    try {
      revokingId = invite.id;
      await revokeWorkspaceInvite(workspaceId, invite.id);
      onupdated?.();
    } catch (err) {
      console.error('Failed to revoke invitation:', err);
    } finally {
      revokingId = null;
    }
  }

  function handleInviteSuccess() {
    showInviteModal = false;
    onupdated?.();
  }

  const pendingInvites = invites.filter((i) => i.status === 'pending' && !isExpired(i));
  const otherInvites = invites.filter((i) => i.status !== 'pending' || isExpired(i));
</script>

<div class="invites-list">
  <div class="list-header">
    <div>
      <h2>Invitations</h2>
      <p>Manage pending and past workspace invitations</p>
    </div>
    {#if canInvite}
      <button class="btn-pill btn-pill-ghost invite-button" onclick={() => (showInviteModal = true)}>
        <UserPlus class="w-4 h-4" />
        Send Invitation
      </button>
    {/if}
  </div>

  <!-- Pending Invites -->
  {#if pendingInvites.length > 0}
    <div class="invites-section">
      <h3 class="section-title">Pending ({pendingInvites.length})</h3>
      <div class="invites-grid">
        {#each pendingInvites as invite (invite.id)}
          <div class="invite-card pending">
            <div class="invite-header">
              <div class="invite-icon">
                <Mail class="w-5 h-5" />
              </div>
              <div class="invite-info">
                <div class="invite-email">{invite.email}</div>
                <div class="invite-role">{invite.role}</div>
              </div>
            </div>

            <div class="invite-meta">
              <div class="meta-row">
                <span class="meta-label">Status</span>
                <span
                  class="status-badge"
                  style="background: {getStatusBadge(invite.status).color}15; color: {getStatusBadge(
                    invite.status
                  ).color}"
                >
                  <svelte:component this={getStatusBadge(invite.status).icon} class="w-3 h-3" />
                  {getStatusBadge(invite.status).label}
                </span>
              </div>
              <div class="meta-row">
                <span class="meta-label">Expires</span>
                <span class="meta-value">
                  {new Date(invite.expires_at).toLocaleDateString()}
                </span>
              </div>
              <div class="meta-row">
                <span class="meta-label">Sent</span>
                <span class="meta-value">
                  {new Date(invite.created_at).toLocaleDateString()}
                </span>
              </div>
            </div>

            <div class="invite-actions">
              <button
                class="btn-pill btn-pill-ghost action-button"
                onclick={() => copyInviteLink(invite)}
                type="button"
              >
                {#if copiedId === invite.id}
                  <Check class="w-4 h-4" />
                  Copied
                {:else}
                  <Copy class="w-4 h-4" />
                  Copy Link
                {/if}
              </button>
              {#if canManage}
                <button
                  class="btn-pill btn-pill-ghost action-button danger"
                  onclick={() => handleRevoke(invite)}
                  disabled={revokingId === invite.id}
                  type="button"
                >
                  {#if revokingId === invite.id}
                    <Loader2 class="w-4 h-4 animate-spin" />
                  {:else}
                    <Trash2 class="w-4 h-4" />
                  {/if}
                  Revoke
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Other Invites -->
  {#if otherInvites.length > 0}
    <div class="invites-section">
      <h3 class="section-title">History ({otherInvites.length})</h3>
      <div class="invites-table-container">
        <table class="invites-table">
          <thead>
            <tr>
              <th>Email</th>
              <th>Role</th>
              <th>Status</th>
              <th>Sent</th>
              <th>Expires</th>
            </tr>
          </thead>
          <tbody>
            {#each otherInvites as invite (invite.id)}
              <tr>
                <td>
                  <div class="email-cell">
                    <Mail class="w-4 h-4 text-gray-400" />
                    {invite.email}
                  </div>
                </td>
                <td>
                  <span class="role-text">{invite.role}</span>
                </td>
                <td>
                  <span
                    class="status-badge small"
                    style="background: {getStatusBadge(invite.status).color}15; color: {getStatusBadge(
                      invite.status
                    ).color}"
                  >
                    {getStatusBadge(invite.status).label}
                  </span>
                </td>
                <td>
                  <span class="date-text">
                    {new Date(invite.created_at).toLocaleDateString()}
                  </span>
                </td>
                <td>
                  <span class="date-text" class:expired={isExpired(invite)}>
                    {new Date(invite.expires_at).toLocaleDateString()}
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

  {#if invites.length === 0}
    <div class="empty-state">
      <Mail class="w-12 h-12" />
      <p>No invitations sent yet</p>
      {#if canInvite}
        <button class="btn-pill btn-pill-ghost empty-action-button" onclick={() => (showInviteModal = true)}>
          <UserPlus class="w-4 h-4" />
          Send First Invitation
        </button>
      {/if}
    </div>
  {/if}
</div>

{#if showInviteModal}
  <InviteMemberModal
    {workspaceId}
    {roles}
    onsuccess={handleInviteSuccess}
    oncancel={() => (showInviteModal = false)}
  />
{/if}

<style>
  .invites-list {
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

  .invite-button {
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

  .invite-button:hover {
    background: #2563eb;
  }

  .invites-section {
    margin-bottom: 2rem;
  }

  .section-title {
    font-size: 1rem;
    font-weight: 600;
    color: #374151;
    margin: 0 0 1rem 0;
  }

  .invites-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 1rem;
  }

  .invite-card {
    padding: 1.5rem;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
    transition: all 0.15s;
  }

  .invite-card.pending {
    border-left: 4px solid #f59e0b;
  }

  .invite-header {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .invite-icon {
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #eff6ff;
    color: #3b82f6;
    border-radius: 0.5rem;
    flex-shrink: 0;
  }

  .invite-info {
    flex: 1;
    min-width: 0;
  }

  .invite-email {
    font-size: 0.875rem;
    font-weight: 500;
    color: #111827;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .invite-role {
    font-size: 0.75rem;
    color: #6b7280;
    text-transform: capitalize;
  }

  .invite-meta {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem;
    background: #f9fafb;
    border-radius: 0.375rem;
    margin-bottom: 1rem;
  }

  .meta-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .meta-label {
    font-size: 0.75rem;
    color: #6b7280;
    font-weight: 500;
  }

  .meta-value {
    font-size: 0.75rem;
    color: #111827;
    font-weight: 500;
  }

  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 9999px;
  }

  .status-badge.small {
    padding: 0.25rem 0.625rem;
  }

  .invite-actions {
    display: flex;
    gap: 0.5rem;
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
  }

  .action-button.danger {
    color: #dc2626;
  }

  .action-button.danger:hover:not(:disabled) {
    background: #fef2f2;
  }

  .action-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .invites-table-container {
    overflow-x: auto;
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
  }

  .invites-table {
    width: 100%;
    border-collapse: collapse;
  }

  .invites-table thead {
    background: #f9fafb;
  }

  .invites-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.75rem;
    font-weight: 600;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #e5e7eb;
  }

  .invites-table td {
    padding: 1rem;
    border-bottom: 1px solid #f3f4f6;
  }

  .invites-table tbody tr:last-child td {
    border-bottom: none;
  }

  .invites-table tbody tr:hover {
    background: #f9fafb;
  }

  .email-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    color: #111827;
  }

  .role-text {
    font-size: 0.875rem;
    color: #6b7280;
    text-transform: capitalize;
  }

  .date-text {
    font-size: 0.875rem;
    color: #6b7280;
  }

  .date-text.expired {
    color: #dc2626;
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

  .empty-action-button {
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

  .empty-action-button:hover {
    background: #2563eb;
  }

  :global(.dark) .list-header h2,
  :global(.dark) .section-title {
    color: #f9fafb;
  }

  :global(.dark) .list-header p {
    color: #9ca3af;
  }

  :global(.dark) .invite-card {
    background: #1f2937;
    border-color: #374151;
  }

  :global(.dark) .invite-email {
    color: #f9fafb;
  }

  :global(.dark) .invite-meta {
    background: #111827;
  }

  :global(.dark) .meta-value {
    color: #f9fafb;
  }

  :global(.dark) .action-button {
    background: #111827;
    border-color: #374151;
    color: #d1d5db;
  }

  :global(.dark) .action-button:hover:not(:disabled) {
    background: #0f172a;
  }

  :global(.dark) .invites-table-container {
    border-color: #374151;
  }

  :global(.dark) .invites-table thead {
    background: #111827;
  }

  :global(.dark) .invites-table th {
    color: #9ca3af;
    border-bottom-color: #374151;
  }

  :global(.dark) .invites-table td {
    border-bottom-color: #1f2937;
  }

  :global(.dark) .invites-table tbody tr:hover {
    background: #111827;
  }

  :global(.dark) .email-cell {
    color: #f9fafb;
  }
</style>
