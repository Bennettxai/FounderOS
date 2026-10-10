<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import MemberCard from './MemberCard.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface TeamMember {
		id: string;
		name: string;
		role: string;
		email?: string;
		avatar?: string;
		status: Status;
		activeProjects: number;
		openTasks: number;
		capacity: number;
	}

	interface Props {
		members: TeamMember[];
		onMemberClick?: (memberId: string) => void;
	}

	let { members, onMemberClick }: Props = $props();
</script>

<div class="flex-1 overflow-y-auto p-6">
	{#if members.length === 0}
		<div class="td-empty" in:fade={{ duration: 200 }}>
			<div class="td-empty__icon-circle">
				<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
				</svg>
			</div>
			<h3 class="td-empty__title">No team members yet</h3>
			<p class="td-empty__text">Add your first team member to get started</p>
		</div>
	{:else}
		<div class="td-member-grid">
			{#each members as member, i (member.id)}
				<div in:fly={{ y: 20, duration: 300, delay: Math.min(i * 30, 300) }}>
					<MemberCard
						{...member}
						onClick={() => onMemberClick?.(member.id)}
					/>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.td-member-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 10px;
	}
	@media (max-width: 1280px) { .td-member-grid { grid-template-columns: repeat(3, 1fr); } }
	@media (max-width: 1024px) { .td-member-grid { grid-template-columns: repeat(2, 1fr); } }
	@media (max-width: 640px)  { .td-member-grid { grid-template-columns: 1fr; } }

	/* ── Empty State ── */
	.td-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 0;
	}
	.td-empty__icon-circle {
		width: 4rem;
		height: 4rem;
		border-radius: 50%;
		background: var(--dbg3);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 1rem;
		color: var(--dt4);
	}
	.td-empty__title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0 0 0.25rem;
	}
	.td-empty__text {
		font-size: 0.875rem;
		color: var(--dt3);
		margin: 0;
	}
</style>
