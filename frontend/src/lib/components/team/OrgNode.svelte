<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface Props {
		id: string;
		name: string;
		role: string;
		avatar?: string;
		status: Status;
		depth?: number;
		onClick?: () => void;
	}

	let { id, name, role, avatar, status, depth = 0, onClick }: Props = $props();

	function getInitials(name: string) {
		return name
			.split(' ')
			.map(n => n.charAt(0))
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}
</script>

<div class="td-org-node-wrap">
	<button
		onclick={onClick}
		class="td-org-node {depth === 0 ? 'td-org-node--root' : ''}"
		aria-label="View {name}'s profile"
	>
		{#if avatar}
			<img src={avatar} alt={name} class="td-avatar td-avatar--md" style="object-fit: cover" />
		{:else}
			<div class="td-avatar td-avatar--md" style="background: var(--bos-avatar-default)">{getInitials(name)}</div>
		{/if}
		<div class="td-org-node__info">
			<span class="td-org-node__name">{name}</span>
			<span class="td-org-node__role">{role}</span>
			<StatusBadge {status} size="sm" />
		</div>
	</button>
</div>

<style>
	.td-org-node {
		display: inline-flex;
		align-items: center;
		gap: 9px;
		padding: 9px 12px;
		border-radius: 11px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg, #fff);
		white-space: nowrap;
		cursor: pointer;
		transition: border-color 0.13s, box-shadow 0.13s;
	}
	.td-org-node:hover {
		border-color: var(--dbd2, #f0f0f0);
		box-shadow: 0 2px 10px rgba(0,0,0,0.06);
	}
	.td-org-node--root {
		border-color: var(--bos-avatar-default);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--bos-avatar-default) 12%, transparent);
	}
	.td-org-node__info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		text-align: left;
	}
	.td-org-node__name {
		font-size: 12px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.01em;
	}
	.td-org-node__role {
		font-size: 10px;
		color: var(--dt3, #888);
		font-weight: 500;
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
	.td-org-node-wrap {
		display: flex;
		flex-direction: column;
		align-items: center;
	}
</style>
