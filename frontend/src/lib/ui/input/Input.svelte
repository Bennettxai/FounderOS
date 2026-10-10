<script lang="ts">
	/**
	 * Input Component - Foundation Design System
	 * Uses Foundation design tokens for consistent theming.
	 */
	import { type Snippet } from 'svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	type InputStatus = 'default' | 'error' | 'success' | 'warning';
	type InputSize = 'default' | 'large';

	interface Props extends Omit<HTMLInputAttributes, 'size' | 'prefix'> {
		status?: InputStatus;
		size?: InputSize;
		class?: string;
		wrapperClass?: string;
		prefix?: Snippet;
		suffix?: Snippet;
	}

	let {
		status = 'default',
		size = 'default',
		class: className = '',
		wrapperClass = '',
		prefix,
		suffix,
		...restProps
	}: Props = $props();

	const hasAdornment = $derived(prefix || suffix);
</script>

{#if hasAdornment}
	<div class="bos-input-wrapper {wrapperClass}">
		{#if prefix}
			<div class="bos-input__prefix">
				{@render prefix()}
			</div>
		{/if}

		<input
			class="bos-input {className}"
			data-status={status}
			data-size={size}
			data-has-prefix={prefix ? true : undefined}
			data-has-suffix={suffix ? true : undefined}
			{...restProps}
		/>

		{#if suffix}
			<div class="bos-input__suffix">
				{@render suffix()}
			</div>
		{/if}
	</div>
{:else}
	<input
		class="bos-input {className}"
		data-status={status}
		data-size={size}
		{...restProps}
	/>
{/if}

<style>
	.bos-input-wrapper {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
	}

	.bos-input {
		width: 100%;
		height: var(--btn-h-default, 32px);
		padding: 0 12px;
		font-size: var(--btn-font-default, 13px);
		font-family: var(--font-text, -apple-system, BlinkMacSystemFont, sans-serif);
		color: var(--dt, #111);
		background-color: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		outline: none;
		transition: border-color 0.2s, box-shadow 0.2s;
	}

	.bos-input[data-size='large'] {
		height: var(--btn-h-xl, 40px);
		font-size: var(--btn-font-xl, 15px);
	}

	.bos-input::placeholder {
		color: var(--dt4, #bbb);
	}

	.bos-input:focus {
		border-color: var(--color-primary, #111827);
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.06);
	}

	.bos-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Status variants */
	.bos-input[data-status='error'] {
		border-color: var(--color-error, #dc2626);
	}

	.bos-input[data-status='error']:focus {
		box-shadow: 0 0 0 2px rgba(220, 38, 38, 0.1);
	}

	.bos-input[data-status='success'] {
		border-color: var(--color-success, #059669);
	}

	.bos-input[data-status='success']:focus {
		box-shadow: 0 0 0 2px rgba(5, 150, 105, 0.1);
	}

	.bos-input[data-status='warning'] {
		border-color: var(--color-warning, #d97706);
	}

	.bos-input[data-status='warning']:focus {
		box-shadow: 0 0 0 2px rgba(217, 119, 6, 0.1);
	}

	/* Prefix/suffix padding */
	.bos-input[data-has-prefix] {
		padding-left: 36px;
	}

	.bos-input[data-has-suffix] {
		padding-right: 36px;
	}

	.bos-input__prefix,
	.bos-input__suffix {
		position: absolute;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3, #888);
		pointer-events: none;
	}

	.bos-input__prefix {
		left: 12px;
	}

	.bos-input__suffix {
		right: 12px;
	}

	.bos-input__prefix :global(svg),
	.bos-input__suffix :global(svg) {
		width: 16px;
		height: 16px;
	}

	/* Dark mode — Foundation tokens auto-switch in .dark */
	:global(.dark) .bos-input:focus {
		box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.08);
	}
</style>
