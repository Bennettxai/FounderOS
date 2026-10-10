<!--
	GlassCard.svelte
	Glassmorphism card wrapper using Foundation glass surface classes.
	Supports: glass-card, glass-card-frosted, glass-card-subtle, glass-panel, glass-panel-dark, glass-surface.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type Padding = 'none' | 'sm' | 'md' | 'lg';
	type GlassVariant = 'default' | 'frosted' | 'subtle' | 'panel' | 'panel-dark' | 'surface';

	interface Props {
		padding?: Padding;
		variant?: GlassVariant;
		hoverable?: boolean;
		class?: string;
		children?: Snippet;
	}

	let {
		padding = 'md',
		variant = 'default',
		hoverable = false,
		class: className = '',
		children
	}: Props = $props();

	const PADDING_MAP: Record<Padding, string> = {
		none: '',
		sm: 'p-2',
		md: 'p-4',
		lg: 'p-6'
	};

	const VARIANT_MAP: Record<GlassVariant, string> = {
		default: 'glass-card',
		frosted: 'glass-card-frosted',
		subtle: 'glass-card-subtle',
		panel: 'glass-panel',
		'panel-dark': 'glass-panel-dark',
		surface: 'glass-surface'
	};

	let paddingClass = $derived(PADDING_MAP[padding]);
	let variantClass = $derived(VARIANT_MAP[variant]);
</script>

<div
	class="{variantClass} {paddingClass} {className}"
	class:transition-shadow={hoverable}
	class:hover:shadow-lg={hoverable}
>
	{#if children}
		{@render children()}
	{/if}
</div>
