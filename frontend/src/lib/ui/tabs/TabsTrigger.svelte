<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		value: string;
		disabled?: boolean;
		class?: string;
		children: Snippet;
	}

	let { value, disabled = false, class: className = '', children }: Props = $props();
</script>

<TabsPrimitive.Trigger
	{value}
	{disabled}
	class={cn('bos-tabs-trigger', className)}
>
	{@render children()}
</TabsPrimitive.Trigger>

<style>
	:global(.bos-tabs-trigger) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		white-space: nowrap;
		border-radius: 6px;
		padding: 4px 12px;
		font-size: 13px;
		font-weight: 500;
		transition: all 0.15s ease;
		outline: none;
		color: var(--dt3);
		background: transparent;
		cursor: pointer;
	}

	:global(.bos-tabs-trigger:focus-visible) {
		outline: 2px solid var(--color-primary, #6366f1);
		outline-offset: 2px;
	}

	:global(.bos-tabs-trigger:disabled) {
		pointer-events: none;
		opacity: 0.5;
	}

	:global(.bos-tabs-trigger[data-state='active']) {
		background-color: var(--dbg);
		color: var(--dt);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	:global(.dark .bos-tabs-trigger[data-state='active']) {
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
	}
</style>
