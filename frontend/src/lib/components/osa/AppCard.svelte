<!--
	AppCard.svelte
	iOS-style app card with circular icon and glassmorphism

	Usage:
	<AppCard
		title="Book Finds"
		description="Discover books..."
		iconUrl="/icon.png"
		usagePercentage={98}
		onclick={handleClick}
	/>
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		iconUrl?: string;
		iconFallback?: string;
		usagePercentage?: number;
		onclick?: (e: MouseEvent) => void;
		class?: string;
		children?: Snippet;
	}

	let {
		title,
		description,
		iconUrl,
		iconFallback,
		usagePercentage,
		onclick,
		class: className = '',
		children
	}: Props = $props();

	const classes = `app-card ${className}`.trim();

	// Get first letter of title for fallback
	const firstLetter = title.charAt(0).toUpperCase();
</script>

<div class={classes} role="button" tabindex="0" {onclick} onkeypress={(e) => e.key === 'Enter' && onclick?.(e as any)}>
	<!-- Usage percentage badge -->
	{#if usagePercentage !== undefined}
		<div class="app-card-usage">
			{usagePercentage}%
		</div>
	{/if}

	<!-- Circular App Icon -->
	<div class="app-card-icon">
		{#if iconUrl}
			<img src={iconUrl} alt={title} />
		{:else}
			<!-- Gradient fallback with first letter -->
			<span class="text-white text-2xl font-bold">{iconFallback || firstLetter}</span>
		{/if}
	</div>

	<!-- App Title -->
	<div class="app-card-title">
		{title}
	</div>

	<!-- App Description -->
	{#if description}
		<div class="app-card-description">
			{description}
		</div>
	{/if}

	<!-- Custom content -->
	{#if children}
		{@render children()}
	{/if}
</div>
