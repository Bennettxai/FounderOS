<script lang="ts">
	/**
	 * Menu Component - BusinessOS Style
	 * Modern document-centric dropdown menu patterns
	 */
	import { DropdownMenu as MenuPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';

	interface Props {
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		class?: string;
		trigger: Snippet;
		children: Snippet;
	}

	let {
		open = $bindable(false),
		onOpenChange,
		class: className = '',
		trigger,
		children
	}: Props = $props();

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange?.(value);
	}
</script>

<MenuPrimitive.Root bind:open onOpenChange={handleOpenChange}>
	<MenuPrimitive.Trigger>
		{#snippet child({ props })}
			<span {...props}>
				{@render trigger()}
			</span>
		{/snippet}
	</MenuPrimitive.Trigger>

	<MenuPrimitive.Portal>
		<MenuPrimitive.Content
			class="bos-menu {className}"
			sideOffset={4}
		>
			{@render children()}
		</MenuPrimitive.Content>
	</MenuPrimitive.Portal>
</MenuPrimitive.Root>

<style>
	:global(.bos-menu) {
		z-index: var(--bos-z-index-popover, 1001);
		min-width: 180px;
		padding: 8px;
		background-color: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 12px;
		box-shadow: 0px 0px 12px rgba(66, 65, 73, 0.14), 0px 0px 0px 0.5px rgba(0, 0, 0, 0.1);
		outline: none;
	}

	/* Animation */
	:global(.bos-menu[data-state='open']) {
		animation: menu-in 0.15s ease-out;
	}

	:global(.bos-menu[data-state='closed']) {
		animation: menu-out 0.1s ease-in;
	}

	@keyframes menu-in {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	@keyframes menu-out {
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
	:global(.dark .bos-menu) {
		background-color: var(--dbg2);
		border-color: var(--dbd);
		box-shadow: 0px 0px 12px rgba(0, 0, 0, 0.5), 0px 0px 0px 0.5px rgba(255, 255, 255, 0.1);
	}
</style>
