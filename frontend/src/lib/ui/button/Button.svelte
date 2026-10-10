<script lang="ts">
	/**
	 * Button Component - Foundation Design System
	 * Uses MIOSA Foundation button classes (btn-pill / btn-rounded / btn-compact / btn-glass)
	 */
	import { type Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type ButtonVariant = 'primary' | 'secondary' | 'plain' | 'error' | 'success';
	type ButtonSize = 'default' | 'large' | 'extraLarge';
	type ButtonShape = 'pill' | 'rounded' | 'compact' | 'glass';

	interface Props extends Omit<HTMLButtonAttributes, 'disabled' | 'prefix'> {
		variant?: ButtonVariant;
		size?: ButtonSize;
		shape?: ButtonShape;
		loading?: boolean;
		disabled?: boolean;
		block?: boolean;
		withoutHover?: boolean;
		class?: string;
		prefix?: Snippet;
		suffix?: Snippet;
		children?: Snippet;
	}

	let {
		variant = 'secondary',
		size = 'default',
		shape = 'rounded',
		loading = false,
		disabled = false,
		block = false,
		withoutHover = false,
		class: className = '',
		prefix,
		suffix,
		children,
		...restProps
	}: Props = $props();

	const variantMap: Record<ButtonVariant, string> = {
		primary: 'primary',
		secondary: 'secondary',
		plain: 'ghost',
		error: 'danger',
		success: 'success'
	};

	/* Foundation sizes: xs=24 sm=28 default=32 lg=36 xl=40 */
	const sizeMap: Record<ButtonSize, string> = {
		default: 'sm',
		large: '',
		extraLarge: 'xl'
	};

	let buttonClasses = $derived.by(() => {
		const s = `btn-${shape}`;
		const parts = [s, `${s}-${variantMap[variant]}`];
		const sz = sizeMap[size];
		if (sz) parts.push(`${s}-${sz}`);
		if (loading) parts.push(`${s}-loading`);
		if (className) parts.push(className);
		return parts.join(' ');
	});
</script>

<button
	class={buttonClasses}
	disabled={disabled || loading}
	style:width={block ? '100%' : undefined}
	style:display={block ? 'flex' : undefined}
	{...restProps}
>
	{#if !loading && prefix}
		<span class="bos-btn-icon">
			{@render prefix()}
		</span>
	{/if}

	{#if children}
		<span class="bos-btn-label">
			{@render children()}
		</span>
	{/if}

	{#if suffix && !loading}
		<span class="bos-btn-icon">
			{@render suffix()}
		</span>
	{/if}
</button>

<style>
	.bos-btn-icon {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		width: 16px;
		height: 16px;
	}

	.bos-btn-icon :global(svg) {
		width: 100%;
		height: 100%;
		display: block;
	}

	.bos-btn-label {
		text-overflow: ellipsis;
		white-space: nowrap;
		overflow: hidden;
	}
</style>
