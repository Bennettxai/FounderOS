<script lang="ts">
  import {
    removeWorkspaceMember,
    type WorkspaceMember,
    type WorkspaceRole,
  } from '$lib/api/workspaces';
  import {
    Users,
    Mail,
    Shield,
    Trash2,
    MoreVertical,
    UserPlus,
    Loader2,
  } from 'lucide-svelte';
  import InviteMemberModal from './InviteMemberModal.svelte';
  import ChangeRoleModal from './ChangeRoleModal.svelte';
  import ConfirmDeleteModal from './ConfirmDeleteModal.svelte';

  interface Props {
    workspaceId: string;
    members: WorkspaceMember[];
    roles: WorkspaceRole[];
    currentUserRole: string;
    currentUserId: string;
    canManage: boolean;
    canInvite: boolean;
    onupdated?: () => void;
  }

  let {
    workspaceId,
    members,
    roles,
    currentUserRole,
    currentUserId,
    canManage,
    canInvite,
    onupdated,
  }: Props = $props();

  // UI state
  let showInviteModal = $state(false);
  let showChangeRoleModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedMember = $state<WorkspaceMember | null>(null);
  let dropdownOpen = $state<string | null>(null);
  let isRemoving = $state(false);

  function getRoleBadgeColor(role: string): string {
    const colors: Record<string, string> = {
      owner: '#8b5cf6',
      admin: '#3b82f6',
      manager: '#10b981',
      member: '#6b7280',
      viewer: '#f59e0b',
      guest: '#9ca3af',
    };
    return colors[role] || colors.member;
  }

  function getStatusBadge(status: string): { label: string; color: string } {
    const badges: Record<string, { label: string; color: string }> = {
      active: { label: 'Active', color: '#10b981' },
      invited: { label: 'Invited', color: '#f59e0b' },
      suspended: { label: 'Suspended', color: '#dc2626' },
      left: { label: 'Left', color: '#6b7280' },
    };
    return badges[status] || badges.active;
  }

  function canModifyMember(member: WorkspaceMember): boolean {
    if (!canManage) return false;
    if (member.user_id === currentUserId) return false;

    const roleHierarchy: Record<string, number> = {
      owner: 100,
      admin: 80,
      manager: 60,
      member: 40,
      viewer: 20,
      guest: 10,
    };

    const currentLevel = roleHierarchy[currentUserRole] || 0;
    const memberLevel = roleHierarchy[member.role] || 0;

    return currentLevel > memberLevel;
  }

  function handleChangeRole(member: WorkspaceMember) {
    selectedMember = member;
    showChangeRoleModal = true;
    dropdownOpen = null;
  }

  function handleRemoveMember(member: WorkspaceMember) {
    selectedMember = member;
    showDeleteModal = true;
    dropdownOpen = null;
  }

  async function confirmRemoveMember() {
    if (!selectedMember) return;

    try {
      isRemoving = true;
      await removeWorkspaceMember(workspaceId, selectedMember.id);
      onupdated?.();
      showDeleteModal = false;
      selectedMember = null;
    } catch (err) {
      console.error('Failed to remove member:', err);
    } finally {
      isRemoving = false;
    }
  }

  function toggleDropdown(memberId: string) {
    dropdownOpen = dropdownOpen === memberId ? null : memberId;
  }

  function handleInviteSuccess() {
    showInviteModal = false;
    onupdated?.();
  }

  function handleRoleChangeSuccess() {
    showChangeRoleModal = false;
    selectedMember = null;
    onupdated?.();
  }
</script>

<div class="members-list">
  <div class="list-header">
    <div>
      <h2>Members</h2>
      <p>Manage workspace members and their roles</p>
    </div>
    {#if canInvite}
      <button class="btn-pill btn-pill-ghost invite-button" onclick={() => (showInviteModal = true)}>
        <UserPlus class="w-4 h-4" />
        Invite Member
      </button>
    {/if}
  </div>

  <div class="members-table-container">
    <table class="members-table">
      <thead>
        <tr>
          <th>Member</th>
          <th>Role</th>
          <th>Status</th>
          <th>Joined</th>
          {#if canManage}
            <th class="actions-column">Actions</th>
          {/if}
        </tr>
      </thead>
      <tbody>
        {#each members as member (member.id)}
          <tr>
            <td>
              <div class="member-cell">
                <div class="member-avatar">
                  {member.user_id.charAt(0).toUpperCase()}
                </div>
                <div class="member-info">
                  <div class="member-name">User {member.user_id.slice(0, 8)}</div>
                  {#if member.user_id === currentUserId}
                    <span class="member-badge">You</span>
                  {/if}
                </div>
              </div>
            </td>
            <td>
              <span
                class="role-badge"
                style="background: {getRoleBadgeColor(member.role)}15; color: {getRoleBadgeColor(
                  member.role
                )}"
              >
                <Shield class="w-3 h-3" />
                {member.role}
              </span>
            </td>
            <td>
              <span
                class="status-badge"
                style="background: {getStatusBadge(member.status).color}15; color: {getStatusBadge(
                  member.status
                ).color}"
              >
                {getStatusBadge(member.status).label}
              </span>
            </td>
            <td>
              <span class="date-text">
                {member.joined_at
                  ? new Date(member.joined_at).toLocaleDateString()
                  : 'Not joined'}
              </span>
            </td>
            {#if canManage}
              <td class="actions-column">
                {#if canModifyMember(member)}
                  <div class="actions-dropdown">
                    <button
                      class="btn-pill btn-pill-ghost actions-button"
                      onclick={() => toggleDropdown(member.id)}
                      type="button"
                    >
                      <MoreVertical class="w-4 h-4" />
                    </button>
                    {#if dropdownOpen === member.id}
                      <div class="dropdown-menu">
                        <button
                          class="dropdown-item"
                          onclick={() => handleChangeRole(member)}
                          type="button"
                        >
                          <Shield class="w-4 h-4" />
                          Change Role
                        </button>
                        <button
                          class="dropdown-item danger"
                          onclick={() => handleRemoveMember(member)}
                          type="button"
                        >
                          <Trash2 class="w-4 h-4" />
                          Remove Member
                        </button>
                      </div>
                    {/if}
                  </div>
                {:else}
                  <span class="no-actions">-</span>
                {/if}
              </td>
            {/if}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  {#if members.length === 0}
    <div class="empty-state">
      <Users class="w-12 h-12" />
      <p>No members yet</p>
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

{#if showChangeRoleModal && selectedMember}
  <ChangeRoleModal
    {workspaceId}
    member={selectedMember}
    {roles}
    currentUserRole={currentUserRole}
    onsuccess={handleRoleChangeSuccess}
    oncancel={() => {
      showChangeRoleModal = false;
      selectedMember = null;
    }}
  />
{/if}

{#if showDeleteModal && selectedMember}
  <ConfirmDeleteModal
    title="Remove Member"
    message="Are you sure you want to remove this member from the workspace? They will lose access to all projects and data."
    confirmText="Remove Member"
    onconfirm={confirmRemoveMember}
    oncancel={() => {
      showDeleteModal = false;
      selectedMember = null;
    }}
  />
{/if}

<style>
  .members-list {
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

  .members-table-container {
    overflow-x: auto;
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
  }

  .members-table {
    width: 100%;
    border-collapse: collapse;
  }

  .members-table thead {
    background: #f9fafb;
  }

  .members-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.75rem;
    font-weight: 600;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #e5e7eb;
  }

  .members-table td {
    padding: 1rem;
    border-bottom: 1px solid #f3f4f6;
  }

  .members-table tbody tr:last-child td {
    border-bottom: none;
  }

  .members-table tbody tr:hover {
    background: #f9fafb;
  }

  .actions-column {
    width: 80px;
    text-align: center;
  }

  .member-cell {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .member-avatar {
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    font-weight: 600;
    font-size: 0.875rem;
    border-radius: 50%;
  }

  .member-info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .member-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: #111827;
  }

  .member-badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    background: #eff6ff;
    color: #1e40af;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 9999px;
  }

  .role-badge,
  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 9999px;
    text-transform: capitalize;
  }

  .date-text {
    font-size: 0.875rem;
    color: #6b7280;
  }

  .actions-dropdown {
    position: relative;
    display: inline-block;
  }

  .actions-button {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.375rem;
    background: transparent;
    border: 1px solid #e5e7eb;
    border-radius: 0.375rem;
    color: #6b7280;
    cursor: pointer;
    transition: all 0.15s;
  }

  .actions-button:hover {
    background: #f3f4f6;
    color: #111827;
  }

  .dropdown-menu {
    position: absolute;
    right: 0;
    top: calc(100% + 0.25rem);
    min-width: 160px;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 0.375rem;
    box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1);
    z-index: 10;
    padding: 0.25rem;
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.625rem 0.75rem;
    background: transparent;
    border: none;
    border-radius: 0.25rem;
    color: #374151;
    font-size: 0.875rem;
    text-align: left;
    cursor: pointer;
    transition: all 0.15s;
  }

  .dropdown-item:hover {
    background: #f3f4f6;
  }

  .dropdown-item.danger {
    color: #dc2626;
  }

  .dropdown-item.danger:hover {
    background: #fef2f2;
  }

  .no-actions {
    color: #d1d5db;
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

  :global(.dark) .list-header h2 {
    color: #f9fafb;
  }

  :global(.dark) .list-header p {
    color: #9ca3af;
  }

  :global(.dark) .members-table-container {
    border-color: #374151;
  }

  :global(.dark) .members-table thead {
    background: #111827;
  }

  :global(.dark) .members-table th {
    color: #9ca3af;
    border-bottom-color: #374151;
  }

  :global(.dark) .members-table td {
    border-bottom-color: #1f2937;
  }

  :global(.dark) .members-table tbody tr:hover {
    background: #111827;
  }

  :global(.dark) .member-name {
    color: #f9fafb;
  }

  :global(.dark) .actions-button {
    border-color: #374151;
    color: #9ca3af;
  }

  :global(.dark) .actions-button:hover {
    background: #111827;
    color: #f9fafb;
  }

  :global(.dark) .dropdown-menu {
    background: #1f2937;
    border-color: #374151;
  }

  :global(.dark) .dropdown-item {
    color: #d1d5db;
  }

  :global(.dark) .dropdown-item:hover {
    background: #111827;
  }
</style>
