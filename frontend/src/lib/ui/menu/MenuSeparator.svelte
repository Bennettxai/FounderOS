<script lang="ts">
	import { DropdownMenu as MenuPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();
</script>

<MenuPrimitive.Separator class={cn('bos-menu-separator', className)} />

<style>
	:global(.bos-menu-separator) {
		margin: 4px -4px;
		height: 1px;
		background-color: var(--dbd2);
	}
</style>
