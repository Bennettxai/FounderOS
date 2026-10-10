<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import CapacityBar from './CapacityBar.svelte';
	import StatusBadge from './StatusBadge.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface TeamMember {
		id: string;
		name: string;
		role: string;
		avatar?: string;
		status: Status;
		capacity: number;
		projects: string[];
		activeProjects?: number;
	}

	interface Props {
		members: TeamMember[];
		onMemberClick?: (memberId: string) => void;
	}

	let { members, onMemberClick }: Props = $props();

	const summary = $derived(() => {
		const overloaded = members.filter(m => m.capacity >= 90).length;
		const atCapacity = members.filter(m => m.capacity >= 70 && m.capacity < 90).length;
		const available = members.filter(m => m.capacity < 70).length;
		return { overloaded, atCapacity, available };
	});

	// Get current week range
	const weekRange = $derived(() => {
		const now = new Date();
		const startOfWeek = new Date(now);
		startOfWeek.setDate(now.getDate() - now.getDay());
		const endOfWeek = new Date(startOfWeek);
		endOfWeek.setDate(startOfWeek.getDate() + 6);

		const formatDate = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		return `Week of ${formatDate(startOfWeek)}-${formatDate(endOfWeek).split(' ')[1]}`;
	});

	function getInitials(name: string) {
		return name
			.split(' ')
			.map(n => n.charAt(0))
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}
</script>

<div class="flex-1 overflow-y-auto p-6">
	<!-- Header -->
	<div class="flex items-center justify-between mb-6">
		<div>
			<h2 class="text-lg font-semibold text-gray-900">Team Capacity Overview</h2>
			<p class="text-sm text-gray-500">{weekRange()}</p>
		</div>
	</div>

	{#if members.length === 0}
		<div class="flex flex-col items-center justify-center py-16" in:fade={{ duration: 200 }}>
			<div class="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
				<svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
				</svg>
			</div>
			<h3 class="text-lg font-medium text-gray-900 mb-1">No capacity data</h3>
			<p class="text-gray-500">Add team members to see capacity overview</p>
		</div>
	{:else}
		<!-- Capacity List -->
		<div class="td-capacity-list">
			{#each members as member, i (member.id)}
				<button
					onclick={() => onMemberClick?.(member.id)}
					class="td-capacity-item"
					in:fly={{ x: -100, duration: 400, delay: i * 50 }}
					aria-label="View {member.name}'s capacity"
				>
					<div class="td-capacity-item__meta">
						{#if member.avatar}
							<img src={member.avatar} alt={member.name} class="td-avatar td-avatar--md" style="object-fit: cover" />
						{:else}
							<div class="td-avatar td-avatar--md" style="background: var(--bos-avatar-default)">{getInitials(member.name)}</div>
						{/if}
						<div class="td-capacity-item__info">
							<span class="td-capacity-item__name">{member.name}</span>
							<span class="td-capacity-item__role">{member.role}</span>
						</div>
						<span class="td-capacity-item__tasks">
							{#if member.projects.length > 0}
								{member.projects.join(', ')}
							{:else if (member.activeProjects ?? 0) > 0}
								{member.activeProjects} project{member.activeProjects === 1 ? '' : 's'}
							{/if}
						</span>
					</div>
					<CapacityBar capacity={member.capacity} size="md" />
					<div class="td-capacity-item__footer">
						<StatusBadge status={member.status} size="sm" />
						<span class="td-capacity-item__pct {member.capacity >= 90 ? 'td-capacity-item__label--overloaded' : member.capacity >= 70 ? 'td-capacity-item__label--caution' : 'td-capacity-item__label--ok'}">{member.capacity}%</span>
					</div>
				</button>
			{/each}
		</div>

		<!-- Summary -->
		<div class="td-capacity-summary">
			<span class="td-capacity-summary__label">Summary:</span>
			{#if summary().overloaded > 0}
				<span class="td-capacity-item__label--overloaded">{summary().overloaded} overloaded</span>
			{/if}
			{#if summary().atCapacity > 0}
				<span class="td-capacity-item__label--caution">{summary().atCapacity} at capacity</span>
			{/if}
			{#if summary().available > 0}
				<span class="td-capacity-item__label--ok">{summary().available} available</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.td-capacity-list {
		display: flex;
		flex-direction: column;
		gap: 14px;
		max-width: 100%;
	}
	.td-capacity-item {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 12px 14px;
		border-radius: 12px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg, #fff);
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: border-color 0.13s, box-shadow 0.13s;
	}
	.td-capacity-item:hover {
		border-color: var(--dbd2, #f0f0f0);
		box-shadow: 0 2px 10px rgba(0,0,0,0.06);
	}
	.td-capacity-item__meta {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.td-capacity-item__info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}
	.td-capacity-item__name {
		font-size: 12px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.01em;
	}
	.td-capacity-item__role {
		font-size: 10px;
		color: var(--dt3, #888);
		font-weight: 500;
	}
	.td-capacity-item__tasks {
		font-size: 11px;
		color: var(--dt3, #888);
		font-weight: 600;
		white-space: nowrap;
		flex-shrink: 0;
	}
	.td-capacity-item__footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.td-capacity-item__pct {
		font-size: 11px;
		font-weight: 700;
	}
	.td-capacity-item__label--overloaded { color: var(--bos-status-error); }
	.td-capacity-item__label--caution    { color: #f59e0b; }
	.td-capacity-item__label--ok         { color: #22c55e; }
	.td-capacity-summary {
		margin-top: 16px;
		padding: 12px 14px;
		border-radius: 12px;
		background: var(--dbg2, #f5f5f5);
		font-size: 12px;
		color: var(--dt2, #555);
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
	.td-capacity-summary__label {
		font-weight: 700;
	}
	.td-avatar {
		border-radius: 9999px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		color: var(--bos-surface-on-color);
		flex-shrink: 0;
		letter-spacing: -0.02em;
	}
	.td-avatar--md { width: 36px; height: 36px; font-size: 13px; }
</style>
