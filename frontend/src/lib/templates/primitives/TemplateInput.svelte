<script lang="ts">
	/**
	 * TemplateInput - Input component for app templates
	 * Aligned with button heights on 4px grid
	 */

	type InputSize = 'sm' | 'md' | 'lg';
	type InputType = 'text' | 'number' | 'email' | 'password' | 'search' | 'tel' | 'url';

	interface Props {
		type?: InputType;
		value?: string | number;
		placeholder?: string;
		size?: InputSize;
		disabled?: boolean;
		readonly?: boolean;
		error?: string;
		prefix?: string;
		suffix?: string;
		name?: string;
		id?: string;
		required?: boolean;
		min?: number;
		max?: number;
		step?: number;
		maxlength?: number;
		oninput?: (e: Event) => void;
		onchange?: (e: Event) => void;
		onblur?: (e: FocusEvent) => void;
		onfocus?: (e: FocusEvent) => void;
	}

	let {
		type = 'text',
		value = $bindable(''),
		placeholder = '',
		size = 'md',
		disabled = false,
		readonly = false,
		error,
		prefix,
		suffix,
		name,
		id,
		required = false,
		min,
		max,
		step,
		maxlength,
		oninput,
		onchange,
		onblur,
		onfocus
	}: Props = $props();
</script>

<div class="tpl-input-wrapper tpl-input-{size}" class:tpl-input-error={error} class:tpl-input-disabled={disabled}>
	{#if prefix}
		<span class="tpl-input-addon tpl-input-prefix">{prefix}</span>
	{/if}
	<input
		{type}
		{name}
		{id}
		{placeholder}
		{disabled}
		{required}
		{min}
		{max}
		{step}
		{maxlength}
		readonly={readonly}
		bind:value
		class="tpl-input"
		class:has-prefix={prefix}
		class:has-suffix={suffix}
		{oninput}
		{onchange}
		{onblur}
		{onfocus}
	/>
	{#if suffix}
		<span class="tpl-input-addon tpl-input-suffix">{suffix}</span>
	{/if}
</div>
{#if error}
	<p class="tpl-input-error-text">{error}</p>
{/if}

<style>
	.tpl-input-wrapper {
		display: flex;
		align-items: stretch;
		background: var(--tpl-input-bg, var(--tpl-bg-secondary));
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		transition: all var(--tpl-transition-fast);
		overflow: hidden;
		box-shadow: var(--tpl-shadow-xs);
	}

	.tpl-input-wrapper:hover:not(.tpl-input-disabled) {
		border-color: var(--tpl-border-hover);
		background: var(--tpl-input-bg-hover, var(--tpl-bg-tertiary));
	}

	.tpl-input-wrapper:focus-within:not(.tpl-input-disabled) {
		border-color: var(--tpl-accent-primary);
		box-shadow: var(--tpl-shadow-focus);
		background: var(--tpl-bg-primary);
	}

	.tpl-input-wrapper.tpl-input-error {
		border-color: var(--tpl-status-error);
		background: var(--tpl-status-error-bg);
	}

	.tpl-input-wrapper.tpl-input-error:focus-within {
		box-shadow: var(--tpl-shadow-focus-error);
		background: var(--tpl-bg-primary);
	}

	.tpl-input-wrapper.tpl-input-disabled {
		background: var(--tpl-bg-tertiary);
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   SIZES (matching button heights with improved padding)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-input-sm {
		height: var(--tpl-size-sm); /* 32px */
	}

	.tpl-input-sm .tpl-input {
		padding: var(--tpl-space-1) var(--tpl-space-3);
		font-size: var(--tpl-text-xs);
	}

	.tpl-input-md {
		height: var(--tpl-size-md); /* 36px */
	}

	.tpl-input-md .tpl-input {
		padding: var(--tpl-space-1-5) var(--tpl-space-3-5);
		font-size: var(--tpl-text-sm);
	}

	.tpl-input-lg {
		height: var(--tpl-size-lg); /* 40px */
	}

	.tpl-input-lg .tpl-input {
		padding: var(--tpl-space-2) var(--tpl-space-4);
		font-size: var(--tpl-text-base);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   INPUT FIELD
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-input {
		flex: 1;
		width: 100%;
		height: 100%;
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		color: var(--tpl-text-primary);
		outline: none;
		min-width: 0;
		letter-spacing: var(--tpl-tracking-normal);
	}

	.tpl-input::placeholder {
		color: var(--tpl-text-muted);
		font-weight: var(--tpl-font-normal);
	}

	.tpl-input:disabled {
		cursor: not-allowed;
		color: var(--tpl-text-muted);
	}

	.tpl-input.has-prefix {
		padding-left: var(--tpl-space-2) !important;
	}

	.tpl-input.has-suffix {
		padding-right: var(--tpl-space-2) !important;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   ADDONS (Prefix/Suffix) - Improved styling
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-input-addon {
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--tpl-text-tertiary);
		background: var(--tpl-bg-tertiary);
		font-size: inherit;
		font-weight: var(--tpl-font-medium);
		white-space: nowrap;
		flex-shrink: 0;
		user-select: none;
	}

	.tpl-input-sm .tpl-input-addon {
		padding: 0 var(--tpl-space-2-5);
		font-size: var(--tpl-text-xs);
		min-width: var(--tpl-space-8);
	}

	.tpl-input-md .tpl-input-addon {
		padding: 0 var(--tpl-space-3);
		font-size: var(--tpl-text-sm);
		min-width: var(--tpl-space-10);
	}

	.tpl-input-lg .tpl-input-addon {
		padding: 0 var(--tpl-space-3-5);
		font-size: var(--tpl-text-base);
		min-width: var(--tpl-space-12);
	}

	.tpl-input-prefix {
		border-right: 1px solid var(--tpl-border-subtle);
	}

	.tpl-input-suffix {
		border-left: 1px solid var(--tpl-border-subtle);
	}

	/* Focus state addon highlighting */
	.tpl-input-wrapper:focus-within .tpl-input-addon {
		color: var(--tpl-text-secondary);
		background: var(--tpl-accent-primary-light);
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   ERROR TEXT
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-input-error-text {
		margin: var(--tpl-space-1-5) 0 0;
		padding-left: var(--tpl-space-1);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-status-error-text);
		line-height: var(--tpl-leading-snug);
	}
</style>
