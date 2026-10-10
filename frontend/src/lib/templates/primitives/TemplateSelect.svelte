<script lang="ts">
	/**
	 * TemplateSelect - Dropdown select component for app templates
	 */

	type SelectSize = 'sm' | 'md' | 'lg';

	interface SelectOption {
		value: string;
		label: string;
		disabled?: boolean;
	}

	interface Props {
		options: SelectOption[];
		value?: string;
		placeholder?: string;
		size?: SelectSize;
		disabled?: boolean;
		error?: string;
		name?: string;
		id?: string;
		required?: boolean;
		onchange?: (e: Event) => void;
	}

	let {
		options,
		value = $bindable(''),
		placeholder = 'Select...',
		size = 'md',
		disabled = false,
		error,
		name,
		id,
		required = false,
		onchange
	}: Props = $props();

	const sizeClasses: Record<SelectSize, string> = {
		sm: 'tpl-select-sm',
		md: 'tpl-select-md',
		lg: 'tpl-select-lg'
	};
</script>

<div class="tpl-select-wrapper {sizeClasses[size]}" class:tpl-select-error={error}>
	<select
		{name}
		{id}
		{disabled}
		{required}
		bind:value
		class="tpl-select"
		{onchange}
	>
		{#if placeholder}
			<option value="" disabled selected={!value}>{placeholder}</option>
		{/if}
		{#each options as option}
			<option value={option.value} disabled={option.disabled}>
				{option.label}
			</option>
		{/each}
	</select>
	<span class="tpl-select-icon">
		<svg viewBox="0 0 20 20" fill="currentColor">
			<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
		</svg>
	</span>
</div>
{#if error}
	<p class="tpl-select-error-text">{error}</p>
{/if}

<style>
	.tpl-select-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}

	.tpl-select {
		width: 100%;
		appearance: none;
		background: var(--tpl-bg-secondary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		font-family: var(--tpl-font-sans);
		color: var(--tpl-text-primary);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
		padding-right: 40px;
		box-shadow: var(--tpl-shadow-xs);
	}

	.tpl-select:hover:not(:disabled) {
		border-color: var(--tpl-border-hover);
		background: var(--tpl-bg-tertiary);
	}

	.tpl-select:focus {
		outline: none;
		border-color: var(--tpl-accent-primary);
		box-shadow: var(--tpl-shadow-focus);
		background: var(--tpl-bg-primary);
	}

	.tpl-select:disabled {
		background: var(--tpl-bg-tertiary);
		color: var(--tpl-text-muted);
		cursor: not-allowed;
		opacity: 0.5;
	}

	.tpl-select-wrapper.tpl-select-error .tpl-select {
		border-color: var(--tpl-status-error);
		background: var(--tpl-status-error-bg);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES (aligned with buttons on 4px grid, improved padding)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-select-sm .tpl-select {
		height: var(--tpl-size-sm); /* 32px */
		padding: var(--tpl-space-1) var(--tpl-space-3);
		padding-right: 36px;
		font-size: var(--tpl-text-xs);
	}

	.tpl-select-md .tpl-select {
		height: var(--tpl-size-md); /* 36px */
		padding: var(--tpl-space-1-5) var(--tpl-space-3-5);
		padding-right: 40px;
		font-size: var(--tpl-text-sm);
	}

	.tpl-select-lg .tpl-select {
		height: var(--tpl-size-lg); /* 40px */
		padding: var(--tpl-space-2) var(--tpl-space-4);
		padding-right: 44px;
		font-size: var(--tpl-text-base);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   ICON (scales with size, improved positioning)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-select-icon {
		position: absolute;
		right: var(--tpl-space-3);
		pointer-events: none;
		color: var(--tpl-text-tertiary);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: color var(--tpl-transition-fast), transform var(--tpl-transition-fast);
	}

	.tpl-select-wrapper:hover .tpl-select-icon {
		color: var(--tpl-text-secondary);
	}

	.tpl-select:focus + .tpl-select-icon {
		color: var(--tpl-accent-primary);
		transform: translateY(1px);
	}

	.tpl-select-sm .tpl-select-icon {
		right: var(--tpl-space-2-5);
	}

	.tpl-select-sm .tpl-select-icon svg {
		width: var(--tpl-icon-xs);
		height: var(--tpl-icon-xs);
	}

	.tpl-select-icon svg {
		width: var(--tpl-icon-sm);
		height: var(--tpl-icon-sm);
	}

	.tpl-select-lg .tpl-select-icon {
		right: var(--tpl-space-3-5);
	}

	.tpl-select-lg .tpl-select-icon svg {
		width: var(--tpl-icon-md);
		height: var(--tpl-icon-md);
	}

	.tpl-select-wrapper.tpl-select-error .tpl-select:focus {
		box-shadow: var(--tpl-shadow-focus-error);
		background: var(--tpl-bg-primary);
	}

	.tpl-select-wrapper.tpl-select-error .tpl-select-icon {
		color: var(--tpl-status-error);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   ERROR TEXT
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-select-error-text {
		margin: var(--tpl-space-1-5) 0 0;
		padding-left: var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-status-error-text);
		line-height: var(--tpl-leading-snug);
	}
</style>
