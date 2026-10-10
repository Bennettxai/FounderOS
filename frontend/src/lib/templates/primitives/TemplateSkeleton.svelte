<script lang="ts">
	/**
	 * TemplateSkeleton - Loading skeleton shapes for app templates
	 */

	type SkeletonVariant = 'text' | 'circular' | 'rectangular' | 'rounded';

	interface Props {
		variant?: SkeletonVariant;
		width?: string;
		height?: string;
		lines?: number;
		animated?: boolean;
		className?: string;
	}

	let {
		variant = 'text',
		width,
		height,
		lines = 1,
		animated = true,
		className = ''
	}: Props = $props();

	const defaultHeights = {
		text: '1em',
		circular: '40px',
		rectangular: '100px',
		rounded: '40px'
	};
</script>

{#if variant === 'text' && lines > 1}
	<div class="tpl-skeleton-lines" style:width={width}>
		{#each Array(lines) as _, i}
			<div
				class="tpl-skeleton tpl-skeleton-text {className}"
				class:tpl-skeleton-animated={animated}
				style:width={i === lines - 1 ? '60%' : '100%'}
				style:height={height || defaultHeights.text}
			></div>
		{/each}
	</div>
{:else}
	<div
		class="tpl-skeleton tpl-skeleton-{variant} {className}"
		class:tpl-skeleton-animated={animated}
		style:width={width || (variant === 'circular' ? defaultHeights.circular : '100%')}
		style:height={height || defaultHeights[variant]}
	></div>
{/if}

<style>
	.tpl-skeleton {
		background: var(--tpl-bg-tertiary);
	}

	.tpl-skeleton-animated {
		background: linear-gradient(
			90deg,
			var(--tpl-bg-tertiary) 25%,
			var(--tpl-bg-secondary) 50%,
			var(--tpl-bg-tertiary) 75%
		);
		background-size: 200% 100%;
		animation: tpl-shimmer 1.5s ease-in-out infinite;
	}

	/* Variants */
	.tpl-skeleton-text {
		border-radius: var(--tpl-radius-sm);
	}

	.tpl-skeleton-circular {
		border-radius: var(--tpl-radius-full);
	}

	.tpl-skeleton-rectangular {
		border-radius: 0;
	}

	.tpl-skeleton-rounded {
		border-radius: var(--tpl-radius-lg);
	}

	/* Lines container */
	.tpl-skeleton-lines {
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-2);
	}
</style>
