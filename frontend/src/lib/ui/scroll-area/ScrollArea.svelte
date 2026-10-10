<script lang="ts">
	import { ScrollArea as ScrollAreaPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	type ScrollbarVisibility = 'auto' | 'always' | 'scroll' | 'hover';

	interface Props {
		orientation?: 'vertical' | 'horizontal' | 'both';
		scrollbarVisibility?: ScrollbarVisibility;
		class?: string;
		viewportClass?: string;
		children: Snippet;
	}

	let {
		orientation = 'vertical',
		scrollbarVisibility = 'hover',
		class: className = '',
		viewportClass = '',
		children
	}: Props = $props();

	const showVertical = $derived(orientation === 'vertical' || orientation === 'both');
	const showHorizontal = $derived(orientation === 'horizontal' || orientation === 'both');
</script>

<ScrollAreaPrimitive.Root class={cn('bos-scroll-area', className)}>
	<ScrollAreaPrimitive.Viewport class={cn('bos-scroll-viewport', viewportClass)}>
		{@render children()}
	</ScrollAreaPrimitive.Viewport>

	{#if showVertical}
		<ScrollAreaPrimitive.Scrollbar
			orientation="vertical"
			class={cn(
				'bos-scrollbar bos-scrollbar--vertical',
				scrollbarVisibility === 'hover' && 'bos-scrollbar--hover',
				scrollbarVisibility === 'auto' && 'bos-scrollbar--auto'
			)}
		>
			<ScrollAreaPrimitive.Thumb class="bos-scrollbar-thumb" />
		</ScrollAreaPrimitive.Scrollbar>
	{/if}

	{#if showHorizontal}
		<ScrollAreaPrimitive.Scrollbar
			orientation="horizontal"
			class={cn(
				'bos-scrollbar bos-scrollbar--horizontal',
				scrollbarVisibility === 'hover' && 'bos-scrollbar--hover',
				scrollbarVisibility === 'auto' && 'bos-scrollbar--auto'
			)}
		>
			<ScrollAreaPrimitive.Thumb class="bos-scrollbar-thumb" />
		</ScrollAreaPrimitive.Scrollbar>
	{/if}

	<ScrollAreaPrimitive.Corner />
</ScrollAreaPrimitive.Root>

<style>
	:global(.bos-scroll-area) {
		position: relative;
		overflow: hidden;
	}

	:global(.bos-scroll-viewport) {
		height: 100%;
		width: 100%;
		border-radius: inherit;
	}

	:global(.bos-scrollbar) {
		display: flex;
		touch-action: none;
		user-select: none;
		transition: opacity 0.15s;
		padding: 1px;
	}

	:global(.bos-scrollbar--vertical) {
		height: 100%;
		width: 10px;
		border-left: 1px solid transparent;
	}

	:global(.bos-scrollbar--horizontal) {
		height: 10px;
		flex-direction: column;
		border-top: 1px solid transparent;
	}

	:global(.bos-scrollbar--hover) {
		opacity: 0;
	}

	:global(.bos-scrollbar--hover:hover),
	:global(.bos-scroll-area:hover .bos-scrollbar--hover) {
		opacity: 1;
	}

	:global(.bos-scrollbar--auto[data-state='hidden']) {
		opacity: 0;
	}

	:global(.bos-scrollbar-thumb) {
		position: relative;
		flex: 1;
		border-radius: 9999px;
		background-color: var(--dbd);
	}
</style>
