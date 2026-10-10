<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		value: string;
		class?: string;
		children: Snippet;
	}

	let { value, class: className = '', children }: Props = $props();
</script>

<TabsPrimitive.Content
	{value}
	class={cn('bos-tabs-content', className)}
>
	{@render children()}
</TabsPrimitive.Content>

<style>
	:global(.bos-tabs-content) {
		margin-top: 8px;
		outline: none;
	}

	:global(.bos-tabs-content:focus-visible) {
		outline: 2px solid var(--color-primary, #6366f1);
		outline-offset: 2px;
	}
</style>
