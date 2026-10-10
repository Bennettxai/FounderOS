<script lang="ts">
	import { DropdownMenu as MenuPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<MenuPrimitive.Group class={cn('', className)}>
	{@render children()}
</MenuPrimitive.Group>
