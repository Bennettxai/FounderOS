<!--
	PillButton.svelte
	Desktop-style pill button with grayscale colors

	Usage:
	<PillButton variant="primary" onclick={handleClick}>
		Sign In
	</PillButton>
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type ButtonVariant = 'primary' | 'secondary' | 'ghost';
	type ButtonSize = 'sm' | 'md' | 'lg';

	interface Props {
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		loading?: boolean;
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
		type = 'button',
		onclick,
		children,
		class: className = '',
		'aria-label': ariaLabel
	}: Props = $props();

	const variantClass = `btn-pill-${variant}`;
	const sizeClass = size !== 'md' ? `btn-pill-${size}` : '';
	const classes = `btn-pill ${variantClass} ${sizeClass} ${className}`.trim();
</script>

<button
	{type}
	class={classes}
	disabled={disabled || loading}
	{onclick}
	aria-label={ariaLabel}
	aria-busy={loading}
>
	{#if loading}
		<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
			<circle
				class="opacity-25"
				cx="12"
				cy="12"
				r="10"
				stroke="currentColor"
				stroke-width="4"
				fill="none"
			/>
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
			/>
		</svg>
	{/if}

	{#if children}
		{@render children()}
	{/if}
</button>
