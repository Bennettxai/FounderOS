<script lang="ts">
	/**
	 * Popover Component - BusinessOS Style
	 * Modern document-centric popover patterns
	 */
	import { Popover as PopoverPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';

	type PopoverSide = 'top' | 'right' | 'bottom' | 'left';

	interface Props {
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		side?: PopoverSide;
		align?: 'start' | 'center' | 'end';
		sideOffset?: number;
		class?: string;
		trigger: Snippet;
		children: Snippet;
	}

	let {
		open = $bindable(false),
		onOpenChange,
		side = 'bottom',
		align = 'center',
		sideOffset = 4,
		class: className = '',
		trigger,
		children
	}: Props = $props();

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange?.(value);
	}
</script>

<PopoverPrimitive.Root bind:open onOpenChange={handleOpenChange}>
	<PopoverPrimitive.Trigger>
		{#snippet child({ props })}
			<span {...props}>
				{@render trigger()}
			</span>
		{/snippet}
	</PopoverPrimitive.Trigger>

	<PopoverPrimitive.Portal>
		<PopoverPrimitive.Content
			{side}
			{align}
			{sideOffset}
			class="bos-popover {className}"
		>
			{@render children()}
		</PopoverPrimitive.Content>
	</PopoverPrimitive.Portal>
</PopoverPrimitive.Root>

<style>
	:global(.bos-popover) {
		z-index: var(--bos-z-index-popover, 1001);
		min-width: 200px;
		padding: 16px;
		background-color: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 12px;
		box-shadow: 0px 0px 12px rgba(66, 65, 73, 0.14), 0px 0px 0px 0.5px rgba(0, 0, 0, 0.1);
		outline: none;
	}

	/* Animation */
	:global(.bos-popover[data-state='open']) {
		animation: popover-in 0.15s ease-out;
	}

	:global(.bos-popover[data-state='closed']) {
		animation: popover-out 0.1s ease-in;
	}

	@keyframes popover-in {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	@keyframes popover-out {
		from {
			opacity: 1;
			transform: scale(1);
		}
		to {
			opacity: 0;
			transform: scale(0.95);
		}
	}

	/* Dark mode */
	:global(.dark .bos-popover) {
		background-color: var(--dbg2);
		border-color: var(--dbd);
		box-shadow: 0px 0px 12px rgba(0, 0, 0, 0.5), 0px 0px 0px 0.5px rgba(255, 255, 255, 0.1);
	}
</style>
