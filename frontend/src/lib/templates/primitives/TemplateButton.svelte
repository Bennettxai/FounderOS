<script lang="ts">
	/**
	 * TemplateButton - Button component for app templates
	 * Refined sizing on 4px grid with proper visual hierarchy
	 */

	type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'outline';
	type ButtonSize = 'xs' | 'sm' | 'md' | 'lg';

	interface Props {
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		loading?: boolean;
		fullWidth?: boolean;
		iconOnly?: boolean;
		type?: 'button' | 'submit' | 'reset';
		onclick?: (e: MouseEvent) => void;
	}

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		fullWidth = false,
		iconOnly = false,
		type = 'button',
		onclick,
		children
	}: Props & { children?: any } = $props();
</script>

<button
	{type}
	class="tpl-btn tpl-btn- {size} tpl-btn-{variant}"
	class:tpl-btn-loading={loading}
	class:tpl-btn-full={fullWidth}
	class:tpl-btn-icon-only={iconOnly}
	disabled={disabled || loading}
	onclick={onclick}
>
	{#if loading}
		<span class="tpl-btn-spinner"></span>
	{/if}
	<span class="tpl-btn-content" class:tpl-btn-content-hidden={loading}>
		{@render children?.()}
	</span>
</button>

<style>
	.tpl-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--tpl-space-1-5);
		font-family: var(--tpl-font-sans);
		font-weight: var(--tpl-font-medium);
		border-radius: var(--tpl-radius-md);
		border: 1px solid transparent;
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
		position: relative;
		white-space: nowrap;
		user-select: none;
		-webkit-user-select: none;
	}

	.tpl-btn:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
		pointer-events: none;
	}

	.tpl-btn:active:not(:disabled) {
		transform: scale(0.98);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES (on 4px grid)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-btn-xs {
		height: var(--tpl-size-xs); /* 24px */
		padding: 0 var(--tpl-space-2);
		font-size: var(--tpl-text-2xs);
		border-radius: var(--tpl-radius-sm);
	}

	.tpl-btn-sm {
		height: var(--tpl-size-sm); /* 32px */
		padding: 0 var(--tpl-space-3);
		font-size: var(--tpl-text-xs);
	}

	.tpl-btn-md {
		height: var(--tpl-size-md); /* 36px */
		padding: 0 var(--tpl-space-4);
		font-size: var(--tpl-text-sm);
	}

	.tpl-btn-lg {
		height: var(--tpl-size-lg); /* 40px */
		padding: 0 var(--tpl-space-5);
		font-size: var(--tpl-text-base);
	}

	/* Icon-only buttons (square) */
	.tpl-btn-icon-only.tpl-btn-xs {
		width: var(--tpl-size-xs);
		padding: 0;
	}
	.tpl-btn-icon-only.tpl-btn-sm {
		width: var(--tpl-size-sm);
		padding: 0;
	}
	.tpl-btn-icon-only.tpl-btn-md {
		width: var(--tpl-size-md);
		padding: 0;
	}
	.tpl-btn-icon-only.tpl-btn-lg {
		width: var(--tpl-size-lg);
		padding: 0;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   VARIANTS
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-btn-primary {
		background: var(--tpl-accent-primary);
		color: white;
		border-color: var(--tpl-accent-primary);
	}

	.tpl-btn-primary:hover:not(:disabled) {
		background: var(--tpl-accent-primary-hover);
		border-color: var(--tpl-accent-primary-hover);
	}

	.tpl-btn-primary:active:not(:disabled) {
		background: var(--tpl-accent-primary-active);
		border-color: var(--tpl-accent-primary-active);
	}

	.tpl-btn-secondary {
		background: var(--tpl-bg-primary);
		color: var(--tpl-text-primary);
		border-color: var(--tpl-border-default);
	}

	.tpl-btn-secondary:hover:not(:disabled) {
		background: var(--tpl-bg-hover);
		border-color: var(--tpl-border-hover);
	}

	.tpl-btn-outline {
		background: transparent;
		color: var(--tpl-accent-primary);
		border-color: var(--tpl-accent-primary);
	}

	.tpl-btn-outline:hover:not(:disabled) {
		background: var(--tpl-accent-primary-light);
	}

	.tpl-btn-ghost {
		background: transparent;
		color: var(--tpl-text-secondary);
		border-color: transparent;
	}

	.tpl-btn-ghost:hover:not(:disabled) {
		background: var(--tpl-bg-hover);
		color: var(--tpl-text-primary);
	}

	.tpl-btn-danger {
		background: var(--tpl-status-error);
		color: white;
		border-color: var(--tpl-status-error);
	}

	.tpl-btn-danger:hover:not(:disabled) {
		background: var(--tpl-status-error-hover);
		border-color: var(--tpl-status-error-hover);
	}

	.tpl-btn-success {
		background: var(--tpl-status-success);
		color: white;
		border-color: var(--tpl-status-success);
	}

	.tpl-btn-success:hover:not(:disabled) {
		background: var(--tpl-status-success-hover);
		border-color: var(--tpl-status-success-hover);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   STATES
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-btn-full {
		width: 100%;
	}

	.tpl-btn-loading {
		pointer-events: none;
	}

	.tpl-btn-content {
		display: inline-flex;
		align-items: center;
		gap: var(--tpl-space-1-5);
	}

	.tpl-btn-content-hidden {
		visibility: hidden;
	}

	.tpl-btn-spinner {
		position: absolute;
		width: 14px;
		height: 14px;
		border: 2px solid currentColor;
		border-top-color: transparent;
		border-radius: 50%;
		animation: tpl-spin 0.6s linear infinite;
	}

	.tpl-btn-sm .tpl-btn-spinner {
		width: 12px;
		height: 12px;
	}

	.tpl-btn-lg .tpl-btn-spinner {
		width: 16px;
		height: 16px;
	}

	/* Icon sizing within buttons */
	.tpl-btn :global(svg) {
		width: var(--tpl-icon-sm);
		height: var(--tpl-icon-sm);
		flex-shrink: 0;
	}

	.tpl-btn-xs :global(svg) {
		width: var(--tpl-icon-xs);
		height: var(--tpl-icon-xs);
	}

	.tpl-btn-lg :global(svg) {
		width: var(--tpl-icon-md);
		height: var(--tpl-icon-md);
	}

	@keyframes tpl-spin {
		to { transform: rotate(360deg); }
	}
</style>
