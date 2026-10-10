<script lang="ts">
	type QuickFilter = 'my-tasks' | 'all' | 'overdue' | 'today' | 'this-week' | 'blocked' | 'unassigned';

	interface Props {
		activeFilter?: QuickFilter;
		counts?: Partial<Record<QuickFilter, number>>;
		onFilterChange?: (filter: QuickFilter) => void;
	}

	let { activeFilter = 'all', counts = {}, onFilterChange }: Props = $props();

	const filters: { value: QuickFilter; label: string }[] = [
		{ value: 'my-tasks', label: 'My Tasks' },
		{ value: 'all', label: 'All Tasks' },
		{ value: 'overdue', label: 'Overdue' },
		{ value: 'today', label: 'Due Today' },
		{ value: 'this-week', label: 'This Week' },
		{ value: 'blocked', label: 'Blocked' }
	];

	function handleFilterClick(filter: QuickFilter) {
		onFilterChange?.(filter);
	}
</script>

<div class="tqf-bar">
	{#each filters as filter}
		{@const count = counts[filter.value]}
		<button
			onclick={() => handleFilterClick(filter.value)}
			class="tqf-pill {activeFilter === filter.value ? 'tqf-pill--active' : ''}"
		>
			{filter.label}
			{#if count !== undefined && count > 0}
				<span class="tqf-count {activeFilter === filter.value ? 'tqf-count--active' : ''}">{count}</span>
			{/if}
		</button>
	{/each}
</div>

<style>
	.tqf-bar {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.5rem 1.5rem;
		background: var(--dbg, #fff);
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
		overflow-x: auto;
	}
	.tqf-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 500;
		border-radius: 0.375rem;
		border: 1px solid transparent;
		background: transparent;
		color: var(--dt3, #888);
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	.tqf-pill:hover {
		color: var(--dt, #111);
		background: var(--dbg2, #f5f5f5);
	}
	.tqf-pill--active {
		color: rgba(59, 130, 246, 1);
		background: rgba(59, 130, 246, 0.1);
		border-color: rgba(59, 130, 246, 0.2);
		font-weight: 600;
	}
	.tqf-pill--active:hover {
		background: rgba(59, 130, 246, 0.15);
		color: rgba(59, 130, 246, 1);
	}
	:global(.dark) .tqf-pill--active {
		color: rgba(96, 165, 250, 1);
		background: rgba(59, 130, 246, 0.15);
		border-color: rgba(59, 130, 246, 0.25);
	}
	:global(.dark) .tqf-pill--active:hover {
		background: rgba(59, 130, 246, 0.2);
		color: rgba(96, 165, 250, 1);
	}
	.tqf-count {
		font-size: 0.6875rem;
		font-weight: 600;
		padding: 0 0.3rem;
		min-width: 1.125rem;
		text-align: center;
		border-radius: 9999px;
		background: var(--dbg3, #eee);
		color: var(--dt2, #555);
		line-height: 1.4;
	}
	.tqf-count--active {
		background: rgba(59, 130, 246, 0.2);
		color: rgba(59, 130, 246, 1);
	}
	:global(.dark) .tqf-count--active {
		background: rgba(59, 130, 246, 0.25);
		color: rgba(96, 165, 250, 1);
	}
</style>
