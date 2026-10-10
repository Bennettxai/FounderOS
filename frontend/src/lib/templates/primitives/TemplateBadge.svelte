<script lang="ts">
	/**
	 * TemplateBadge - Badge/Tag component for app templates
	 * Refined proportions with proper token usage
	 */

	type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' | 'outline' | 'custom';
	type BadgeSize = 'xs' | 'sm' | 'md';

	interface Props {
		variant?: BadgeVariant;
		size?: BadgeSize;
		color?: string;
		backgroundColor?: string;
		dot?: boolean;
		removable?: boolean;
		onremove?: () => void;
	}

	let {
		variant = 'default',
		size = 'sm',
		color,
		backgroundColor,
		dot = false,
		removable = false,
		onremove,
		children
	}: Props & { children?: any } = $props();

	const customStyle = $derived(
		variant === 'custom' && (color || backgroundColor)
			? `color: ${color || 'inherit'}; background-color: ${backgroundColor || 'transparent'}`
			: ''
	);
</script>

<span
	class="tpl-badge tpl-badge-{size} tpl-badge-{variant}"
	style={customStyle}
>
	{#if dot}
		<span class="tpl-badge-dot"></span>
	{/if}
	<span class="tpl-badge-text">
		{@render children?.()}
	</span>
	{#if removable}
		<button type="button" class="tpl-badge-remove" onclick={onremove} aria-label="Remove">
			<svg viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M3 3l6 6M9 3l-6 6" />
			</svg>
		</button>
	{/if}
</span>

<style>
	.tpl-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-weight: var(--tpl-font-medium);
		border-radius: var(--tpl-radius-full);
		white-space: nowrap;
		transition: all var(--tpl-transition-fast);
		line-height: 1;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES (proportional scaling)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-badge-xs {
		height: 18px;
		padding: 0 var(--tpl-space-1-5);
		font-size: var(--tpl-text-2xs);
	}

	.tpl-badge-sm {
		height: 22px;
		padding: 0 var(--tpl-space-2);
		font-size: 11px;
	}

	.tpl-badge-md {
		height: 26px;
		padding: 0 var(--tpl-space-2-5);
		font-size: var(--tpl-text-xs);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   VARIANTS
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-badge-default {
		background: var(--tpl-bg-tertiary);
		color: var(--tpl-text-secondary);
	}

	.tpl-badge-success {
		background: var(--tpl-status-success-bg);
		color: var(--tpl-status-success-text);
	}

	.tpl-badge-warning {
		background: var(--tpl-status-warning-bg);
		color: var(--tpl-status-warning-text);
	}

	.tpl-badge-error {
		background: var(--tpl-status-error-bg);
		color: var(--tpl-status-error-text);
	}

	.tpl-badge-info {
		background: var(--tpl-status-info-bg);
		color: var(--tpl-status-info-text);
	}

	.tpl-badge-outline {
		background: transparent;
		color: var(--tpl-text-secondary);
		box-shadow: inset 0 0 0 1px var(--tpl-border-default);
	}

	.tpl-badge-custom {
		/* Uses inline styles */
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   DOT INDICATOR (scales with size)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-badge-dot {
		border-radius: 50%;
		background: currentColor;
		flex-shrink: 0;
		opacity: 0.9;
	}

	.tpl-badge-xs .tpl-badge-dot {
		width: 4px;
		height: 4px;
	}

	.tpl-badge-sm .tpl-badge-dot {
		width: 5px;
		height: 5px;
	}

	.tpl-badge-md .tpl-badge-dot {
		width: 6px;
		height: 6px;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   REMOVE BUTTON
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-badge-remove {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		margin-left: var(--tpl-space-0-5);
		background: transparent;
		border: none;
		border-radius: 50%;
		cursor: pointer;
		opacity: 0.5;
		transition: opacity var(--tpl-transition-fast);
		color: currentColor;
	}

	.tpl-badge-remove:hover {
		opacity: 1;
	}

	.tpl-badge-xs .tpl-badge-remove {
		width: 12px;
		height: 12px;
	}

	.tpl-badge-xs .tpl-badge-remove svg {
		width: 8px;
		height: 8px;
	}

	.tpl-badge-sm .tpl-badge-remove {
		width: 14px;
		height: 14px;
	}

	.tpl-badge-sm .tpl-badge-remove svg {
		width: 9px;
		height: 9px;
	}

	.tpl-badge-md .tpl-badge-remove {
		width: 16px;
		height: 16px;
	}

	.tpl-badge-md .tpl-badge-remove svg {
		width: 10px;
		height: 10px;
	}
</style>
