<script lang="ts">
	import { fly } from 'svelte/transition';
	import StatusBadge from './StatusBadge.svelte';
	import CapacityBar from './CapacityBar.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface Props {
		id: string;
		name: string;
		role: string;
		email?: string;
		avatar?: string;
		status: Status;
		activeProjects: number;
		openTasks: number;
		capacity: number;
		onClick?: () => void;
	}

	let {
		id,
		name,
		role,
		email,
		avatar,
		status,
		activeProjects,
		openTasks,
		capacity,
		onClick
	}: Props = $props();

	function getInitials(name: string) {
		return name
			.split(' ')
			.map(n => n.charAt(0))
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}
</script>

<div
	class="td-member-card"
	role="button"
	tabindex="0"
	onkeydown={(e) => e.key === 'Enter' && onClick?.()}
>
	<!-- Avatar -->
	<div class="td-member-card__avatar-wrap">
		{#if avatar}
			<img src={avatar} alt={name} class="td-avatar td-avatar--lg" style="object-fit: cover" />
		{:else}
			<div class="td-avatar td-avatar--lg" style="background: var(--bos-avatar-default)">{getInitials(name)}</div>
		{/if}
		<span class="td-status-dot td-status-dot--{status}"></span>
	</div>

	<div class="td-member-card__info">
		<span class="td-member-card__name">{name}</span>
		<span class="td-member-card__role">{role}</span>
	</div>

	<!-- Stats -->
	<div class="td-member-card__stats">
		<div class="td-member-card__stat">
			<span class="td-member-card__stat-label">Projects</span>
			<span class="td-member-card__stat-value">{activeProjects}</span>
		</div>
		<div class="td-member-card__stat">
			<span class="td-member-card__stat-label">Tasks</span>
			<span class="td-member-card__stat-value">{openTasks}</span>
		</div>
	</div>

	<!-- Capacity -->
	<div class="td-member-card__capacity">
		<CapacityBar {capacity} size="sm" showPercentage={true} />
	</div>

	<div class="td-member-card__actions">
		<button
			class="btn-pill btn-pill-soft btn-pill-sm"
			onclick={(e) => { e.stopPropagation(); onClick?.(); }}
		>View Profile</button>
	</div>
</div>

<style>
	.td-member-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		padding: 18px 14px 14px;
		border-radius: 14px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg, #fff);
		text-align: center;
		cursor: pointer;
		transition: border-color 0.13s, box-shadow 0.13s;
	}
	.td-member-card:hover {
		border-color: var(--dbd2, #f0f0f0);
		box-shadow: 0 4px 16px rgba(0,0,0,0.07);
	}
	.td-member-card__avatar-wrap {
		position: relative;
		display: inline-block;
	}
	.td-member-card__avatar-wrap .td-status-dot {
		position: absolute;
		bottom: 1px;
		right: 1px;
	}
	.td-member-card__info {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
	}
	.td-member-card__name {
		font-size: 13px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.02em;
		line-height: 1.2;
	}
	.td-member-card__role {
		font-size: 11px;
		color: var(--dt3, #888);
		font-weight: 500;
	}
	.td-member-card__stats {
		display: flex;
		gap: 16px;
		width: 100%;
		justify-content: center;
		padding: 8px 0;
		border-top: 1px solid var(--dbd2, #f0f0f0);
	}
	.td-member-card__stat {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
	}
	.td-member-card__stat-label {
		font-size: 10px;
		color: var(--dt3, #888);
		font-weight: 500;
	}
	.td-member-card__stat-value {
		font-size: 14px;
		font-weight: 700;
		color: var(--dt, #111);
	}
	.td-member-card__capacity {
		width: 100%;
		padding: 0 4px;
	}
	.td-member-card__actions {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
	}
	.td-member-card__actions :global(button) {
		width: 100%;
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
	.td-avatar--lg { width: 44px; height: 44px; font-size: 15px; }
	.td-status-dot {
		width: 9px;
		height: 9px;
		border-radius: 9999px;
		border: 2px solid var(--dbg, #fff);
		display: block;
		flex-shrink: 0;
	}
	.td-status-dot--available { background: #22c55e; }
	.td-status-dot--busy { background: #f59e0b; }
	.td-status-dot--overloaded { background: #ef4444; }
	.td-status-dot--ooo { background: #9ca3af; }
</style>
