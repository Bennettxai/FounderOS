<!--
	PillButton.svelte
	Desktop-style pill button using Foundation btn-pill classes.
	Supports all Foundation pill variants and sizes.

	Usage:
	<PillButton variant="primary" onclick={handleClick}>
		Sign In
	</PillButton>
	<PillButton variant="danger" size="icon" onclick={handleStop}>
		<StopIcon />
	</PillButton>
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type ButtonVariant =
		| 'primary'
		| 'cta'
		| 'secondary'
		| 'ghost'
		| 'danger'
		| 'success'
		| 'warning'
		| 'outline'
		| 'soft'
		| 'link';
	type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'icon';

	interface Props {
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		loading?: boolean;
		block?: boolean;
		type?: 'button' | 'submit' | 'reset';
		onclick?: (e: MouseEvent) => void;
		children?: Snippet;
		class?: string;
		'aria-label'?: string;
	}

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		block = false,
		type = 'button',
		onclick,
		children,
		class: className = '',
		'aria-label': ariaLabel
	}: Props = $props();

	let classes = $derived.by(() => {
		const parts = ['btn-pill', `btn-pill-${variant}`];
		if (size === 'icon') parts.push('btn-pill-icon');
		else if (size !== 'md') parts.push(`btn-pill-${size}`);
		if (loading) parts.push('btn-pill-loading');
		if (block) parts.push('btn-pill-block');
		if (className) parts.push(className);
		return parts.join(' ');
	});
</script>

<button
	{type}
	class={classes}
	disabled={disabled || loading}
	{onclick}
	aria-label={ariaLabel}
	aria-busy={loading}
>
	{#if children}
		{@render children()}
	{/if}
</button>
