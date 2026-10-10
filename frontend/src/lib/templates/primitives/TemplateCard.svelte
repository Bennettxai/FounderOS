<script lang="ts">
	/**
	 * TemplateCard - Card container component for app templates
	 * Provides header, body, and footer sections
	 */

	type CardVariant = 'default' | 'outlined' | 'elevated' | 'filled';

	interface Props {
		variant?: CardVariant;
		padding?: 'none' | 'sm' | 'md' | 'lg';
		hoverable?: boolean;
		clickable?: boolean;
		selected?: boolean;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		variant = 'default',
		padding = 'md',
		hoverable = false,
		clickable = false,
		selected = false,
		onclick,
		children
	}: Props & { children?: any } = $props();
</script>

{#if clickable}
	<button
		type="button"
		class="tpl-card tpl-card- {variant} tpl-card-pad-{padding}"
		class:tpl-card-hoverable={hoverable || clickable}
		class:tpl-card-selected={selected}
		{onclick}
	>
		{@render children?.()}
	</button>
{:else}
	<div
		class="tpl-card tpl-card-{variant} tpl-card-pad-{padding}"
		class:tpl-card-hoverable={hoverable}
		class:tpl-card-selected={selected}
	>
		{@render children?.()}
	</div>
{/if}

<style>
	.tpl-card {
		display: flex;
		flex-direction: column;
		background: var(--tpl-card-bg);
		border: 1px solid var(--tpl-card-border);
		border-radius: var(--tpl-card-radius);
		font-family: var(--tpl-font-sans);
		text-align: left;
		transition: all var(--tpl-transition-normal);
	}

	button.tpl-card {
		cursor: pointer;
		width: 100%;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   VARIANTS
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-card-default {
		box-shadow: var(--tpl-card-shadow);
	}

	.tpl-card-outlined {
		background: transparent;
		box-shadow: none;
	}

	.tpl-card-elevated {
		box-shadow: var(--tpl-shadow-md);
	}

	.tpl-card-filled {
		background: var(--tpl-bg-secondary);
		border-color: transparent;
		box-shadow: none;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   PADDING
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-card-pad-none {
		padding: 0;
	}

	.tpl-card-pad-sm {
		padding: var(--tpl-space-3);
	}

	.tpl-card-pad-md {
		padding: var(--tpl-space-4);
	}

	.tpl-card-pad-lg {
		padding: var(--tpl-space-6);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   STATES
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-card-hoverable:hover {
		transform: translateY(-2px);
		box-shadow: var(--tpl-card-shadow-hover);
	}

	.tpl-card-selected {
		border-color: var(--tpl-accent-primary);
		box-shadow: 0 0 0 1px var(--tpl-accent-primary);
	}

	button.tpl-card:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SECTIONS (via global classes)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-card :global(.tpl-card-header) {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--tpl-space-3);
		padding-bottom: var(--tpl-space-3);
		border-bottom: 1px solid var(--tpl-border-subtle);
		margin-bottom: var(--tpl-space-3);
	}

	.tpl-card :global(.tpl-card-title) {
		margin: 0;
		font-size: var(--tpl-text-base);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-card :global(.tpl-card-subtitle) {
		margin: var(--tpl-space-1) 0 0;
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-card :global(.tpl-card-body) {
		flex: 1;
	}

	.tpl-card :global(.tpl-card-footer) {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: var(--tpl-space-2);
		padding-top: var(--tpl-space-3);
		border-top: 1px solid var(--tpl-border-subtle);
		margin-top: var(--tpl-space-3);
	}

	/* No padding variant adjustments */
	.tpl-card-pad-none :global(.tpl-card-header) {
		padding: var(--tpl-space-4);
		margin-bottom: 0;
	}

	.tpl-card-pad-none :global(.tpl-card-body) {
		padding: var(--tpl-space-4);
	}

	.tpl-card-pad-none :global(.tpl-card-footer) {
		padding: var(--tpl-space-4);
		margin-top: 0;
	}
</style>
