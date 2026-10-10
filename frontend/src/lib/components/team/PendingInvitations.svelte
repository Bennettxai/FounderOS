<script lang="ts">
  import { Clock, X, Mail, Copy, Check } from 'lucide-svelte';
  import { currentWorkspace } from '$lib/stores/workspaces';
  import { getWorkspaceInvites, revokeWorkspaceInvite, type WorkspaceInvite } from '$lib/api/workspaces';
  import { onMount } from 'svelte';

  let invitations = $state<WorkspaceInvite[]>([]);
  let loading = $state(true);
  let copiedId = $state<string | null>(null);

  onMount(() => {
    loadInvitations();
  });

  async function loadInvitations() {
    const workspaceId = $currentWorkspace?.id;
    if (!workspaceId) {
      invitations = [];
      loading = false;
      return;
    }

    try {
      loading = true;
      invitations = await getWorkspaceInvites(workspaceId);
    } catch (err) {
      console.error('Failed to load invitations:', err);
      invitations = [];
    } finally {
      loading = false;
    }
  }

  async function revokeInvite(id: string) {
    const workspaceId = $currentWorkspace?.id;
    
    if (!workspaceId) return;

    try {
      await revokeWorkspaceInvite(workspaceId, id);
      invitations = invitations.filter(i => i.id !== id);
    } catch (err) {
      console.error('Failed to revoke invitation:', err);
    }
  }

  async function copyInviteLink(invite: WorkspaceInvite) {
    const link = `${window.location.origin}/invite/${invite.token}`;
    await navigator.clipboard.writeText(link);
    copiedId = invite.id;
    setTimeout(() => copiedId = null, 2000);
  }

  function formatExpiry(date: string) {
    const d = new Date(date);
    const now = new Date();
    const days = Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    if (days <= 0) return 'Expired';
    if (days === 1) return 'Expires tomorrow';
    return `Expires in ${days} days`;
  }

  // Only show pending invitations
  const pendingInvitations = $derived(invitations.filter(i => i.status === 'pending'));
</script>

{#if loading}
  <div class="td-pending-banner td-pending-banner--loading">
    <div class="td-skeleton" style="width:160px; height:18px;"></div>
    <div class="td-skeleton" style="width:100%; height:48px; margin-top:10px;"></div>
  </div>
{:else if pendingInvitations.length > 0}
  <div class="td-pending-banner">
    <h3 class="td-pending-banner__title">
      <Clock class="w-4 h-4" />
      Pending Invitations ({pendingInvitations.length})
    </h3>

    <div class="td-pending-list">
      {#each pendingInvitations as invite (invite.id)}
        <div class="td-pending-item">
          <div class="td-pending-item__left">
            <div class="td-pending-item__icon">
              <Mail class="w-4 h-4" />
            </div>
            <div class="td-pending-item__info">
              <span class="td-pending-item__email">{invite.email}</span>
              <div class="td-pending-item__meta">
                <span class="td-role-badge td-role-badge--{invite.role}">
                  {invite.role}
                </span>
                <span class="td-pending-item__expiry">
                  {formatExpiry(invite.expires_at)}
                </span>
              </div>
            </div>
          </div>
          <div class="td-pending-item__actions">
            <button
              onclick={() => copyInviteLink(invite)}
              class="td-pending-item__btn"
              title="Copy invite link"
            >
              {#if copiedId === invite.id}
                <Check class="w-4 h-4" style="color: #22c55e;" />
              {:else}
                <Copy class="w-4 h-4" />
              {/if}
            </button>
            <button
              onclick={() => revokeInvite(invite.id)}
              class="td-pending-item__btn td-pending-item__btn--danger"
              title="Revoke invitation"
            >
              <X class="w-4 h-4" />
            </button>
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .td-pending-banner {
    margin: 0 20px;
    margin-top: 14px;
    padding: 14px;
    background: var(--bos-status-warning-bg);
    border: 1px solid #fde68a;
    border-radius: 12px;
  }
  :global(.dark) .td-pending-banner { background: var(--dbg2); border-color: var(--dbd); }

  .td-pending-banner--loading { animation: pulse 1.5s ease-in-out infinite; }
  @keyframes pulse { 0%,100% { opacity:1; } 50% { opacity:0.5; } }

  .td-skeleton { background: var(--dbg2, #e5e5e5); border-radius: 6px; }

  .td-pending-banner__title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 600;
    color: #92400e;
    margin-bottom: 10px;
  }
  :global(.dark) .td-pending-banner__title { color: var(--bos-status-warning); }

  .td-pending-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .td-pending-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: var(--dbg, #fff);
    border-radius: 10px;
    border: 1px solid #fde68a;
    box-shadow: 0 1px 2px rgba(0,0,0,0.04);
  }
  :global(.dark) .td-pending-item { background: var(--dbg2); border-color: var(--dbd); }

  .td-pending-item__left {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .td-pending-item__icon {
    width: 32px;
    height: 32px;
    border-radius: 9999px;
    background: #fef3c7;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #d97706;
    flex-shrink: 0;
  }
  :global(.dark) .td-pending-item__icon { background: var(--dbg3); color: var(--bos-status-warning); }

  .td-pending-item__info {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .td-pending-item__email {
    font-size: 13px;
    font-weight: 600;
    color: var(--dt, #111);
    letter-spacing: -0.01em;
  }

  .td-pending-item__meta {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .td-role-badge {
    display: inline-flex;
    align-items: center;
    padding: 1px 8px;
    border-radius: 9999px;
    font-size: 10px;
    font-weight: 600;
    text-transform: capitalize;
  }
  .td-role-badge--admin { background: #f3e8ff; color: #7c3aed; }
  .td-role-badge--manager { background: #fef3c7; color: #b45309; }
  .td-role-badge--member { background: var(--dbg2, #f5f5f5); color: var(--dt2, #555); }

  .td-pending-item__expiry {
    font-size: 11px;
    color: var(--dt3, #888);
  }

  .td-pending-item__actions {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .td-pending-item__btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    border: none;
    background: transparent;
    color: var(--dt3, #888);
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }
  .td-pending-item__btn:hover { background: var(--dbg2, #f5f5f5); color: var(--dt, #111); }
  .td-pending-item__btn--danger:hover { background: #fef2f2; color: #ef4444; }
</style>
