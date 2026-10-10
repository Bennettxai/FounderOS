<script lang="ts">
	interface Props {
		title: string;
		count: number;
		color?: string;
		collapsed?: boolean;
		showAddButton?: boolean;
		onToggle?: () => void;
		onAdd?: () => void;
	}

	let {
		title,
		count,
		color = 'var(--status-todo)',
		collapsed = false,
		showAddButton = true,
		onToggle,
		onAdd
	}: Props = $props();
</script>

<div class="tb-group flex items-center justify-between py-2.5 px-4 sticky top-0 z-10">
	<button
		onclick={onToggle}
		class="flex items-center gap-2.5"
		aria-label="Toggle {title} section"
	>
		<svg
			class="w-3.5 h-3.5 tb-group-chevron transition-transform duration-200 {collapsed ? '-rotate-90' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
		<span class="tb-group-pill">
			<span class="tb-group-dot" style="background: {color}"></span>
			{title}
		</span>
		<span class="tb-group-count">{count}</span>
	</button>

	{#if showAddButton && !collapsed}
		<button
			onclick={onAdd}
			class="tb-group-add flex items-center gap-1"
			aria-label="Add task to {title}"
		>
			<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			Add
		</button>
	{/if}
</div>

<style>
	.tb-group {
		background: var(--dbg, #fff);
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.tb-group-chevron {
		color: var(--dt4, #bbb);
	}
	.tb-group-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 3px 10px;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		font-weight: 600;
		line-height: 1.4;
		color: var(--dt, #111);
		background: var(--dbg3, #eee);
	}
	.tb-group-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.tb-group-count {
		font-size: 0.75rem;
		color: var(--dt3, #888);
		font-weight: 400;
	}
	.tb-group-add {
		font-size: 0.8125rem;
		color: var(--dt3, #888);
		transition: color 200ms ease;
	}
	.tb-group-add:hover {
		color: var(--dt, #111);
	}
</style>
