<script lang="ts">
	import { fly } from 'svelte/transition';

	interface Props {
		capacity: number; // 0-100
		showPercentage?: boolean;
		size?: 'sm' | 'md' | 'lg';
		animated?: boolean;
	}

	let { capacity, showPercentage = true, size = 'md', animated = true }: Props = $props();

	const getColorClass = (value: number) => {
		if (value < 70) return 'td-cap-fill--ok';
		if (value < 90) return 'td-cap-fill--caution';
		return 'td-cap-fill--overloaded';
	};

	const clampedCapacity = $derived(Math.min(100, Math.max(0, capacity)));
	const colorClass = $derived(getColorClass(capacity));
</script>

<div class="td-cap-row">
	<div class="td-cap-track {size === 'lg' ? 'td-cap-track--lg' : size === 'sm' ? 'td-cap-track--sm' : ''}">
		{#if animated}
			<div
				class="td-cap-fill {colorClass}"
				style="width: {clampedCapacity}%"
				in:fly={{ x: -100, duration: 600 }}
			></div>
		{:else}
			<div
				class="td-cap-fill {colorClass}"
				style="width: {clampedCapacity}%"
			></div>
		{/if}
	</div>
	{#if showPercentage}
		<span class="td-cap-label">{clampedCapacity}%</span>
	{/if}
</div>

<style>
	.td-cap-row {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
	}
	.td-cap-track {
		height: 6px;
		border-radius: 9999px;
		background: var(--dbg2, #f5f5f5);
		overflow: hidden;
		flex: 1;
	}
	.td-cap-track--sm { height: 4px; }
	.td-cap-track--lg { height: 10px; }
	.td-cap-fill {
		height: 100%;
		border-radius: 9999px;
		transition: width 0.3s ease;
	}
	.td-cap-fill--ok { background: #22c55e; }
	.td-cap-fill--caution { background: #f59e0b; }
	.td-cap-fill--overloaded { background: #ef4444; }
	.td-cap-label {
		font-size: 11px;
		color: var(--dt3, #888);
		white-space: nowrap;
		font-weight: 600;
		min-width: 32px;
		text-align: right;
	}
</style>
