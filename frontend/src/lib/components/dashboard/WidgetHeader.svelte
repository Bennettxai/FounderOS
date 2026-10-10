<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		/** Widget title text */
		title: string;
		/** Icon snippet to render inside the icon circle */
		icon?: Snippet;
		/** Icon color variant (kept for API compatibility, no longer renders a background circle) */
		iconColor?: 'gray' | 'blue' | 'purple' | 'green' | 'cyan' | 'orange';
		/** Action button label (e.g., "View All", "Edit") */
		actionLabel?: string;
		/** Action button click handler */
		onAction?: () => void;
		/** Show chevron arrow after action label */
		showArrow?: boolean;
	}

	let {
		title,
		icon,
		iconColor = 'gray',
		actionLabel,
		onAction,
		showArrow = true
	}: Props = $props();
</script>

<div class="widget-header">
	<div class="widget-header-left">
		{#if icon}
			<span class="widget-icon-inline">
				{@render icon()}
			</span>
		{/if}
		<h2 class="widget-title">{title}</h2>
	</div>

	{#if actionLabel && onAction}
		<button class="widget-action" onclick={onAction}>
			{actionLabel}
			{#if showArrow}
				<svg class="widget-action-arrow" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			{/if}
		</button>
	{/if}
</div>

<style>
	.widget-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.widget-header-left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.widget-icon-inline {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.widget-icon-inline :global(svg) {
		width: 1.1rem;
		height: 1.1rem;
		color: var(--bos-text-secondary, var(--dt2));
	}

	.widget-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--bos-text-primary);
		line-height: 1.25;
	}

	.widget-action {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border-radius: 0.375rem;
		font-size: 0.75rem;
		color: var(--bos-text-tertiary);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: color 0.15s, background 0.15s;
	}

	.widget-action:hover {
		color: var(--bos-text-primary);
		background: var(--bos-hover);
	}

	.widget-action-arrow {
		width: 0.75rem;
		height: 0.75rem;
	}
</style>
