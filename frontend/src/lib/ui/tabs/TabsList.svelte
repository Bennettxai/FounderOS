<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<TabsPrimitive.List class={cn('bos-tabs-list', className)}>
	{@render children()}
</TabsPrimitive.List>

<style>
	:global(.bos-tabs-list) {
		display: inline-flex;
		height: 36px;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		padding: 4px;
		background-color: var(--dbg2);
		color: var(--dt3);
	}
</style>
