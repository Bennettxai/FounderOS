<script lang="ts">
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Props {
		priority: Priority | null | undefined;
		size?: 'sm' | 'md';
		showLabel?: boolean;
	}

	let { priority, size = 'sm', showLabel = true }: Props = $props();

	const priorityDots: Record<Priority, string> = {
		critical: 'var(--priority-critical)',
		high: 'var(--priority-high)',
		medium: 'var(--priority-medium)',
		low: 'var(--priority-low)'
	};

	const priorityLabels: Record<Priority, string> = {
		critical: 'Critical',
		high: 'High',
		medium: 'Medium',
		low: 'Low'
	};

	const normalizedPriority = $derived(priority && priorityDots[priority] ? priority : 'medium');
	const sizeClasses = size === 'sm' ? 'pb-sm' : 'pb-md';
</script>

<span class="pb-badge {sizeClasses}">
	<span class="pb-dot" style="background: {priorityDots[normalizedPriority]}"></span>
	{#if showLabel}
		{priorityLabels[normalizedPriority]}
	{/if}
</span>

<style>
	.pb-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-weight: 500;
		color: var(--dt2, #555);
		background: var(--dbg2, #f5f5f5);
		border-radius: 9999px;
	}
	.pb-sm {
		padding: 0.125rem 0.5rem;
		font-size: 0.75rem;
	}
	.pb-md {
		padding: 0.25rem 0.625rem;
		font-size: 0.8125rem;
	}
	.pb-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}
</style>
