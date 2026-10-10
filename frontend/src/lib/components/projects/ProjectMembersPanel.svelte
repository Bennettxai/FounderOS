<script lang="ts">
	import { onMount } from 'svelte';
	import type { ProjectMember, ProjectRole } from '$lib/api/projects/types';
	import {
		listProjectMembers,
		addProjectMember,
		updateProjectMemberRole,
		removeProjectMember
	} from '$lib/api/projects/members';
	import MemberCard from './MemberCard.svelte';
	import AddMemberModal from './AddMemberModal.svelte';
	import { Users, UserPlus, Loader2, AlertCircle, Search } from 'lucide-svelte';

	interface Props {
		projectId: string;
		workspaceId: string;
		currentUserId: string;
		userRole?: ProjectRole;
		canInvite?: boolean;
	}

	let { projectId, workspaceId, currentUserId, userRole = 'viewer', canInvite = false }: Props = $props();

	let members = $state<ProjectMember[]>([]);
	let loading = $state(true);
	let error = $state('');
	let addModalOpen = $state(false);
	let searchQuery = $state('');

	// Derived states
	const filteredMembers = $derived(
		members.filter((member) => {
			if (!searchQuery.trim()) return true;
			const query = searchQuery.toLowerCase();
			return (
				member.user_name?.toLowerCase().includes(query) ||
				member.user_email?.toLowerCase().includes(query) ||
				member.user_id.toLowerCase().includes(query) ||
				member.role.toLowerCase().includes(query)
			);
		})
	);

	const membersByRole = $derived(
		filteredMembers.reduce(
			(acc, member) => {
				acc[member.role] = (acc[member.role] || 0) + 1;
				return acc;
			},
			{} as Record<ProjectRole, number>
		)
	);

	const canEditMembers = $derived(userRole === 'lead' || canInvite);
	const canRemoveMembers = $derived(userRole === 'lead');

	onMount(async () => {
		await loadMembers();
	});

	async function loadMembers() {
		loading = true;
		error = '';
		try {
			members = await listProjectMembers(projectId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load members';
			console.error('Failed to load project members:', err);
		} finally {
			loading = false;
		}
	}

	async function handleAddMember(data: { user_id: string; role: ProjectRole; workspace_id: string }) {
		try {
			const newMember = await addProjectMember(projectId, data);
			members = [...members, newMember];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to add member';
			console.error('Failed to add member:', err);
			// Re-throw to show error in modal
			throw err;
		}
	}

	async function handleRoleChange(memberId: string, newRole: ProjectRole) {
		try {
			const updatedMember = await updateProjectMemberRole(projectId, memberId, { role: newRole });
			members = members.map((m) => (m.id === memberId ? updatedMember : m));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update member role';
			console.error('Failed to update member role:', err);
			// Reload to revert optimistic update
			await loadMembers();
		}
	}

	async function handleRemoveMember(memberId: string) {
		try {
			await removeProjectMember(projectId, memberId);
			members = members.filter((m) => m.id !== memberId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to remove member';
			console.error('Failed to remove member:', err);
			// Reload to revert optimistic update
			await loadMembers();
		}
	}

	function clearError() {
		error = '';
	}
</script>

<div class="project-members-panel">
	<!-- Header -->
	<div class="flex items-center justify-between mb-6">
		<div class="flex items-center gap-3">
			<div class="p-2 prm-mp__icon-bg rounded-lg">
				<Users class="w-5 h-5 prm-mp__accent-icon" />
			</div>
			<div>
				<h2 class="prm-mp__title">Project Members</h2>
				<p class="text-sm prm-mp__muted">
					{members.length} {members.length === 1 ? 'member' : 'members'}
				</p>
			</div>
		</div>

		{#if canInvite}
			<button
				onclick={() => (addModalOpen = true)}
				class="btn-pill btn-pill-primary flex items-center gap-2"
			>
				<UserPlus class="w-4 h-4" />
				Add Member
			</button>
		{/if}
	</div>

	<!-- Error Alert -->
	{#if error}
		<div class="mb-4 flex items-start gap-3 p-4 prm-mp__error rounded-lg">
			<AlertCircle class="w-5 h-5 prm-mp__error-icon flex-shrink-0 mt-0.5" />
			<div class="flex-1">
				<p class="text-sm prm-mp__error-text">{error}</p>
			</div>
			<button
				onclick={clearError}
				class="btn-pill btn-pill-ghost btn-pill-icon"
				aria-label="Dismiss error"
			>
				<svg class="w-4 h-4 prm-mp__error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	<!-- Search and Stats -->
	<div class="mb-6 space-y-4">
		<!-- Search -->
		<div class="relative">
			<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 prm-mp__icon" />
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search members by name, email, or role..."
				class="w-full pl-10 pr-4 py-2.5 text-sm prm-mp__search-input"
			/>
		</div>

		<!-- Role Distribution -->
		{#if members.length > 0}
			<div class="flex flex-wrap gap-2">
				{#if membersByRole.lead > 0}
					<div class="px-3 py-1.5 prm-mp__role-badge prm-mp__role-badge--lead rounded-lg text-sm font-medium">
						{membersByRole.lead} Lead{membersByRole.lead > 1 ? 's' : ''}
					</div>
				{/if}
				{#if membersByRole.contributor > 0}
					<div class="px-3 py-1.5 prm-mp__role-badge prm-mp__role-badge--contributor rounded-lg text-sm font-medium">
						{membersByRole.contributor} Contributor{membersByRole.contributor > 1 ? 's' : ''}
					</div>
				{/if}
				{#if membersByRole.reviewer > 0}
					<div class="px-3 py-1.5 prm-mp__role-badge prm-mp__role-badge--reviewer rounded-lg text-sm font-medium">
						{membersByRole.reviewer} Reviewer{membersByRole.reviewer > 1 ? 's' : ''}
					</div>
				{/if}
				{#if membersByRole.viewer > 0}
					<div class="px-3 py-1.5 prm-mp__viewer-badge rounded-lg text-sm font-medium">
						{membersByRole.viewer} Viewer{membersByRole.viewer > 1 ? 's' : ''}
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Loading State -->
	{#if loading}
		<div class="flex flex-col items-center justify-center py-12">
			<Loader2 class="w-8 h-8 prm-mp__accent-icon animate-spin mb-3" />
			<p class="text-sm prm-mp__muted">Loading members...</p>
		</div>

		<!-- Empty State -->
	{:else if members.length === 0}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="p-4 prm-mp__empty-circle rounded-full mb-4">
				<Users class="w-8 h-8 prm-mp__icon" />
			</div>
			<h3 class="prm-mp__title mb-2">No members yet</h3>
			<p class="text-sm prm-mp__muted mb-4 max-w-sm">
				Start collaborating by adding team members to this project.
			</p>
			{#if canInvite}
				<button
					onclick={() => (addModalOpen = true)}
					class="btn-pill btn-pill-primary flex items-center gap-2"
				>
					<UserPlus class="w-4 h-4" />
					Add First Member
				</button>
			{/if}
		</div>

		<!-- Members List -->
	{:else if filteredMembers.length === 0}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="p-4 prm-mp__empty-circle rounded-full mb-4">
				<Search class="w-8 h-8 prm-mp__icon" />
			</div>
			<h3 class="prm-mp__title mb-2">No members found</h3>
			<p class="text-sm prm-mp__muted mb-4">
				Try adjusting your search query to find the member you're looking for.
			</p>
			<button
				onclick={() => (searchQuery = '')}
				class="btn-pill btn-pill-ghost btn-pill-xs"
			>
				Clear search
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredMembers as member (member.id)}
				<MemberCard
					{member}
					canEdit={canEditMembers}
					canRemove={canRemoveMembers}
					{currentUserId}
					onRoleChange={handleRoleChange}
					onRemove={handleRemoveMember}
				/>
			{/each}
		</div>
	{/if}
</div>

<!-- Add Member Modal -->
<AddMemberModal bind:open={addModalOpen} {workspaceId} onAdd={handleAddMember} />

<style>
	.project-members-panel {
		/* Container styling if needed */
	}
	.prm-mp__title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt, #111);
	}
	.prm-mp__muted {
		color: var(--dt2, #555);
	}
	.prm-mp__icon {
		color: var(--dt3, #888);
	}
	.prm-mp__empty-circle {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-mp__search-input {
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		background: var(--dbg, #fff);
		color: var(--dt, #111);
	}
	.prm-mp__viewer-badge {
		background: var(--dbg2, #f5f5f5);
		color: var(--dt2, #555);
	}
	.prm-mp__icon-bg { background: var(--dbg3, #eee); }
	.prm-mp__accent-icon { color: var(--dt2, #555); }
	.prm-mp__error { background: color-mix(in srgb, var(--bos-status-error, #ef4444) 10%, var(--dbg)); border: 1px solid color-mix(in srgb, var(--bos-status-error, #ef4444) 25%, var(--dbd)); }
	.prm-mp__error-icon { color: var(--bos-status-error, #ef4444); }
	.prm-mp__error-text { color: var(--bos-status-error, #ef4444); }
	.prm-mp__search-input:focus { outline: none; box-shadow: 0 0 0 2px var(--dbd, #e0e0e0); border-color: var(--dt, #111); }
	.prm-mp__role-badge { border-radius: 0.5rem; }
	.prm-mp__role-badge--lead { background: var(--dbg3, #eee); color: var(--dt, #111); font-weight: 600; }
	.prm-mp__role-badge--contributor { background: var(--dbg3, #eee); color: var(--dt2, #555); }
	.prm-mp__role-badge--reviewer { background: var(--dbg3, #eee); color: var(--dt2, #555); }
</style>
