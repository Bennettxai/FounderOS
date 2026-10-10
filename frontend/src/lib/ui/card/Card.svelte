<script lang="ts">
	/**
	 * Card Component - Glass card with hover effects
	 * Extracted from BusinessOS2 GlassCard + AppCard patterns
	 *
	 * Usage:
	 *   <Card>Content here</Card>
	 *   <Card padding="lg" hoverable>Interactive card</Card>
	 *   <Card variant="glass" onclick={handleClick}>Glass card</Card>
	 */
	import { type Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type CardVariant = 'default' | 'glass' | 'outline' | 'elevated';
	type CardPadding = 'none' | 'sm' | 'md' | 'lg' | 'xl';

	interface Props extends Omit<HTMLAttributes<HTMLDivElement>, 'class'> {
		variant?: CardVariant;
		padding?: CardPadding;
		hoverable?: boolean;
		class?: string;
		children?: Snippet;
	}

	let {
		variant = 'default',
		padding = 'md',
		hoverable = false,
		class: className = '',
		children,
		...restProps
	}: Props = $props();

	const paddingClasses: Record<CardPadding, string> = {
		none: '',
		sm: 'p-3',
		md: 'p-6',
		lg: 'p-8',
		xl: 'p-12'
	};
</script>

<div
	class="card {paddingClasses[padding]} {className}"
	class:hoverable
	data-variant={variant}
	role={restProps.onclick ? 'button' : undefined}
	tabindex={restProps.onclick ? 0 : undefined}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</div>

<style>
	.card {
		border-radius: 0.75rem;
		transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	}

	/* Default - solid white */
	.card[data-variant='default'] {
		background: white;
		border: 1px solid #e5e7eb;
	}

	/* Glass - glassmorphic */
	.card[data-variant='glass'] {
		background: rgba(255, 255, 255, 0.7);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border: 1px solid rgba(255, 255, 255, 0.3);
	}

	/* Outline - transparent with border */
	.card[data-variant='outline'] {
		background: transparent;
		border: 1px solid #e5e7eb;
	}

	/* Elevated - with shadow */
	.card[data-variant='elevated'] {
		background: white;
		border: 1px solid #f3f4f6;
		box-shadow:
			0 1px 3px 0 rgba(0, 0, 0, 0.1),
			0 1px 2px -1px rgba(0, 0, 0, 0.1);
	}

	/* Hover effect */
	.card.hoverable {
		cursor: pointer;
	}

	.card.hoverable:hover {
		transform: translateY(-2px);
		box-shadow:
			0 12px 40px 0 rgba(31, 38, 135, 0.12),
			0 4px 12px 0 rgba(0, 0, 0, 0.05);
	}

	/* Dark mode */
	:global(.dark) .card[data-variant='default'] {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .card[data-variant='glass'] {
		background: rgba(44, 44, 46, 0.7);
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .card[data-variant='outline'] {
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .card[data-variant='elevated'] {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.08);
		box-shadow:
			0 1px 3px 0 rgba(0, 0, 0, 0.3),
			0 1px 2px -1px rgba(0, 0, 0, 0.2);
	}

	:global(.dark) .card.hoverable:hover {
		box-shadow:
			0 12px 40px 0 rgba(0, 0, 0, 0.4),
			0 4px 12px 0 rgba(0, 0, 0, 0.2);
	}
</style>
