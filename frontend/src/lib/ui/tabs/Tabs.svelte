<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		value?: string;
		onValueChange?: (value: string) => void;
		class?: string;
		children: Snippet;
	}

	let {
		value = $bindable(''),
		onValueChange,
		class: className = '',
		children
	}: Props = $props();

	function handleValueChange(newValue: string) {
		value = newValue;
		onValueChange?.(newValue);
	}
</script>

<TabsPrimitive.Root bind:value onValueChange={handleValueChange} class={cn('', className)}>
	{@render children()}
</TabsPrimitive.Root>
